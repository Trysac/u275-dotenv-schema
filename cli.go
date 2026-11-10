package dotenvschema

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

// Run executes the dotenv-schema command line tool. It returns 0 when no
// issues are found, 1 when one or more issues are reported, and 2 on usage or
// I/O errors. Output is written to stdout; errors are written to stderr.
func Run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dotenv-schema", flag.ContinueOnError)
	fs.SetOutput(stderr)
	schemaPath := fs.String("schema", "", "path to schema file (required, '-' for stdin)")
	jsonOut := fs.Bool("json", false, "output issues as a JSON array")
	exampleOut := fs.Bool("example", false, "print generated .env.example and exit")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *schemaPath == "" {
		fmt.Fprintln(stderr, "dotenv-schema: --schema is required")
		return 2
	}
	schema, err := loadSchemaFile(*schemaPath)
	if err != nil {
		fmt.Fprintf(stderr, "dotenv-schema: %v\n", err)
		return 2
	}
	if *exampleOut {
		fmt.Fprint(stdout, Example(schema))
		return 0
	}
	files := fs.Args()
	if len(files) == 0 {
		files = []string{"-"}
	}
	var allIssues []Issue
	for _, file := range files {
		entries, err := readEnvFile(file)
		if err != nil {
			fmt.Fprintf(stderr, "dotenv-schema: %s: %v\n", file, err)
			return 2
		}
		issues := Validate(entries, schema)
		if !*jsonOut {
			prefix := ""
			if file != "-" {
				prefix = file + ":"
			}
			for _, iss := range issues {
				fmt.Fprintf(stdout, "%s%d: %s %s\n", prefix, iss.Line, iss.Code, iss.Message)
			}
		}
		allIssues = append(allIssues, issues...)
	}
	if *jsonOut {
		enc := json.NewEncoder(stdout)
		if err := enc.Encode(allIssues); err != nil {
			fmt.Fprintf(stderr, "dotenv-schema: %v\n", err)
			return 2
		}
	}
	if len(allIssues) > 0 {
		return 1
	}
	return 0
}

func loadSchemaFile(path string) (Schema, error) {
	if path == "-" {
		return LoadSchema(os.Stdin)
	}
	f, err := os.Open(path)
	if err != nil {
		return Schema{}, err
	}
	defer f.Close()
	return LoadSchema(f)
}

func readEnvFile(path string) ([]Entry, error) {
	if path == "-" {
		return Parse(os.Stdin, Options{})
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f, Options{})
}
