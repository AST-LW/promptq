package promptq

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func editorCommand() []string {
	if ed := strings.TrimSpace(os.Getenv("VISUAL")); ed != "" {
		return strings.Fields(ed)
	}
	if ed := strings.TrimSpace(os.Getenv("EDITOR")); ed != "" {
		return strings.Fields(ed)
	}
	if runtime.GOOS == "windows" {
		return []string{"notepad"}
	}
	return []string{"vi"}
}

func openInEditor(path string) error {
	parts := editorCommand()
	if len(parts) == 0 {
		return fmt.Errorf("editor is not configured")
	}
	args := append(append([]string{}, parts[1:]...), path)
	cmd := exec.Command(parts[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}
	return nil
}
