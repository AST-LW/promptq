package tests

import (
	"io"
	"os"
	"strings"
	"testing"

	promptq "github.com/ast-lw/promptq/internal/promptq"
)

func TestPreviewPreservesVariables(t *testing.T) {
	useTempPromptHome(t)
	want := "Rewrite {{text}} in a {{tone}} tone."
	if err := promptq.Save(promptq.Prompt{Name: "rewrite", Body: want}); err != nil {
		t.Fatal(err)
	}

	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = write
	t.Cleanup(func() { os.Stdout = old })

	if err := promptq.Run([]string{"preview", "@rewrite"}); err != nil {
		t.Fatal(err)
	}
	_ = write.Close()
	got, err := io.ReadAll(read)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSuffix(string(got), "\n") != want {
		t.Fatalf("preview = %q, want unchanged %q", got, want)
	}
}
