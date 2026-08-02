// Package cheatsheet holds the cheat sheet model and the rules used to
// normalize the many shapes a user file can have into a single one.
package cheatsheet

import (
	"regexp"
	"strings"
)

// Item is a single command and the help text that goes with it.
type Item struct {
	Command     string `yaml:"command" json:"command"`
	Description string `yaml:"description" json:"description"`
}

// Group is a named set of items, used to split a sheet in smaller blocks.
type Group struct {
	Name  string `yaml:"name" json:"name"`
	Items []Item `yaml:"items" json:"items"`
}

// Sheet is everything registered under one id, ex: "tmux".
type Sheet struct {
	ID          string  `yaml:"id" json:"id"`
	Description string  `yaml:"description" json:"description"`
	Groups      []Group `yaml:"groups" json:"groups"`
	// Items keeps the old flat format working: items declared directly on the
	// sheet, without groups.
	Items []Item `yaml:"items" json:"items"`
}

// Colors holds the hex colors used to paint the table. Empty fields keep the
// built-in color.
type Colors struct {
	Title       string `yaml:"title" json:"title"`
	Group       string `yaml:"group" json:"group"`
	Command     string `yaml:"command" json:"command"`
	Description string `yaml:"description" json:"description"`
	Border      string `yaml:"border" json:"border"`
}

// Config is the extra settings area of the file.
type Config struct {
	Colors Colors `yaml:"colors" json:"colors"`
}

// Book is the root of the file.
type Book struct {
	Config Config  `yaml:"config" json:"config"`
	Sheets []Sheet `yaml:"sheets" json:"sheets"`
}

// separatorItem matches the trick used before groups existed: an item with no
// description whose command is something like "-- SESSIONS --".
var separatorItem = regexp.MustCompile(`^\s*[-=*#]{2,}\s*(.*?)\s*[-=*#]{2,}\s*$`)

// NormalizedGroups returns the sheet as groups, whatever format it was written
// in. Flat items become groups when they use the "-- NAME --" separator, and
// anything before the first separator lands in a leading unnamed group.
func (s Sheet) NormalizedGroups() []Group {
	groups := groupsFromFlatItems(s.Items)

	for _, g := range s.Groups {
		items := make([]Item, 0, len(g.Items))
		for _, item := range g.Items {
			if strings.TrimSpace(item.Command) == "" && strings.TrimSpace(item.Description) == "" {
				continue
			}
			items = append(items, item)
		}

		if len(items) == 0 {
			continue
		}

		groups = append(groups, Group{Name: strings.TrimSpace(g.Name), Items: items})
	}

	return groups
}

func groupsFromFlatItems(items []Item) []Group {
	var groups []Group
	current := Group{}

	flush := func() {
		if len(current.Items) > 0 {
			groups = append(groups, current)
		}
	}

	for _, item := range items {
		command := strings.TrimSpace(item.Command)
		description := strings.TrimSpace(item.Description)

		if command == "" && description == "" {
			continue
		}

		if description == "" {
			if match := separatorItem.FindStringSubmatch(command); match != nil {
				flush()
				current = Group{Name: strings.TrimSpace(match[1])}
				continue
			}
		}

		current.Items = append(current.Items, item)
	}

	flush()

	return groups
}

// Find returns the sheet with the given id. Ids are matched case-insensitively.
func (b Book) Find(id string) (Sheet, bool) {
	for _, sheet := range b.Sheets {
		if strings.EqualFold(strings.TrimSpace(sheet.ID), strings.TrimSpace(id)) {
			return sheet, true
		}
	}

	return Sheet{}, false
}

// Suggest returns sheet ids similar to the given one, to help on typos.
func (b Book) Suggest(id string) []string {
	needle := strings.ToLower(strings.TrimSpace(id))
	if needle == "" {
		return nil
	}

	var suggestions []string
	for _, sheet := range b.Sheets {
		candidate := strings.TrimSpace(sheet.ID)
		lowered := strings.ToLower(candidate)

		if strings.Contains(lowered, needle) || strings.Contains(needle, lowered) {
			suggestions = append(suggestions, candidate)
		}
	}

	return suggestions
}

// Filter keeps only the items matching the given term, on the command or on the
// description. Empty groups are dropped.
func Filter(groups []Group, term string) []Group {
	needle := strings.ToLower(strings.TrimSpace(term))
	if needle == "" {
		return groups
	}

	var filtered []Group
	for _, group := range groups {
		var items []Item
		for _, item := range group.Items {
			haystack := strings.ToLower(item.Command + " " + item.Description)
			if strings.Contains(haystack, needle) {
				items = append(items, item)
			}
		}

		if len(items) > 0 {
			filtered = append(filtered, Group{Name: group.Name, Items: items})
		}
	}

	return filtered
}

// CountItems returns how many items the sheet has, across every group.
func (s Sheet) CountItems() int {
	total := 0
	for _, group := range s.NormalizedGroups() {
		total += len(group.Items)
	}

	return total
}
