package promptq

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const clipboardTimeout = 5 * time.Second

// copyToClipboard writes text to the OS clipboard using native tools.
func copyToClipboard(text string) error {
	var name string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		name = "pbcopy"
	case "windows":
		name, args = "clip", nil
	default: // linux, wsl
		if path, err := exec.LookPath("wl-copy"); err == nil {
			name = path
		} else if path, err := exec.LookPath("xclip"); err == nil {
			name, args = path, []string{"-selection", "clipboard"}
		} else if path, err := exec.LookPath("xsel"); err == nil {
			name, args = path, []string{"--clipboard", "--input"}
		} else {
			return fmt.Errorf("no clipboard tool found (install wl-clipboard, xclip, or xsel)")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), clipboardTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("clipboard command timed out")
		}
		return fmt.Errorf("clipboard unavailable: %w", err)
	}
	return nil
}
