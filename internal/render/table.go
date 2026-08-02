// Package render prints cheat sheets to the terminal as a table.
package render

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/fatih/color"

	"github.com/bhrott/ccs/internal/cheatsheet"
)

// Style selects how the table borders are drawn.
type Style int

const (
	// StyleBox draws the table with box characters.
	StyleBox Style = iota
	// StylePlain drops the borders and keeps only the aligned columns. Useful
	// on terminals that mess up box drawing.
	StylePlain
)

// Options controls a single render call.
type Options struct {
	Style Style
	// Width is the terminal width used to wrap descriptions. Zero means the
	// detected width, and a negative value disables wrapping.
	Width int
	// Theme paints the table. The zero value uses the built-in colors, and so
	// does every color left empty on it.
	Theme Theme
}

// minDescriptionWidth is how narrow the description column may get before we
// give up on wrapping and just let the terminal do whatever it wants.
const minDescriptionWidth = 12

// Theme is the set of colors used to paint the table. Nil colors fall back to
// the built-in ones, so the zero value is a valid theme.
type Theme struct {
	title       *color.Color
	subtitle    *color.Color
	border      *color.Color
	header      *color.Color
	group       *color.Color
	command     *color.Color
	description *color.Color
}

// DefaultTheme returns the colors used when nothing is configured.
func DefaultTheme() Theme {
	return Theme{
		title:       color.New(color.FgCyan, color.Bold),
		subtitle:    color.New(color.Faint),
		border:      color.New(color.Faint),
		header:      color.New(color.FgWhite, color.Bold),
		group:       color.New(color.FgMagenta, color.Bold),
		command:     color.New(color.FgYellow),
		description: color.New(color.FgWhite),
	}
}

// NewTheme builds a theme from the hex colors configured in the file. Colors
// left empty keep the default, and invalid ones are reported as errors while
// the default is kept, so a typo never stops the output.
func NewTheme(colors cheatsheet.Colors) (Theme, []error) {
	theme := DefaultTheme()

	var errs []error
	apply := func(name, value string, target **color.Color, extra ...color.Attribute) {
		if strings.TrimSpace(value) == "" {
			return
		}

		parsed, err := parseHexColor(value)
		if err != nil {
			errs = append(errs, fmt.Errorf("config.colors.%s: %w", name, err))
			return
		}

		*target = parsed.Add(extra...)
	}

	apply("title", colors.Title, &theme.title, color.Bold)
	apply("group", colors.Group, &theme.group, color.Bold)
	apply("command", colors.Command, &theme.command)
	apply("description", colors.Description, &theme.description)
	apply("border", colors.Border, &theme.border)

	return theme, errs
}

// resolve fills the colors left nil with the default ones.
func (t Theme) resolve() Theme {
	fallback := DefaultTheme()

	for _, pair := range [][2]**color.Color{
		{&t.title, &fallback.title},
		{&t.subtitle, &fallback.subtitle},
		{&t.border, &fallback.border},
		{&t.header, &fallback.header},
		{&t.group, &fallback.group},
		{&t.command, &fallback.command},
		{&t.description, &fallback.description},
	} {
		if *pair[0] == nil {
			*pair[0] = *pair[1]
		}
	}

	return t
}

// parseHexColor accepts "#rgb", "#rrggbb" and the same values without the "#".
func parseHexColor(value string) (*color.Color, error) {
	hex := strings.TrimPrefix(strings.TrimSpace(value), "#")

	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}

	if len(hex) != 6 {
		return nil, fmt.Errorf("invalid hex color %q, expected something like \"#5fd7ff\"", value)
	}

	rgb, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid hex color %q, expected something like \"#5fd7ff\"", value)
	}

	return color.RGB(int(rgb>>16), int(rgb>>8&0xff), int(rgb&0xff)), nil
}

// Sheet prints the sheet header and every group as a table.
func Sheet(w io.Writer, sheet cheatsheet.Sheet, groups []cheatsheet.Group, opts Options) {
	p := opts.Theme.resolve()

	fmt.Fprintln(w, p.title.Sprint(sheet.ID))
	if description := strings.TrimSpace(sheet.Description); description != "" {
		fmt.Fprintln(w, p.subtitle.Sprint(description))
	}
	fmt.Fprintln(w)

	rows := make([][2]string, 0, 16)
	sections := make(map[int]string)

	for _, group := range groups {
		if name := strings.TrimSpace(group.Name); name != "" {
			sections[len(rows)] = name
		}

		for _, item := range group.Items {
			rows = append(rows, [2]string{
				strings.TrimSpace(item.Command),
				strings.TrimSpace(item.Description),
			})
		}
	}

	writeTable(w, p, [2]string{"COMMAND", "DESCRIPTION"}, rows, sections, opts)
}

// List prints every sheet id with its description and item count.
func List(w io.Writer, book cheatsheet.Book, opts Options) {
	p := opts.Theme.resolve()

	rows := make([][2]string, 0, len(book.Sheets))
	for _, sheet := range book.Sheets {
		description := strings.TrimSpace(sheet.Description)
		if description == "" {
			description = fmt.Sprintf("%d items", sheet.CountItems())
		}

		rows = append(rows, [2]string{strings.TrimSpace(sheet.ID), description})
	}

	writeTable(w, p, [2]string{"SHEET", "DESCRIPTION"}, rows, nil, opts)
}

