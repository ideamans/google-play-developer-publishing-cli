// Package llmdocs embeds the reference that `gplay llm` prints.
//
// 00-guide.md and 10-pitfalls.md are hand-written. 90-commands.md is generated
// from the cobra command tree by `go generate ./...` and committed, because
// go:embed needs real files at build time. CI regenerates it and fails if the
// committed copy is stale.
package llmdocs

import (
	"embed"

	kit "github.com/ideamans/go-llm-cli-kit/llmdocs"
)

//go:generate go run ../gen-llmdocs

//go:embed *.md
var files embed.FS

// Docs is the embedded reference bundle.
func Docs() *kit.Docs { return kit.New(files, ".") }
