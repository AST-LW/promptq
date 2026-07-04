package promptq

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

var (
	cAccent = lipgloss.Color("80")
	cVar    = lipgloss.Color("215")
	cText   = lipgloss.Color("252")
	cMeta   = lipgloss.Color("245")
	cDim    = lipgloss.Color("240")
	cFaint  = lipgloss.Color("237")
	cTag    = lipgloss.Color("66")
	cOK     = lipgloss.Color("114")
	cBgSel  = lipgloss.Color("236")
)

var (
	stTitle  = lipgloss.NewStyle().Bold(true).Foreground(cAccent)
	stCount  = lipgloss.NewStyle().Foreground(cDim)
	stItem   = lipgloss.NewStyle().Foreground(cText)
	stSel    = lipgloss.NewStyle().Foreground(cAccent).Bold(true).Background(cBgSel)
	stMeta   = lipgloss.NewStyle().Foreground(cMeta)
	stTags   = lipgloss.NewStyle().Foreground(cTag)
	stRule   = lipgloss.NewStyle().Foreground(cFaint)
	stHelp   = lipgloss.NewStyle().Foreground(cDim)
	stPanel  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cFaint).Padding(1, 2)
	stStatus = lipgloss.NewStyle().Foreground(cOK)
	stAccent = lipgloss.NewStyle().Foreground(cAccent)
	stSearch = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cFaint).Padding(0, 1)
)

var gradient = []lipgloss.Color{"43", "44", "80", "74", "75", "111", "147"}

func banner() string {
	word := "promptq"
	var b strings.Builder
	for i, r := range word {
		c := gradient[i%len(gradient)]
		b.WriteString(lipgloss.NewStyle().Foreground(c).Bold(true).Render(string(r)))
	}
	badge := lipgloss.NewStyle().
		Foreground(lipgloss.Color("16")).
		Background(cAccent).
		Bold(true).
		Padding(0, 1).
		Render("STUDIO")
	mark := lipgloss.NewStyle().Foreground(cAccent).Render("◆ ")
	return mark + b.String() + " " + badge
}

type mode int

const (
	modeList mode = iota
	modeForm
	modeConfirm
)

type studio struct {
	all      []Prompt
	filtered []Prompt
	cursor   int
	search   textinput.Model
	w, h     int
	status   string
	quit     bool
	expanded map[string]bool

	mode       mode
	editing    bool
	standalone bool
	origName   string
	field      int
	fName      textinput.Model
	fDesc      textinput.Model
	fTags      textinput.Model
	fBody      textarea.Model
	formErr    string
}

func newStudio(prompts []Prompt) studio {
	ti := textinput.New()
	ti.Placeholder = "Search your prompt directory"
	ti.Prompt = ""
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(cDim)
	ti.TextStyle = lipgloss.NewStyle().Foreground(cText)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(cAccent)
	ti.Focus()
	s := studio{all: prompts, search: ti, mode: modeList, expanded: map[string]bool{}}
	for _, p := range prompts {
		parts := strings.Split(p.Folder(), "/")
		for i := range parts {
			if parts[i] != "" {
				s.expanded[strings.Join(parts[:i+1], "/")] = true
			}
		}
	}
	s.fName = textinput.New()
	s.fName.Placeholder = "@name or folder/@name"
	s.fName.Prompt = ""
	s.fDesc = textinput.New()
	s.fDesc.Placeholder = "one-line summary (optional)"
	s.fDesc.Prompt = ""
	s.fTags = textinput.New()
	s.fTags.Placeholder = "comma, separated (optional)"
	s.fTags.Prompt = ""
	s.fBody = textarea.New()
	s.fBody.Placeholder = "write your prompt - {{variables}} stay unchanged"
	s.fBody.Prompt = ""
	s.fBody.ShowLineNumbers = false
	s.refilter()
	return s
}

func (s studio) Init() tea.Cmd { return textinput.Blink }

