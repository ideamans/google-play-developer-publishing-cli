package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ideamans/go-llm-cli-kit/llmcmd"

	"github.com/ideamans/google-play-developer-publishing-cli/internal/llmdocs"
)

// TestEmbeddedReference checks that the three things an agent needs are all
// present: the credential and edit model, the hard-won pitfalls, and the
// command catalog.
func TestEmbeddedReference(t *testing.T) {
	g, err := llmdocs.Docs().Markdown()
	if err != nil {
		t.Fatalf("Markdown: %v", err)
	}
	for _, want := range []string{
		"reference for AI agents",
		"GPLAY_PROFILE",    // credential model
		"The edit model",   // the transactional model
		"# Known pitfalls", // hand-written pitfalls chapter
		"Version codes must increase",
		"# Command catalog", // generated
		"### `gplay apps list`",
	} {
		if !strings.Contains(g, want) {
			t.Errorf("embedded reference missing %q", want)
		}
	}
}

func TestChapterOrder(t *testing.T) {
	sections, err := llmdocs.Docs().Sections()
	if err != nil {
		t.Fatalf("Sections: %v", err)
	}
	var files []string
	for _, s := range sections {
		files = append(files, s.File)
	}
	want := "00-guide.md,10-pitfalls.md,90-commands.md"
	if got := strings.Join(files, ","); got != want {
		t.Errorf("chapters = %s, want %s", got, want)
	}
}

func TestLLMSubcommand(t *testing.T) {
	var out bytes.Buffer
	root := Root()
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"llm"})
	if err := root.Execute(); err != nil {
		t.Fatalf("gplay llm: %v", err)
	}
	if !strings.Contains(out.String(), "# Command catalog") {
		t.Error("gplay llm did not print the reference")
	}
}

// TestLegacyLLMFlag guards the compatibility promise: --llm used to work at any
// position on the command line, and callers still rely on it.
func TestLegacyLLMFlag(t *testing.T) {
	for _, args := range [][]string{{"--llm"}, {"apps", "list", "--llm"}} {
		var out bytes.Buffer
		handled, err := llmcmd.HandleLegacy(args, LLMConfig(), &out)
		if err != nil {
			t.Fatalf("HandleLegacy(%v): %v", args, err)
		}
		if !handled {
			t.Errorf("HandleLegacy(%v) did not handle --llm", args)
		}
		if !strings.Contains(out.String(), "GPLAY_PROFILE") {
			t.Errorf("HandleLegacy(%v) printed the wrong thing", args)
		}
	}
}
