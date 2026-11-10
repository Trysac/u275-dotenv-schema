package main

import (
	"os"

	"github.com/Trysac/u275-dotenv-schema"
)

func main() {
	os.Exit(dotenvschema.Run(os.Args[1:], os.Stdout, os.Stderr))
}
