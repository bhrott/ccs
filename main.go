package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/jedib0t/go-pretty/table"
	"gopkg.in/yaml.v2"
)

type Item struct {
	Command     string `yaml:"command"`
	Description string `yaml:"description"`
}

type Sheet struct {
	ID    string `yaml:"id"`
	Items []Item `yaml:"items"`
}

type Sheets struct {
	Sheets []Sheet `yaml:"sheets"`
}

const CHEAT_SHEET_FILE_NAME = "cheat-sheets.yaml"
const CHEAT_SHEET_DIR_PATH = "/.cli-cheat-sheets/"

func getFolderPath(fullPath string) string {
	return filepath.Dir(fullPath)
}

func getSheetId() string {
	if len(os.Args) < 2 {
		errorAndExit("Please provide a cheat sheet ID (ex: 'ccs go')")
	}

	return os.Args[1]
}

func getCheatSheetsFilePath() string {
	yamlFilePath := os.Getenv("CHEAT_SHEETS_FILE_PATH")

	if yamlFilePath == "" {
		yamlFilePath = os.Getenv("HOME") + CHEAT_SHEET_DIR_PATH + CHEAT_SHEET_FILE_NAME
	}

	return yamlFilePath
}

func getCheatSheets() Sheets {
	yamlFilePath := getCheatSheetsFilePath()

	yamlFile, err := os.ReadFile(yamlFilePath)
	if err != nil {
		errorAndExit("Failed to read cheat sheets file at path: " + yamlFilePath)
	}

	var sheets Sheets
	err = yaml.Unmarshal(yamlFile, &sheets)
	if err != nil {
		panic(err)
	}

	return sheets
}

func createCheatSheetsFileIfNotExists() {
	yamlFilePath := getCheatSheetsFilePath()

	baseYamlFile := []string{
		"sheets:",
		"  - id: cli-cheat-sheets",
		"    items:",
		"      - command: \"ccs --ls\"",
		"        description: \"List cheat sheets\"",
	}

	separator := "\n"
	yamlContent := strings.Join(baseYamlFile, separator)

	if _, err := os.Stat(yamlFilePath); os.IsNotExist(err) {
		folderPath := getFolderPath(yamlFilePath)

		os.MkdirAll(folderPath, 0755)
		os.WriteFile(yamlFilePath, []byte(yamlContent), 0644)
	}
}

func handleListCommand(sheets Sheets) {
	t := createDefaultTable()
	t.AppendHeader(table.Row{"Cheat Sheet"})

	for _, sheet := range sheets.Sheets {
		t.AppendRow(table.Row{sheet.ID})
	}

	t.Render()
}

func printCheatSheet(sheetCheets Sheets, sheetID string) {
	t := createDefaultTable()
	t.AppendHeader(table.Row{"Command", "Description"})

	found := false

	for _, sheet := range sheetCheets.Sheets {
		if sheet.ID == sheetID {
			found = true
			for _, item := range sheet.Items {
				t.AppendRow(table.Row{item.Command, item.Description})
			}
		}
	}

	if !found {
		errorAndExit("Cheat sheet not found")
	}

	t.Render()
}

func createDefaultTable() table.Writer {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleLight)
	t.Style().Options.SeparateRows = true

	return t
}

func errorAndExit(message string) {
	println(message)
	os.Exit(1)
}

func main() {
	createCheatSheetsFileIfNotExists()

	sheetID := getSheetId()
	sheetCheets := getCheatSheets()

	if sheetID == "--ls" {
		handleListCommand(sheetCheets)
		os.Exit(0)
	}

	printCheatSheet(sheetCheets, sheetID)

}
