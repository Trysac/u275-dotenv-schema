package dotenvschema_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Trysac/u275-dotenv-schema"
)

func stringPtr(s string) *string { return &s }
func intPtr(i int) *int          { return &i }
func int64Ptr(i int64) *int64    { return &i }

func TestParseAcceptance(t *testing.T) {
	input := "export PORT=8080\nNAME='a b' # c\nMSG=\"line1\\nline2\"\n# comment\nEMPTY=\n"
	entries, err := dotenvschema.Parse(strings.NewReader(input), dotenvschema.Options{})
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	want := []dotenvschema.Entry{
		{Key: "PORT", Value: "8080", Line: 1},
		{Key: "NAME", Value: "a b", Line: 2},
		{Key: "MSG", Value: "line1\nline2", Line: 3},
		{Key: "EMPTY", Value: "", Line: 5},
	}
	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d", len(entries), len(want))
	}
	for i := range want {
		if entries[i] != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, entries[i], want[i])
		}
	}
}

func TestParseCommentsAndQuotes(t *testing.T) {
	input := "URL=http://x/#frag\nA=b #c\nB='x\\ny'\n"
	entries, err := dotenvschema.Parse(strings.NewReader(input), dotenvschema.Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := []dotenvschema.Entry{
		{Key: "URL", Value: "http://x/#frag", Line: 1},
		{Key: "A", Value: "b", Line: 2},
		{Key: "B", Value: "x\\ny", Line: 3},
	}
	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d", len(entries), len(want))
	}
	for i := range want {
		if entries[i] != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, entries[i], want[i])
		}
	}
}

func TestParseEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []dotenvschema.Entry
	}{
		{
			name:  "space before comment marker",
			input: "A= #c\n",
			want:  []dotenvschema.Entry{{Key: "A", Value: "", Line: 1}},
		},
		{
			name:  "tab before comment marker",
			input: "A=\t#c\n",
			want:  []dotenvschema.Entry{{Key: "A", Value: "", Line: 1}},
		},
		{
			name:  "no whitespace before comment marker",
			input: "A=#c\n",
			want:  []dotenvschema.Entry{{Key: "A", Value: "#c", Line: 1}},
		},
		{
			name:  "value containing equals",
			input: "A=b=c\n",
			want:  []dotenvschema.Entry{{Key: "A", Value: "b=c", Line: 1}},
		},
		{
			name:  "export used as key name",
			input: "export=1\n",
			want:  []dotenvschema.Entry{{Key: "export", Value: "1", Line: 1}},
		},
		{
			name:  "CRLF line endings",
			input: "A=1\r\nB=2\r\n",
			want: []dotenvschema.Entry{
				{Key: "A", Value: "1", Line: 1},
				{Key: "B", Value: "2", Line: 2},
			},
		},
		{
			name:  "empty file",
			input: "",
			want:  nil,
		},
		{
			name:  "UTF-8 BOM",
			input: "\ufeffA=1\n",
			want:  []dotenvschema.Entry{{Key: "A", Value: "1", Line: 1}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := dotenvschema.Parse(strings.NewReader(tt.input), dotenvschema.Options{})
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			if len(entries) != len(tt.want) {
				t.Fatalf("got %d entries, want %d: %+v", len(entries), len(tt.want), entries)
			}
			for i := range tt.want {
				if entries[i] != tt.want[i] {
					t.Errorf("entry %d = %+v, want %+v", i, entries[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
		line    int
	}{
		{name: "bad line", input: "BAD LINE\n", wantErr: dotenvschema.ErrSyntax, line: 1},
		{name: "unterminated single quote", input: "A='x\n", wantErr: dotenvschema.ErrSyntax, line: 1},
		{name: "unterminated double quote", input: "A=\"x\n", wantErr: dotenvschema.ErrSyntax, line: 1},
		{name: "invalid escape", input: "A=\"x\\q\"\n", wantErr: dotenvschema.ErrSyntax, line: 1},
		{name: "missing equals", input: "A\n", wantErr: dotenvschema.ErrSyntax, line: 1},
		{name: "invalid key", input: "1A=x\n", wantErr: dotenvschema.ErrSyntax, line: 1},
		{name: "unexpected after quote", input: "A='x' junk\n", wantErr: dotenvschema.ErrSyntax, line: 1},
		{name: "newline in double quote", input: "A=\"x\ny\"\n", wantErr: dotenvschema.ErrSyntax, line: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := dotenvschema.Parse(strings.NewReader(tt.input), dotenvschema.Options{})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Parse error = %v, want %v", err, tt.wantErr)
			}
			var syn *dotenvschema.SyntaxError
			if errors.As(err, &syn) {
				if syn.Line != tt.line {
					t.Errorf("SyntaxError.Line = %d, want %d", syn.Line, tt.line)
				}
			}
		})
	}
}

func TestParseInputTooLarge(t *testing.T) {
	big := strings.Repeat("A=x\n", 60000) // ~300KB, under max but many lines
	_, err := dotenvschema.Parse(strings.NewReader(big), dotenvschema.Options{MaxLines: 100})
	if !errors.Is(err, dotenvschema.ErrInputTooLarge) {
		t.Fatalf("Parse error = %v, want ErrInputTooLarge", err)
	}
	big = strings.Repeat("A", 1_048_577)
	_, err = dotenvschema.Parse(strings.NewReader(big), dotenvschema.Options{})
	if !errors.Is(err, dotenvschema.ErrInputTooLarge) {
		t.Fatalf("Parse error = %v, want ErrInputTooLarge", err)
	}
}

func TestParseValueTooLong(t *testing.T) {
	longVal := strings.Repeat("a", 65_537)
	input := "KEY=" + longVal + "\n"
	_, err := dotenvschema.Parse(strings.NewReader(input), dotenvschema.Options{})
	if !errors.Is(err, dotenvschema.ErrSyntax) {
		t.Fatalf("Parse error = %v, want ErrSyntax", err)
	}
	var syn *dotenvschema.SyntaxError
	if errors.As(err, &syn) && syn.Line != 1 {
		t.Errorf("SyntaxError.Line = %d, want 1", syn.Line)
	}
}

func TestLoadSchemaAcceptance(t *testing.T) {
	valid := `{"vars":{"PORT":{"type":"int","min":1,"max":65535},"NAME":{"type":"string","pattern":"^[a-z]+$"}}}`
	_, err := dotenvschema.LoadSchema(strings.NewReader(valid))
	if err != nil {
		t.Fatalf("LoadSchema error: %v", err)
	}
}

func TestLoadSchemaErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "invalid type", input: `{"vars":{"A":{"type":"float"}}}`},
		{name: "enum no values", input: `{"vars":{"A":{"type":"enum"}}}`},
		{name: "int min>max", input: `{"vars":{"A":{"type":"int","min":10,"max":1}}}`},
		{name: "invalid pattern", input: `{"vars":{"A":{"type":"string","pattern":"["}}}`},
		{name: "default invalid", input: `{"vars":{"A":{"type":"int","default":"abc"}}}`},
		{name: "unknown field", input: `{"vars":{},"foo":1}`},
		{name: "too many vars", input: func() string {
			var sb strings.Builder
			sb.WriteString(`{"vars":{`)
			for i := 0; i < 1001; i++ {
				if i > 0 {
					sb.WriteByte(',')
				}
				sb.WriteString(`"V` + strconv.Itoa(i) + `":{"type":"string"}`)
			}
			sb.WriteString(`}}`)
			return sb.String()
		}()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := dotenvschema.LoadSchema(strings.NewReader(tt.input))
			if !errors.Is(err, dotenvschema.ErrSchema) {
				t.Fatalf("LoadSchema error = %v, want ErrSchema", err)
			}
		})
	}
	// schema exceeds 1 MiB
	big := `{"vars":{` + strings.Repeat(`"A":{"type":"string"},`, 60000) + `}}`
	_, err := dotenvschema.LoadSchema(strings.NewReader(big))
	if !errors.Is(err, dotenvschema.ErrSchema) {
		t.Fatalf("LoadSchema big error = %v, want ErrSchema", err)
	}
}

