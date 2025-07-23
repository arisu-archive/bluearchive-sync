package main

import (
	"os"

	"github.com/arisu-archive/bluearchive-data-sync/internal/cmd/root"
)

var Version = "1.0.0"

func main() {
	root.Execute(Version, os.Exit, os.Stdin, os.Stdout, os.Stderr, os.Args[1:])
}
