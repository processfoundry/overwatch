package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func clientDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), ".overwatch")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".overwatch")
}

func IsSetup() bool {
	_, err := os.Stat(clientDir())
	return err == nil
}

func Run() error {
	if !IsSetup() {
		return runSetup()
	}
	fmt.Println("overwatch TUI — not yet implemented")
	fmt.Println("use 'overwatch serve' to start the server, or 'overwatch status' to inspect configuration")
	return nil
}

func runSetup() error {
	dir := clientDir()
	fmt.Println("Welcome to Overwatch!")
	fmt.Println()
	fmt.Println("No client configuration found. Creating", dir)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	fmt.Println()
	fmt.Println("To connect to a self-hosted server:")
	fmt.Println("  overwatch serve            # start a server")
	fmt.Println()
	fmt.Println("To connect to Overwatch Cloud:")
	fmt.Println("  Visit https://overwatch.dev to sign up")
	fmt.Println()
	fmt.Println("Run 'overwatch' again to launch the TUI.")
	return nil
}
