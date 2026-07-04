package promptq

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// parseUserName enforces the public CLI spelling: @name or folder/@name.
func parseUserName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", fmt.Errorf("prompt name is empty")
	}
	if strings.Contains(name, "/") {
		if strings.HasPrefix(name, "@") || !strings.HasPrefix(path.Base(name), "@") {
			return "", fmt.Errorf("invalid prompt name %q: use folder/@prompt", raw)
		}
	} else if !strings.HasPrefix(name, "@") {
		return "", fmt.Errorf("invalid prompt name %q: use @prompt", raw)
	}
	return cleanName(name)
}

// resolvePromptName finds an existing prompt. A root-style @name may fall back
// to a unique leaf with that name in any folder.
func resolvePromptName(raw string) (string, error) {
	wanted, err := parseUserName(raw)
	if err != nil {
		return "", err
	}
	prompts, err := list()
	if err != nil {
		return "", err
	}
	for _, p := range prompts {
		if p.Name == wanted {
			return wanted, nil
		}
	}
	if strings.Contains(wanted, "/") {
		return "", fmt.Errorf("prompt %s not found", displayName(wanted))
	}
	var matches []string
	for _, p := range prompts {
		if path.Base(p.Name) == wanted {
			matches = append(matches, p.Name)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("prompt %s not found", displayName(wanted))
	}
	sort.Strings(matches)
	displays := make([]string, len(matches))
	for i, name := range matches {
		displays[i] = displayName(name)
	}
	return "", fmt.Errorf("prompt %s is ambiguous; use one of: %s", displayName(wanted), strings.Join(displays, ", "))
}
