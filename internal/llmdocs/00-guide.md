# gplay — Google Play Developer Publishing API CLI (reference for AI agents)

gplay is a non-interactive CLI for the Google Play Developer Publishing API,
also known as the Android Publisher API v3
(https://androidpublisher.googleapis.com). Every command reads flags and
environment variables only; nothing prompts for input, so it is safe to run
from scripts and agents. Human-readable summaries go to stdout, errors go to
stderr prefixed with "Error:", and the exit code is 0 on success / 1 on any
failure. --json makes list and show commands print the raw API JSON.

## Credential model

The API authenticates with an OAuth 2.0 access token obtained through the
service account JWT-bearer flow:

- a service account JSON key (client_email + private_key) created in Google
  Cloud Console for a project with the "Google Play Android Developer API"
  enabled
- the same service account invited in Play Console > Users and permissions,
  with access to the apps it must manage

gplay signs an RS256 assertion and exchanges it for an access token (about one
hour of validity) on every invocation; the token is cached in memory for the
duration of the process only. A "profile" is a named key plus optional default
package name and developer account id.

Two propagation delays regularly confuse people: a newly invited service
account can take minutes to gain access, and a newly enabled API can take a
minute before requests stop returning 403.

## Credential resolution order (first match wins)

1. Environment variables:
   - GPLAY_SERVICE_ACCOUNT_BASE64 (base64 of the JSON key; for CI)
   - GPLAY_SERVICE_ACCOUNT_JSON (path to the JSON key)
   - GOOGLE_APPLICATION_CREDENTIALS (path; the standard Google variable)
2. Profile lookup: --profile flag, else GPLAY_PROFILE, else default_profile in
   config.toml.

The package name comes from --package/-p, else GPLAY_PACKAGE, else the
profile's package. .env files are never read implicitly.

## Configuration files

~/.config/google-play-developer-publishing/   (or $XDG_CONFIG_HOME/..., mode 0700)
├── config.toml                               (mode 0600)
└── keys/<profile>.json                       (mode 0600)

config.toml format:

    default_profile = "default"

    [profiles.default]
    service_account = "keys/default.json"   # absolute, or relative to the config dir
    package = "com.example.app"             # optional default for --package
    developer_id = "1234567890123456789"    # optional, for users/grants

## The edit model (the thing to understand first)

Store listing text, graphics, binaries and tracks are never changed directly.
They are changed inside an "edit": a staging transaction that is created,
filled in, then committed atomically. Nothing reaches Google Play until the
commit.

gplay hides this for single operations — every edit-scoped command opens an
edit, applies the change and commits. To batch several changes into one commit
(which is also faster and avoids conflicting edits), chain them:

    EDIT=$(gplay edits create)
    gplay listings set --edit $EDIT --language ja --title "..."
    gplay images replace --edit $EDIT --language ja --type phoneScreenshots --file 01.png
    gplay tracks set --edit $EDIT --track internal --version-code 42
    gplay edits commit $EDIT

Rules that matter:
- --edit <id> joins an existing edit and never commits it.
- --no-commit creates an edit, applies the change, prints the edit id on stdout
  and leaves it open.
- Read-only edit-scoped commands (listings list, tracks get, ...) create a
  throwaway edit and delete it afterwards.
- Only one edit can be committed against a given app state; an edit becomes
  invalid when the app changes elsewhere, and commits then fail with 409.
- Edits expire on their own, so an abandoned edit is harmless.

Which commands are edit-scoped: listings, details, images, bundles, apks,
mapping, expansion, tracks, testers, release. Everything else (products,
subscriptions, one-time-products, purchases, orders, reviews, users, grants,
device-tier-configs, app-recovery, external-transactions, internal-sharing)
changes state immediately.

## Using "gplay api" effectively

"gplay api" is the escape hatch for the full API surface
(https://developers.google.com/android-publisher/api-ref/rest). Notes:

- Paths may omit the "/androidpublisher/v3" prefix; it is added automatically.
- Responses are plain REST JSON, not JSON:API: no "data" envelope.
- Pagination comes in two flavours. Newer endpoints return nextPageToken and
  accept pageToken; older ones (reviews, voidedpurchases, inappproducts) return
  tokenPagination.nextPageToken and accept token. "gplay api --paginate --items
  <field>" handles both.
- Errors return {"error":{"code","message","status"}}; gplay formats them into
  the stderr message and exits 1.
- HTTP 401 means the token was rejected (bad or deleted key). HTTP 403 usually
  means the service account is not invited in Play Console, lacks a permission,
  or the API is not enabled in the Cloud project. HTTP 404 on a package usually
  means the app has never had a binary published in this account.

## Typical flows

Register credentials, then verify:
    gplay configure --key ~/Downloads/service-account.json --package com.example.app
    gplay details get

Ship a build to internal testing:
    gplay release --track internal --aab app-release.aab

Staged production rollout, then widen it:
    gplay release --track production --aab app-release.aab --mapping mapping.txt \
      --user-fraction 0.1 --notes ja=@notes-ja.txt
    gplay tracks rollout --track production --user-fraction 0.5 --changes-not-sent-for-review
    gplay tracks complete --track production

Refresh the Japanese store listing in one commit:
    EDIT=$(gplay edits create)
    gplay listings set --edit $EDIT --language ja --title "..." --full-description @desc-ja.txt
    gplay images replace --edit $EDIT --language ja --type phoneScreenshots --file 01.png --file 02.png
    gplay edits commit $EDIT

Verify a purchase from a server:
    gplay purchases subscription-v2 get --token "$PURCHASE_TOKEN" --json

Ad-hoc curl with an access token:
    curl -H "Authorization: Bearer $(gplay token)" \
      "https://androidpublisher.googleapis.com/androidpublisher/v3/applications/com.example.app/reviews"

## Before doing anything that reaches users

`--dry-run` prints mutating requests instead of sending them. Use it first for
anything that changes a release, a track, a listing or pricing.

Promoting a release to `production`, changing a rollout percentage, or halting a
release **affects what real users receive**. State the package, the track, the
version code and the rollout fraction, get the user's agreement, and only then
run it for real.
