//go:build windows

package render

import (
	"os"
	"strconv"
)

// TerminalWidth returns the width of the terminal, or 0 when it is unknown.
func TerminalWidth() int {
	width, err := strconv.Atoi(os.Getenv("COLUMNS"))
	if err != nil {
		return 0
	}

	return width
}
