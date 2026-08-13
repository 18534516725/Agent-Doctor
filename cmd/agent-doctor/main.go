package main

import (
	"os"

	"github.com/18534516725/Agent-Doctor/internal/app"
)

func main() {
	os.Exit(app.Run(os.Args[1:], os.Stdout, os.Stderr))
}
