package main

import (
	"fmt"
	"os"

	"github.com/calendar/yuman/internal/cli"
)

var (
	version = "0.2.0"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	cli.SetVersion(fmt.Sprintf("%s (%s) built %s", version, commit, date), version)
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}