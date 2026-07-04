package tests

import (
	"fmt"
	"strings"
	"testing"

	promptq "github.com/ast-lw/promptq/internal/promptq"
)

func TestResolvePromptName(t *testing.T) {
	useTempPromptHome(t)
	for _, p := range []promptq.Prompt{
		{Name: "root", Body: "root"},
		{Name: "writing/rewrite", Body: "writing"},
		{Name: "engineering/review", Body: "engineering"},
	} {
		if err := promptq.Save(p); err != nil {
			t.Fatal(err)
		}
	}
	cases := map[string]string{
		"@root":               "root",
		"@rewrite":            "writing/rewrite",
		"writing/@rewrite":    "writing/rewrite",
		"engineering/@review": "engineering/review",
	}
	for input, want := range cases {
		got, err := promptq.ResolvePromptName(input)
		if err != nil {
			t.Fatalf("ResolvePromptName(%q): %v", input, err)
		}
		if got != want {
			t.Errorf("ResolvePromptName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestResolvePromptNameAmbiguous(t *testing.T) {
	useTempPromptHome(t)
	for _, folder := range []string{"engineering", "writing"} {
		if err := promptq.Save(promptq.Prompt{Name: folder + "/review", Body: folder}); err != nil {
			t.Fatal(err)
		}
	}
	_, err := promptq.ResolvePromptName("@review")
	if err == nil || !strings.Contains(err.Error(), "engineering/@review, writing/@review") {
		t.Fatalf("ambiguity error = %v", err)
	}
}

func TestResolvePromptNamePrefersExactRoot(t *testing.T) {
	useTempPromptHome(t)
	for _, name := range []string{"review", "writing/review"} {
		if err := promptq.Save(promptq.Prompt{Name: name, Body: name}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := promptq.ResolvePromptName("@review")
	if err != nil || got != "review" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestDuplicateLeafAcrossFolders(t *testing.T) {
	useTempPromptHome(t)
	if err := promptq.Save(promptq.Prompt{Name: "one/shared", Body: "body"}); err != nil {
		t.Fatal(err)
	}
	src, err := promptq.ResolvePromptName("one/@shared")
	if err != nil {
		t.Fatal(err)
	}
	dst, err := promptq.ParseUserName("two/@shared")
	if err != nil {
		t.Fatal(err)
	}
	if err := promptq.DuplicatePrompt(src, dst); err != nil {
		t.Fatal(err)
	}
	if p, err := promptq.Load("two/shared"); err != nil || p.Body != "body" {
		t.Fatalf("duplicate = %#v, %v", p, err)
	}
}

func TestResolveAmongMoreThanOneThousandPrompts(t *testing.T) {
	useTempPromptHome(t)
	for i := 0; i < 1200; i++ {
		name := fmt.Sprintf("folder-%04d/prompt-%04d", i, i)
		if err := promptq.Save(promptq.Prompt{Name: name, Body: name}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := promptq.ResolvePromptName("@prompt-1199")
	if err != nil {
		t.Fatal(err)
	}
	if got != "folder-1199/prompt-1199" {
		t.Fatalf("got %q", got)
	}
}

func TestParseUserNameRejectsNonCanonicalForms(t *testing.T) {
	for _, input := range []string{"name", "folder/name", "@folder/name", "folder/"} {
		if _, err := promptq.ParseUserName(input); err == nil {
			t.Errorf("ParseUserName(%q) succeeded", input)
		}
	}
}
