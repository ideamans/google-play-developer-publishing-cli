---
name: gplay-install
description: Make the gplay command available, installing it only if it is missing. Use when another skill reports that `gplay` is not on PATH, or when the user asks to install, update or upgrade the ideamans Google Play publishing CLI. Prefers an already-installed binary, then the latest GitHub release, then a build from source with go install.
license: MIT
compatibility: Requires curl (or wget) and tar to install from a release, or a Go toolchain for the source fallback. Standalone — does not need gplay to be present already. Installs from the public repository github.com/ideamans/google-play-developer-publishing-cli, so no GitHub authentication is needed.
allowed-tools: Bash(curl:*) Bash(wget:*) Bash(tar:*) Bash(unzip:*) Bash(go:*) Bash(uname:*) Bash(command:*) Bash(which:*) Bash(mkdir:*) Bash(mv:*) Bash(cp:*) Bash(rm:*) Bash(chmod:*) Bash(ls:*) Bash(test:*) Bash(echo:*) Read
---

# gplay-install

Make the `gplay` command usable, doing the least work that achieves it.

## Route 1 — an existing installation on PATH

```bash
command -v gplay && gplay --version
```

If that resolves, **use it and stop here.** Do not check for a newer release —
it costs an API call and the user did not ask for an upgrade.

Two checks before trusting the hit:

- **It is the right tool.** `gplay` is a short name that other things may own.
  `gplay llm | head -1` must read
  `# gplay — Google Play Developer Publishing API CLI (reference for AI agents)`. If
  something else owns the name, tell the user and use an explicit path rather
  than shadowing theirs.
- **It is recent enough.** If `gplay llm` is not a known command, the binary
  predates the embedded reference. Say so and continue to route 2 to upgrade it.

Continue past this section only when the command is missing, is the wrong tool,
is too old, or the user explicitly asked to update.

## Route 2 — the latest GitHub release

The repository is public, so no authentication is needed.

```bash
VERSION=$(curl -fsSL https://api.github.com/repos/ideamans/google-play-developer-publishing-cli/releases/latest \
  | grep '"tag_name"' | head -1 | cut -d'"' -f4)   # e.g. v0.5.0
```

**The archive is named after the goreleaser project, not the repository** —
`google-play-developer-publishing`, without the `-cli` suffix:

```
google-play-developer-publishing_<version-without-v>_<os>_<arch>.tar.gz
```

`<os>` is `darwin`, `linux` or `windows` (lowercase); `<arch>` is `amd64` or
`arm64`, so `uname -m` reporting `x86_64` maps to `amd64`. Windows ships a
`.zip`.

```bash
OS=$(uname -s | tr '[:upper:]' '[:lower:]')            # darwin | linux
ARCH=$(uname -m); [ "$ARCH" = "x86_64" ] && ARCH=amd64  # amd64 | arm64
curl -fsSL -o /tmp/gplay.tar.gz \
  "https://github.com/ideamans/google-play-developer-publishing-cli/releases/download/${VERSION}/google-play-developer-publishing_${VERSION#v}_${OS}_${ARCH}.tar.gz"
```

If the download 404s, list the actual assets on the release page rather than
retrying variations.

### Install onto PATH

```bash
tar -xzf /tmp/gplay.tar.gz -C /tmp
mkdir -p ~/.local/bin && mv /tmp/gplay ~/.local/bin/ && chmod +x ~/.local/bin/gplay
```

Prefer the first writable directory already on PATH — `~/.local/bin`, then
`/usr/local/bin`. Two things not to do on your own initiative:

- If nothing on PATH is writable, leave the binary in `/tmp`, print the exact
  `sudo mv` command and let the user run it. Do not run `sudo` yourself.
- If `~/.local/bin` is not on PATH, give the user the line to add to their shell
  profile. Do not edit the profile for them.

## Route 3 — build from source

Needs a Go toolchain and compiles rather than downloads, so it is the last
resort — but it covers platforms the release assets miss. Note the `/cmd/gplay`
suffix; installing the module root would not build anything.

```bash
go install github.com/ideamans/google-play-developer-publishing-cli/cmd/gplay@latest
```

The binary lands in `$(go env GOPATH)/bin` and is named `gplay`.

## Verify

```bash
gplay --version
gplay llm | head -5
```

Report which route was taken, the version and the install path.

Then say what is still needed: `gplay` cannot do anything without a Google Play
service account key. If the user has not run `gplay configure`, tell them now
rather than letting the first real command fail — they will need a service
account JSON key from Google Cloud, granted API access in Play Console under
Users and permissions.

Mention too that an app must have had at least one build uploaded through the
Play Console UI before the API can touch it; otherwise every call 404s on the
package.
