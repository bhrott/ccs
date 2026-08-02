//go:build !windows

package render

import (
	"os"
	"strconv"

	"golang.org/x/sys/unix"
)

// TerminalWidth returns the width of the terminal, or 0 when it is unknown.
func TerminalWidth() int {
	if width := widthFromEnv(); width > 0 {
		return width
	}

	for _, file := range []*os.File{os.Stdout, os.Stderr} {
		size, err := unix.IoctlGetWinsize(int(file.Fd()), unix.TIOCGWINSZ)
		if err == nil && size.Col > 0 {
			return int(size.Col)
		}
	}

	return 0
}

func widthFromEnv() int {
	width, err := strconv.Atoi(os.Getenv("COLUMNS"))
	if err != nil {
		return 0
	}

	return width
}
