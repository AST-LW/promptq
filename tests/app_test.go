package tests

import (
	"reflect"
	"strings"
	"testing"

	promptq "github.com/ast-lw/promptq/internal/promptq"
)

func TestParseSaveArgs(t *testing.T) {
	got, err := promptq.ParseSaveArgs([]string{
		"email/@followup",
		"-d", "Follow-up",
		"--tags=sales, draft",
		"--message=Hello",
	})
	if err != nil {
		t.Fatalf("ParseSaveArgs returned error: %v", err)
	}
	want := promptq.SaveArgs{
		Name:           "email/@followup",
		Description:    "Follow-up",
		DescriptionSet: true,
		Tags:           "sales, draft",
		TagsSet:        true,
		Body:           "Hello",
	}
	if got != want {
		t.Fatalf("ParseSaveArgs = %#v, want %#v", got, want)
	}

	errorCases := [][]string{
		{"@name", "extra"},
		{"@name", "--bad"},
		{"@name", "-m"},
		{"@name", "--description"},
	}
	for _, args := range errorCases {
		if _, err := promptq.ParseSaveArgs(args); err == nil {
			t.Fatalf("ParseSaveArgs(%v) succeeded, want error", args)
		}
	}
}

func TestCommandArgumentValidation(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"get missing", []string{"get"}},
		{"get extra", []string{"get", "@one", "@two"}},
		{"get unknown flag", []string{"get", "--unknown", "@one"}},
		{"get repeated preview", []string{"get", "--preview", "--preview", "@one"}},
		{"preview missing", []string{"preview"}},
		{"preview extra", []string{"preview", "@one", "@two"}},
		{"show missing", []string{"show"}},
		{"show extra", []string{"show", "@one", "@two"}},
		{"search missing", []string{"search"}},
		{"search blank", []string{"search", "  "}},
		{"rename missing", []string{"rename", "@one"}},
		{"rename extra", []string{"rename", "@one", "@two", "@three"}},
		{"duplicate missing", []string{"duplicate", "@one"}},
		{"duplicate extra", []string{"duplicate", "@one", "@two", "@three"}},
		{"remove missing", []string{"rm"}},
		{"remove extra", []string{"rm", "@one", "@two"}},
		{"edit missing", []string{"edit"}},
		{"edit extra", []string{"edit", "@one", "@two"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := promptq.Run(tt.args); err == nil {
				t.Fatal("command succeeded, want argument error")
			}
		})
	}
}

func TestSaveUpdatePreservesUnspecifiedMetadataAndUsage(t *testing.T) {
	useTempPromptHome(t)
	original := promptq.Prompt{
		Name: "rewrite", Description: "description", Tags: []string{"one", "two"},
		Body: "old", UseCount: 7, LastUsed: 123,
	}
	if err := promptq.Save(original); err != nil {
		t.Fatal(err)
	}
	if err := promptq.Run([]string{"save", "@rewrite", "-m", "new"}); err != nil {
		t.Fatal(err)
	}
	got, err := promptq.Load("rewrite")
	if err != nil {
		t.Fatal(err)
	}
	if got.Body != "new" || got.Description != original.Description ||
		!reflect.DeepEqual(got.Tags, original.Tags) || got.UseCount != 7 || got.LastUsed != 123 {
		t.Fatalf("updated prompt lost data: %#v", got)
	}
	if strings.Contains(got.Body, "old") {
		t.Fatalf("body was not replaced: %q", got.Body)
	}
}

func TestSplitTags(t *testing.T) {
	got := promptq.SplitTags(" sales, ,draft,ai ")
	want := []string{"sales", "draft", "ai"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SplitTags = %#v, want %#v", got, want)
	}
}

func TestEditorCommand(t *testing.T) {
	t.Setenv("VISUAL", "code --wait")
	t.Setenv("EDITOR", "vim -f")
	if got, want := promptq.EditorCommand(), []string{"code", "--wait"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("EditorCommand with VISUAL = %#v, want %#v", got, want)
	}

	t.Setenv("VISUAL", "")
	if got, want := promptq.EditorCommand(), []string{"vim", "-f"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("EditorCommand with EDITOR = %#v, want %#v", got, want)
	}
}
