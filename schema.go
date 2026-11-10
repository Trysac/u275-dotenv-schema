package dotenvschema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	maxSchemaBytes = 1 << 20 // 1 MiB
	maxSchemaVars  = 1000
)

// Schema defines the validation rules for a set of environment variables.
type Schema struct {
	Vars         map[string]Var `json:"vars"`
	AllowUnknown bool           `json:"allowUnknown"`
	compiled     map[string]*regexp.Regexp
}

// Var describes the validation rules for a single variable. Type must be one of
// "string", "int", "bool", "enum", "url", or "duration".
type Var struct {
	Type      string   `json:"type"`
	Required  bool     `json:"required"`
	Secret    bool     `json:"secret"`
	Default   *string  `json:"default"`
	Min       *int64   `json:"min"`
	Max       *int64   `json:"max"`
	Values    []string `json:"values"`
	Pattern   string   `json:"pattern"`
	MinLength *int     `json:"minLength"`
	MaxLength *int     `json:"maxLength"`
}

// Issue is a validation problem for a specific variable. Code, Key, and Message
// are suitable for JSON serialization; Line is 0 for issues not tied to a file
// line (e.g. missing required keys).
type Issue struct {
	Code    string `json:"code"`
	Key     string `json:"key"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

// LoadSchema reads a JSON schema from r (at most 1 MiB). It rejects unknown
// JSON fields, enforces a maximum of 1000 variables, validates the schema's
// rules, and verifies that any default value satisfies its own type constraint.
func LoadSchema(r io.Reader) (Schema, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxSchemaBytes+1))
	if err != nil {
		return Schema{}, err
	}
	if len(data) > maxSchemaBytes {
		return Schema{}, fmt.Errorf("%w: schema exceeds 1 MiB", ErrSchema)
	}
	var s Schema
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&s); err != nil {
		return Schema{}, fmt.Errorf("%w: %v", ErrSchema, err)
	}
	if len(s.Vars) > maxSchemaVars {
		return Schema{}, fmt.Errorf("%w: too many variables (max %d)", ErrSchema, maxSchemaVars)
	}
	s.compiled = make(map[string]*regexp.Regexp)
	for key, v := range s.Vars {
		if err := validateVar(key, v, &s); err != nil {
			return Schema{}, err
		}
	}
	return s, nil
}

func validateVar(key string, v Var, s *Schema) error {
	switch v.Type {
	case "string", "int", "bool", "enum", "url", "duration":
	default:
		return fmt.Errorf("%w: variable %q: invalid type %q", ErrSchema, key, v.Type)
	}
	if v.Type == "enum" && len(v.Values) == 0 {
		return fmt.Errorf("%w: variable %q: enum requires at least one value", ErrSchema, key)
	}
	if v.Type == "int" && v.Min != nil && v.Max != nil && *v.Min > *v.Max {
		return fmt.Errorf("%w: variable %q: min > max", ErrSchema, key)
	}
	if v.Pattern != "" {
		re, err := regexp.Compile(v.Pattern)
		if err != nil {
			return fmt.Errorf("%w: variable %q: invalid pattern: %v", ErrSchema, key, err)
		}
		s.compiled[key] = re
	}
	if v.Default != nil {
		issues := validateValue(key, v, *v.Default, 0, s)
		if len(issues) > 0 {
			return fmt.Errorf("%w: variable %q: default %q: %s", ErrSchema, key, *v.Default, issues[0].Message)
		}
	}
	return nil
}

// Validate checks entries against schema and returns a sorted slice of issues.
// Issues are sorted by Line, then Key, then Code. Values are never included in
// issue messages.
func Validate(entries []Entry, s Schema) []Issue {
	var issues []Issue
	firstSeen := make(map[string]bool)
	for _, e := range entries {
		key := e.Key
		if _, ok := s.Vars[key]; !ok {
			if !s.AllowUnknown {
				issues = append(issues, Issue{Code: "E006", Key: key, Line: e.Line, Message: key + " is not in the schema"})
			}
			if firstSeen[key] {
				issues = append(issues, Issue{Code: "E007", Key: key, Line: e.Line, Message: key + " is defined more than once"})
			}
			firstSeen[key] = true
			continue
		}
		v := s.Vars[key]
		if firstSeen[key] {
			issues = append(issues, Issue{Code: "E007", Key: key, Line: e.Line, Message: key + " is defined more than once"})
		}
		firstSeen[key] = true
		issues = append(issues, validateValue(key, v, e.Value, e.Line, &s)...)
	}
	for key, v := range s.Vars {
		if v.Required && v.Default == nil && !firstSeen[key] {
			issues = append(issues, Issue{Code: "E001", Key: key, Line: 0, Message: key + " is required"})
		}
	}
	sortIssues(issues)
	return issues
}

func validateValue(key string, v Var, val string, line int, s *Schema) []Issue {
	var issues []Issue
	switch v.Type {
	case "int":
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			issues = append(issues, Issue{Code: "E002", Key: key, Line: line, Message: key + " must be an int"})
		} else {
			if v.Min != nil && n < *v.Min {
				issues = append(issues, Issue{Code: "E003", Key: key, Line: line, Message: fmt.Sprintf("%s must be >= %d", key, *v.Min)})
			}
			if v.Max != nil && n > *v.Max {
				issues = append(issues, Issue{Code: "E003", Key: key, Line: line, Message: fmt.Sprintf("%s must be <= %d", key, *v.Max)})
			}
		}
	case "bool":
		if val != "true" && val != "false" && val != "1" && val != "0" {
			issues = append(issues, Issue{Code: "E002", Key: key, Line: line, Message: key + " must be a bool"})
		}
	case "url":
		u, err := url.Parse(val)
		if err != nil || u.Scheme == "" || u.Host == "" {
			issues = append(issues, Issue{Code: "E002", Key: key, Line: line, Message: key + " must be a url"})
		}
	case "duration":
		if _, err := time.ParseDuration(val); err != nil {
			issues = append(issues, Issue{Code: "E002", Key: key, Line: line, Message: key + " must be a duration"})
		}
	case "enum":
		ok := false
		for _, allowed := range v.Values {
			if val == allowed {
				ok = true
				break
			}
		}
		if !ok {
			issues = append(issues, Issue{Code: "E005", Key: key, Line: line, Message: fmt.Sprintf("%s must be one of %s", key, strings.Join(v.Values, "|"))})
		}
	case "string":
		if v.MinLength != nil && len(val) < *v.MinLength {
			issues = append(issues, Issue{Code: "E003", Key: key, Line: line, Message: fmt.Sprintf("%s must be >= %d", key, *v.MinLength)})
		}
		if v.MaxLength != nil && len(val) > *v.MaxLength {
			issues = append(issues, Issue{Code: "E003", Key: key, Line: line, Message: fmt.Sprintf("%s must be <= %d", key, *v.MaxLength)})
		}
		if v.Pattern != "" {
			matched := false
			if re, ok := s.compiled[key]; ok {
				matched = re.MatchString(val)
			} else {
				matched, _ = regexp.MatchString(v.Pattern, val)
			}
			if !matched {
				issues = append(issues, Issue{Code: "E004", Key: key, Line: line, Message: key + " does not match the required pattern"})
			}
		}
	default:
		// Unreachable if LoadSchema was used; handle defensively.
		issues = append(issues, Issue{Code: "E002", Key: key, Line: line, Message: key + " must be a valid value"})
	}
	return issues
}

func sortIssues(issues []Issue) {
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Line != issues[j].Line {
			return issues[i].Line < issues[j].Line
		}
		if issues[i].Key != issues[j].Key {
			return issues[i].Key < issues[j].Key
		}
		return issues[i].Code < issues[j].Code
	})
}

// Example generates a .env.example from schema. Variables are sorted by name.
// Each variable is preceded by a comment "# <type>[, required][, secret]" and
// followed by "KEY=<default>" unless the variable is secret or has no default,
// in which case the value is empty.
func Example(s Schema) string {
	keys := make([]string, 0, len(s.Vars))
	for k := range s.Vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		v := s.Vars[k]
		b.WriteString("# ")
		b.WriteString(v.Type)
		if v.Required {
			b.WriteString(", required")
		}
		if v.Secret {
			b.WriteString(", secret")
		}
		b.WriteByte('\n')
		b.WriteString(k)
		b.WriteByte('=')
		if !v.Secret && v.Default != nil {
			b.WriteString(*v.Default)
		}
		b.WriteByte('\n')
	}
	return b.String()
}
