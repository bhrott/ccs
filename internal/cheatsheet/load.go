package cheatsheet

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v2"
)

// FilePathEnv is the env var pointing to the cheat sheets folder, or to a
// single file when you prefer to keep everything together.
const FilePathEnv = "CHEAT_SHEETS_FILE_PATH"

// DefaultDirName is the folder created under $HOME when nothing is configured.
const DefaultDirName = ".cheat-sheets"

// sheetExtensions are the file extensions read inside the folder.
var sheetExtensions = []string{".yaml", ".yml", ".json"}

// configBaseName is the file holding the settings shared by every sheet.
const configBaseName = "config"

const starterConfigFile = `# Settings shared by every cheat sheet. Colors are hex and paint the table.
config:
  colors:
    title: "#5fd7ff"
    group: "#ff87d7"
    command: "#ffd75f"
    description: "#dadada"
    border: "#5f5f5f"
`

// starterSheetFile is one cheat sheet: the file name is the id, so this one is
// printed with "ccs ccs".
const starterSheetFile = `description: Commands of this cheat sheet tool

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
        description: Show which folder is being read
      - command: export CHEAT_SHEETS_FILE_PATH=./my-sheets
        description: Read the sheets from another folder
      - command: ccs reset
        description: Download the default cheat sheets from github
`

// ResolvePath returns the cheat sheets folder, from $CHEAT_SHEETS_FILE_PATH or
// from ~/.cheat-sheets. The env var may also point to a single file, and then
// every sheet is read from it.
func ResolvePath() string {
	if path := strings.TrimSpace(os.Getenv(FilePathEnv)); path != "" {
		return path
	}

	return filepath.Join(homeDir(), DefaultDirName)
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		home = os.Getenv("HOME")
	}

	return home
}

// IsFolder tells how the path is read: as a folder with one file per sheet, or
// as a single file with every sheet inside. An existing path answers for
// itself, and a missing one is a folder unless it looks like a file.
func IsFolder(path string) bool {
	if info, err := os.Stat(path); err == nil {
		return info.IsDir()
	}

	return !hasSheetExtension(path)
}

// SheetFilePath returns where the sheet with the given id is expected to live,
// used on the messages that tell the user where to write. On a single file
// setup it is the file itself.
func SheetFilePath(path, id string) string {
	if !IsFolder(path) {
		return path
	}

	id = strings.TrimSpace(id)
	if id == "" {
		id = "tmux"
	}

	return filepath.Join(path, id+".yaml")
}

// Ensure creates the cheat sheets folder with a starter sheet when there is
// nothing there yet. It never overwrites an existing file.
func Ensure(path string) error {
	if !IsFolder(path) {
		return ensureFile(path)
	}

	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("create cheat sheets folder %q: %w", path, err)
	}

	files, err := sheetFiles(path)
	if err != nil {
		return err
	}

	if len(files) > 0 {
		return nil
	}

	if err := writeIfMissing(filepath.Join(path, "config.yaml"), starterConfigFile); err != nil {
		return err
	}

	return writeIfMissing(filepath.Join(path, "ccs.yaml"), starterSheetFile)
}

func ensureFile(path string) error {
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

	return writeIfMissing(path, "config:\n  colors: {}\n\nsheets: []\n")
}

func writeIfMissing(path, content string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check cheat sheets file %q: %w", path, err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("create cheat sheets file %q: %w", path, err)
	}

	return nil
}

// Load reads the cheat sheets from a folder, one file per sheet, or from a
// single file holding every sheet. YAML and JSON are both accepted.
func Load(path string) (Book, error) {
	if IsFolder(path) {
		return loadDir(path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return Book{}, fmt.Errorf("read cheat sheets file %q: %w", path, err)
	}

	return ParseFile(content, path)
}

// loadDir reads every sheet file of the folder, in file name order. The
// "config" file is the shared settings, every other file is one cheat sheet.
func loadDir(dir string) (Book, error) {
	files, err := sheetFiles(dir)
	if err != nil {
		return Book{}, err
	}

	var book Book

	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			return Book{}, fmt.Errorf("read cheat sheets file %q: %w", path, err)
		}

		if strings.TrimSpace(string(content)) == "" {
			continue
		}

		parsed, err := ParseFile(content, path)
		if err != nil {
			return Book{}, err
		}

		book.Config.Colors = mergeColors(book.Config.Colors, parsed.Config.Colors)
		book.Sheets = append(book.Sheets, parsed.Sheets...)
	}

	return book, nil
}

