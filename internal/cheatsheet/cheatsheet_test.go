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

func TestParseFileTakesTheSheetIDFromTheFileName(t *testing.T) {
	content := []byte(`
description: Terminal multiplexer

groups:
  - name: Sessions
    items:
      - command: tmux ls
        description: List sessions
`)

	book, err := ParseFile(content, filepath.Join("sheets", "tmux.yaml"))
	if err != nil {
		t.Fatalf("parse file: %v", err)
	}

	sheet, found := book.Find("tmux")
	if !found {
		t.Fatalf("expected the id to come from the file name, got %#v", book.Sheets)
	}

	if sheet.Description != "Terminal multiplexer" || sheet.CountItems() != 1 {
		t.Fatalf("unexpected sheet: %#v", sheet)
	}
}

func TestParseFileKeepsAnExplicitID(t *testing.T) {
	content := []byte(`
id: tmux
items:
  - command: tmux ls
    description: List sessions
`)

	book, err := ParseFile(content, filepath.Join("sheets", "my-notes.yaml"))
	if err != nil {
		t.Fatalf("parse file: %v", err)
	}

	if _, found := book.Find("tmux"); !found {
		t.Fatalf("expected the explicit id to win, got %#v", book.Sheets)
	}
}

func TestParseFileStillReadsTheWholeBookFormat(t *testing.T) {
	content := []byte(`
config:
  colors:
    title: "#5fd7ff"

sheets:
  - id: tmux
    items:
      - command: tmux ls
        description: List sessions
  - id: go
    items:
      - command: go test ./...
        description: Run the tests
`)

	book, err := ParseFile(content, filepath.Join("sheets", "everything.yaml"))
	if err != nil {
		t.Fatalf("parse file: %v", err)
	}

	if len(book.Sheets) != 2 {
		t.Fatalf("expected 2 sheets, got %#v", book.Sheets)
	}

	if book.Config.Colors.Title != "#5fd7ff" {
		t.Fatalf("expected the config to be kept, got %#v", book.Config)
	}
}

func TestLoadFolderReadsOneFilePerSheet(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "config.yaml", `
config:
  colors:
    title: "#5fd7ff"
`)
	writeFile(t, dir, "tmux.yaml", `
description: Terminal multiplexer
items:
  - command: tmux ls
    description: List sessions
`)
	writeFile(t, dir, "go.json", `{"items":[{"command":"go test ./...","description":"Run the tests"}]}`)
	writeFile(t, dir, "notes.txt", "not a sheet")
	writeFile(t, dir, ".hidden.yaml", "items: []")
	writeFile(t, dir, "empty.yaml", "\n")

	book, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(book.Sheets) != 2 {
		t.Fatalf("expected the tmux and go sheets, got %#v", book.Sheets)
	}

	if book.Config.Colors.Title != "#5fd7ff" {
		t.Fatalf("expected config.yaml to be read, got %#v", book.Config)
	}

	tmux, found := book.Find("tmux")
	if !found || tmux.Description != "Terminal multiplexer" {
		t.Fatalf("unexpected tmux sheet: %#v", tmux)
	}

	if _, found := book.Find("go"); !found {
		t.Fatalf("expected the go sheet from the json file, got %#v", book.Sheets)
	}
}

func TestLoadFolderKeepsALegacyFileWithEverythingInside(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "cheat-sheets.yaml", `
config:
  colors:
    group: "#ff87d7"

sheets:
  - id: tmux
    items:
      - command: tmux ls
        description: List sessions
`)

	book, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if _, found := book.Find("tmux"); !found {
		t.Fatalf("expected the legacy file to keep working, got %#v", book.Sheets)
	}

	if book.Config.Colors.Group != "#ff87d7" {
		t.Fatalf("expected the legacy config to be kept, got %#v", book.Config)
	}
}

func TestLoadSingleFileFromTheEnvVar(t *testing.T) {
	path := filepath.Join(t.TempDir(), "my-sheets.yaml")
	if err := os.WriteFile(path, []byte("sheets:\n  - id: tmux\n    items:\n      - command: tmux ls\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	book, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if _, found := book.Find("tmux"); !found {
		t.Fatalf("expected the single file to be read, got %#v", book.Sheets)
	}
}

func TestResolvePathPrefersTheEnvVar(t *testing.T) {
	t.Setenv(FilePathEnv, "/tmp/from-env")

	if got := ResolvePath(); got != "/tmp/from-env" {
		t.Fatalf("expected %s to win, got %q", FilePathEnv, got)
	}
}

func TestResolvePathDefaultsToTheDefaultFolder(t *testing.T) {
	home := t.TempDir()
	t.Setenv(FilePathEnv, "")
	t.Setenv("HOME", home)

	want := filepath.Join(home, DefaultDirName)
	if got := ResolvePath(); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestIsFolder(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "tmux.yaml", "items: []")

	for path, want := range map[string]bool{
		dir:                                true,
		filepath.Join(dir, "tmux.yaml"):    false,
		filepath.Join(dir, "missing"):      true,
		filepath.Join(dir, "missing.yaml"): false,
	} {
		if got := IsFolder(path); got != want {
			t.Fatalf("IsFolder(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestSheetFilePath(t *testing.T) {
	dir := t.TempDir()

	if got, want := SheetFilePath(dir, "tmux"), filepath.Join(dir, "tmux.yaml"); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}

	file := filepath.Join(dir, "everything.yaml")
	if got := SheetFilePath(file, "tmux"); got != file {
		t.Fatalf("expected the single file itself, got %q", got)
	}
}

func TestEnsureCreatesTheStarterFolderAndKeepsExistingContent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", ".cheat-sheets")

	if err := Ensure(dir); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	book, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(book.Sheets) == 0 {
		t.Fatal("expected the starter folder to have a sheet")
	}

	if book.Config.Colors.Title == "" {
		t.Fatalf("expected the starter config to be read, got %#v", book.Config)
	}

	if err := os.WriteFile(filepath.Join(dir, "ccs.yaml"), []byte("items: []"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Ensure(dir); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "ccs.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "items: []" {
		t.Fatalf("existing file was overwritten: %q", content)
	}
}

func TestEnsureKeepsWorkingWithASingleFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "cheat-sheets.yaml")

	if err := Ensure(path); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	if _, err := Load(path); err != nil {
		t.Fatalf("load: %v", err)
	}

	if err := os.WriteFile(path, []byte("sheets: []"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Ensure(path); err != nil {
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

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
