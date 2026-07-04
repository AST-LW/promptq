# Security

PromptQ is local-first. It does not call a hosted API, start a server, or upload prompt content.

Security expectations:

- Prompt files are private by default on Unix-like systems.
- Prompt names are validated to prevent path traversal and unsafe cross-platform filenames.
- Clipboard and editor integrations use direct process execution, not shell evaluation.
- `PROMPTQ_HOME` can isolate storage for tests, automation, or separate workspaces.

Do not store production secrets in prompts unless your local machine, backups, and sync tools are approved for that data.

Report security issues privately through the project maintainer channel before opening a public issue.