// sheetFiles lists the readable files of the folder, sorted by name, with the
// config file first so its colors win over the ones of a sheet file.
func sheetFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("read cheat sheets folder %q: %w", dir, err)
	}

	var config string
	var sheets []string

	for _, entry := range entries {
		name := entry.Name()

		if entry.IsDir() || strings.HasPrefix(name, ".") || !hasSheetExtension(name) {
			continue
		}

		if isConfigName(name) {
			config = filepath.Join(dir, name)
			continue
		}

		sheets = append(sheets, filepath.Join(dir, name))
	}

	if config == "" {
		return sheets, nil
	}

	return append([]string{config}, sheets...), nil
}

func hasSheetExtension(name string) bool {
	return slices.Contains(sheetExtensions, strings.ToLower(filepath.Ext(name)))
}

func isConfigName(name string) bool {
	return strings.EqualFold(strings.TrimSuffix(name, filepath.Ext(name)), configBaseName)
}

// ParseFile decodes one file. A file with a "sheets:" list is a whole book,
// anything else is a single cheat sheet whose id defaults to the file name.
func ParseFile(content []byte, name string) (Book, error) {
	if looksLikeBook(content, name) {
		return Parse(content, name)
	}

	var file sheetFile
	if err := decode(content, name, &file); err != nil {
		return Book{}, err
	}

	sheet := file.Sheet
	if strings.TrimSpace(sheet.ID) == "" {
		sheet.ID = sheetIDFromName(name)
	}

	book := Book{Config: file.Config}
	if sheet.CountItems() > 0 || strings.TrimSpace(sheet.Description) != "" {
		book.Sheets = []Sheet{sheet}
	}

	return book, nil
}

// Parse decodes a file holding the whole book. The name is only used to pick
// the format and to build error messages.
func Parse(content []byte, name string) (Book, error) {
	var book Book

	if err := decode(content, name, &book); err != nil {
		return Book{}, err
	}

	for index, sheet := range book.Sheets {
		if strings.TrimSpace(sheet.ID) == "" {
			book.Sheets[index].ID = sheetIDFromName(name)
		}
	}

	return book, nil
}

// sheetFile is the shape of a single sheet file: the sheet fields on the root,
// plus an optional config so a folder with one sheet still can be themed.
type sheetFile struct {
	Sheet  `yaml:",inline" json:",inline"`
	Config Config `yaml:"config" json:"config"`
}

// looksLikeBook reports whether the file declares a "sheets:" list, the format
// used before every sheet had its own file.
func looksLikeBook(content []byte, name string) bool {
	var probe struct {
		Sheets *[]Sheet `yaml:"sheets" json:"sheets"`
	}

	if err := decode(content, name, &probe); err != nil {
		// A broken file is decoded again by the caller, which reports the error.
		return false
	}

	return probe.Sheets != nil
}

func decode(content []byte, name string, target any) error {
	if isJSON(content, name) {
		if err := json.Unmarshal(content, target); err != nil {
			return fmt.Errorf("parse cheat sheets file %q as json: %w", name, err)
		}

		return nil
	}

	if err := yaml.Unmarshal(content, target); err != nil {
		return fmt.Errorf("parse cheat sheets file %q as yaml: %w", name, err)
	}

	return nil
}

// sheetIDFromName turns "~/.cheat-sheets/tmux.yaml" into "tmux".
func sheetIDFromName(name string) string {
	base := filepath.Base(name)

	return strings.TrimSuffix(base, filepath.Ext(base))
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

// mergeColors keeps the colors already set and fills the empty ones.
func mergeColors(base, extra Colors) Colors {
	for _, pair := range [][2]*string{
		{&base.Title, &extra.Title},
		{&base.Group, &extra.Group},
		{&base.Command, &extra.Command},
		{&base.Description, &extra.Description},
		{&base.Border, &extra.Border},
	} {
		if strings.TrimSpace(*pair[0]) == "" {
			*pair[0] = *pair[1]
		}
	}

	return base
}
