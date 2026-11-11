package dotenvschema

import (
	"io"
	"regexp"
	"strings"
)

const (
	defaultMaxBytes = 1_048_576
	defaultMaxLines = 10_000
	maxValueBytes   = 65_536
)

var keyRegexp = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Entry is a parsed key/value pair from a .env file with its 1-based line number.
type Entry struct {
	Key   string
	Value string
	Line  int
}

// Options controls Parse limits. Zero values use defaults (1 MiB, 10,000 lines).
type Options struct {
	MaxBytes int
	MaxLines int
}

// Parse reads r and returns the parsed entries. The input size is limited by
// opts.MaxBytes (default 1 MiB) and the number of lines by opts.MaxLines
// (default 10,000). Individual values may not exceed 65,536 bytes.
func Parse(r io.Reader, opts Options) ([]Entry, error) {
	maxBytes := opts.MaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
	}
	maxLines := opts.MaxLines
	if maxLines <= 0 {
		maxLines = defaultMaxLines
	}
	data, err := io.ReadAll(io.LimitReader(r, int64(maxBytes)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxBytes {
		return nil, ErrInputTooLarge
	}
	text := strings.TrimPrefix(string(data), "\ufeff")
	lines := strings.Split(text, "\n")

	var entries []Entry
	lineNo := 0
	for i, raw := range lines {
		if i == len(lines)-1 && raw == "" {
			continue
		}
		lineNo++
		if lineNo > maxLines {
			return nil, ErrInputTooLarge
		}
		line := strings.TrimSuffix(raw, "\r")
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		rest := trimmed
		if strings.HasPrefix(rest, "export ") || strings.HasPrefix(rest, "export\t") {
			after := strings.TrimLeft(rest[len("export"):], " \t")
			if after == "" {
				return nil, &SyntaxError{Line: lineNo, Reason: "expected key after export"}
			}
			rest = after
		}
		eqIdx := strings.Index(rest, "=")
		if eqIdx < 0 {
			return nil, &SyntaxError{Line: lineNo, Reason: "missing '='"}
		}
		key := rest[:eqIdx]
		if !keyRegexp.MatchString(key) {
			return nil, &SyntaxError{Line: lineNo, Reason: "invalid key"}
		}
		value, err := parseValue(rest[eqIdx+1:], lineNo)
		if err != nil {
			return nil, err
		}
		if len(value) > maxValueBytes {
			return nil, &SyntaxError{Line: lineNo, Reason: "value exceeds 65536 bytes"}
		}
		entries = append(entries, Entry{Key: key, Value: value, Line: lineNo})
	}
	return entries, nil
}

func parseValue(part string, lineNo int) (string, error) {
	if part == "" {
		return "", nil
	}
	if part[0] == '\'' {
		end := strings.Index(part[1:], "'")
		if end < 0 {
			return "", &SyntaxError{Line: lineNo, Reason: "unterminated single-quoted value"}
		}
		idx := end + 1
		value := part[1:idx]
		rest := strings.TrimLeft(part[idx+1:], " \t")
		if rest != "" && !strings.HasPrefix(rest, "#") {
			return "", &SyntaxError{Line: lineNo, Reason: "unexpected content after closing quote"}
		}
		return value, nil
	}
	if part[0] == '"' {
		var b strings.Builder
		i := 1
		for i < len(part) {
			c := part[i]
			if c == '"' {
				rest := strings.TrimLeft(part[i+1:], " \t")
				if rest != "" && !strings.HasPrefix(rest, "#") {
					return "", &SyntaxError{Line: lineNo, Reason: "unexpected content after closing quote"}
				}
				return b.String(), nil
			}
			if c == '\n' || c == '\r' {
				return "", &SyntaxError{Line: lineNo, Reason: "newline in double-quoted value"}
			}
			if c == '\\' {
				if i+1 >= len(part) {
					return "", &SyntaxError{Line: lineNo, Reason: "unterminated escape"}
				}
				next := part[i+1]
				switch next {
				case 'n':
					b.WriteByte('\n')
				case 't':
					b.WriteByte('\t')
				case '"':
					b.WriteByte('"')
				case '\\':
					b.WriteByte('\\')
				default:
					return "", &SyntaxError{Line: lineNo, Reason: "invalid escape sequence"}
				}
				i += 2
				continue
			}
			b.WriteByte(c)
			i++
		}
		return "", &SyntaxError{Line: lineNo, Reason: "unterminated double-quoted value"}
	}
	for i := 0; i < len(part); i++ {
		if part[i] == '#' && i > 0 && (part[i-1] == ' ' || part[i-1] == '\t') {
			part = part[:i]
			break
		}
	}
	return strings.Trim(part, " \t"), nil
}
