package render

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/fatih/color"

	"github.com/bhrott/ccs/internal/cheatsheet"
)

func TestMain(m *testing.M) {
	color.NoColor = true
	m.Run()
}

func sampleGroups() []cheatsheet.Group {
	return []cheatsheet.Group{
		{Name: "Sessions", Items: []cheatsheet.Item{
			{Command: "tmux ls", Description: "List sessions"},
			{Command: "tmux a -t mysession", Description: "Attach"},
		}},
		{Name: "Panes", Items: []cheatsheet.Item{
			{Command: "ctrl+b z", Description: "Zoom pane"},
		}},
	}
}

func TestSheetRendersAlignedBoxTable(t *testing.T) {
	var buf bytes.Buffer

	sheet := cheatsheet.Sheet{ID: "tmux", Description: "Terminal multiplexer"}
	Sheet(&buf, sheet, sampleGroups(), Options{Width: -1})

	output := buf.String()
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	if lines[0] != "tmux" || lines[1] != "Terminal multiplexer" {
		t.Fatalf("unexpected header:\n%s", output)
	}

	width := utf8.RuneCountInString(lines[3])
	for _, line := range lines[3:] {
		if got := utf8.RuneCountInString(line); got != width {
			t.Fatalf("line %q has width %d, expected %d:\n%s", line, got, width, output)
		}
	}

	for _, want := range []string{"COMMAND", "DESCRIPTION", "Sessions", "Panes", "tmux a -t mysession"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q:\n%s", want, output)
		}
	}
}

func TestSheetWrapsDescriptionToTerminalWidth(t *testing.T) {
	var buf bytes.Buffer

	groups := []cheatsheet.Group{{Items: []cheatsheet.Item{
		{Command: "ctrl+b %", Description: "Split the current pane with a vertical line to create a horizontal layout"},
	}}}

	Sheet(&buf, cheatsheet.Sheet{ID: "tmux"}, groups, Options{Width: 40})

	for _, line := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
		if utf8.RuneCountInString(line) > 40 {
			t.Fatalf("line wider than the terminal (%d):\n%s", utf8.RuneCountInString(line), buf.String())
		}
	}
}

func TestSheetPlainStyleHasNoBorders(t *testing.T) {
	var buf bytes.Buffer

	Sheet(&buf, cheatsheet.Sheet{ID: "tmux"}, sampleGroups(), Options{Style: StylePlain, Width: -1})

	output := buf.String()
	if strings.ContainsAny(output, "│┌┐└┘├┤┬┴┼") {
		t.Fatalf("plain style should not draw borders:\n%s", output)
	}

	if !strings.Contains(output, "tmux ls              List sessions") {
		t.Fatalf("expected aligned columns:\n%s", output)
	}
}

func TestListPrintsSheetIDsAndCounts(t *testing.T) {
	var buf bytes.Buffer

	book := cheatsheet.Book{Sheets: []cheatsheet.Sheet{
		{ID: "tmux", Description: "Terminal multiplexer"},
		{ID: "go", Items: []cheatsheet.Item{{Command: "go test ./...", Description: "Run tests"}}},
	}}

	List(&buf, book, Options{Width: -1})

	output := buf.String()
	for _, want := range []string{"SHEET", "tmux", "Terminal multiplexer", "go", "1 items"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q:\n%s", want, output)
		}
	}
}

func TestNewThemeUsesTheConfiguredHexColors(t *testing.T) {
	theme, errs := NewTheme(cheatsheet.Colors{
		Title:       "#5fd7ff",
		Group:       "ff87d7",
		Command:     "#fd0",
		Description: "  #dadada  ",
		Border:      "#5f5f5f",
	})

	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	for name, expected := range map[string]struct {
		colored *color.Color
		want    string
	}{
		"title":       {theme.title, "\x1b[38;2;95;215;255;1m"},
		"group":       {theme.group, "\x1b[38;2;255;135;215;1m"},
		"command":     {theme.command, "\x1b[38;2;255;221;0m"},
		"description": {theme.description, "\x1b[38;2;218;218;218m"},
		"border":      {theme.border, "\x1b[38;2;95;95;95m"},
	} {
		expected.colored.EnableColor()

		if got := expected.colored.Sprint("x"); !strings.HasPrefix(got, expected.want+"x") {
			t.Fatalf("%s: got %q, want prefix %q", name, got, expected.want)
		}
	}
}

func TestNewThemeKeepsDefaultsOnEmptyAndInvalidColors(t *testing.T) {
	theme, errs := NewTheme(cheatsheet.Colors{Command: "not-a-color", Group: "#12345"})

	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %v", errs)
	}

	for _, err := range errs {
		if !strings.Contains(err.Error(), "config.colors.") {
			t.Fatalf("error should point at the config key: %v", err)
		}
	}

	defaults := DefaultTheme()
	resolved := theme.resolve()

	for name, pair := range map[string][2]*color.Color{
		"command":     {resolved.command, defaults.command},
		"group":       {resolved.group, defaults.group},
		"title":       {resolved.title, defaults.title},
		"description": {resolved.description, defaults.description},
	} {
		pair[0].EnableColor()
		pair[1].EnableColor()

		if pair[0].Sprint("x") != pair[1].Sprint("x") {
			t.Fatalf("%s should have kept the default color", name)
		}
	}
}

func TestZeroThemeRendersWithDefaults(t *testing.T) {
	var buf bytes.Buffer

	Sheet(&buf, cheatsheet.Sheet{ID: "tmux"}, sampleGroups(), Options{Width: -1})

	if !strings.Contains(buf.String(), "tmux ls") {
		t.Fatalf("expected the zero theme to render:\n%s", buf.String())
	}
}

func TestWrapBreaksLongWords(t *testing.T) {
	lines := wrap("supercalifragilistic word", 10)

	for _, line := range lines {
		if utf8.RuneCountInString(line) > 10 {
			t.Fatalf("line %q is wider than 10", line)
		}
	}

	if strings.Join(lines, "") != "supercalifragilisticword" {
		t.Fatalf("wrap lost content: %#v", lines)
	}
}
