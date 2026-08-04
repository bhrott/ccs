package cheatsheet

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DefaultsURL is the folder holding the default cheat sheets, printed on the
// messages so the user knows what is being downloaded.
const DefaultsURL = "https://github.com/bhrott/ccs/tree/main/cheat-sheets"

// defaultsRepoURL is the repository cloned to read that folder, and
// defaultsBranch and defaultsDir say where the sheets live inside it. They are
// vars so the tests can clone a local repository instead.
var (
	defaultsRepoURL = "https://github.com/bhrott/ccs.git"
	defaultsBranch  = "main"
	defaultsDir     = "cheat-sheets"
)

// cloneTimeout stops a clone that hangs, so the cli always gives an answer.
var cloneTimeout = 2 * time.Minute

// RemoteFile is one cheat sheet read from the clone, ready to be written to
// the cheat sheets folder.
type RemoteFile struct {
	Name    string
	Content []byte
}

// FetchDefaults clones the repository into a temporary folder and reads the
// cheat sheets published there. The clone is removed before returning, the
// files are kept in memory so nothing is written when a step fails halfway.
func FetchDefaults() ([]RemoteFile, error) {
	tmp, err := os.MkdirTemp("", "ccs-defaults-")
	if err != nil {
		return nil, fmt.Errorf("create a temporary folder for the clone: %w", err)
	}
	defer os.RemoveAll(tmp)

	// git wants to create the folder itself, so the clone goes in a child of it.
	clone := filepath.Join(tmp, "repo")

	if err := cloneDefaults(clone); err != nil {
		return nil, err
	}

	files, err := readDefaults(filepath.Join(clone, defaultsDir))
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no default cheat sheets found at %s", DefaultsURL)
	}

	return files, nil
}

// cloneDefaults runs the git of the machine, with the shallowest clone that
// still gives the folder: one branch, one commit.
func cloneDefaults(dir string) error {
	git, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("git is needed to download the default cheat sheets: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cloneTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, git, "clone",
		"--depth", "1",
		"--single-branch",
		"--branch", defaultsBranch,
		defaultsRepoURL, dir,
	)

	// Never stop on a credentials prompt, the repository is public.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("clone %s: timed out after %s", defaultsRepoURL, cloneTimeout)
		}

		return fmt.Errorf("clone %s: %w%s", defaultsRepoURL, err, gitError(stderr.String()))
	}

	return nil
}

// gitError appends what git wrote to stderr, so the reason of a failed clone
// (no network, unknown host, ...) reaches the user.
func gitError(stderr string) string {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return ""
	}

	return "\n" + stderr
}

// readDefaults reads the cheat sheet files of the cloned folder, in file name
// order. Anything that is not a sheet file is left behind.
func readDefaults(dir string) ([]RemoteFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no default cheat sheets found at %s", DefaultsURL)
		}

		return nil, fmt.Errorf("read the cloned cheat sheets: %w", err)
	}

	var files []RemoteFile

	for _, entry := range entries {
		name := entry.Name()

		if entry.IsDir() || strings.HasPrefix(name, ".") || !hasSheetExtension(name) {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("read the cloned cheat sheets: %w", err)
		}

		files = append(files, RemoteFile{Name: name, Content: content})
	}

	return files, nil
}

// ExistingDefaults returns the names of the files that already exist in the
// folder and would be overwritten by a reset.
func ExistingDefaults(dir string, files []RemoteFile) []string {
	var existing []string

	for _, file := range files {
		if _, err := os.Stat(filepath.Join(dir, file.Name)); err == nil {
			existing = append(existing, file.Name)
		}
	}

	return existing
}

// WriteDefaults copies the files read from the clone into the cheat sheets
// folder, overwriting the ones already there. The other files of the folder
// are left untouched.
func WriteDefaults(dir string, files []RemoteFile) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create cheat sheets folder %q: %w", dir, err)
	}

	for _, file := range files {
		path := filepath.Join(dir, file.Name)

		if err := os.WriteFile(path, file.Content, 0o644); err != nil {
			return fmt.Errorf("write cheat sheets file %q: %w", path, err)
		}
	}

	return nil
}