func TestValidateAcceptance(t *testing.T) {
	schema := dotenvschema.Schema{
		Vars: map[string]dotenvschema.Var{
			"PORT":         {Type: "int", Min: int64Ptr(1), Max: int64Ptr(65535)},
			"DATABASE_URL": {Type: "url", Secret: true},
			"LOG_LEVEL":    {Type: "enum", Values: []string{"debug", "info"}},
			"NAME":         {Type: "string", Pattern: "^[a-z]+$"},
			"OPTIONAL":     {Type: "string", Default: stringPtr("fallback")},
			"REQUIRED":     {Type: "string", Required: true},
		},
	}
	entries := []dotenvschema.Entry{
		{Key: "PORT", Value: "70000", Line: 1},
		{Key: "DATABASE_URL", Value: "postgres//x", Line: 2},
		{Key: "LOG_LEVEL", Value: "trace", Line: 3},
		{Key: "NAME", Value: "ABC", Line: 4},
		{Key: "PORT", Value: "abc", Line: 5},
		{Key: "UNKNOWN", Value: "x", Line: 6},
	}
	issues := dotenvschema.Validate(entries, schema)
	// Check specific issues exist
	wantCodes := []string{"E003", "E002", "E005", "E004", "E007", "E002", "E006", "E001"}
	seen := make(map[string]bool)
	for _, iss := range issues {
		seen[iss.Code] = true
	}
	for _, code := range wantCodes {
		if !seen[code] {
			t.Errorf("missing issue code %s in %v", code, issues)
		}
	}
	// Ensure secret value never appears in messages
	for _, iss := range issues {
		if strings.Contains(iss.Message, "postgres//x") {
			t.Errorf("secret value leaked in message: %s", iss.Message)
		}
	}
	// Check sorted order: Line, then Key, then Code (missing required keys use line 0).
	if len(issues) < 3 {
		t.Fatalf("expected multiple issues, got %d", len(issues))
	}
	for i := 1; i < len(issues); i++ {
		a, b := issues[i-1], issues[i]
		if a.Line > b.Line || (a.Line == b.Line && (a.Key > b.Key || (a.Key == b.Key && a.Code > b.Code))) {
			t.Errorf("issues not sorted: %+v before %+v", a, b)
		}
	}
	if !hasIssue(issues, "E003", "PORT", 1) {
		t.Errorf("expected E003 for PORT at line 1, got %v", issues)
	}
}

func TestValidateIssueMessages(t *testing.T) {
	schema := dotenvschema.Schema{
		Vars: map[string]dotenvschema.Var{
			"PORT":         {Type: "int", Min: int64Ptr(1), Max: int64Ptr(65535)},
			"DATABASE_URL": {Type: "url", Secret: true},
		},
	}

	issues := dotenvschema.Validate([]dotenvschema.Entry{
		{Key: "PORT", Value: "70000", Line: 1},
	}, schema)
	wantMax := dotenvschema.Issue{Code: "E003", Key: "PORT", Line: 1, Message: "PORT must be <= 65535"}
	if len(issues) != 1 || issues[0] != wantMax {
		t.Errorf("PORT=70000 issues = %+v, want exactly %+v", issues, wantMax)
	}

	issues = dotenvschema.Validate([]dotenvschema.Entry{
		{Key: "PORT", Value: "abc", Line: 1},
	}, schema)
	wantInt := dotenvschema.Issue{Code: "E002", Key: "PORT", Line: 1, Message: "PORT must be an int"}
	if len(issues) != 1 || issues[0] != wantInt {
		t.Errorf("PORT=abc issues = %+v, want exactly %+v", issues, wantInt)
	}

	issues = dotenvschema.Validate([]dotenvschema.Entry{
		{Key: "DATABASE_URL", Value: "postgres//x", Line: 1},
	}, schema)
	wantURL := dotenvschema.Issue{Code: "E002", Key: "DATABASE_URL", Line: 1, Message: "DATABASE_URL must be a url"}
	if len(issues) != 1 || issues[0] != wantURL {
		t.Errorf("DATABASE_URL issues = %+v, want exactly %+v", issues, wantURL)
	}
	for _, iss := range issues {
		if strings.Contains(iss.Message, "postgres//x") {
			t.Errorf("secret value leaked in message: %s", iss.Message)
		}
	}
}

func TestValidateDuplicateAndRequiredDefault(t *testing.T) {
	schema := dotenvschema.Schema{
		Vars: map[string]dotenvschema.Var{
			"A": {Type: "string", Required: true, Default: stringPtr("d")},
			"B": {Type: "string", Required: true},
		},
	}
	entries := []dotenvschema.Entry{
		{Key: "A", Value: "x", Line: 1},
		{Key: "A", Value: "y", Line: 2},
		{Key: "B", Value: "z", Line: 3},
	}
	issues := dotenvschema.Validate(entries, schema)
	// E007 for duplicate A at line2; no E001 for A because default; no E001 for B because present
	if !hasIssue(issues, "E007", "A", 2) {
		t.Errorf("expected E007 for A at line 2, got %v", issues)
	}
	if hasIssue(issues, "E001", "A", 0) {
		t.Errorf("unexpected E001 for A with default")
	}

	// A duplicate key whose second value is invalid yields both E007 and the type issue.
	intSchema := dotenvschema.Schema{Vars: map[string]dotenvschema.Var{"PORT": {Type: "int"}}}
	dupIssues := dotenvschema.Validate([]dotenvschema.Entry{
		{Key: "PORT", Value: "1", Line: 1},
		{Key: "PORT", Value: "abc", Line: 2},
	}, intSchema)
	if !hasIssue(dupIssues, "E007", "PORT", 2) {
		t.Errorf("expected E007 for PORT at line 2, got %v", dupIssues)
	}
	if !hasIssue(dupIssues, "E002", "PORT", 2) {
		t.Errorf("expected E002 for PORT at line 2, got %v", dupIssues)
	}
}

