---
name: regen-ai
description: Regenerate the embedded LLM reference and verify the result. Use after changing commands, flags, help text, or the hand-written guide and pitfalls chapters.
allowed-tools: Bash(go generate:*) Bash(go test:*) Bash(go build:*) Bash(git status:*) Bash(git diff:*) Read
---

# regen-ai

Bring `internal/llmdocs/` back in line with the code.

1. `git status --short` — note what is already dirty, so the regeneration diff
   can be told apart from work in progress.
2. `go generate ./...` — rewrites `90-commands.md`.
3. `go build ./... && go test ./...`.
4. Report which commands appeared, disappeared or changed shape. With 143
   commands the diff is easy to skim past — say explicitly if a command was
   removed, since that is a breaking change for anyone scripting against it.

This skill is Claude Code-local; it is not part of the distributed plugin.
