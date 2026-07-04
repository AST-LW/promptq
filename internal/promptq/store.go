package promptq

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const promptqHomeEnv = "PROMPTQ_HOME"

var validNameSegmentRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Prompt is a single stored prompt with optional metadata.
type Prompt struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Body        string   `json:"body"`
	UseCount    int      `json:"use_count"`
	LastUsed    int64    `json:"last_used"` // unix seconds; 0 = never
}

// Display returns a folder path with @ attached only to the prompt leaf.
func (p Prompt) Display() string { return displayName(p.Name) }

func displayName(name string) string {
	folder, _, found := strings.Cut(strings.TrimSpace(name), "/")
	if !found {
		return "@" + strings.TrimPrefix(folder, "@")
	}
	last := strings.LastIndex(name, "/")
	return name[:last+1] + "@" + strings.TrimPrefix(name[last+1:], "@")
}

// Folder returns the directory portion of a namespaced name, or "" if top-level.
func (p Prompt) Folder() string {
	if i := strings.LastIndex(p.Name, "/"); i >= 0 {
		return p.Name[:i]
	}
	return ""
}

// promptsDir returns the prompt storage directory, creating it if missing.
func promptsDir() (string, error) {
	root, err := promptqHome()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "prompts")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func promptqHome() (string, error) {
	if dir := strings.TrimSpace(os.Getenv(promptqHomeEnv)); dir != "" {
		clean := filepath.Clean(dir)
		if filepath.IsAbs(clean) {
			return clean, nil
		}
		return filepath.Abs(clean)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".promptq"), nil
}

// cleanName accepts @name or folder/@name. Internally stored names omit @.
func cleanName(name string) (string, error) {
	raw := strings.TrimSpace(name)
	if strings.HasPrefix(raw, "@") && strings.Contains(raw, "/") {
		return "", fmt.Errorf("invalid prompt name %q: use folder/@prompt", name)
	}
	n := strings.TrimPrefix(raw, "@")
	if n == "" {
		return "", fmt.Errorf("prompt name is empty")
	}
	if strings.Contains(n, "\\") || strings.HasPrefix(n, "/") || strings.HasSuffix(n, "/") || strings.Contains(n, "//") {
		return "", fmt.Errorf("invalid prompt name: %q", name)
	}
	parts := strings.Split(n, "/")
	parts[len(parts)-1] = strings.TrimPrefix(parts[len(parts)-1], "@")
	n = strings.Join(parts, "/")
	for _, seg := range parts {
		if !validNameSegmentRE.MatchString(seg) || strings.HasSuffix(seg, ".") || isWindowsReservedName(seg) {
			return "", fmt.Errorf("invalid prompt name: %q", name)
		}
	}
	return n, nil
}

func isWindowsReservedName(seg string) bool {
	stem, _, _ := strings.Cut(seg, ".")
	switch strings.ToUpper(stem) {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return true
	default:
		return false
	}
}

func promptPath(name string) (string, error) {
	clean, err := cleanName(name)
	if err != nil {
		return "", err
	}
	dir, err := promptsDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, filepath.FromSlash(clean)+".txt")
	if err := ensureInside(dir, path); err != nil {
		return "", err
	}
	return path, nil
}

func ensureInside(base, path string) error {
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return err
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(baseAbs, pathAbs)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("prompt path escapes storage root")
	}
	return nil
}

// save writes a prompt to disk, encoding metadata as a frontmatter header.
func save(p Prompt) error {
	if strings.ContainsAny(p.Description, "\r\n") {
		return fmt.Errorf("description must be a single line")
	}
	for _, tag := range p.Tags {
		if strings.ContainsAny(tag, "\r\n,") {
			return fmt.Errorf("tag %q contains an invalid character", tag)
		}
	}
	path, err := promptPath(p.Name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	var b strings.Builder
	// Always emit an empty-capable header so bodies beginning with "---" remain
	// unambiguously prompt content when loaded again.
	b.WriteString("---\n")
	if p.Description != "" {
		fmt.Fprintf(&b, "description: %s\n", p.Description)
	}
	if len(p.Tags) > 0 {
		fmt.Fprintf(&b, "tags: %s\n", strings.Join(p.Tags, ", "))
	}
	if p.UseCount > 0 {
		fmt.Fprintf(&b, "uses: %d\n", p.UseCount)
	}
	if p.LastUsed > 0 {
		fmt.Fprintf(&b, "last_used: %d\n", p.LastUsed)
	}
	b.WriteString("---\n")
	b.WriteString(p.Body)
	return writeFileAtomic(path, []byte(b.String()), 0o600)
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Chmod(perm); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		_ = os.Remove(path)
	}
	return os.Rename(tmp, path)
}

// bumpUsage records that a prompt was used: count+1, last_used=now.
func bumpUsage(name string) {
	p, err := load(name)
	if err != nil {
		return
	}
	p.UseCount++
	p.LastUsed = time.Now().Unix()
	_ = save(p)
}

// load reads and parses a single prompt by name.
func load(name string) (Prompt, error) {
	path, err := promptPath(name)
	if err != nil {
		return Prompt{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Prompt{}, fmt.Errorf("prompt %s not found", displayName(name))
		}
		return Prompt{}, err
	}
	return parse(name, string(data)), nil
}

// parse splits an optional frontmatter header from the body.
func parse(name, content string) Prompt {
	p := Prompt{Name: name}
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		end := -1
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				end = i
				break
			}
		}
		if end > 0 {
			for _, l := range lines[1:end] {
				k, v, ok := strings.Cut(l, ":")
				if !ok {
					continue
				}
				k, v = strings.TrimSpace(k), strings.TrimSpace(v)
				switch k {
				case "description":
					p.Description = v
				case "tags":
					for _, t := range strings.Split(v, ",") {
						if t = strings.TrimSpace(t); t != "" {
							p.Tags = append(p.Tags, t)
						}
					}
				case "uses":
					p.UseCount, _ = strconv.Atoi(v)
				case "last_used":
					p.LastUsed, _ = strconv.ParseInt(v, 10, 64)
				}
			}
			lines = lines[end+1:]
		}
	}
	p.Body = strings.TrimRight(strings.Join(lines, "\n"), "\n")
	return p
}

// list returns all stored prompts (recursively) sorted by name.
func list() ([]Prompt, error) {
	dir, err := promptsDir()
	if err != nil {
		return nil, err
	}
	var prompts []Prompt
	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".txt") {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(strings.TrimSuffix(rel, ".txt"))
		if _, err := cleanName(name); err != nil {
			return nil
		}
		if p, e := load(name); e == nil {
			prompts = append(prompts, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(prompts, func(i, j int) bool { return prompts[i].Name < prompts[j].Name })
	return prompts, nil
}

// remove deletes a prompt by name.
func remove(name string) error {
	path, err := promptPath(name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("prompt %s not found", displayName(name))
		}
		return err
	}
	return nil
}

func renamePrompt(oldName, newName string) error {
	if oldName == newName {
		return nil
	}
	p, err := load(oldName)
	if err != nil {
		return err
	}
	if _, err := load(newName); err == nil {
		return fmt.Errorf("prompt %s already exists", displayName(newName))
	}
	p.Name = newName
	if err := save(p); err != nil {
		return err
	}
	return remove(oldName)
}

func duplicatePrompt(srcName, dstName string) error {
	p, err := load(srcName)
	if err != nil {
		return err
	}
	if _, err := load(dstName); err == nil {
		return fmt.Errorf("prompt %s already exists", displayName(dstName))
	}
	p.Name = dstName
	p.UseCount = 0
	p.LastUsed = 0
	return save(p)
}
