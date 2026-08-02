package cheatsheet

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseYAMLWithGroups(t *testing.T) {
	content := []byte(`
sheets:
  - id: tmux
    description: Terminal multiplexer
    groups:
      - name: Sessions
        items:
          - command: tmux ls
            description: List sessions
`)

	book, err := Parse(content, "cheat-sheets.yaml")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	sheet, found := book.Find("TMUX")
	if !found {
		t.Fatal("expected to find the tmux sheet, case-insensitively")
	}

	groups := sheet.NormalizedGroups()
	if len(groups) != 1 || groups[0].Name != "Sessions" || len(groups[0].Items) != 1 {
		t.Fatalf("unexpected groups: %#v", groups)
	}

	if groups[0].Items[0].Command != "tmux ls" {
		t.Fatalf("unexpected command: %q", groups[0].Items[0].Command)
	}
}

func TestParseJSON(t *testing.T) {
	content := []byte(`{"sheets":[{"id":"go","groups":[{"name":"Build","items":[{"command":"go build ./...","description":"Build everything"}]}]}]}`)

	book, err := Parse(content, "cheat-sheets.json")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	sheet, found := book.Find("go")
	if !found {
		t.Fatal("expected to find the go sheet")
	}

	if got := sheet.CountItems(); got != 1 {
		t.Fatalf("expected 1 item, got %d", got)
	}
}

func TestParseSniffsJSONWithoutExtension(t *testing.T) {
	content := []byte(`  {"sheets":[{"id":"go"}]}`)

	book, err := Parse(content, "sheets")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(book.Sheets) != 1 {
		t.Fatalf("expected 1 sheet, got %d", len(book.Sheets))
	}
}

func TestNormalizedGroupsFromLegacyFlatItems(t *testing.T) {
	content := []byte(`
sheets:
  - id: tmux
    items:
      - command: tmux a
        description: Attach
      - command: -- SESSIONS --
        description:
      - command: tmux ls
        description: List sessions
      - command: -- PANES --
        description:
      - command: ctrl+b z
        description: Zoom pane
`)

	book, err := Parse(content, "legacy.yaml")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	sheet, _ := book.Find("tmux")
	groups := sheet.NormalizedGroups()

	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d: %#v", len(groups), groups)
	}

	if groups[0].Name != "" || len(groups[0].Items) != 1 {
		t.Fatalf("expected a leading unnamed group with 1 item, got %#v", groups[0])
	}

	if groups[1].Name != "SESSIONS" || groups[2].Name != "PANES" {
		t.Fatalf("unexpected group names: %q, %q", groups[1].Name, groups[2].Name)
	}

	if got := sheet.CountItems(); got != 3 {
		t.Fatalf("expected 3 items, got %d", got)
	}
}

func TestFilter(t *testing.T) {
	groups := []Group{
		{Name: "Sessions", Items: []Item{
			{Command: "tmux ls", Description: "List sessions"},
			{Command: "tmux a", Description: "Attach"},
		}},
		{Name: "Panes", Items: []Item{
			{Command: "ctrl+b z", Description: "Zoom pane"},
		}},
	}

	filtered := Filter(groups, "ZOOM")
	if len(filtered) != 1 || filtered[0].Name != "Panes" || len(filtered[0].Items) != 1 {
		t.Fatalf("expected only the zoom item, got %#v", filtered)
	}

	if got := Filter(groups, "  "); len(got) != 2 {
		t.Fatalf("empty term should keep every group, got %#v", got)
	}
}

func TestSuggest(t *testing.T) {
	book := Book{Sheets: []Sheet{{ID: "tmux"}, {ID: "superfile"}}}

	if got := book.Suggest("tmu"); len(got) != 1 || got[0] != "tmux" {
		t.Fatalf("unexpected suggestions: %#v", got)
	}

	if got := book.Suggest("docker"); len(got) != 0 {
		t.Fatalf("expected no suggestions, got %#v", got)
	}
}

func TestResolvePathPrefersTheEnvVar(t *testing.T) {
	t.Setenv(FilePathEnv, "/tmp/from-env.yaml")

	if got := ResolvePath(); got != "/tmp/from-env.yaml" {
		t.Fatalf("expected %s to win, got %q", FilePathEnv, got)
	}
}

func TestResolvePathFindsExistingDefaultFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv(FilePathEnv, "")
	t.Setenv("HOME", home)

	want := writeSheetFile(t, home, DefaultDirName, "cheat-sheets.json")

	if got := ResolvePath(); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolvePathDefaultsToTheDefaultFolder(t *testing.T) {
	home := t.TempDir()
	t.Setenv(FilePathEnv, "")
	t.Setenv("HOME", home)

	want := filepath.Join(home, DefaultDirName, "cheat-sheets.yaml")
	if got := ResolvePath(); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func writeSheetFile(t *testing.T, home, dir, name string) string {
	t.Helper()

	path := filepath.Join(home, dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("sheets: []"), 0o644); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestEnsureFileCreatesStarterAndKeepsExistingContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "cheat-sheets.yaml")

	if err := EnsureFile(path); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	book, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(book.Sheets) == 0 {
		t.Fatal("expected the starter file to have sheets")
	}

	if err := os.WriteFile(path, []byte("sheets: []"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureFile(path); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "sheets: []" {
		t.Fatalf("existing file was overwritten: %q", content)
	}
}
