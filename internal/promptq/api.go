package promptq

// SaveArgs is the public parse result for `promptq save`.
type SaveArgs struct {
	Name           string
	Description    string
	DescriptionSet bool
	Tags           string
	TagsSet        bool
	Body           string
}

func ParseSaveArgs(args []string) (SaveArgs, error) {
	opts, err := parseSaveArgs(args)
	if err != nil {
		return SaveArgs{}, err
	}
	return SaveArgs{
		Name:           opts.name,
		Description:    opts.description,
		DescriptionSet: opts.descriptionSet,
		Tags:           opts.tags,
		TagsSet:        opts.tagsSet,
		Body:           opts.body,
	}, nil
}

func SplitTags(tags string) []string { return splitTags(tags) }

func EditorCommand() []string { return editorCommand() }

func CleanName(name string) (string, error) { return cleanName(name) }

func ParseUserName(name string) (string, error) { return parseUserName(name) }

func ResolvePromptName(name string) (string, error) { return resolvePromptName(name) }

func Save(p Prompt) error { return save(p) }

func Load(name string) (Prompt, error) { return load(name) }

func List() ([]Prompt, error) { return list() }

func ParseStoredPrompt(name, content string) Prompt { return parse(name, content) }

func PromptPath(name string) (string, error) { return promptPath(name) }

func RenamePrompt(oldName, newName string) error { return renamePrompt(oldName, newName) }

func DuplicatePrompt(srcName, dstName string) error { return duplicatePrompt(srcName, dstName) }

func Remove(name string) error { return remove(name) }

func FormatTable(headers []string, rows [][]string) string { return formatTable(headers, rows) }
