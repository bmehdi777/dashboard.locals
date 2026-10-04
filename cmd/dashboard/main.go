package main

import (
	"dashboard.locals/internal/cli"
	"os"
)

func main() {
	os.Exit(cli.Execute())
}
