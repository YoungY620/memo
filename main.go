package main

import (
	"os"

	"github.com/YoungY620/memo/cmd"
)

var Version = "dev"

func main() {
	cmd.SetVersion(Version)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