// writeTable renders a two column table. sections maps a row index to a section
// title printed right before that row.
func writeTable(w io.Writer, p Theme, header [2]string, rows [][2]string, sections map[int]string, opts Options) {
	leftWidth := displayWidth(header[0])
	rightWidth := displayWidth(header[1])

	for _, row := range rows {
		leftWidth = max(leftWidth, displayWidth(row[0]))
		rightWidth = max(rightWidth, displayWidth(row[1]))
	}

	rightWidth = wrapWidth(leftWidth, rightWidth, opts)

	if opts.Style == StylePlain {
		writePlain(w, p, header, rows, sections, leftWidth, rightWidth)
		return
	}

	writeBox(w, p, header, rows, sections, leftWidth, rightWidth)
}

// wrapWidth shrinks the description column so the table fits the terminal.
func wrapWidth(leftWidth, rightWidth int, opts Options) int {
	width := opts.Width
	if width < 0 {
		return rightWidth
	}

	if width == 0 {
		width = TerminalWidth()
	}

	if width <= 0 {
		return rightWidth
	}

	// Borders and paddings: "| " + left + " | " + right + " |".
	available := width - leftWidth - 7
	if available < minDescriptionWidth || available >= rightWidth {
		return rightWidth
	}

	return available
}

func writeBox(w io.Writer, p Theme, header [2]string, rows [][2]string, sections map[int]string, leftWidth, rightWidth int) {
	line := func(left, middle, right string) string {
		return p.border.Sprint(left + strings.Repeat("─", leftWidth+2) + middle + strings.Repeat("─", rightWidth+2) + right)
	}

	bar := p.border.Sprint("│")
	spanWidth := leftWidth + rightWidth + 3

	fmt.Fprintln(w, line("┌", "┬", "┐"))
	fmt.Fprintf(w, "%s %s %s %s %s\n",
		bar, p.header.Sprint(pad(header[0], leftWidth)),
		bar, p.header.Sprint(pad(header[1], rightWidth)), bar)

	// A section starting on the first row draws its own separator, so the
	// header one would be a duplicated line.
	if _, startsWithSection := sections[0]; !startsWithSection {
		fmt.Fprintln(w, line("├", "┼", "┤"))
	}

	for index, row := range rows {
		if title, ok := sections[index]; ok {
			fmt.Fprintln(w, line("├", "┴", "┤"))
			fmt.Fprintf(w, "%s %s %s\n", bar, p.group.Sprint(pad(title, spanWidth)), bar)
			fmt.Fprintln(w, line("├", "┬", "┤"))
		}

		lines := wrap(row[1], rightWidth)
		for i, description := range lines {
			command := ""
			if i == 0 {
				command = row[0]
			}

			fmt.Fprintf(w, "%s %s %s %s %s\n",
				bar, p.command.Sprint(pad(command, leftWidth)),
				bar, p.description.Sprint(pad(description, rightWidth)), bar)
		}
	}

	fmt.Fprintln(w, line("└", "┴", "┘"))
}

func writePlain(w io.Writer, p Theme, header [2]string, rows [][2]string, sections map[int]string, leftWidth, rightWidth int) {
	fmt.Fprintf(w, "%s  %s\n", p.header.Sprint(pad(header[0], leftWidth)), p.header.Sprint(header[1]))
	fmt.Fprintln(w, p.border.Sprint(strings.Repeat("-", leftWidth+2+rightWidth)))

	for index, row := range rows {
		if title, ok := sections[index]; ok {
			if index > 0 {
				fmt.Fprintln(w)
			}
			fmt.Fprintln(w, p.group.Sprint(title))
		}

		lines := wrap(row[1], rightWidth)
		for i, description := range lines {
			command := ""
			if i == 0 {
				command = row[0]
			}

			fmt.Fprintf(w, "%s  %s\n", p.command.Sprint(pad(command, leftWidth)), p.description.Sprint(description))
		}
	}
}

// pad right pads the text with spaces up to width.
func pad(text string, width int) string {
	missing := width - displayWidth(text)
	if missing <= 0 {
		return text
	}

	return text + strings.Repeat(" ", missing)
}

// wrap breaks the text in lines of at most width, on word boundaries when it
// can. It always returns at least one line.
func wrap(text string, width int) []string {
	if width <= 0 || displayWidth(text) <= width {
		return []string{text}
	}

	var lines []string
	current := ""

	for _, word := range strings.Fields(text) {
		switch {
		case current == "":
			current = word
		case displayWidth(current)+1+displayWidth(word) <= width:
			current += " " + word
		default:
			lines = append(lines, current)
			current = word
		}

		for displayWidth(current) > width {
			head, tail := splitAt(current, width)
			lines = append(lines, head)
			current = tail
		}
	}

	if current != "" || len(lines) == 0 {
		lines = append(lines, current)
	}

	return lines
}

// splitAt cuts the text in two at the given display width.
func splitAt(text string, width int) (string, string) {
	runes := []rune(text)
	if width >= len(runes) {
		return text, ""
	}

	return string(runes[:width]), string(runes[width:])
}

func displayWidth(text string) int {
	return utf8.RuneCountInString(text)
}
