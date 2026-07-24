---
paths:
  - "cmd/*.go"
  - "internal/llmdocs/00-guide.md"
  - "internal/llmdocs/10-pitfalls.md"
---

# You just touched the source of the embedded LLM reference

If you changed a command, a flag, or a `Short` / `Long` / `Example`, run
`/regen-ai` before finishing so `internal/llmdocs/90-commands.md` matches. CI
regenerates it and fails on a dirty tree.

Do not edit `90-commands.md` directly.

If what you learned was a trap that the API documentation does not state, it
belongs in `10-pitfalls.md` — that chapter is why this CLI is worth using.
