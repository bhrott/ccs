// Package cli parses the arguments and wires the cheat sheet loading with the
// terminal rendering.
package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"

	"github.com/bhrott/ccs/internal/cheatsheet"
	"github.com/bhrott/ccs/internal/render"
)

// Version is the ccs version, overridable at build time with -ldflags.
var Version = "dev"

const usage = `ccs - your own cheat sheets in the terminal

Usage:
  ccs <sheet>          Print a cheat sheet as a table
  ccs <sheet> <term>   Print only the items matching a term
  ccs ls               List all cheat sheets

Options:
  --plain              Render without table borders
  --no-color           Render without colors
  --path               Print the cheat sheets path and exit
  -h, --help           Show this help
  -v, --version        Show the version

Files:
  Every cheat sheet is one file inside $CHEAT_SHEETS_FILE_PATH, or inside
  ~/.cheat-sheets when the variable is not set. The file name is the sheet
  id (tmux.yaml -> "ccs tmux") and "config.yaml" holds the colors.
  YAML and JSON are both supported, and the variable may also point to a
  single file with every sheet inside.`

type options struct {
	plain   bool
	noColor bool
	path    bool
	help    bool
	version bool
	args    []string
}

// Run executes the cli and returns the process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	opts, err := parse(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		fmt.Fprintln(stderr, "\nRun 'ccs --help' to see the available options.")
		return 1
	}

	if opts.noColor {
		color.NoColor = true
	}

	switch {
	case opts.help:
		fmt.Fprintln(stdout, usage)
		return 0
	case opts.version:
		fmt.Fprintln(stdout, "ccs "+Version)
		return 0
	}

	path := cheatsheet.ResolvePath()

	if opts.path {
		fmt.Fprintln(stdout, path)
		return 0
	}

	if err := cheatsheet.Ensure(path); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	book, err := cheatsheet.Load(path)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	theme, themeErrs := render.NewTheme(book.Config.Colors)
	for _, themeErr := range themeErrs {
		fmt.Fprintln(stderr, "warning:", themeErr)
	}

	renderOpts := render.Options{Theme: theme}
	if opts.plain {
		renderOpts.Style = render.StylePlain
	}

	if len(opts.args) == 0 {
		fmt.Fprintln(stderr, "Please provide a cheat sheet id (ex: 'ccs tmux') or run 'ccs ls'.")
		return 1
	}

	if isListCommand(opts.args[0]) {
		if len(book.Sheets) == 0 {
			fmt.Fprintf(stderr, "No cheat sheets registered yet. Create one like %s.\n", cheatsheet.SheetFilePath(path, "tmux"))
			return 1
		}

		render.List(stdout, book, renderOpts)
		return 0
	}

	return printSheet(stdout, stderr, book, opts, renderOpts, path)
}

func printSheet(stdout, stderr io.Writer, book cheatsheet.Book, opts options, renderOpts render.Options, path string) int {
	id := opts.args[0]
	term := strings.Join(opts.args[1:], " ")

	sheet, found := book.Find(id)
	if !found {
		fmt.Fprintf(stderr, "Cheat sheet %q not found in %s.\n", id, path)
		if suggestions := book.Suggest(id); len(suggestions) > 0 {
			fmt.Fprintf(stderr, "Did you mean: %s?\n", strings.Join(suggestions, ", "))
		}
		fmt.Fprintln(stderr, "Run 'ccs ls' to see all cheat sheets.")
		return 1
	}

	groups := cheatsheet.Filter(sheet.NormalizedGroups(), term)
	if len(groups) == 0 {
		if term != "" {
			fmt.Fprintf(stderr, "No items matching %q in cheat sheet %q.\n", term, sheet.ID)
		} else {
			fmt.Fprintf(stderr, "Cheat sheet %q has no items yet. Add some to %s.\n", sheet.ID, cheatsheet.SheetFilePath(path, sheet.ID))
		}
		return 1
	}

	render.Sheet(stdout, sheet, groups, renderOpts)
	return 0
}

func isListCommand(arg string) bool {
	switch strings.ToLower(arg) {
	case "ls", "list", "--ls", "--list", "-l":
		return true
	}

	return false
}

func parse(args []string) (options, error) {
	opts := options{}

	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") || isListCommand(arg) {
			opts.args = append(opts.args, arg)
			continue
		}

		switch arg {
		case "--plain":
			opts.plain = true
		case "--no-color":
			opts.noColor = true
		case "--path":
			opts.path = true
		case "-h", "--help":
			opts.help = true
		case "-v", "--version":
			opts.version = true
		default:
			return options{}, fmt.Errorf("unknown option: %s", arg)
		}
	}

	if !opts.plain && isTruthy(os.Getenv("CCS_PLAIN")) {
		opts.plain = true
	}

	return opts, nil
}

func isTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	}

	return false
}
