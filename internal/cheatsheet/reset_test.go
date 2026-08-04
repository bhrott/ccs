package cheatsheet

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// defaultsRepo builds a git repository with the given files inside the cheat
// sheets folder, and points the reset at it. A local path is a valid clone
// source, so no network is needed.
func defaultsRepo(t *testing.T, files map[string]string) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	repo := t.TempDir()
	dir := filepath.Join(repo, defaultsDir)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	git(t, repo, "init", "--initial-branch", defaultsBranch)
	git(t, repo, "config", "user.email", "ccs@example.com")
	git(t, repo, "config", "user.name", "ccs")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-m", "cheat sheets")

	previous := defaultsRepoURL
	defaultsRepoURL = repo
	t.Cleanup(func() { defaultsRepoURL = previous })
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestFetchDefaults(t *testing.T) {
	defaultsRepo(t, map[string]string{
		"config.yaml": "config:\n  colors: {}\n",
		"tmux.yaml":   "description: Terminal multiplexer\n",
		"README.md":   "ignored",
	})

	files, err := FetchDefaults()
	if err != nil {
		t.Fatalf("fetch defaults: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected only the sheet files, got %#v", files)
	}

	if files[0].Name != "config.yaml" || files[1].Name != "tmux.yaml" {
		t.Fatalf("expected the files sorted by name, got %q and %q", files[0].Name, files[1].Name)
	}

	if string(files[1].Content) != "description: Terminal multiplexer\n" {
		t.Fatalf("unexpected content: %q", files[1].Content)
	}
}

func TestFetchDefaultsWithoutSheets(t *testing.T) {
	defaultsRepo(t, map[string]string{"README.md": "nothing here"})

	if _, err := FetchDefaults(); err == nil {
		t.Fatal("expected an error when the folder has no cheat sheets")
	}
}

func TestFetchDefaultsWithoutTheFolder(t *testing.T) {
	defaultsRepo(t, map[string]string{"tmux.yaml": "description: Terminal multiplexer\n"})

	previous := defaultsDir
	defaultsDir = "missing"
	t.Cleanup(func() { defaultsDir = previous })

	if _, err := FetchDefaults(); err == nil {
		t.Fatal("expected an error when the repository has no cheat sheets folder")
	}
}

func TestFetchDefaultsWhenTheCloneFails(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	previous := defaultsRepoURL
	defaultsRepoURL = filepath.Join(t.TempDir(), "not-a-repo")
	t.Cleanup(func() { defaultsRepoURL = previous })

	if _, err := FetchDefaults(); err == nil {
		t.Fatal("expected an error when the clone fails")
	}
}

func TestWriteDefaultsOverwritesAndKeepsTheOtherFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sheets")

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	mine := filepath.Join(dir, "mine.yaml")
	if err := os.WriteFile(mine, []byte("description: Mine\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "tmux.yaml"), []byte("old"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	files := []RemoteFile{{Name: "tmux.yaml", Content: []byte("new")}}

	if existing := ExistingDefaults(dir, files); len(existing) != 1 || existing[0] != "tmux.yaml" {
		t.Fatalf("unexpected existing files: %#v", existing)
	}

	if err := WriteDefaults(dir, files); err != nil {
		t.Fatalf("write defaults: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "tmux.yaml"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if string(content) != "new" {
		t.Fatalf("expected the file to be overwritten, got %q", content)
	}

	if _, err := os.Stat(mine); err != nil {
		t.Fatalf("expected the other files to be kept: %v", err)
	}
}

func TestWriteDefaultsCreatesTheFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing", "sheets")

	if err := WriteDefaults(dir, []RemoteFile{{Name: "ccs.yaml", Content: []byte("description: ccs\n")}}); err != nil {
		t.Fatalf("write defaults: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "ccs.yaml")); err != nil {
		t.Fatalf("expected the file to be created: %v", err)
	}
}
