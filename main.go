package main

import (
	cmd "github.com/gregoirelafitte/packman/cmd/packman"

	// Register ecosystem plugins
	_ "github.com/gregoirelafitte/packman/internal/plugin/nodejs"
)

func main() {
	cmd.Execute()
}
