package main

import (
	"fmt"
	"os"

	"github.com/middlendian/gh-app-auth/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