func (s *studio) refilter() {
	raw := strings.TrimSpace(s.search.Value())
	field, q := "", strings.ToLower(raw)
	if len(raw) > 0 {
		switch raw[0] {
		case '@':
			field, q = "name", strings.ToLower(raw[1:])
		case '#':
			field, q = "tag", strings.ToLower(raw[1:])
		case '/':
			field, q = "folder", strings.ToLower(raw[1:])
		}
	}
	if q == "" {
		s.filtered = append([]Prompt(nil), s.all...)
		sort.SliceStable(s.filtered, func(i, j int) bool {
			a, b := s.filtered[i], s.filtered[j]
			if a.UseCount != b.UseCount {
				return a.UseCount > b.UseCount
			}
			if a.LastUsed != b.LastUsed {
				return a.LastUsed > b.LastUsed
			}
			return a.Name < b.Name
		})
	} else {
		type sc struct {
			p Prompt
			n int
		}
		var hits []sc
		for _, p := range s.all {
			var n int
			switch field {
			case "name":
				n = fuzzyScore(strings.ToLower(p.Name), q)
			case "tag":
				n = fuzzyScore(strings.ToLower(strings.Join(p.Tags, " ")), q)
			case "folder":
				n = fuzzyScore(strings.ToLower(p.Folder()), q)
			default:
				n = bestScore(p, q)
			}
			if n >= 0 {
				n -= p.UseCount
				hits = append(hits, sc{p, n})
			}
		}
		sort.SliceStable(hits, func(i, j int) bool { return hits[i].n < hits[j].n })
		s.filtered = s.filtered[:0]
		for _, h := range hits {
			s.filtered = append(s.filtered, h.p)
		}
	}
	if rows := s.treeRows(); s.cursor >= len(rows) {
		s.cursor = max(0, len(rows)-1)
	}
}

// bestScore weights name highest, then tags, then description.
func bestScore(p Prompt, q string) int {
	best := -1
	if n := fuzzyScore(strings.ToLower(p.Name), q); n >= 0 {
		best = n
	}
	if n := fuzzyScore(strings.ToLower(strings.Join(p.Tags, " ")), q); n >= 0 {
		if best < 0 || n+10 < best {
			best = n + 10
		}
	}
	if n := fuzzyScore(strings.ToLower(p.Description), q); n >= 0 {
		if best < 0 || n+25 < best {
			best = n + 25
		}
	}
	return best
}

// fuzzyScore returns subsequence match cost, or -1 if no match. Lower is better.
func fuzzyScore(hay, needle string) int {
	hi, cost, gap := 0, 0, 0
	for _, r := range needle {
		idx := strings.IndexRune(hay[hi:], r)
		if idx < 0 {
			return -1
		}
		cost += idx
		if idx > 0 {
			gap++
		}
		hi += idx + 1
	}
	return cost + gap
}

func (s studio) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		s.w, s.h = sz.Width, sz.Height
	}
	switch s.mode {
	case modeForm:
		return s.updateForm(msg)
	case modeConfirm:
		return s.updateConfirm(msg)
	default:
		return s.updateList(msg)
	}
}

func (s studio) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m, ok := msg.(tea.KeyMsg); ok {
		switch m.String() {
		case "ctrl+c", "esc":
			s.quit = true
			return s, tea.Quit
		case "down", "ctrl+j":
			if s.cursor < len(s.treeRows())-1 {
				s.cursor++
			}
			return s, nil
		case "up", "ctrl+k":
			if s.cursor > 0 {
				s.cursor--
			}
			return s, nil
		case "pgdown":
			s.cursor = min(s.cursor+5, len(s.treeRows())-1)
			return s, nil
		case "pgup":
			s.cursor = max(s.cursor-5, 0)
			return s, nil
		case "enter":
			if row, ok := s.currentRow(); ok && row.folder != "" {
				s.expanded[row.folder] = !s.expanded[row.folder]
				return s, nil
			}
			if p, ok := s.current(); ok {
				if err := copyToClipboard(p.Body); err != nil {
					s.status = "clipboard unavailable"
				} else {
					bumpUsage(p.Name)
					s.reload()
					s.status = "copied " + p.Display()
				}
			}
			return s, nil
		case "ctrl+n":
			return s.openForm(false), nil
		case "ctrl+e":
			return s.openForm(true), nil
		case "ctrl+d":
			if _, ok := s.current(); ok {
				s.mode = modeConfirm
			}
			return s, nil
		}
	}
	var cmd tea.Cmd
	s.search, cmd = s.search.Update(msg)
	s.refilter()
	return s, cmd
}

