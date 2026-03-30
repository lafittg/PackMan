package main

import (
	cmd "github.com/gregoirelafitte/packman/cmd/packman"

	// Register ecosystem plugins
	_ "github.com/gregoirelafitte/packman/internal/plugin/golang"
	_ "github.com/gregoirelafitte/packman/internal/plugin/nodejs"
	_ "github.com/gregoirelafitte/packman/internal/plugin/python"
)

func main() {
	cmd.Execute()
}
