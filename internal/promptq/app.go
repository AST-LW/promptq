package promptq

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

var Version = "0.1.0"

var ErrUsage = errors.New("usage")

var varRE = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_-]+)\s*\}\}`)

const asciiLogo = `
 ██████╗  ██████╗   ██████╗  ███╗   ███╗ ██████╗  ████████╗  ██████╗
 ██╔══██╗ ██╔══██╗ ██╔═══██╗ ████╗ ████║ ██╔══██╗ ╚══██╔══╝ ██╔═══██╗
 ██████╔╝ ██████╔╝ ██║   ██║ ██╔████╔██║ ██████╔╝    ██║    ██║   ██║
 ██╔═══╝  ██╔══██╗ ██║   ██║ ██║╚██╔╝██║ ██╔═══╝     ██║    ██║▄▄ ██║
 ██║      ██║  ██║ ╚██████╔╝ ██║ ╚═╝ ██║ ██║         ██║    ╚██████╔╝
 ╚═╝      ╚═╝  ╚═╝  ╚═════╝  ╚═╝     ╚═╝ ╚═╝         ╚═╝     ╚══▀▀═╝
`

func Run(args []string) error {
	if len(args) == 0 {
		usage()
		return ErrUsage
	}
	switch args[0] {
	case "studio":
		return runStudio()
	case "save":
		return cmdSave(args[1:])
	case "edit":
		return cmdEdit(args[1:])
	case "get":
		return cmdGet(args[1:])
	case "preview":
		return cmdPreview(args[1:])
	case "show":
		return cmdShow(args[1:])
	case "list":
		return cmdList()
	case "search":
		return cmdSearch(args[1:])
	case "recent":
		return cmdRecent()
	case "rename":
		return cmdRename(args[1:])
	case "duplicate":
		return cmdDuplicate(args[1:])
	case "rm":
		return cmdRemove(args[1:])
	case "help":
		usage()
		return nil
	case "version":
		fmt.Println("promptq", Version)
		return nil
	default:
		return fmt.Errorf("unknown command %q - run `promptq help`", args[0])
	}
}

func ErrorLine(err error) string {
	return errMark("Error:") + " " + err.Error()
}

func cmdSave(args []string) error {
	opts, err := parseSaveArgs(args)
	if err != nil {
		return err
	}
	name, desc, tags, body := opts.name, opts.description, opts.tags, opts.body

	if name == "" {
		return fmt.Errorf("name required - usage: promptq save <name> [-d desc] [-t tags] [-m body]")
	}
	n, err := parseUserName(name)
	if err != nil {
		return err
	}
	if body == "" {
		if isTerminal(os.Stdin) {
			fmt.Fprintln(os.Stderr, "Type the prompt body, then press Ctrl-D to save:")
		}
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read prompt body: %w", err)
		}
		body = strings.TrimRight(string(data), "\n")
	}
	if body == "" {
		return fmt.Errorf("prompt body is empty - pass -m \"text\" or pipe content")
	}
	p := Prompt{Name: n, Description: desc, Tags: splitTags(tags), Body: body}
	if existing, loadErr := load(n); loadErr == nil {
		p.UseCount = existing.UseCount
		p.LastUsed = existing.LastUsed
		if !opts.descriptionSet {
			p.Description = existing.Description
		}
		if !opts.tagsSet {
			p.Tags = existing.Tags
		}
	}
	if err := save(p); err != nil {
		return err
	}
	fmt.Println(okMark("✓"), "Saved "+displayName(n))
	return nil
}

type saveOptions struct {
	name           string
	description    string
	descriptionSet bool
	tags           string
	tagsSet        bool
	body           string
}

func parseSaveArgs(args []string) (saveOptions, error) {
	var opts saveOptions
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case strings.HasPrefix(arg, "--description="):
			opts.description = strings.TrimPrefix(arg, "--description=")
			opts.descriptionSet = true
		case strings.HasPrefix(arg, "--tags="):
			opts.tags = strings.TrimPrefix(arg, "--tags=")
			opts.tagsSet = true
		case strings.HasPrefix(arg, "--message="):
			opts.body = strings.TrimPrefix(arg, "--message=")
		case arg == "-d" || arg == "--description":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("%s requires a value", arg)
			}
			opts.description = args[i]
			opts.descriptionSet = true
		case arg == "-t" || arg == "--tags":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("%s requires a value", arg)
			}
			opts.tags = args[i]
			opts.tagsSet = true
		case arg == "-m" || arg == "--message":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("%s requires a value", arg)
			}
			opts.body = args[i]
		case strings.HasPrefix(arg, "-"):
			return opts, fmt.Errorf("unknown flag %q", arg)
		default:
			if opts.name != "" {
				return opts, fmt.Errorf("unexpected argument %q", arg)
			}
			opts.name = arg
		}
	}
	return opts, nil
}

// splitTags normalizes a comma-separated tag string.
func splitTags(tags string) []string {
	var out []string
	for _, t := range strings.Split(tags, ",") {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func cmdGet(args []string) error {
	preview := false
	var name string
	for _, arg := range args {
		if arg == "--preview" {
			if preview {
				return fmt.Errorf("--preview specified more than once")
			}
			preview = true
			continue
		}
		if strings.HasPrefix(arg, "-") {
			return fmt.Errorf("unknown flag %q", arg)
		}
		if name != "" {
			return fmt.Errorf("unexpected argument %q", arg)
		}
		name = arg
	}
	if name == "" {
		return fmt.Errorf("usage: promptq get @name")
	}
	n, err := resolvePromptName(name)
	if err != nil {
		return err
	}
	p, err := load(n)
	if err != nil {
		return err
	}
	out := p.Body
	if preview {
		fmt.Println(out)
		if isTerminal(os.Stdin) {
			fmt.Fprint(os.Stderr, "copy to clipboard? [y/N]: ")
			in := bufio.NewReader(os.Stdin)
			answer, _ := in.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "y" && answer != "yes" {
				return nil
			}
		}
	}
	if err := copyToClipboard(out); err != nil {
		fmt.Fprintln(os.Stderr, errMark("Warning:"), "clipboard unavailable:", err)
		fmt.Println(out)
		return nil
	}
	bumpUsage(n)
	fmt.Println(okMark("✓"), "Copied "+displayName(n))
	return nil
}

func cmdPreview(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: promptq preview @name")
	}
	n, err := resolvePromptName(args[0])
	if err != nil {
		return err
	}
	p, err := load(n)
	if err != nil {
		return err
	}
	fmt.Println(p.Body)
	return nil
}

func cmdShow(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: promptq show @name")
	}
	n, err := resolvePromptName(args[0])
	if err != nil {
		return err
	}
	p, err := load(n)
	if err != nil {
		return err
	}
	fmt.Println(p.Body)
	return nil
}

func cmdList() error {
	prompts, err := list()
	if err != nil {
		return err
	}
	if len(prompts) == 0 {
		fmt.Println("No prompts found. Create one with `promptq save @name`.")
		return nil
	}
	rows := make([][]string, 0, len(prompts))
	for _, p := range prompts {
		rows = append(rows, []string{
			p.Display(),
			truncate(p.Description, 48),
			strings.Join(p.Tags, ", "),
		})
	}
	fmt.Println(formatTable([]string{"NAME", "DESCRIPTION", "TAGS"}, rows))
	return nil
}

func cmdSearch(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: promptq search <query>")
	}
	q := strings.ToLower(strings.TrimSpace(strings.Join(args, " ")))
	if q == "" {
		return fmt.Errorf("search query is empty")
	}
	prompts, err := list()
	if err != nil {
		return err
	}
	var matches []Prompt
	for _, p := range prompts {
		if bestScore(p, q) >= 0 {
			matches = append(matches, p)
		}
	}
	if len(matches) == 0 {
		fmt.Println("No matching prompts.")
		return nil
	}
	rows := make([][]string, 0, len(matches))
	for _, p := range matches {
		rows = append(rows, []string{
			p.Display(),
			truncate(p.Description, 48),
			strings.Join(p.Tags, ", "),
		})
	}
	fmt.Println(formatTable([]string{"NAME", "DESCRIPTION", "TAGS"}, rows))
	return nil
}

func cmdRecent() error {
	prompts, err := list()
	if err != nil {
		return err
	}
	sort.SliceStable(prompts, func(i, j int) bool {
		if prompts[i].LastUsed != prompts[j].LastUsed {
			return prompts[i].LastUsed > prompts[j].LastUsed
		}
		return prompts[i].UseCount > prompts[j].UseCount
	})
	var recent []Prompt
	for _, p := range prompts {
		if p.UseCount == 0 && p.LastUsed == 0 {
			continue
		}
		recent = append(recent, p)
	}
	if len(recent) == 0 {
		fmt.Println("No recently used prompts.")
		return nil
	}
	rows := make([][]string, 0, len(recent))
	for _, p := range recent {
		lastUsed := "—"
		if p.LastUsed > 0 {
			lastUsed = time.Unix(p.LastUsed, 0).Local().Format("2006-01-02 15:04")
		}
		rows = append(rows, []string{p.Display(), fmt.Sprintf("%d", p.UseCount), lastUsed})
	}
	fmt.Println(formatTable([]string{"NAME", "USES", "LAST USED"}, rows))
	return nil
}

func cmdRename(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: promptq rename @old @new")
	}
	oldName, err := resolvePromptName(args[0])
	if err != nil {
		return err
	}
	newName, err := parseUserName(args[1])
	if err != nil {
		return err
	}
	if err := renamePrompt(oldName, newName); err != nil {
		return err
	}
	fmt.Println(okMark("✓"), "Renamed "+displayName(oldName)+" → "+displayName(newName))
	return nil
}

func cmdDuplicate(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: promptq duplicate @source @new")
	}
	src, err := resolvePromptName(args[0])
	if err != nil {
		return err
	}
	dst, err := parseUserName(args[1])
	if err != nil {
		return err
	}
	if err := duplicatePrompt(src, dst); err != nil {
		return err
	}
	fmt.Println(okMark("✓"), "Duplicated "+displayName(src)+" → "+displayName(dst))
	return nil
}

func cmdRemove(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: promptq rm @name")
	}
	n, err := resolvePromptName(args[0])
	if err != nil {
		return err
	}
	if err := remove(n); err != nil {
		return err
	}
	fmt.Println(okMark("✓"), "Removed "+displayName(n))
	return nil
}

func cmdEdit(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: promptq edit @name")
	}
	n, err := resolvePromptName(args[0])
	if err != nil {
		return err
	}
	path, err := promptPath(n)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}
	return openInEditor(path)
}

// isTerminal reports whether f is an interactive terminal (not a pipe/file).
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func usage() {
	fmt.Print(renderLogo() + `
PromptQ is a local prompt library with a terminal interface for saving, organizing, finding, and copying reusable prompts.

Studio:
  promptq studio                                     open PromptQ Studio

Terminal commands:
  promptq save @name [-d desc] [-t "a,b"] [-m body]   create or update a prompt
  promptq edit @name                                  edit a prompt in $VISUAL or $EDITOR
  promptq get  [--preview] @name                      copy the stored prompt unchanged
  promptq preview @name                               preview the stored prompt unchanged
  promptq show @name                                  print prompt body
  promptq list                                        list prompts
  promptq search <query>                              search prompts
  promptq recent                                      show recent prompts
  promptq rename @old @new                            rename a prompt
  promptq duplicate @source @new                      duplicate a prompt
  promptq rm @name                                    delete a prompt
  promptq version                                     show version
  promptq help                                        show help
`)
}