// openForm switches to the editor; edit=true loads the current prompt.
func (s studio) openForm(edit bool) studio {
	s.mode = modeForm
	s.editing = edit
	s.field = 0
	s.formErr = ""
	s.fName.SetValue("")
	s.fDesc.SetValue("")
	s.fTags.SetValue("")
	s.fBody.SetValue("")
	if edit {
		if p, ok := s.current(); ok {
			s.origName = p.Name
			s.fName.SetValue(p.Display())
			s.fDesc.SetValue(p.Description)
			s.fTags.SetValue(strings.Join(p.Tags, ", "))
			s.fBody.SetValue(p.Body)
		}
	} else {
		s.origName = ""
	}
	s.focusField()
	return s
}

func (s *studio) focusField() {
	s.fName.Blur()
	s.fDesc.Blur()
	s.fTags.Blur()
	s.fBody.Blur()
	switch s.field {
	case 0:
		s.fName.Focus()
	case 1:
		s.fDesc.Focus()
	case 2:
		s.fTags.Focus()
	case 3:
		s.fBody.Focus()
	}
}

func (s studio) folders() []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range s.all {
		f := p.Folder()
		if f != "" && !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	sort.Strings(out)
	return out
}

// folderHint shows matching existing folders while typing a namespaced name.
func (s studio) folderHint(w int) string {
	if s.field != 0 {
		return ""
	}
	val := s.fName.Value()
	i := strings.LastIndex(val, "/")
	if i < 0 {
		return ""
	}
	prefix := strings.ToLower(val[:i+1])
	var matches []string
	for _, f := range s.folders() {
		if strings.HasPrefix(strings.ToLower(f+"/"), prefix) {
			matches = append(matches, f+"/")
		}
		if len(matches) >= 4 {
			break
		}
	}
	if len(matches) == 0 {
		return "↳ new folder"
	}
	return "↳ " + strings.Join(matches, "  ")
}

func (s studio) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m, ok := msg.(tea.KeyMsg); ok {
		switch m.String() {
		case "esc":
			if s.standalone {
				s.quit = true
				return s, tea.Quit
			}
			s.mode = modeList
			return s, nil
		case "tab":
			s.field = (s.field + 1) % 4
			s.focusField()
			return s, nil
		case "shift+tab":
			s.field = (s.field + 3) % 4
			s.focusField()
			return s, nil
		case "ctrl+s":
			return s.commitForm()
		}
	}
	var cmd tea.Cmd
	switch s.field {
	case 0:
		s.fName, cmd = s.fName.Update(msg)
	case 1:
		s.fDesc, cmd = s.fDesc.Update(msg)
	case 2:
		s.fTags, cmd = s.fTags.Update(msg)
	case 3:
		s.fBody, cmd = s.fBody.Update(msg)
	}
	return s, cmd
}

