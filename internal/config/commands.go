package config

import "fmt"

// CommandEntry is a user-defined shell command on a remote host.
type CommandEntry struct {
	Name  string // display name (dedupe key within group)
	Group string // logical grouping label
	Run   string // shell command string
}

// MergeCommands merges command entries from defaults, group, and host levels.
// Dedupe key is group+"/"+name: same name in different groups = different commands.
// Host overrides group, group overrides defaults. New entries are additive.
func MergeCommands(defaults, group, host []CommandEntry) []CommandEntry {
	seen := make(map[string]int) // "group/name" → index in result
	var result []CommandEntry

	add := func(entries []CommandEntry) {
		for _, e := range entries {
			key := e.Group + "/" + e.Name
			if idx, ok := seen[key]; ok {
				result[idx] = e
			} else {
				seen[key] = len(result)
				result = append(result, e)
			}
		}
	}

	add(defaults)
	add(group)
	add(host)

	return result
}

// ValidateCommand checks that a command entry has all required fields.
func ValidateCommand(c CommandEntry) error {
	if c.Name == "" {
		return fmt.Errorf("command missing required field 'name'")
	}
	if c.Group == "" {
		return fmt.Errorf("command %q missing required field 'group'", c.Name)
	}
	if c.Run == "" {
		return fmt.Errorf("command %q missing required field 'run'", c.Name)
	}
	return nil
}
