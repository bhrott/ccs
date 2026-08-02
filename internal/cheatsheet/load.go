package cheatsheet

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// FilePathEnv is the env var pointing to the cheat sheets file.
const FilePathEnv = "CHEAT_SHEETS_FILE_PATH"

// DefaultDirName is the folder created under $HOME when no file is configured.
const DefaultDirName = ".cheat-sheets"

// defaultFileNames are looked up, in order, inside the default folder.
var defaultFileNames = []string{
	"cheat-sheets.yaml",
	"cheat-sheets.yml",
	"cheat-sheets.json",
}

const starterFile = `# Extra settings. Colors are hex and paint the commands table.
config:
  colors:
    title: "#5fd7ff"
    group: "#ff87d7"
    command: "#ffd75f"
    description: "#dadada"
    border: "#5f5f5f"

sheets:
  - id: ccs
    description: Commands of this cheat sheet tool
    groups:
      - name: Basics
        items:
          - command: ccs ls
            description: List all cheat sheets
          - command: ccs <sheet>
            description: Print a cheat sheet as a table
          - command: ccs <sheet> <term>
            description: Print only the items matching a term
      - name: Config
        items:
          - command: ccs --path
            description: Show which file is being read
          - command: export CHEAT_SHEETS_FILE_PATH=./my.yaml
            description: Read the sheets from another file
`

// ResolvePath returns the cheat sheets file path, from $CHEAT_SHEETS_FILE_PATH
// or from ~/.cheat-sheets. When nothing exists yet it returns the default yaml
// path, which is created on the first run.
func ResolvePath() string {
	if path := strings.TrimSpace(os.Getenv(FilePathEnv)); path != "" {
		return path
	}

	dir := filepath.Join(homeDir(), DefaultDirName)

	for _, name := range defaultFileNames {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return filepath.Join(dir, defaultFileNames[0])
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		home = os.Getenv("HOME")
	}

	return home
}

// EnsureFile creates the cheat sheets file with a starter content when it does
// not exist yet. It never overwrites an existing file.
func EnsureFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check cheat sheets file %q: %w", path, err)
	}

	if strings.EqualFold(filepath.Ext(path), ".json") {
		// A user pointing at a .json file is expected to bring their own.
		return nil
	}

	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create cheat sheets folder %q: %w", dir, err)
		}
	}

	if err := os.WriteFile(path, []byte(starterFile), 0o644); err != nil {
		return fmt.Errorf("create cheat sheets file %q: %w", path, err)
	}

	return nil
}

// Load reads and parses the cheat sheets file. YAML and JSON are both accepted:
// the format comes from the extension, falling back to sniffing the content.
func Load(path string) (Book, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Book{}, fmt.Errorf("read cheat sheets file %q: %w", path, err)
	}

	book, err := Parse(content, path)
	if err != nil {
		return Book{}, err
	}

	return book, nil
}

// Parse decodes the file content. The name is only used to pick the format and
// to build error messages.
func Parse(content []byte, name string) (Book, error) {
	var book Book

	if isJSON(content, name) {
		if err := json.Unmarshal(content, &book); err != nil {
			return Book{}, fmt.Errorf("parse cheat sheets file %q as json: %w", name, err)
		}

		return book, nil
	}

	if err := yaml.Unmarshal(content, &book); err != nil {
		return Book{}, fmt.Errorf("parse cheat sheets file %q as yaml: %w", name, err)
	}

	return book, nil
}

func isJSON(content []byte, name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".json":
		return true
	case ".yaml", ".yml":
		return false
	}

	return strings.HasPrefix(strings.TrimSpace(string(content)), "{")
}
