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
	fmt.Println("overwatch TUI — not yet implemented")
	fmt.Println("use 'overwatch serve' to start the server, or 'overwatch status' to inspect configuration")
	return nil
}