func hasIssue(issues []dotenvschema.Issue, code, key string, line int) bool {
	for _, iss := range issues {
		if iss.Code == code && iss.Key == key && iss.Line == line {
			return true
		}
	}
	return false
}

func TestExampleAcceptance(t *testing.T) {
	schema := dotenvschema.Schema{
		Vars: map[string]dotenvschema.Var{
			"PORT":         {Type: "int", Required: true, Default: stringPtr("8080")},
			"DATABASE_URL": {Type: "url", Required: true, Secret: true},
		},
	}
	got := dotenvschema.Example(schema)
	want := "# url, required, secret\nDATABASE_URL=\n# int, required\nPORT=8080\n"
	if got != want {
		t.Errorf("Example = %q, want %q", got, want)
	}
}

func TestRunAcceptance(t *testing.T) {
	// Create schema and env file
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.json")
	envPath := filepath.Join(dir, ".env")
	schemaContent := `{"vars":{"PORT":{"type":"int","min":1,"max":65535}}}`
	envContent := "PORT=70000\n"
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := dotenvschema.Run([]string{"--schema", schemaPath, envPath}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run code = %d, want 1", code)
	}
	wantLine := envPath + ":1: E003 PORT must be <= 65535\n"
	if stdout.String() != wantLine {
		t.Errorf("stdout = %q, want %q", stdout.String(), wantLine)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunLiteralPaths(t *testing.T) {
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	}()

	if err := os.WriteFile("s.json", []byte(`{"vars":{"PORT":{"type":"int","min":1,"max":65535}}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".env", []byte("PORT=70000\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := dotenvschema.Run([]string{"--schema", "s.json", ".env"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run code = %d, want 1", code)
	}
	want := ".env:1: E003 PORT must be <= 65535\n"
	if stdout.String() != want {
		t.Errorf("stdout = %q, want %q", stdout.String(), want)
	}
	if !strings.Contains(stdout.String(), ".env:1: E003 PORT must be <= 65535") {
		t.Errorf("stdout does not contain literal issue line: %q", stdout.String())
	}
}

func TestRunValidFileReturnsZero(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.json")
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(schemaPath, []byte(`{"vars":{"PORT":{"type":"int","min":1,"max":65535}}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envPath, []byte("PORT=8080\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := dotenvschema.Run([]string{"--schema", schemaPath, envPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run code = %d, want 0", code)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunExample(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.json")
	schemaContent := `{"vars":{"A":{"type":"string","required":true}}}`
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := dotenvschema.Run([]string{"--schema", schemaPath, "--example"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run code = %d, want 0", code)
	}
	want := "# string, required\nA=\n"
	if stdout.String() != want {
		t.Errorf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunJSONAndMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.json")
	env1 := filepath.Join(dir, "env1")
	env2 := filepath.Join(dir, "env2")
	os.WriteFile(schemaPath, []byte(`{"vars":{"A":{"type":"int"}}}`), 0644)
	os.WriteFile(env1, []byte("A=abc\n"), 0644)
	os.WriteFile(env2, []byte("A=1\n"), 0644)
	var stdout, stderr bytes.Buffer
	code := dotenvschema.Run([]string{"--schema", schemaPath, "--json", env1, env2}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run code = %d, want 1", code)
	}
	// JSON output should be an array with one issue from env1
	expected := `[{"code":"E002","key":"A","line":1,"message":"A must be an int"}]` + "\n"
	if stdout.String() != expected {
		t.Errorf("stdout = %q, want %q", stdout.String(), expected)
	}
}

func TestRunUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := dotenvschema.Run([]string{}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("Run code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--schema is required") {
		t.Errorf("stderr should mention --schema required, got %q", stderr.String())
	}
}

func TestRunIOError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := dotenvschema.Run([]string{"--schema", "nonexistent.json", "env"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("Run code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "nonexistent.json") {
		t.Errorf("stderr should mention file, got %q", stderr.String())
	}
}