func (s studio) commitForm() (tea.Model, tea.Cmd) {
	name, err := parseUserName(s.fName.Value())
	if err != nil {
		s.formErr = err.Error()
		return s, nil
	}
	body := strings.TrimRight(s.fBody.Value(), "\n")
	if body == "" {
		s.formErr = "prompt body is empty"
		return s, nil
	}
	p := Prompt{Name: name, Description: s.fDesc.Value(), Tags: splitTags(s.fTags.Value()), Body: body}
	if s.editing && s.origName != "" {
		if s.origName != name {
			if _, err := load(name); err == nil {
				s.formErr = "prompt " + displayName(name) + " already exists"
				return s, nil
			}
		}
		if existing, err := load(s.origName); err == nil {
			p.UseCount = existing.UseCount
			p.LastUsed = existing.LastUsed
		}
	} else if _, err := load(name); err == nil {
		s.formErr = "prompt " + displayName(name) + " already exists"
		return s, nil
	}
	if err := save(p); err != nil {
		s.formErr = err.Error()
		return s, nil
	}
	if s.editing && s.origName != "" && s.origName != name {
		_ = remove(s.origName)
	}
	if s.standalone {
		s.quit = true
		return s, tea.Quit
	}
	s.reload()
	s.mode = modeList
	s.status = "saved " + displayName(name)
	return s, nil
}

func (s studio) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m, ok := msg.(tea.KeyMsg); ok {
		switch m.String() {
		case "y", "Y", "enter":
			if p, ok := s.current(); ok {
				_ = remove(p.Name)
				s.reload()
				s.status = "deleted " + p.Display()
			}
			s.mode = modeList
		default:
			s.mode = modeList
		}
	}
	return s, nil
}

// reload re-reads prompts from disk after a mutation.
func (s *studio) reload() {
	var keep string
	if p, ok := s.current(); ok {
		keep = p.Name
	}
	if ps, err := list(); err == nil {
		s.all = ps
	}
	s.refilter()
	if keep != "" {
		for i, row := range s.treeRows() {
			if row.prompt != nil && row.prompt.Name == keep {
				s.cursor = i
				break
			}
		}
	}
}

func (s studio) current() (Prompt, bool) {
	if row, ok := s.currentRow(); ok && row.prompt != nil {
		return *row.prompt, true
	}
	return Prompt{}, false
}

type treeRow struct {
	folder string
	prompt *Prompt
	guides []bool
	last   bool
}

func (s studio) currentRow() (treeRow, bool) {
	rows := s.treeRows()
	if s.cursor >= 0 && s.cursor < len(rows) {
		return rows[s.cursor], true
	}
	return treeRow{}, false
}

// treeRows builds an Explorer-style tree. Searching expands matching paths.
func (s studio) treeRows() []treeRow {
	type node struct {
		folders map[string]*node
		prompts []Prompt
		path    string
	}
	root := &node{folders: map[string]*node{}}
	for _, p := range s.filtered {
		n := root
		parts := strings.Split(p.Name, "/")
		for _, part := range parts[:len(parts)-1] {
			if n.folders[part] == nil {
				path := part
				if n.path != "" {
					path = n.path + "/" + part
				}
				n.folders[part] = &node{folders: map[string]*node{}, path: path}
			}
			n = n.folders[part]
		}
		n.prompts = append(n.prompts, p)
	}
	var rows []treeRow
	searching := strings.TrimSpace(s.search.Value()) != ""
	var walk func(*node, []bool)
	walk = func(n *node, guides []bool) {
		folderNames := make([]string, 0, len(n.folders))
		for name := range n.folders {
			folderNames = append(folderNames, name)
		}
		sort.Strings(folderNames)
		sort.SliceStable(n.prompts, func(i, j int) bool { return n.prompts[i].Name < n.prompts[j].Name })
		total := len(folderNames) + len(n.prompts)
		position := 0
		for _, name := range folderNames {
			child := n.folders[name]
			last := position == total-1
			rows = append(rows, treeRow{folder: child.path, guides: append([]bool(nil), guides...), last: last})
			if searching || s.expanded[child.path] {
				walk(child, append(append([]bool(nil), guides...), !last))
			}
			position++
		}
		for i := range n.prompts {
			last := position == total-1
			rows = append(rows, treeRow{prompt: &n.prompts[i], guides: append([]bool(nil), guides...), last: last})
			position++
		}
	}
	walk(root, nil)
	return rows
}

