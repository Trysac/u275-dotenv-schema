package dotenvschema

import (
	"errors"
	"strconv"
)

// Sentinel errors returned by the package.
var (
	ErrSyntax        = errors.New("dotenvschema: syntax error")
	ErrInputTooLarge = errors.New("dotenvschema: input too large")
	ErrSchema        = errors.New("dotenvschema: invalid schema")
)

// SyntaxError describes a syntax error at a specific line of a .env file.
// It unwraps to [ErrSyntax].
type SyntaxError struct {
	Line   int
	Reason string
}

func (e *SyntaxError) Error() string {
	return "dotenvschema: line " + strconv.Itoa(e.Line) + ": " + e.Reason
}

func (e *SyntaxError) Unwrap() error {
	return ErrSyntax
}
