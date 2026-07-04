package tests

import (
	"errors"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"

	promptq "github.com/ast-lw/promptq/internal/promptq"
)

func useTempPromptHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("PROMPTQ_HOME", dir)
	return dir
}

func TestCleanName(t *testing.T) {
	t.Parallel()
	valid := map[string]string{
		"@daily":           "daily",
		"team/review_v1":   "team/review_v1",
		"Team/Release-1.2": "Team/Release-1.2",
		"folder/@item":     "folder/item",
		"alpha_beta/gamma": "alpha_beta/gamma",
	}
	for input, want := range valid {
		got, err := promptq.CleanName(input)
		if err != nil {
			t.Fatalf("CleanName(%q) returned error: %v", input, err)
		}
		if got != want {
			t.Fatalf("CleanName(%q) = %q, want %q", input, got, want)
		}
	}

	invalid := []string{
		"", "@", "../x", "x/../y", "x//y", "/x", "x/", "x y",
		`x\y`, ".hidden", "x.", "CON", "com1", "nul.txt", "x/$bad", "@folder/item",
	}
	for _, input := range invalid {
		if got, err := promptq.CleanName(input); err == nil {
			t.Fatalf("CleanName(%q) = %q, want error", input, got)
		}
	}
}

func TestPromptDisplayKeepsAtOnLeaf(t *testing.T) {
	cases := map[string]string{
		"abc":          "@abc",
		"abc/def":      "abc/@def",
		"one/two/last": "one/two/@last",
	}
	for name, want := range cases {
		if got := (promptq.Prompt{Name: name}).Display(); got != want {
			t.Errorf("Prompt{%q}.Display() = %q, want %q", name, got, want)
		}
	}
}

func TestSaveLoadListRoundTrip(t *testing.T) {
	useTempPromptHome(t)

	want := promptq.Prompt{
		Name:        "work/email",
		Description: "Follow-up email",
		Tags:        []string{"sales", "draft"},
		Body:        "Hello {{name}},\nThanks for the time.",
		UseCount:    7,
		LastUsed:    12345,
	}
	if err := promptq.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := promptq.Load(want.Name)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("loaded prompt mismatch:\n got: %#v\nwant: %#v", got, want)
	}

	all, err := promptq.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 1 || all[0].Name != want.Name {
		t.Fatalf("List returned %#v", all)
	}

	if runtime.GOOS != "windows" {
		path, err := promptq.PromptPath(want.Name)
		if err != nil {
			t.Fatalf("PromptPath: %v", err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat prompt file: %v", err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("prompt file mode = %v, want 0600", got)
		}
	}
}

func TestParseLongLineDoesNotTruncate(t *testing.T) {
	long := strings.Repeat("x", 2<<20)
	got := promptq.ParseStoredPrompt("long", "---\ndescription: big\n---\n"+long)
	if got.Description != "big" {
		t.Fatalf("description = %q, want big", got.Description)
	}
	if got.Body != long {
		t.Fatalf("body length = %d, want %d", len(got.Body), len(long))
	}
}

func TestBodyBeginningWithFrontmatterDelimiterRoundTrips(t *testing.T) {
	useTempPromptHome(t)
	want := "---\nthis is prompt content\n---\nnot metadata"
	if err := promptq.Save(promptq.Prompt{Name: "delimiter", Body: want}); err != nil {
		t.Fatal(err)
	}
	got, err := promptq.Load("delimiter")
	if err != nil {
		t.Fatal(err)
	}
	if got.Body != want {
		t.Fatalf("body = %q, want %q", got.Body, want)
	}
}

func TestSaveRejectsMetadataInjection(t *testing.T) {
	useTempPromptHome(t)
	for _, p := range []promptq.Prompt{
		{Name: "bad-description", Description: "first\nuses: 999", Body: "body"},
		{Name: "bad-tag", Tags: []string{"safe\nlast_used: 999"}, Body: "body"},
	} {
		if err := promptq.Save(p); err == nil {
			t.Errorf("Save(%#v) succeeded", p)
		}
	}
}

func TestRenameDuplicateRemove(t *testing.T) {
	useTempPromptHome(t)

	source := promptq.Prompt{Name: "source", Body: "body", UseCount: 5, LastUsed: 99}
	if err := promptq.Save(source); err != nil {
		t.Fatalf("Save source: %v", err)
	}
	if err := promptq.DuplicatePrompt("source", "copy"); err != nil {
		t.Fatalf("DuplicatePrompt: %v", err)
	}
	copyPrompt, err := promptq.Load("copy")
	if err != nil {
		t.Fatalf("Load copy: %v", err)
	}
	if copyPrompt.UseCount != 0 || copyPrompt.LastUsed != 0 {
		t.Fatalf("duplicate retained usage: %#v", copyPrompt)
	}

	if err := promptq.RenamePrompt("source", "renamed"); err != nil {
		t.Fatalf("RenamePrompt: %v", err)
	}
	if _, err := promptq.Load("source"); err == nil {
		t.Fatalf("old prompt still loads after rename")
	}
	renamed, err := promptq.Load("renamed")
	if err != nil {
		t.Fatalf("Load renamed: %v", err)
	}
	if renamed.Body != source.Body {
		t.Fatalf("renamed body = %q, want %q", renamed.Body, source.Body)
	}

	if err := promptq.RenamePrompt("copy", "renamed"); err == nil {
		t.Fatalf("rename over existing prompt succeeded")
	}
	if err := promptq.Remove("copy"); err != nil {
		t.Fatalf("Remove copy: %v", err)
	}
	if _, err := promptq.Load("copy"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("load removed prompt error = %v", err)
	}
	if err := promptq.Remove("copy"); err == nil || errors.Is(err, os.ErrNotExist) {
		t.Fatalf("remove missing error = %v", err)
	}
}