func (s studio) View() string {
	if s.w == 0 {
		return "loading…"
	}
	if s.mode == modeForm {
		return s.viewForm()
	}
	contentW := max(20, s.w-4)
	bodyH := max(3, s.h-11)
	listW := s.w * 42 / 100
	if listW < 30 {
		listW = 30
	}

	pad2 := lipgloss.NewStyle().PaddingLeft(2)
	var sub string
	if len(s.filtered) == len(s.all) {
		sub = fmt.Sprintf("%d prompts", len(s.all))
	} else {
		sub = fmt.Sprintf("%d of %d", len(s.filtered), len(s.all))
	}
	header := pad2.Render(banner() + "   " + stCount.Render(sub))
	searchInner := stAccent.Render("⌕ ") + s.search.View()
	searchBox := stSearch.Width(max(10, s.w-10)).Render(searchInner)
	search := pad2.Render(searchBox)
	hints := "↑↓ move · enter open/copy · ^n new · ^e edit · ^d delete · esc quit"
	var help string
	if s.mode == modeConfirm {
		if p, ok := s.current(); ok {
			help = lipgloss.NewStyle().Foreground(cVar).Render("delete " + p.Display() + "? y/N · esc cancel")
		}
	} else if s.status != "" {
		help = stStatus.Render("✓ "+s.status) + stHelp.Render("  ·  "+hints)
	} else {
		help = stHelp.Render(hints)
	}
	help = pad2.Render(help)
	var cols string
	if s.w < 80 {
		cols = stPanel.Width(contentW).Height(bodyH).Render(s.renderList(max(8, contentW-6), bodyH))
	} else {
		prevW := max(24, s.w-listW-8)
		list := stPanel.Width(listW).Height(bodyH).Render(s.renderList(max(8, listW-6), bodyH))
		prev := stPanel.Width(prevW).Height(bodyH).Render(s.renderPreview(max(8, prevW-6)))
		cols = lipgloss.JoinHorizontal(lipgloss.Top, list, "  ", prev)
	}
	return strings.Join([]string{"", header, search, "", cols, "", help}, "\n")
}

func (s studio) viewForm() string {
	innerW := s.w - 12
	if innerW < 20 {
		innerW = 20
	}
	s.fName.Width = innerW
	s.fDesc.Width = innerW
	s.fTags.Width = innerW
	s.fBody.SetWidth(innerW)
	s.fBody.SetHeight(max(4, s.h-18))

	title := "◆ new prompt"
	if s.editing {
		title = "◆ edit " + displayName(s.origName)
	}
	field := func(i int, label, view, hint string) string {
		gutter, l := stMeta, stMeta
		if s.field == i {
			gutter, l = stTitle, stTitle
		}
		rows := []string{
			gutter.Render("│ ") + l.Render(label),
			lipgloss.NewStyle().PaddingLeft(2).Render(view),
		}
		if hint != "" {
			rows = append(rows, stMeta.Render("  "+hint))
		}
		return strings.Join(rows, "\n")
	}
	gap := ""
	rows := []string{
		stTitle.Render(title),
		"",
		field(0, "name", s.fName.View(), s.folderHint(innerW)),
		gap,
		field(1, "description", s.fDesc.View(), ""),
		gap,
		field(2, "tags", s.fTags.View(), ""),
		gap,
		field(3, "body", s.fBody.View(), ""),
	}
	help := stHelp.Render("tab/⇧tab move · ^s save · esc cancel")
	if s.formErr != "" {
		help = lipgloss.NewStyle().Foreground(cVar).Render("! "+s.formErr) + "   " + help
	}
	rows = append(rows, "", help)
	return stPanel.Width(max(20, s.w-4)).Render(strings.Join(rows, "\n"))
}

