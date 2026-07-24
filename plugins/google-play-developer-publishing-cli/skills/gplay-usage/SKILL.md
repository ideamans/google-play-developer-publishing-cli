---
name: gplay-usage
description: Publish and manage Android apps on Google Play from the command line with the gplay CLI — upload AABs and APKs, move builds between tracks, run and halt staged rollouts, edit store listings and images, manage in-app products and subscriptions, and read reviews and orders. Use when the user asks about their Android app on Google Play, a release or rollout, Play Console metadata, or Play Developer API calls.
license: MIT
compatibility: Requires the `gplay` binary on PATH — run the gplay-install skill if it is missing. Needs a Google Play service account key with API access, registered via `gplay configure`. The app must already exist in Play Console with at least one build uploaded through the UI. Release commands reach real users.
allowed-tools: Bash(gplay:*) Bash(jq:*) Bash(command:*) Read Write
---

# gplay-usage

Drive the Google Play Developer Publishing API (Android Publisher v3).

## 1. Confirm the tool and the credentials

```bash
command -v gplay && gplay --version
gplay apps list
```

Missing binary? Run the `gplay-install` skill.

Credentials come from a Google service account key registered with
`gplay configure`, selected by `--profile` or `GPLAY_PROFILE`. The package name
comes from `-p` / `--package` or `GPLAY_PACKAGE`. **Never print the contents of
the service account JSON.**

A 404 on every call for a package usually means the app has never had a build
uploaded through the Play Console UI — the API cannot touch it until then.

## 2. Understand the edit model before changing anything

Google Play changes are **transactional**: you open an edit, make changes
against it, then commit. An edit that is never committed changes nothing, and an
edit left open can go stale and be rejected. Read this before your first write:

```bash
gplay llm | sed -n '/## The edit model/,/## Using/p'
```

## 3. Read the pitfalls chapter before a release

```bash
gplay llm | sed -n '/# Known pitfalls/,/# Command catalog/p'
```

It records what the API reference does not: version codes must increase, the
first upload has to happen in the UI, and the several ways a release silently
does nothing. Every one of them costs a debugging cycle if you learn it the hard
way.

## 4. Dry-run, then act — releases reach real users

```bash
gplay <command> --dry-run
```

Promoting to `production`, changing a rollout percentage, or halting a release
**changes what real users receive, immediately**. Before running one, state the
package, the track, the version code and the rollout fraction, and get the
user's agreement. A staged rollout in particular is a decision about a
percentage of the user base, not a routine command.

Use `--json` when you are going to parse output; the default is a human table.

## 5. Report

Give back the track, the version code, the rollout percentage and the resulting
status. For uploads say what the assigned version code was — it is what every
later command addresses the build by.

## Failure modes

| Symptom | Cause | Fix |
| --- | --- | --- |
| `command not found: gplay` | not installed | run the `gplay-install` skill |
| 404 on the package | no build ever uploaded via the Play Console UI, or wrong package name | check `-p`, and tell the user the first upload must be manual |
| 401 / 403 | service account lacks API access or the right role | ask the user to grant it in Play Console |
| upload rejected | version code not greater than the last one | bump the version code — Play orders by code, not name |
| a change "did not apply" | the edit was never committed | commit the edit; see the edit-model chapter |
| rollout appears stuck | staged rollout percentage, or a halted release | check the track state before assuming a bug |
