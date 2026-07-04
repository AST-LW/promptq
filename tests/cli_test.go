package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var cliPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "promptq-cli-test-")
	if err != nil {
		panic(err)
	}
	name := "promptq"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	cliPath = filepath.Join(dir, name)
	build := exec.Command("go", "build", "-trimpath", "-o", cliPath, "../cmd/promptq")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func runCLI(t *testing.T, home string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(cliPath, args...)
	cmd.Env = append(os.Environ(), "PROMPTQ_HOME="+home)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func requireSuccess(t *testing.T, home string, args ...string) string {
	t.Helper()
	out, err := runCLI(t, home, args...)
	if err != nil {
		t.Fatalf("promptq %v failed: %v\n%s", args, err, out)
	}
	return out
}

func TestCLICompletePromptLifecycle(t *testing.T) {
	home := t.TempDir()

	out := requireSuccess(t, home, "save", "writing/@review", "-d", "Review text", "-t", "writing", "-m", "body {{text}}")
	if !strings.Contains(out, "Saved writing/@review") {
		t.Fatalf("unexpected save output: %s", out)
	}

	if out := requireSuccess(t, home, "preview", "@review"); out != "body {{text}}" {
		t.Fatalf("preview changed body: %q", out)
	}
	if out := requireSuccess(t, home, "show", "writing/@review"); out != "body {{text}}" {
		t.Fatalf("show changed body: %q", out)
	}

	out = requireSuccess(t, home, "list")
	for _, want := range []string{"writing/@review", "Review text", "writing"} {
		if !strings.Contains(out, want) {
			t.Fatalf("list output missing %q:\n%s", want, out)
		}
	}

	out = requireSuccess(t, home, "search", "review")
	if !strings.Contains(out, "writing/@review") {
		t.Fatalf("search output missing prompt: %s", out)
	}

	requireSuccess(t, home, "duplicate", "writing/@review", "engineering/@review")
	if _, err := runCLI(t, home, "preview", "@review"); err == nil {
		t.Fatal("ambiguous leaf lookup succeeded")
	}
	requireSuccess(t, home, "rename", "engineering/@review", "engineering/@approved")
	requireSuccess(t, home, "rm", "engineering/@approved")

	if out := requireSuccess(t, home, "preview", "@review"); out != "body {{text}}" {
		t.Fatalf("unique lookup after removal = %q", out)
	}
}

func TestCLIAmbiguityExplainsCanonicalPaths(t *testing.T) {
	home := t.TempDir()
	requireSuccess(t, home, "save", "one/@shared", "-m", "one")
	requireSuccess(t, home, "save", "two/@shared", "-m", "two")

	out, err := runCLI(t, home, "preview", "@shared")
	if err == nil {
		t.Fatal("ambiguous preview succeeded")
	}
	for _, want := range []string{"prompt @shared is ambiguous", "one/@shared", "two/@shared"} {
		if !strings.Contains(out, want) {
			t.Fatalf("ambiguity output missing %q: %s", want, out)
		}
	}
}

func TestCLIRejectsInvalidCommandsAndNames(t *testing.T) {
	home := t.TempDir()
	tests := [][]string{
		{"save", "plain", "-m", "body"},
		{"save", "folder/plain", "-m", "body"},
		{"preview"},
		{"preview", "@one", "@two"},
		{"search", "  "},
		{"rename", "@one"},
		{"duplicate", "@one"},
		{"rm"},
		{"unknown"},
	}
	for _, args := range tests {
		if out, err := runCLI(t, home, args...); err == nil {
			t.Errorf("promptq %v succeeded: %s", args, out)
		}
	}
}

func TestCLIUpdatePreservesMetadata(t *testing.T) {
	home := t.TempDir()
	requireSuccess(t, home, "save", "@rewrite", "-d", "Description", "-t", "one,two", "-m", "old")
	requireSuccess(t, home, "save", "@rewrite", "-m", "new")

	out := requireSuccess(t, home, "list")
	for _, want := range []string{"Description", "one, two"} {
		if !strings.Contains(out, want) {
			t.Fatalf("updated prompt lost %q:\n%s", want, out)
		}
	}
	if out := requireSuccess(t, home, "show", "@rewrite"); out != "new" {
		t.Fatalf("updated body = %q", out)
	}
}