func (s studio) renderList(w, h int) string {
	rows := s.treeRows()
	if len(rows) == 0 {
		return stMeta.Render("no matches")
	}
	// Each tree entry gets a content row plus one spacer row.
	vis := max(1, (h-2)/2)
	top := 0
	if s.cursor >= vis {
		top = s.cursor - vis + 1
	}
	end := top + vis
	if end > len(rows) {
		end = len(rows)
	}
	var b strings.Builder
	if top > 0 {
		b.WriteString(stMeta.Render("  ↑ "+fmt.Sprintf("%d more", top)) + "\n")
	} else {
		b.WriteString("\n")
	}
	for i := top; i < end; i++ {
		row := rows[i]
		var prefix strings.Builder
		for depth, continued := range row.guides {
			if depth == len(row.guides)-1 {
				if row.last {
					prefix.WriteString("╰── ")
				} else {
					prefix.WriteString("├── ")
				}
			} else if continued {
				prefix.WriteString("│  ")
			} else {
				prefix.WriteString("   ")
			}
		}
		name := ""
		if row.folder != "" {
			leaf := row.folder
			if j := strings.LastIndex(leaf, "/"); j >= 0 {
				leaf = leaf[j+1:]
			}
			arrow := "▸"
			if strings.TrimSpace(s.search.Value()) != "" || s.expanded[row.folder] {
				arrow = "▾"
			}
			name = prefix.String() + arrow + " " + leaf + "/"
		} else {
			leaf := row.prompt.Name
			if j := strings.LastIndex(leaf, "/"); j >= 0 {
				leaf = leaf[j+1:]
			}
			name = prefix.String() + "@" + leaf
		}
		name = truncW(name, w-2)
		if i == s.cursor {
			b.WriteString(stSel.Width(w).Render(name) + "\n")
		} else {
			b.WriteString(stItem.Render(name) + "\n")
		}
		if row.folder != "" && (strings.TrimSpace(s.search.Value()) != "" || s.expanded[row.folder]) {
			guide := strings.Repeat("   ", len(row.guides)) + "│"
			b.WriteString(stRule.Render(guide) + "\n")
		} else {
			b.WriteString("\n")
		}
	}
	if end < len(rows) {
		b.WriteString(stMeta.Render("  ↓ " + fmt.Sprintf("%d more", len(rows)-end)))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (s studio) renderPreview(w int) string {
	w = max(1, w)
	p, ok := s.current()
	if !ok {
		return stMeta.Render("select a prompt")
	}
	wrap := lipgloss.NewStyle().Width(w)
	var b strings.Builder
	b.WriteString(stTitle.Render(p.Display()) + "\n\n")
	if p.Description != "" {
		b.WriteString(wrap.Foreground(cMeta).Render(p.Description) + "\n\n")
	}
	if len(p.Tags) > 0 {
		b.WriteString(stTags.Render("#"+strings.Join(p.Tags, " #")) + "\n\n")
	}
	b.WriteString(stRule.Render(strings.Repeat("─", w)) + "\n\n")
	body := highlightVars(p.Body)
	b.WriteString(wrap.Foreground(cText).Render(body))
	return b.String()
}

func highlightVars(s string) string {
	return varRE.ReplaceAllStringFunc(s, func(m string) string {
		return lipgloss.NewStyle().Foreground(cVar).Bold(true).Render(m)
	})
}

func pad(s string, w int) string {
	if w < 0 {
		w = 0
	}
	d := w - runewidth.StringWidth(s)
	if d <= 0 {
		return truncW(s, w)
	}
	return s + strings.Repeat(" ", d)
}

func truncW(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= w {
		return s
	}
	return runewidth.Truncate(s, w, "…")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// runStudio launches the full-screen TUI.
func runStudio() error {
	prompts, err := list()
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(newStudio(prompts), tea.WithAltScreen()).Run()
	return err
}

// runEditor opens a single-shot editor to create a prompt, optionally pre-named.
func runEditor(name string) error {
	prompts, _ := list()
	s := newStudio(prompts)
	s.standalone = true
	s = s.openForm(false)
	if name != "" {
		s.fName.SetValue(strings.TrimPrefix(name, "@"))
		s.field = 3
		s.focusField()
	}
	_, err := tea.NewProgram(s, tea.WithAltScreen()).Run()
	return err
}
