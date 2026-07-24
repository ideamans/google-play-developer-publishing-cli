# Generated artifacts — do not hand-edit

| Generated file | Source of truth |
| --- | --- |
| `internal/llmdocs/90-commands.md` | the cobra command definitions in `cmd/*.go`, rendered by `internal/gen-llmdocs` |

Hand-written and safe to edit:

- `internal/llmdocs/00-guide.md` — credential model, the edit model, `gplay api`, typical flows
- `internal/llmdocs/10-pitfalls.md` — traps learned from real releases
- `plugins/google-play-developer-publishing-cli/skills/*/SKILL.md`
- `context7.json`

Editing a generated file is always wrong: the next `go generate ./...`
overwrites it, and CI fails on the stale diff in the meantime. To improve the
catalog, improve the command's `Short` / `Long` / `Example` instead.

Regenerate with `/regen-ai`, or `go generate ./... && go test ./...`.
