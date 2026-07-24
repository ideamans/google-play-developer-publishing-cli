# Command catalog

Generated from the cobra command tree by `go generate ./...`.
Do not edit by hand — edit the command definitions instead.

## Global flags

| flag | type | default | description |
| --- | --- | --- | --- |
| `--dry-run` | bool | `false` | print mutating requests instead of sending them |
| `--json` | bool | `false` | print raw API JSON instead of a table |
| `-p`, `--package` | string | — | Android package name, e.g. com.example.app (env: GPLAY_PACKAGE) |
| `--profile` | string | — | profile name (env: GPLAY_PROFILE) |

## `gplay api`

Send a raw request to the Android Publisher API

Sends an authenticated request to
https://androidpublisher.googleapis.com and prints the JSON response. Use this
for endpoints not yet covered by a dedicated subcommand.

The "/androidpublisher/v3" prefix is added when the path omits it, so
"/applications/com.example.app/reviews" and
"/androidpublisher/v3/applications/com.example.app/reviews" are equivalent.

```
gplay api <path>
```

Example:

```
gplay api /applications/com.example.app/reviews
  gplay api -X POST /applications/com.example.app/edits
  gplay api -X PATCH "/applications/com.example.app/edits/EDIT_ID/listings/ja" -d @listing.json
  gplay api --paginate --items reviews "/applications/com.example.app/reviews?maxResults=100"
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `-d`, `--data` | string | — | JSON request body, or @file to read from a file |
| `--items` | string | — | response field holding the page items (with --paginate) |
| `-X`, `--method` | string | `GET` | HTTP method |
| `--paginate` | bool | `false` | follow pagination and merge every page |

## `gplay apks`

APKs (.apk)

### `gplay apks add-externally-hosted`

Register an externally hosted APK (enterprise-only)

Registers an APK that Google Play links to but does not host. This is only
available to organizations using Google Play for private (enterprise) apps.

The JSON body is an ExternallyHostedApk resource; it is long enough that
--from-json @file is the practical way to pass it.

Example:

```
gplay apks add-externally-hosted --from-json @externally-hosted.json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review (only for changes that do not require review) |
| `--edit` | string | — | apply to an existing edit id instead of creating one (leaves it open) |
| `--from-json` | string | — | ExternallyHostedApk JSON, or @file (required) |
| `--no-commit` | bool | `false` | create the edit and apply changes but do not commit; prints the edit id |

### `gplay apks list`

List APKs in the current edit

| flag | type | default | description |
| --- | --- | --- | --- |
| `--edit` | string | — | read from an existing edit id instead of creating a throwaway one |

### `gplay apks upload`

Upload an .apk to the current edit

Example:

```
gplay apks upload --file app-release.apk
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review (only for changes that do not require review) |
| `--edit` | string | — | apply to an existing edit id instead of creating one (leaves it open) |
| `--file` | string | — | path to the .apk (required) |
| `--no-commit` | bool | `false` | create the edit and apply changes but do not commit; prints the edit id |

## `gplay app-recovery`

App recovery actions for broken releases

App recovery lets you push a remote in-app update to users stuck on a broken
version, without a full release. Create a draft action, then deploy it; cancel
stops an action that is already running.

### `gplay app-recovery add-targeting`

Widen the targeting of a running recovery action

Takes an AddTargetingRequest body; targeting can only be added, never removed.

| flag | type | default | description |
| --- | --- | --- | --- |
| `--from-json` | string | — | request JSON, or @file (required) |
| `--id` | string | — | app recovery action id (required) |

### `gplay app-recovery cancel`

Cancel a running recovery action

| flag | type | default | description |
| --- | --- | --- | --- |
| `--id` | string | — | app recovery action id (required) |

### `gplay app-recovery create`

Create a draft recovery action

Takes a CreateDraftAppRecoveryRequest body: the targeting and the remote in-app update to apply.

Example:

```
gplay app-recovery create --from-json @recovery.json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--from-json` | string | — | request JSON, or @file (required) |

### `gplay app-recovery deploy`

Deploy a draft recovery action to users

Example:

```
gplay app-recovery deploy --id 123456
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--id` | string | — | app recovery action id (required) |

### `gplay app-recovery list`

List recovery actions

| flag | type | default | description |
| --- | --- | --- | --- |
| `--version-code` | string | — | only actions targeting this version code |

## `gplay app-store`

Alternative app store operator APIs (hosted apps, catalog export)

These commands are for operators of an app store other than Google Play, not
for app developers. They cover two surfaces:

  apps     register apps your store hosts, submit their details for Google's
           policy review, and upload their APKs, images and declaration files
  catalog  read the Play catalog export: metadata about recently updated apps
           and a feed of update events

Access is granted by Google to approved app store partners; other accounts get
403 here. Everything is keyed by --store-package, the package name of the app
store application itself.

### `gplay app-store apps`

Apps hosted by your app store

#### `gplay app-store apps create`

Register a hosted app record

Creates the app record. This must be called before any other command for
that hosted app.

Example:

```
gplay app-store apps create --store-package com.example.appstore --app-package com.example.hostedapp
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--app-package` | string | — | package name of the app (defaults to --package) |
| `--store-package` | string | — | package name of your app store application (required) |

#### `gplay app-store apps publish-status`

Publish or unpublish a hosted app

Sets the publish state. An app is PUBLISHED by default after a successful
update, so this is only needed to unpublish or to re-publish.

Example:

```
gplay app-store apps publish-status --store-package com.example.appstore --app-package com.example.hostedapp --state UNPUBLISHED
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--app-package` | string | — | package name of the app (defaults to --package) |
| `--state` | string | — | PUBLISHED or UNPUBLISHED (required) |
| `--store-package` | string | — | package name of your app store application (required) |

#### `gplay app-store apps update`

Submit or update a hosted app's details for review

Sends the app's details, localized store listings, active APK sets and policy
declarations. The update is sent for Google's review immediately.

The body is an UpdateAppStoreHostedAppRequest. Image and APK ids in it come
from "app-store apps upload-image" and "app-store apps upload-apk".

Example:

```
gplay app-store apps update --store-package com.example.appstore --app-package com.example.hostedapp --from-json @hosted-app.json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--app-package` | string | — | package name of the app (defaults to --package) |
| `--from-json` | string | — | UpdateAppStoreHostedAppRequest JSON, or @file (required) |
| `--store-package` | string | — | package name of your app store application (required) |

#### `gplay app-store apps upload-apk`

Upload an APK for a hosted app and print its id

Returns an APK id to reference from the activeApks field of "app-store apps update".

Example:

```
gplay app-store apps upload-apk --store-package com.example.appstore --app-package com.example.hostedapp --file app.apk
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--app-package` | string | — | package name of the app (defaults to --package) |
| `--file` | string | — | file to upload (required) |
| `--store-package` | string | — | package name of your app store application (required) |

#### `gplay app-store apps upload-image`

Upload an icon or screenshot for a hosted app and print its id

Returns an image id to reference from appIconId or screenshotId in "app-store apps update".

Example:

```
gplay app-store apps upload-image --store-package com.example.appstore --app-package com.example.hostedapp --file icon.png
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--app-package` | string | — | package name of the app (defaults to --package) |
| `--file` | string | — | file to upload (required) |
| `--store-package` | string | — | package name of your app store application (required) |

#### `gplay app-store apps upload-policy-file`

Upload a policy declaration file for a hosted app

Uploads supporting documentation for a policy declaration and prints its id.
The file type is sent alongside the bytes, so this uses a multipart upload.

Example:

```
gplay app-store apps upload-policy-file --store-package com.example.appstore --app-package com.example.hostedapp --file declaration.pdf
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--app-package` | string | — | package name of the app (defaults to --package) |
| `--file` | string | — | file to upload (required) |
| `--store-package` | string | — | package name of your app store application (required) |
| `--type` | string | `DOCUMENT` | policy declaration file type |

### `gplay app-store catalog`

Google Play catalog export for app stores

#### `gplay app-store catalog app`

Show catalog metadata for one Play app

Example:

```
gplay app-store catalog app --store-package com.example.appstore --app-package com.example.playapp
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--app-package` | string | — | package name of the app (defaults to --package) |
| `--store-package` | string | — | package name of your app store application (required) |

#### `gplay app-store catalog updates`

List update events for eligible apps in a time range

Example:

```
gplay app-store catalog updates --store-package com.example.appstore --start 2026-07-01T00:00:00Z
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--end` | string | — | end of the range as an RFC 3339 timestamp |
| `--page-size` | int | `0` | events per page |
| `--start` | string | — | start of the range as an RFC 3339 timestamp |
| `--store-package` | string | — | package name of your app store application (required) |

## `gplay apps`

Apps the service account can reach

### `gplay apps list`

List the apps the service account has access to

The Android Publisher API has no endpoint that enumerates apps: every call
takes a package name. This command therefore queries the Play Developer
Reporting API instead, which does expose the list.

That means it needs two extra things compared with the other commands:
the "Google Play Developer Reporting API" enabled in the key's Google Cloud
project, and the playdeveloperreporting scope. If either is missing the command
explains what to enable; the rest of gplay keeps working regardless.

Example:

```
gplay apps list
```

## `gplay bundles`

Android App Bundles (.aab)

### `gplay bundles list`

List app bundles in the current edit

Lists the bundles the edit knows about. Note this reflects the edit, not the
full upload history of the app; a freshly created edit lists the bundles
already associated with the app.

| flag | type | default | description |
| --- | --- | --- | --- |
| `--edit` | string | — | read from an existing edit id instead of creating a throwaway one |

### `gplay bundles upload`

Upload an .aab to the current edit

Uploads an Android App Bundle. Files above 16 MiB use the resumable upload
protocol with progress on stderr.

The upload alone does not release anything: assign the resulting version code
to a track with "gplay tracks set", or use "gplay release" to do both in one
edit.

Example:

```
gplay bundles upload --file app-release.aab
  gplay bundles upload --file app-release.aab --no-commit
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--ack-installation-warning` | bool | `false` | acknowledge that the bundle is installable only via Play (large bundles) |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review (only for changes that do not require review) |
| `--device-tier-config-id` | string | — | device tier config id to build the bundle against |
| `--edit` | string | — | apply to an existing edit id instead of creating one (leaves it open) |
| `--file` | string | — | path to the .aab (required) |
| `--no-commit` | bool | `false` | create the edit and apply changes but do not commit; prints the edit id |

## `gplay configure`

Add a profile from a service account JSON key

Copies the service account JSON key into
~/.config/google-play-developer-publishing/keys/ (with 0600 permissions) and
registers a profile in config.toml. The first registered profile becomes the
default profile.

The key must be a service account key created in Google Cloud Console
(IAM & Admin > Service Accounts > Keys > Add key > JSON) for a project where the
"Google Play Android Developer API" is enabled. The same service account must
then be invited in Play Console > Users and permissions with access to the apps
you intend to manage.

--package and --developer-id are optional defaults; with them you can omit
--package on every later command. Re-run with --force to update an existing
profile (--key may be omitted when only changing the defaults).

Example:

```
gplay configure --key ~/Downloads/play-publisher-abc123.json --package com.example.app
  gplay configure --profile client-a --key ~/Downloads/client-a.json --package com.client.app
  gplay configure --force --package com.example.other
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--developer-id` | string | — | Play Console developer account id (for the users/grants commands) |
| `--force` | bool | `false` | overwrite an existing profile |
| `--key` | string | — | path to the service account JSON key |

## `gplay data-safety`

Upload the Data safety (privacy) declaration

Replaces the app's Data safety section from a CSV export.

The API takes the exact CSV that Play Console's Data safety page exports and
imports; export it once from the console, edit it, and submit it here. The CSV
content is sent as the safetyLabels string, so this command reads a file rather
than taking individual answers.

This is a whole-section replacement: anything missing from the CSV is cleared.

Example:

```
gplay data-safety --file data-safety.csv
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--file` | string | — | Data safety CSV exported from Play Console (required) |

## `gplay details`

App-level details (support contacts, default language)

### `gplay details get`

Show the app details

Shows contact email, phone, website and default language.

This is also the cheapest way to verify that credentials and permissions work:
it opens an edit, reads the details and discards the edit.

Example:

```
gplay details get --package com.example.app
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--edit` | string | — | read from an existing edit id instead of creating a throwaway one |

### `gplay details set`

Update the app details

Updates the fields you pass and leaves the rest untouched (PATCH). Pass
--replace to send a full update (PUT), which clears fields you omit.

Example:

```
gplay details set --contact-email support@example.com --contact-website https://example.com/support
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review (only for changes that do not require review) |
| `--contact-email` | string | — | user-visible support email |
| `--contact-phone` | string | — | user-visible support phone number |
| `--contact-website` | string | — | user-visible support website |
| `--default-language` | string | — | default language, e.g. ja |
| `--edit` | string | — | apply to an existing edit id instead of creating one (leaves it open) |
| `--no-commit` | bool | `false` | create the edit and apply changes but do not commit; prints the edit id |
| `--replace` | bool | `false` | send a full update (PUT), clearing omitted fields |

## `gplay device-tier-configs`

Device tier configs for Play Asset Delivery / device targeting

Device tier configs describe device groups (by RAM, SoC, system feature) and
tiers, so an app bundle can ship different assets to different devices. A
config is immutable: creating one returns a new id, which you then pass to
"gplay bundles upload --device-tier-config-id".

Aliases: dtc

### `gplay device-tier-configs create`

Create a device tier config

Example:

```
gplay device-tier-configs create --from-json @device-tiers.json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--allow-unknown-devices` | bool | `false` | accept device models Google does not recognise |
| `--from-json` | string | — | DeviceTierConfig JSON, or @file (required) |

### `gplay device-tier-configs get`

Show one device tier config

| flag | type | default | description |
| --- | --- | --- | --- |
| `--id` | string | — | device tier config id (required) |

### `gplay device-tier-configs list`

List device tier configs

## `gplay edits`

Manage edit transactions

Every change to an app's store listing, tracks or binaries happens inside an
"edit": a staging area that is created, filled in, and then committed as one
atomic change. Nothing reaches Google Play until the edit is committed.

The dedicated commands (listings, tracks, images, bundles, ...) open and commit
an edit for you. Use these subcommands when you want to batch several commands
into one edit:

    EDIT=$(gplay edits create)
    gplay listings set --edit $EDIT --locale ja --title "..."
    gplay tracks set   --edit $EDIT --track internal --version-code 42
    gplay edits commit $EDIT

Edits expire (typically after 7 days, sooner if the app changes elsewhere) and
a commit fails once expired; just create a new one.

### `gplay edits commit`

Commit an edit and send the changes to Google Play

Commits the edit. Changes that require review are submitted for review unless
--changes-not-sent-for-review is given.

Use --changes-not-sent-for-review only for changes that do not require review
(for example a staged rollout percentage). Google rejects the commit when the
edit contains changes that do need review, and changes held back this way stay
pending until a later submission carries them along.

```
gplay edits commit [edit-id]
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review |
| `--edit` | string | — | read from an existing edit id instead of creating a throwaway one |

### `gplay edits create`

Create an edit and print its id

Example:

```
EDIT=$(gplay edits create --package com.example.app)
```

### `gplay edits delete`

Discard an edit without publishing anything

```
gplay edits delete [edit-id]
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--edit` | string | — | read from an existing edit id instead of creating a throwaway one |

### `gplay edits get`

Show an edit and its expiry

```
gplay edits get [edit-id]
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--edit` | string | — | read from an existing edit id instead of creating a throwaway one |

### `gplay edits validate`

Check an edit for errors without committing it

```
gplay edits validate [edit-id]
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--edit` | string | — | read from an existing edit id instead of creating a throwaway one |

## `gplay expansion`

APK expansion files (OBB, legacy APK-only feature)

Expansion files apply to APK releases only; app bundles use Play Asset
Delivery instead. The type is "main" or "patch".

### `gplay expansion get`

Show the expansion file attached to a version code

| flag | type | default | description |
| --- | --- | --- | --- |
| `--edit` | string | — | read from an existing edit id instead of creating a throwaway one |
| `--type` | string | `main` | main or patch |
| `--version-code` | int64 | `0` | APK version code (required) |

### `gplay expansion reuse`

Point a version code at another version's expansion file

Avoids re-uploading an unchanged OBB when publishing a new APK.

Example:

```
gplay expansion reuse --version-code 43 --type main --references-version 42
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review (only for changes that do not require review) |
| `--edit` | string | — | apply to an existing edit id instead of creating one (leaves it open) |
| `--no-commit` | bool | `false` | create the edit and apply changes but do not commit; prints the edit id |
| `--references-version` | int64 | `0` | version code whose expansion file to reuse (required) |
| `--replace` | bool | `false` | send a full update (PUT), clearing omitted fields |
| `--type` | string | `main` | main or patch |
| `--version-code` | int64 | `0` | APK version code (required) |

### `gplay expansion upload`

Upload an expansion file for a version code

Example:

```
gplay expansion upload --version-code 42 --type main --file main.42.com.example.app.obb
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review (only for changes that do not require review) |
| `--edit` | string | — | apply to an existing edit id instead of creating one (leaves it open) |
| `--file` | string | — | path to the .obb (required) |
| `--no-commit` | bool | `false` | create the edit and apply changes but do not commit; prints the edit id |
| `--type` | string | `main` | main or patch |
| `--version-code` | int64 | `0` | APK version code (required) |

## `gplay external-transactions`

Report transactions made with an alternative billing system

Apps that use an alternative billing system (or user choice billing) must
report each transaction to Google. These commands wrap that reporting API.

Aliases: ext

### `gplay external-transactions create`

Report a new external transaction

Takes an ExternalTransaction body describing the amounts, tax and whether the
transaction is one-time or recurring.

Example:

```
gplay external-transactions create --id order-123 --from-json @transaction.json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--from-json` | string | — | request JSON, or @file |
| `--id` | string | — | external transaction id (required) |

### `gplay external-transactions get`

Show a reported external transaction

| flag | type | default | description |
| --- | --- | --- | --- |
| `--id` | string | — | external transaction id (required) |

### `gplay external-transactions refund`

Report a refund of an external transaction

Takes a RefundExternalTransactionRequest body: a full refund or a partial one with the refunded amount.

Example:

```
gplay external-transactions refund --id order-123 --from-json @refund.json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--from-json` | string | — | request JSON, or @file |
| `--id` | string | — | external transaction id (required) |

## `gplay generated-apks`

APKs Google Play generated from an uploaded app bundle

Lists and downloads the split, standalone and universal APKs that Google Play
derived from an app bundle, signed with the app signing key. This is how you
get an installable artifact that matches what users receive.

### `gplay generated-apks download`

Download one generated APK

Example:

```
gplay generated-apks download --version-code 42 --download-id <id> --output universal.apk
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--download-id` | string | — | download id from "generated-apks list" (required) |
| `-o`, `--output` | string | — | output file (default: derived from the package and version code) |
| `--version-code` | int64 | `0` | version code (required) |

### `gplay generated-apks list`

List the APKs generated for a version code

Example:

```
gplay generated-apks list --version-code 42
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--version-code` | int64 | `0` | version code (required) |

## `gplay grants`

Per-app permissions for a user

A grant is one user's access to one app. App-level permissions are enum
values such as CAN_MANAGE_PUBLIC_APKS, CAN_VIEW_FINANCIAL_DATA,
CAN_REPLY_TO_REVIEWS or CAN_MANAGE_STORE_PRESENCE.

### `gplay grants create`

Grant a user access to an app

Example:

```
gplay grants create --email dev@example.com --grant-package com.example.app --permission CAN_MANAGE_PUBLIC_APKS
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--developer-id` | string | — | Play Console developer account id (env: GPLAY_DEVELOPER_ID) |
| `--email` | string | — | user email address (required) |
| `--grant-package` | string | — | package the grant applies to (defaults to --package) |
| `--permission` | stringArray | `[]` | app-level permission enum; repeatable (required) |

### `gplay grants delete`

Remove a user's access to an app

| flag | type | default | description |
| --- | --- | --- | --- |
| `--developer-id` | string | — | Play Console developer account id (env: GPLAY_DEVELOPER_ID) |
| `--email` | string | — | user email address (required) |
| `--grant-package` | string | — | package the grant applies to (defaults to --package) |

### `gplay grants update`

Change a user's permissions on an app

| flag | type | default | description |
| --- | --- | --- | --- |
| `--developer-id` | string | — | Play Console developer account id (env: GPLAY_DEVELOPER_ID) |
| `--email` | string | — | user email address (required) |
| `--grant-package` | string | — | package the grant applies to (defaults to --package) |
| `--permission` | stringArray | `[]` | app-level permission enum; repeatable (required) |

## `gplay images`

Store listing graphics (icon, feature graphic, screenshots)

Uploads and manages the graphic assets of a localized store listing.

Image types: featureGraphic, icon, phoneScreenshots, sevenInchScreenshots, tenInchScreenshots, tvBanner, tvScreenshots, wearScreenshots

Requirements enforced before upload (bypass with --skip-validation):
  icon            32-bit PNG, exactly 512x512
  featureGraphic  PNG or JPEG, exactly 1024x500
  tvBanner        PNG or JPEG, exactly 1280x720
  *Screenshots    each side between 320 and 3840 px
  every image     at most 15 MiB (API limit)

### `gplay images delete`

Delete one image by id

Example:

```
gplay images delete --language ja --type phoneScreenshots --id AbCdEf123
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review (only for changes that do not require review) |
| `--edit` | string | — | apply to an existing edit id instead of creating one (leaves it open) |
| `--id` | string | — | image id from "gplay images list" (required) |
| `-l`, `--language` | string | — | BCP-47 language code, e.g. ja (required) |
| `--no-commit` | bool | `false` | create the edit and apply changes but do not commit; prints the edit id |
| `--type` | string | — | image type: featureGraphic, icon, phoneScreenshots, sevenInchScreenshots, tenInchScreenshots, tvBanner, tvScreenshots, wearScreenshots (required) |

### `gplay images delete-all`

Delete every image of one type

Example:

```
gplay images delete-all --language ja --type phoneScreenshots
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review (only for changes that do not require review) |
| `--edit` | string | — | apply to an existing edit id instead of creating one (leaves it open) |
| `-l`, `--language` | string | — | BCP-47 language code, e.g. ja (required) |
| `--no-commit` | bool | `false` | create the edit and apply changes but do not commit; prints the edit id |
| `--type` | string | — | image type: featureGraphic, icon, phoneScreenshots, sevenInchScreenshots, tenInchScreenshots, tvBanner, tvScreenshots, wearScreenshots (required) |

### `gplay images list`

List uploaded images of one type

Example:

```
gplay images list --language ja --type phoneScreenshots
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--edit` | string | — | read from an existing edit id instead of creating a throwaway one |
| `-l`, `--language` | string | — | BCP-47 language code, e.g. ja (required) |
| `--type` | string | — | image type: featureGraphic, icon, phoneScreenshots, sevenInchScreenshots, tenInchScreenshots, tvBanner, tvScreenshots, wearScreenshots (required) |

### `gplay images replace`

Delete every image of a type, then upload the given files

Deletes all existing images of the type and uploads the files in the order
given, so the listing ends up with exactly those images. Both steps happen in
one edit, so the listing is never left empty.

Example:

```
gplay images replace --language ja --type phoneScreenshots --file 01.png --file 02.png --file 03.png
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--ai-generated` | bool | `false` | attest that the images were generated by AI |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review (only for changes that do not require review) |
| `--edit` | string | — | apply to an existing edit id instead of creating one (leaves it open) |
| `--file` | stringArray | `[]` | image file; repeat for several images, in order (required) |
| `-l`, `--language` | string | — | BCP-47 language code, e.g. ja (required) |
| `--no-commit` | bool | `false` | create the edit and apply changes but do not commit; prints the edit id |
| `--skip-validation` | bool | `false` | upload without checking format and dimensions |
| `--type` | string | — | image type: featureGraphic, icon, phoneScreenshots, sevenInchScreenshots, tenInchScreenshots, tvBanner, tvScreenshots, wearScreenshots (required) |

### `gplay images upload`

Upload one or more images

Adds images to a listing. Screenshots accumulate in upload order, so use
"gplay images replace" when you want the listing to contain exactly the files
you pass. Single-image types (icon, featureGraphic, tvBanner) are replaced by
the upload.

Example:

```
gplay images upload --language ja --type icon --file icon-512.png
  gplay images upload --language ja --type phoneScreenshots --file 01.png --file 02.png
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--ai-generated` | bool | `false` | attest that the images were generated by AI |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review (only for changes that do not require review) |
| `--edit` | string | — | apply to an existing edit id instead of creating one (leaves it open) |
| `--file` | stringArray | `[]` | image file; repeat for several images, in order (required) |
| `-l`, `--language` | string | — | BCP-47 language code, e.g. ja (required) |
| `--no-commit` | bool | `false` | create the edit and apply changes but do not commit; prints the edit id |
| `--skip-validation` | bool | `false` | upload without checking format and dimensions |
| `--type` | string | — | image type: featureGraphic, icon, phoneScreenshots, sevenInchScreenshots, tenInchScreenshots, tvBanner, tvScreenshots, wearScreenshots (required) |

## `gplay internal-sharing`

Internal app sharing uploads (shareable link, no edit or review)

Uploads an artifact for internal app sharing and prints a download URL that
testers with access can install directly. This bypasses tracks, review and
version code rules entirely, so it is the quickest way to hand a build to
someone. It is not an edit operation.

### `gplay internal-sharing upload`

Upload an .aab or .apk for internal app sharing

Example:

```
gplay internal-sharing upload --file app-debug.apk
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--file` | string | — | path to the .aab or .apk (required) |

## `gplay listings`

Store listing text per language (title, descriptions, promo video)

### `gplay listings delete`

Delete one localized store listing

| flag | type | default | description |
| --- | --- | --- | --- |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review (only for changes that do not require review) |
| `--edit` | string | — | apply to an existing edit id instead of creating one (leaves it open) |
| `-l`, `--language` | string | — | BCP-47 language code, e.g. ja or en-US (required) |
| `--no-commit` | bool | `false` | create the edit and apply changes but do not commit; prints the edit id |

### `gplay listings delete-all`

Delete every localized store listing

| flag | type | default | description |
| --- | --- | --- | --- |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review (only for changes that do not require review) |
| `--edit` | string | — | apply to an existing edit id instead of creating one (leaves it open) |
| `--no-commit` | bool | `false` | create the edit and apply changes but do not commit; prints the edit id |

### `gplay listings get`

Show one localized store listing

Example:

```
gplay listings get --language ja
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--edit` | string | — | read from an existing edit id instead of creating a throwaway one |
| `-l`, `--language` | string | — | BCP-47 language code, e.g. ja or en-US (required) |

### `gplay listings list`

List every localized store listing

Example:

```
gplay listings list --package com.example.app
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--edit` | string | — | read from an existing edit id instead of creating a throwaway one |

### `gplay listings set`

Create or update a localized store listing

Updates the fields you pass and leaves the rest untouched (PATCH). Pass
--replace to send a full update (PUT) instead, which clears fields you omit.

Text flags accept @file to read the value from a file, which is the practical
way to supply the 4000-character full description.

Google Play limits: title 30 characters, short description 80, full description
4000.

Example:

```
gplay listings set --language ja --title "レシート読取" --short-description "..." --full-description @desc-ja.txt
  gplay listings set --language en-US --video https://www.youtube.com/watch?v=XXXXXXXXXXX
  gplay listings set --language ja --from-json @listing-ja.json --replace
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review (only for changes that do not require review) |
| `--edit` | string | — | apply to an existing edit id instead of creating one (leaves it open) |
| `--from-json` | string | — | listing resource as JSON, or @file |
| `--full-description` | string | — | full description (max 4000 chars), or @file |
| `-l`, `--language` | string | — | BCP-47 language code, e.g. ja or en-US (required) |
| `--no-commit` | bool | `false` | create the edit and apply changes but do not commit; prints the edit id |
| `--replace` | bool | `false` | send a full update (PUT), clearing omitted fields |
| `--short-description` | string | — | short description (max 80 chars), or @file |
| `--title` | string | — | app title (max 30 chars), or @file |
| `--video` | string | — | promotional YouTube URL |

## `gplay mapping`

ProGuard/R8 mapping and native debug symbol files

### `gplay mapping upload`

Upload a mapping.txt or native debug symbols for a version code

Attaches a deobfuscation file to an already uploaded APK or bundle so that
crash stack traces in Play Console are readable.

  --type proguard    mapping.txt produced by R8/ProGuard
  --type nativeCode  a zip of native debug symbols (.so with symbols)

Example:

```
gplay mapping upload --version-code 42 --file mapping.txt
  gplay mapping upload --version-code 42 --type nativeCode --file symbols.zip
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review (only for changes that do not require review) |
| `--edit` | string | — | apply to an existing edit id instead of creating one (leaves it open) |
| `--file` | string | — | path to mapping.txt or the native symbols zip (required) |
| `--no-commit` | bool | `false` | create the edit and apply changes but do not commit; prints the edit id |
| `--type` | string | `proguard` | proguard or nativeCode |
| `--version-code` | int64 | `0` | version code the file belongs to (required) |

## `gplay one-time-products`

One-time products with purchase options and offers (monetization API)

Manages the newer one-time product model, which replaces plain managed
products for apps that need purchase options (for example rentals versus
purchases) and time-limited offers on them.

Existing managed products created with "gplay products" also appear here; the
"gplay products" commands remain the simpler way to manage them.

Purchase options and offers are nested in the product resource, so they are
written with "one-time-products update --update-mask purchaseOptions"; the
dedicated subcommands cover the state changes that have their own endpoints.

Aliases: otp

### `gplay one-time-products batch-delete`

Delete several one-time products in one request

| flag | type | default | description |
| --- | --- | --- | --- |
| `--product` | stringArray | `[]` | one-time product id; repeatable (required) |

### `gplay one-time-products batch-get`

Fetch several one-time products in one request

| flag | type | default | description |
| --- | --- | --- | --- |
| `--product` | stringArray | `[]` | one-time product id; repeatable (required) |

### `gplay one-time-products batch-update`

Create or update several one-time products in one request

| flag | type | default | description |
| --- | --- | --- | --- |
| `--from-json` | string | — | request JSON, or @file |

### `gplay one-time-products delete`

Delete a one-time product

| flag | type | default | description |
| --- | --- | --- | --- |
| `--product` | string | — | one-time product id (required) |

### `gplay one-time-products get`

Show one one-time product

| flag | type | default | description |
| --- | --- | --- | --- |
| `--product` | string | — | one-time product id (required) |

### `gplay one-time-products list`

List one-time products

### `gplay one-time-products offers`

Offers on a purchase option

#### `gplay one-time-products offers activate`

Activate a one-time product offer

| flag | type | default | description |
| --- | --- | --- | --- |
| `--offer` | string | — | offer id (required) |
| `--product` | string | — | one-time product id (required) |
| `--purchase-option` | string | — | purchase option id (required) |

#### `gplay one-time-products offers batch-delete`

Delete several one-time product offers in one request

Example:

```
gplay one-time-products offers batch-delete --product rental_48h --purchase-option standard --offer old-promo
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--from-json` | string | — | batch request JSON, or @file |
| `--offer` | stringArray | `[]` | offer id; repeatable |
| `--product` | string | — | one-time product id (required) |
| `--purchase-option` | string | — | purchase option id (required) |

#### `gplay one-time-products offers batch-get`

Fetch several offers of a purchase option in one request

Pass --offer repeatedly, or a full BatchGetOneTimeProductOffersRequest with --from-json.

Example:

```
gplay one-time-products offers batch-get --product rental_48h --purchase-option standard --offer launch
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--from-json` | string | — | batch request JSON, or @file |
| `--offer` | stringArray | `[]` | offer id; repeatable |
| `--product` | string | — | one-time product id (required) |
| `--purchase-option` | string | — | purchase option id (required) |

#### `gplay one-time-products offers batch-set-states`

Activate, deactivate and cancel several offers in one request

Build the batch from --activate / --deactivate / --cancel, or pass a full
BatchUpdateOneTimeProductOfferStatesRequest with --from-json.

Example:

```
gplay one-time-products offers batch-set-states --product rental_48h --purchase-option standard --activate launch --cancel old-promo
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--activate` | stringArray | `[]` | offer id to activate; repeatable |
| `--cancel` | stringArray | `[]` | offer id to cancel; repeatable |
| `--deactivate` | stringArray | `[]` | offer id to deactivate; repeatable |
| `--from-json` | string | — | batch request JSON, or @file |
| `--product` | string | — | one-time product id (required) |
| `--purchase-option` | string | — | purchase option id (required) |

#### `gplay one-time-products offers batch-update`

Create or update several one-time product offers in one request

Takes a BatchUpdateOneTimeProductOffersRequest body:
{"requests":[{"oneTimeProductOffer":{...},"updateMask":"regionalPricingAndAvailabilityConfigs","allowMissing":true}, ...]}.

| flag | type | default | description |
| --- | --- | --- | --- |
| `--from-json` | string | — | batch request JSON, or @file |
| `--product` | string | — | one-time product id (required) |
| `--purchase-option` | string | — | purchase option id (required) |

#### `gplay one-time-products offers cancel`

Cancel a one-time product offer that is currently running

| flag | type | default | description |
| --- | --- | --- | --- |
| `--offer` | string | — | offer id (required) |
| `--product` | string | — | one-time product id (required) |
| `--purchase-option` | string | — | purchase option id (required) |

#### `gplay one-time-products offers deactivate`

Deactivate a one-time product offer

| flag | type | default | description |
| --- | --- | --- | --- |
| `--offer` | string | — | offer id (required) |
| `--product` | string | — | one-time product id (required) |
| `--purchase-option` | string | — | purchase option id (required) |

#### `gplay one-time-products offers list`

List the offers of a purchase option

Pass --purchase-option "-" to list the offers of every purchase option.

| flag | type | default | description |
| --- | --- | --- | --- |
| `--product` | string | — | one-time product id (required) |
| `--purchase-option` | string | — | purchase option id (required) |

### `gplay one-time-products purchase-options`

Purchase options of a one-time product

#### `gplay one-time-products purchase-options delete`

Delete purchase options in bulk

| flag | type | default | description |
| --- | --- | --- | --- |
| `--from-json` | string | — | request JSON, or @file |
| `--product` | string | — | one-time product id (required) |
| `--purchase-option` | string | — | purchase option id (required) |

#### `gplay one-time-products purchase-options set-states`

Activate or deactivate purchase options in bulk

Takes a BatchUpdatePurchaseOptionStatesRequest body, whose requests each
activate or deactivate one purchase option.

Example:

```
gplay one-time-products purchase-options set-states --product rental_48h --from-json @states.json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--from-json` | string | — | request JSON, or @file |
| `--product` | string | — | one-time product id (required) |

### `gplay one-time-products update`

Create or update a one-time product

Patches a one-time product. Pass --allow-missing to create it when it does
not exist yet. The update mask names the top-level fields to replace, e.g.
--update-mask listings,purchaseOptions.

Note the API path for this method is lower-case "onetimeproducts" while the
other methods use "oneTimeProducts"; that quirk is handled here.

Example:

```
gplay one-time-products update --product rental_48h --allow-missing --update-mask listings,purchaseOptions --from-json @product.json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--allow-missing` | bool | `false` | create the product when it does not exist |
| `--from-json` | string | — | request JSON, or @file |
| `--product` | string | — | one-time product id (required) |
| `--regions-version` | string | `2022/02` | region catalogue version for price changes |
| `--update-mask` | string | — | comma-separated fields to replace (inferred from the body when omitted) |

## `gplay orders`

Orders and refunds

Reads order details and issues refunds. Orders are identified by the order id
that appears in the purchase receipt and in Play Console (for example
GPA.1234-5678-9012-34567).

### `gplay orders batch-get`

Fetch up to 1000 orders in one request

Example:

```
gplay orders batch-get --order GPA.1111 --order GPA.2222
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--order` | stringArray | `[]` | order id; repeatable (required) |

### `gplay orders get`

Show one order

Example:

```
gplay orders get --order GPA.1234-5678-9012-34567
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--order` | string | — | order id, e.g. GPA.1234-5678-9012-34567 (required) |

### `gplay orders refund`

Refund an order

Refunds the order. With --revoke the entitlement is also withdrawn, which is
what you want for a purchase the user should no longer have.

Example:

```
gplay orders refund --order GPA.1234-5678-9012-34567
  gplay orders refund --order GPA.1234-5678-9012-34567 --revoke
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--order` | string | — | order id, e.g. GPA.1234-5678-9012-34567 (required) |
| `--revoke` | bool | `false` | also revoke the entitlement |

### `gplay orders review-refund`

Answer a pending refund request with a recommendation

Responds to a user-initiated refund request that Google has forwarded, using
the pending refund token from the notification. The body is an
OrdersReviewRefundRequest: refundPreference (APPROVE / DECLINE / NEUTRAL),
consumption evidence, and the pending refund token.

Example:

```
gplay orders review-refund --order GPA.1234 --from-json @review.json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--from-json` | string | — | OrdersReviewRefundRequest JSON, or @file (required) |
| `--order` | string | — | order id, e.g. GPA.1234-5678-9012-34567 (required) |

## `gplay pricing`

Price conversion across regions

### `gplay pricing convert`

Convert one price into every other region's price

Asks Google to convert a price into the equivalent price for every region,
the same calculation the Play Console offers when you set a product price. It
computes prices only — nothing is written, so this is safe to run any time.

Feed the result into "gplay products update --from-json" or a subscription
base plan's regionalConfigs.

Example:

```
gplay pricing convert --price USD:4.99
  gplay pricing convert --price JPY:480 --tax-category withdrawalRightEu --json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--price` | string | — | price to convert as CURRENCY:AMOUNT, e.g. USD:4.99 (required) |
| `--tax-category` | string | — | product tax category code to apply |

## `gplay products`

Managed in-app products (one-time purchases)

Manages the classic inappproducts resource: managed products and legacy
subscriptions. These commands are not edit-scoped — changes take effect
immediately.

Prices are written as CURRENCY:AMOUNT (for example JPY:150 or USD:1.99) and are
converted to the micros the API expects. Regional prices use REGION=CURRENCY:AMOUNT
(for example JP=JPY:150).

Aliases: inappproducts

### `gplay products batch-delete`

Delete several products in one request

Example:

```
gplay products batch-delete --sku a --sku b
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--sku` | stringArray | `[]` | product id / SKU; repeatable (required) |

### `gplay products batch-get`

Fetch several products in one request

Example:

```
gplay products batch-get --sku a --sku b --sku c
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--sku` | stringArray | `[]` | product id / SKU; repeatable (required) |

### `gplay products batch-update`

Update up to 100 products in one request

Takes an InappproductsBatchUpdateRequest body: {"requests":[{"inappproduct":{...}}, ...]}.

Example:

```
gplay products batch-update --from-json @products.json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--from-json` | string | — | InappproductsBatchUpdateRequest JSON, or @file (required) |

### `gplay products create`

Create an in-app product

Creates a managed product. A product needs a default language, a title and
description in that language, and a default price.

Use --auto-convert-prices to let Google fill in the other regions from the
default price.

Example:

```
gplay products create --sku premium_upgrade --default-language ja \
    --title ja="プレミアム" --description ja="広告非表示" \
    --default-price JPY:480 --auto-convert-prices
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--auto-convert-prices` | bool | `false` | let Google convert the default price into missing regions |
| `--default-language` | string | — | default language, e.g. ja |
| `--default-price` | string | — | default price as CURRENCY:AMOUNT, e.g. JPY:480 |
| `--description` | stringArray | `[]` | <language>=<description\|@file>; repeatable |
| `--from-json` | string | — | InAppProduct JSON, or @file; other flags override it |
| `--price` | stringArray | `[]` | regional price as REGION=CURRENCY:AMOUNT, e.g. JP=JPY:480; repeatable |
| `--purchase-type` | string | — | managedUser (one-time) or subscription (legacy) |
| `--sku` | string | — | product id / SKU (required) |
| `--status` | string | — | active or inactive |
| `--title` | stringArray | `[]` | <language>=<title>; repeatable |

### `gplay products delete`

Delete an in-app product

Example:

```
gplay products delete --sku premium_upgrade
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--sku` | string | — | product id / SKU (required) |

### `gplay products get`

Show one in-app product

Example:

```
gplay products get --sku premium_upgrade
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--sku` | string | — | product id / SKU (required) |

### `gplay products list`

List in-app products

Example:

```
gplay products list --package com.example.app
```

### `gplay products update`

Update an in-app product

Patches the fields you pass and leaves the rest untouched.

--replace sends a full update (PUT) instead, clearing omitted fields, and
--allow-missing turns the call into an upsert (which the API only offers on the
full update, so it implies --replace).

Example:

```
gplay products update --sku premium_upgrade --default-price JPY:580 --auto-convert-prices
  gplay products update --sku premium_upgrade --status inactive
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--allow-missing` | bool | `false` | create the product when it does not exist (implies --replace) |
| `--auto-convert-prices` | bool | `false` | let Google convert the default price into missing regions |
| `--default-language` | string | — | default language, e.g. ja |
| `--default-price` | string | — | default price as CURRENCY:AMOUNT, e.g. JPY:480 |
| `--description` | stringArray | `[]` | <language>=<description\|@file>; repeatable |
| `--from-json` | string | — | InAppProduct JSON, or @file; other flags override it |
| `--latency-tolerance` | string | — | PRODUCT_UPDATE_LATENCY_TOLERANCE_LATENCY_TOLERANT for bulk updates |
| `--price` | stringArray | `[]` | regional price as REGION=CURRENCY:AMOUNT, e.g. JP=JPY:480; repeatable |
| `--purchase-type` | string | — | managedUser (one-time) or subscription (legacy) |
| `--replace` | bool | `false` | send a full update (PUT), clearing omitted fields |
| `--sku` | string | — | product id / SKU (required) |
| `--status` | string | — | active or inactive |
| `--title` | stringArray | `[]` | <language>=<title>; repeatable |

## `gplay profiles`

Manage credential profiles

### `gplay profiles list`

List configured profiles

### `gplay profiles remove`

Remove a profile (the key file is kept)

```
gplay profiles remove <name>
```

### `gplay profiles use`

Set the default profile

```
gplay profiles use <name>
```

## `gplay purchases`

Verify and manage purchases made in the app

Server-side purchase verification and management. These commands take the
purchase token the app receives from Google Play Billing.

The v2 endpoints ("purchases product-v2", "purchases subscription-v2") do not
need a product id and return the richer current-state model; prefer them for
new integrations.

### `gplay purchases product`

One-time product purchases (v1)

#### `gplay purchases product acknowledge`

Acknowledge a one-time product purchase

Google Play refunds and revokes purchases that are not acknowledged within
three days, so a server-side integration must acknowledge every purchase it
grants entitlement for.

| flag | type | default | description |
| --- | --- | --- | --- |
| `--developer-payload` | string | — | opaque payload to store with the purchase |
| `--product` | string | — | product id the token belongs to (required) |
| `--token` | string | — | purchase token from Google Play Billing (required) |

#### `gplay purchases product consume`

Consume a one-time product purchase so it can be bought again

| flag | type | default | description |
| --- | --- | --- | --- |
| `--product` | string | — | product id the token belongs to (required) |
| `--token` | string | — | purchase token from Google Play Billing (required) |

#### `gplay purchases product get`

Show the state of a one-time product purchase

Example:

```
gplay purchases product get --product premium_upgrade --token <purchase-token>
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--product` | string | — | product id the token belongs to (required) |
| `--token` | string | — | purchase token from Google Play Billing (required) |

### `gplay purchases product-v2`

One-time product purchases (v2, no product id needed)

#### `gplay purchases product-v2 get`

Show the state of a one-time product purchase (v2)

Example:

```
gplay purchases product-v2 get --token <purchase-token>
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--token` | string | — | purchase token from Google Play Billing (required) |

### `gplay purchases subscription`

Subscription purchases (v1)

#### `gplay purchases subscription acknowledge`

Acknowledge a subscription purchase

| flag | type | default | description |
| --- | --- | --- | --- |
| `--developer-payload` | string | — | opaque payload to store with the purchase |
| `--product` | string | — | product id the token belongs to (required) |
| `--token` | string | — | purchase token from Google Play Billing (required) |

#### `gplay purchases subscription cancel`

Cancel a subscription at the end of the current billing period

| flag | type | default | description |
| --- | --- | --- | --- |
| `--product` | string | — | product id the token belongs to (required) |
| `--token` | string | — | purchase token from Google Play Billing (required) |

#### `gplay purchases subscription defer`

Push a subscription's next billing date out

Extends a subscription for free by moving its expiry time. Both timestamps
are milliseconds since epoch, or RFC 3339 timestamps that are converted for
you. --from must be the current expiry time.

Example:

```
gplay purchases subscription defer --product premium_monthly --token <token> \
    --from 2026-08-01T00:00:00Z --to 2026-09-01T00:00:00Z
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--from` | string | — | current expiry time (required) |
| `--product` | string | — | product id the token belongs to (required) |
| `--to` | string | — | desired expiry time (required) |
| `--token` | string | — | purchase token from Google Play Billing (required) |

### `gplay purchases subscription-v2`

Subscription purchases (v2, no product id needed)

#### `gplay purchases subscription-v2 cancel`

Cancel a subscription at the end of the billing period (v2)

| flag | type | default | description |
| --- | --- | --- | --- |
| `--from-json` | string | — | request JSON, or @file |
| `--token` | string | — | purchase token from Google Play Billing (required) |

#### `gplay purchases subscription-v2 defer`

Push a subscription's next billing date out (v2)

Takes a DeferSubscriptionPurchaseRequest body naming the line item and the new expiry.

| flag | type | default | description |
| --- | --- | --- | --- |
| `--from-json` | string | — | request JSON, or @file |
| `--token` | string | — | purchase token from Google Play Billing (required) |

#### `gplay purchases subscription-v2 get`

Show the current state of a subscription purchase (v2)

Example:

```
gplay purchases subscription-v2 get --token <purchase-token>
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--token` | string | — | purchase token from Google Play Billing (required) |

#### `gplay purchases subscription-v2 revoke`

Revoke a subscription immediately (v2)

Ends a subscription right away, optionally refunding it. The request body
selects the refund behaviour; without --from-json the subscription is revoked
with no refund and immediate loss of access.

Example:

```
gplay purchases subscription-v2 revoke --token <token> --from-json @revoke.json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--from-json` | string | — | request JSON, or @file |
| `--token` | string | — | purchase token from Google Play Billing (required) |

### `gplay purchases voided`

Refunded, charged-back or revoked purchases

Lists purchases that were voided, so entitlements can be withdrawn. Google
retains 30 days of history by default and up to 60 months when a start time is
given.

#### `gplay purchases voided list`

List voided purchases

Example:

```
gplay purchases voided list
  gplay purchases voided list --start 2026-06-01T00:00:00Z --type 1
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--end` | string | — | latest void time (RFC 3339 or epoch millis) |
| `--include-partial-refunds` | bool | `false` | include quantity-based partial refunds |
| `--start` | string | — | earliest void time (RFC 3339 or epoch millis) |
| `--type` | int | `0` | 0 = voided one-time purchases only, 1 = also voided subscriptions |

## `gplay release`

Upload a build and release it to a track in one edit

Does the whole publishing round trip in a single edit: upload the artifacts,
attach the mapping file, write the release into the track, and commit.

Version codes come from the uploads; pass --version-code as well to include
builds that were uploaded earlier (for example one APK per ABI).

The status defaults to "inProgress" when --user-fraction is given and
"completed" otherwise. Use --status draft to stage a release without
publishing it.

Example:

```
gplay release --track internal --aab app-release.aab
  gplay release --track production --aab app-release.aab --mapping mapping.txt \
    --user-fraction 0.1 --notes ja=@notes-ja.txt --notes en-US=@notes-en.txt
  gplay release --track production --aab app-release.aab --status draft --dry-run
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--aab` | stringArray | `[]` | app bundle to upload; repeatable |
| `--apk` | stringArray | `[]` | APK to upload; repeatable |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review (only for changes that do not require review) |
| `--country` | stringSlice | `[]` | restrict the release to these CLDR country codes, e.g. JP,US |
| `--edit` | string | — | apply to an existing edit id instead of creating one (leaves it open) |
| `--from-json` | string | — | TrackRelease JSON, or @file; other flags override it |
| `--in-app-update-priority` | int64 | `0` | in-app update priority, 0-5 |
| `--include-rest-of-world` | bool | `false` | also target "rest of world" alongside --country |
| `--keep-existing` | bool | `false` | merge into the track's existing releases instead of replacing them |
| `--mapping` | string | — | ProGuard/R8 mapping.txt for the uploaded artifact |
| `--no-commit` | bool | `false` | create the edit and apply changes but do not commit; prints the edit id |
| `--notes` | stringArray | `[]` | release notes as <language>=<text> or <language>=@file; repeatable |
| `--put` | bool | `false` | send a full track update (PUT) instead of a patch |
| `--release-name` | string | — | release name shown in Play Console |
| `--status` | string | — | draft, inProgress, halted or completed |
| `--symbols` | string | — | native debug symbols zip for the uploaded artifact |
| `--track` | string | — | target track: internal, alpha, beta, production or a closed track (required) |
| `--user-fraction` | float64 | `0` | staged rollout fraction, 0 < f < 1 (implies --status inProgress) |
| `--validate` | bool | `false` | validate the edit before committing |
| `--version-code` | int64Slice | `[]` | version code to release; repeat for several (e.g. an APK per ABI) |

## `gplay reviews`

User reviews and developer replies

Reads reviews that have a written comment and posts developer replies.

Google only exposes reviews from the last week or so through this API; use the
review export in Play Console for the full history.

### `gplay reviews get`

Show one review with its full comment thread

Example:

```
gplay reviews get --review <review-id>
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--review` | string | — | review id (required) |
| `--translate-to` | string | — | translate reviews into this language, e.g. ja |

### `gplay reviews list`

List recent reviews

Example:

```
gplay reviews list
  gplay reviews list --limit 20 --translate-to ja
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--limit` | int | `0` | stop after this many reviews |
| `--translate-to` | string | — | translate reviews into this language, e.g. ja |

### `gplay reviews reply`

Reply to a review

Posts (or replaces) the developer reply. A reply is limited to 350
characters and is public.

Example:

```
gplay reviews reply --review <review-id> --text "ご報告ありがとうございます。次のアップデートで修正します。"
  gplay reviews reply --review <review-id> --text @reply.txt
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--review` | string | — | review id (required) |
| `--text` | string | — | reply text (max 350 chars), or @file (required) |

## `gplay subscriptions`

Subscriptions, base plans and offers (monetization API)

Manages the modern subscription model: a subscription holds localized
listings and one or more base plans, and each base plan can carry offers.

Base plans and their regional prices are nested inside the subscription
resource, so they are edited through "subscriptions update" with an
--update-mask (for example --update-mask basePlans). The base-plans and offers
subcommands cover the operations that have dedicated endpoints: activate,
deactivate, delete and price migration.

Subscriptions are not edit-scoped — changes take effect immediately.

Aliases: subs

### `gplay subscriptions archive`

Archive a subscription

Archiving hides a subscription from new purchases; existing subscribers keep it.

| flag | type | default | description |
| --- | --- | --- | --- |
| `--product` | string | — | subscription product id (required) |

### `gplay subscriptions base-plans`

Base plans of a subscription

#### `gplay subscriptions base-plans activate`

Activate a base plan

Example:

```
gplay subscriptions base-plans activate --product premium_monthly --base-plan monthly
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--base-plan` | string | — | base plan id (required) |
| `--product` | string | — | subscription product id (required) |

#### `gplay subscriptions base-plans batch-migrate-prices`

Migrate subscriber prices for several base plans in one request

Takes a BatchMigrateBasePlanPricesRequest body:
{"requests":[{"basePlanId":"monthly","regionalPriceMigrations":[...],"regionsVersion":{"version":"2022/02"}}, ...]}.

Example:

```
gplay subscriptions base-plans batch-migrate-prices --product premium --from-json @migrations.json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--from-json` | string | — | batch request JSON, or @file |
| `--product` | string | — | subscription product id (required) |

#### `gplay subscriptions base-plans batch-set-states`

Activate and deactivate several base plans in one request

Activates and/or deactivates base plans of one subscription atomically.
Build the batch from --activate / --deactivate, or pass a full
BatchUpdateBasePlanStatesRequest with --from-json.

Example:

```
gplay subscriptions base-plans batch-set-states --product premium --activate monthly --deactivate legacy-monthly
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--activate` | stringArray | `[]` | id to activate; repeatable |
| `--deactivate` | stringArray | `[]` | id to deactivate; repeatable |
| `--from-json` | string | — | batch request JSON, or @file |
| `--product` | string | — | subscription product id (required) |

#### `gplay subscriptions base-plans deactivate`

Deactivate a base plan (existing subscribers keep it)

| flag | type | default | description |
| --- | --- | --- | --- |
| `--base-plan` | string | — | base plan id (required) |
| `--product` | string | — | subscription product id (required) |

#### `gplay subscriptions base-plans delete`

Delete a draft base plan

| flag | type | default | description |
| --- | --- | --- | --- |
| `--base-plan` | string | — | base plan id (required) |
| `--product` | string | — | subscription product id (required) |

#### `gplay subscriptions base-plans list`

List a subscription's base plans

Example:

```
gplay subscriptions base-plans list --product premium_monthly
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--product` | string | — | subscription product id (required) |

#### `gplay subscriptions base-plans migrate-prices`

Migrate existing subscribers to the current base plan prices

Applies the base plan's current regional prices to subscribers who are still
on an older price. The request body names the regions and the oldest price
version to migrate; see MigrateBasePlanPricesRequest.

Example:

```
gplay subscriptions base-plans migrate-prices --product premium_monthly --base-plan monthly --from-json @migrate.json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--base-plan` | string | — | base plan id (required) |
| `--from-json` | string | — | resource JSON, or @file |
| `--product` | string | — | subscription product id (required) |
| `--regions-version` | string | `2022/02` | region catalogue version for price changes |

### `gplay subscriptions batch-get`

Fetch several subscriptions in one request

Example:

```
gplay subscriptions batch-get --product monthly --product yearly
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--product` | stringArray | `[]` | subscription product id; repeatable (required) |

### `gplay subscriptions batch-update`

Create or update up to 100 subscriptions in one request

Takes a BatchUpdateSubscriptionsRequest body:
{"requests":[{"subscription":{...},"updateMask":"listings","allowMissing":true}, ...]}.

Every request in the batch must carry the same packageName as --package.

Example:

```
gplay subscriptions batch-update --from-json @subscriptions.json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--from-json` | string | — | batch request JSON, or @file |

### `gplay subscriptions create`

Create a subscription

Creates a subscription from a Subscription resource. Base plans, prices and
regional availability are nested in that resource, so --from-json is the
practical way to supply them; --title/--description/--benefit fill in a single
listing for simple cases.

Example:

```
gplay subscriptions create --product premium_monthly --from-json @subscription.json
  gplay subscriptions create --product premium_monthly --language ja --title "プレミアム" --description "広告非表示"
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--benefit` | stringArray | `[]` | listing benefit line; repeatable |
| `--description` | string | — | listing description, or @file |
| `--from-json` | string | — | resource JSON, or @file |
| `-l`, `--language` | string | — | listing language, e.g. ja |
| `--product` | string | — | subscription product id (required) |
| `--regions-version` | string | `2022/02` | region catalogue version for price changes |
| `--title` | string | — | listing title |

### `gplay subscriptions delete`

Delete a draft subscription

Only subscriptions that have never been activated can be deleted; archive the others.

| flag | type | default | description |
| --- | --- | --- | --- |
| `--product` | string | — | subscription product id (required) |

### `gplay subscriptions get`

Show one subscription

Example:

```
gplay subscriptions get --product premium_monthly
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--product` | string | — | subscription product id (required) |

### `gplay subscriptions list`

List subscriptions

Example:

```
gplay subscriptions list --package com.example.app
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--show-archived` | bool | `false` | include archived subscriptions |

### `gplay subscriptions offers`

Introductory and promotional offers on a base plan

#### `gplay subscriptions offers activate`

Activate an offer

| flag | type | default | description |
| --- | --- | --- | --- |
| `--base-plan` | string | — | base plan id (required) |
| `--offer` | string | — | offer id (required) |
| `--product` | string | — | subscription product id (required) |

#### `gplay subscriptions offers batch-get`

Fetch several offers of a base plan in one request

Pass --offer repeatedly, or a full BatchGetSubscriptionOffersRequest with --from-json.

Example:

```
gplay subscriptions offers batch-get --product premium --base-plan monthly --offer intro-7d --offer winback
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--base-plan` | string | — | base plan id (required) |
| `--from-json` | string | — | batch request JSON, or @file |
| `--offer` | stringArray | `[]` | offer id; repeatable |
| `--product` | string | — | subscription product id (required) |

#### `gplay subscriptions offers batch-set-states`

Activate and deactivate several offers in one request

Build the batch from --activate / --deactivate, or pass a full
BatchUpdateSubscriptionOfferStatesRequest with --from-json.

Example:

```
gplay subscriptions offers batch-set-states --product premium --base-plan monthly --activate intro-7d --deactivate old-promo
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--activate` | stringArray | `[]` | id to activate; repeatable |
| `--base-plan` | string | — | base plan id (required) |
| `--deactivate` | stringArray | `[]` | id to deactivate; repeatable |
| `--from-json` | string | — | batch request JSON, or @file |
| `--product` | string | — | subscription product id (required) |

#### `gplay subscriptions offers batch-update`

Create or update several offers in one request

Takes a BatchUpdateSubscriptionOffersRequest body:
{"requests":[{"subscriptionOffer":{...},"updateMask":"phases","allowMissing":true}, ...]}.

Example:

```
gplay subscriptions offers batch-update --product premium --base-plan monthly --from-json @offers.json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--base-plan` | string | — | base plan id (required) |
| `--from-json` | string | — | batch request JSON, or @file |
| `--product` | string | — | subscription product id (required) |

#### `gplay subscriptions offers create`

Create an offer on a base plan

Example:

```
gplay subscriptions offers create --product premium_monthly --base-plan monthly --offer intro-7d --from-json @offer.json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--base-plan` | string | — | base plan id (required) |
| `--from-json` | string | — | resource JSON, or @file |
| `--offer` | string | — | offer id (required) |
| `--product` | string | — | subscription product id (required) |
| `--regions-version` | string | `2022/02` | region catalogue version for price changes |

#### `gplay subscriptions offers deactivate`

Deactivate an offer

| flag | type | default | description |
| --- | --- | --- | --- |
| `--base-plan` | string | — | base plan id (required) |
| `--offer` | string | — | offer id (required) |
| `--product` | string | — | subscription product id (required) |

#### `gplay subscriptions offers delete`

Delete a draft offer

| flag | type | default | description |
| --- | --- | --- | --- |
| `--base-plan` | string | — | base plan id (required) |
| `--offer` | string | — | offer id (required) |
| `--product` | string | — | subscription product id (required) |

#### `gplay subscriptions offers get`

Show one offer

| flag | type | default | description |
| --- | --- | --- | --- |
| `--base-plan` | string | — | base plan id (required) |
| `--offer` | string | — | offer id (required) |
| `--product` | string | — | subscription product id (required) |

#### `gplay subscriptions offers list`

List the offers of a base plan

Example:

```
gplay subscriptions offers list --product premium_monthly --base-plan monthly
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--base-plan` | string | — | base plan id (required) |
| `--product` | string | — | subscription product id (required) |

#### `gplay subscriptions offers update`

Update an offer

Example:

```
gplay subscriptions offers update --product p --base-plan monthly --offer intro-7d --update-mask phases --from-json @offer.json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--base-plan` | string | — | base plan id (required) |
| `--from-json` | string | — | resource JSON, or @file |
| `--offer` | string | — | offer id (required) |
| `--product` | string | — | subscription product id (required) |
| `--regions-version` | string | `2022/02` | region catalogue version for price changes |
| `--update-mask` | string | — | comma-separated fields to replace (inferred from the body when omitted) |

### `gplay subscriptions update`

Update a subscription

Patches a subscription. The API requires an update mask naming the top-level
fields to replace, e.g. --update-mask listings or --update-mask basePlans.

Example:

```
gplay subscriptions update --product premium_monthly --update-mask basePlans --from-json @subscription.json
  gplay subscriptions update --product premium_monthly --update-mask listings --language ja --title "プレミアム"
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--allow-missing` | bool | `false` | create the subscription when it does not exist |
| `--benefit` | stringArray | `[]` | listing benefit line; repeatable |
| `--description` | string | — | listing description, or @file |
| `--from-json` | string | — | resource JSON, or @file |
| `-l`, `--language` | string | — | listing language, e.g. ja |
| `--latency-tolerance` | string | — | PRODUCT_UPDATE_LATENCY_TOLERANCE_LATENCY_TOLERANT for bulk updates |
| `--product` | string | — | subscription product id (required) |
| `--regions-version` | string | `2022/02` | region catalogue version for price changes |
| `--title` | string | — | listing title |
| `--update-mask` | string | — | comma-separated fields to replace (inferred from the body when omitted) |

## `gplay system-apks`

System image APK variants (device manufacturers only)

Creates and downloads APK variants suitable for preloading into a system
image. This is only available to accounts that Google has enabled for system
app distribution.

### `gplay system-apks create`

Create a variant for a device specification

Takes a Variant body describing the device spec (SDK version, ABIs, screen density, locales).

Example:

```
gplay system-apks create --version-code 42 --from-json @variant.json
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--from-json` | string | — | Variant JSON, or @file (required) |
| `--version-code` | int64 | `0` | version code (required) |

### `gplay system-apks download`

Download a variant as an APK

| flag | type | default | description |
| --- | --- | --- | --- |
| `-o`, `--output` | string | — | output file (default: derived from the package and version code) |
| `--variant` | int64 | `0` | variant id (required) |
| `--version-code` | int64 | `0` | version code (required) |

### `gplay system-apks get`

Show one variant

| flag | type | default | description |
| --- | --- | --- | --- |
| `--variant` | int64 | `0` | variant id (required) |
| `--version-code` | int64 | `0` | version code (required) |

### `gplay system-apks list`

List the variants created for a version code

| flag | type | default | description |
| --- | --- | --- | --- |
| `--version-code` | int64 | `0` | version code (required) |

## `gplay testers`

Google Groups allowed to test a closed track

Manages the Google Groups that can join a closed testing track. Individual
tester email addresses can only be managed in the Play Console UI; the API
works with groups.

### `gplay testers get`

Show the tester groups of a track

Example:

```
gplay testers get --track alpha
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--edit` | string | — | read from an existing edit id instead of creating a throwaway one |
| `--track` | string | — | track name: internal, alpha, beta, production or a closed track (required) |

### `gplay testers set`

Add tester groups to a track (or replace them)

Example:

```
gplay testers set --track alpha --group qa@example.com
  gplay testers set --track alpha --group qa@example.com --replace
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review (only for changes that do not require review) |
| `--edit` | string | — | apply to an existing edit id instead of creating one (leaves it open) |
| `--group` | stringArray | `[]` | Google Group email address; repeatable (required) |
| `--no-commit` | bool | `false` | create the edit and apply changes but do not commit; prints the edit id |
| `--replace` | bool | `false` | replace the group list instead of adding to it |
| `--track` | string | — | track name: internal, alpha, beta, production or a closed track (required) |

## `gplay token`

Print an OAuth 2.0 access token for use with curl etc.

Exchanges the profile's service account key for a short-lived OAuth 2.0
access token (valid for about an hour) and prints it.

Example:

```
curl -H "Authorization: Bearer $(gplay token)" \
    https://androidpublisher.googleapis.com/androidpublisher/v3/applications/com.example.app/edits -X POST
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--scope` | string | `https://www.googleapis.com/auth/androidpublisher` | OAuth scope to request |

## `gplay tracks`

Release tracks and staged rollouts

Manages the releases in a track (internal, alpha, beta, production, or a
closed testing track's own name).

A release couples version codes with a status:

  draft       staged in Play Console, not released
  inProgress  rolling out to a fraction of users (requires --user-fraction)
  halted      rollout paused
  completed   rolled out to everyone

Updating a track replaces its release list. "gplay tracks set" therefore sends
exactly the release you describe, which is what a normal deploy wants; pass
--keep-existing to merge into the releases already in the track instead.

### `gplay tracks complete`

Roll a track's release out to 100% of users

Sets the release status to "completed" and drops the user fraction.

Example:

```
gplay tracks complete --track production
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review (only for changes that do not require review) |
| `--edit` | string | — | apply to an existing edit id instead of creating one (leaves it open) |
| `--no-commit` | bool | `false` | create the edit and apply changes but do not commit; prints the edit id |
| `--track` | string | — | track name: internal, alpha, beta, production or a closed track (required) |

### `gplay tracks country-availability`

Show the countries a track's artifacts are available in

Example:

```
gplay tracks country-availability --track production
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--edit` | string | — | read from an existing edit id instead of creating a throwaway one |
| `--track` | string | — | track name: internal, alpha, beta, production or a closed track (required) |

### `gplay tracks create`

Create a new closed testing track

Creates an additional closed testing track. The built-in tracks (internal,
alpha, beta, production) always exist and do not need to be created.

Example:

```
gplay tracks create --track qa-team --type CLOSED_TESTING
  gplay tracks create --track wear-qa --type CLOSED_TESTING --form-factor WEAR
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review (only for changes that do not require review) |
| `--edit` | string | — | apply to an existing edit id instead of creating one (leaves it open) |
| `--form-factor` | string | — | DEFAULT, WEAR or AUTOMOTIVE |
| `--no-commit` | bool | `false` | create the edit and apply changes but do not commit; prints the edit id |
| `--track` | string | — | track name: internal, alpha, beta, production or a closed track (required) |
| `--type` | string | `CLOSED_TESTING` | track type |

### `gplay tracks get`

Show one track

Example:

```
gplay tracks get --track production
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--edit` | string | — | read from an existing edit id instead of creating a throwaway one |
| `--track` | string | — | track name: internal, alpha, beta, production or a closed track (required) |

### `gplay tracks halt`

Pause a track's staged rollout

Example:

```
gplay tracks halt --track production
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review (only for changes that do not require review) |
| `--edit` | string | — | apply to an existing edit id instead of creating one (leaves it open) |
| `--no-commit` | bool | `false` | create the edit and apply changes but do not commit; prints the edit id |
| `--track` | string | — | track name: internal, alpha, beta, production or a closed track (required) |

### `gplay tracks list`

List every track and its releases

Example:

```
gplay tracks list --package com.example.app
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--edit` | string | — | read from an existing edit id instead of creating a throwaway one |

### `gplay tracks promote`

Move a release from one track to another

Reads the source track's release, writes it into the target track, and
removes it from the source track (which is what promoting means in Play).

Without --version-code the version codes of the source track's newest release
are used. Release notes carry over unless --notes is given.

Example:

```
gplay tracks promote --from internal --to production --user-fraction 0.1
  gplay tracks promote --from beta --to production --version-code 42
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review (only for changes that do not require review) |
| `--country` | stringSlice | `[]` | restrict the release to these CLDR country codes, e.g. JP,US |
| `--edit` | string | — | apply to an existing edit id instead of creating one (leaves it open) |
| `--from` | string | — | source track (required) |
| `--from-json` | string | — | TrackRelease JSON, or @file; other flags override it |
| `--in-app-update-priority` | int64 | `0` | in-app update priority, 0-5 |
| `--include-rest-of-world` | bool | `false` | also target "rest of world" alongside --country |
| `--keep-existing` | bool | `false` | merge into the track's existing releases instead of replacing them |
| `--no-commit` | bool | `false` | create the edit and apply changes but do not commit; prints the edit id |
| `--notes` | stringArray | `[]` | release notes as <language>=<text> or <language>=@file; repeatable |
| `--put` | bool | `false` | send a full track update (PUT) instead of a patch |
| `--release-name` | string | — | release name shown in Play Console |
| `--status` | string | — | draft, inProgress, halted or completed |
| `--to` | string | — | target track (required) |
| `--user-fraction` | float64 | `0` | staged rollout fraction, 0 < f < 1 (implies --status inProgress) |
| `--version-code` | int64Slice | `[]` | version code to release; repeat for several (e.g. an APK per ABI) |

### `gplay tracks releases`

List a track's releases that are ready for review, in review, or rejected

Shows the review state of the track's releases. Unlike the other track
commands this reads published state directly and needs no edit.

Example:

```
gplay tracks releases --track production
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--track` | string | — | track name: internal, alpha, beta, production or a closed track (required) |

### `gplay tracks rollout`

Change the staged rollout percentage of a track's release

Updates the user fraction of the track's in-progress (or halted) release and
resumes it. A rollout change does not require review, so it can be committed
with --changes-not-sent-for-review.

Example:

```
gplay tracks rollout --track production --user-fraction 0.5
  gplay tracks rollout --track production --user-fraction 0.5 --changes-not-sent-for-review
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review (only for changes that do not require review) |
| `--edit` | string | — | apply to an existing edit id instead of creating one (leaves it open) |
| `--no-commit` | bool | `false` | create the edit and apply changes but do not commit; prints the edit id |
| `--track` | string | — | track name: internal, alpha, beta, production or a closed track (required) |
| `--user-fraction` | float64 | `0` | staged rollout fraction, 0 < f < 1 (required) |

### `gplay tracks set`

Put a release into a track

Writes one release into the track. The status defaults to "inProgress" when
--user-fraction is given and "completed" otherwise.

Release notes are per language and accept files:
  --notes ja=@notes-ja.txt --notes en-US="Bug fixes"

Example:

```
gplay tracks set --track internal --version-code 42
  gplay tracks set --track production --version-code 42 --user-fraction 0.1 --notes ja=@notes-ja.txt
  gplay tracks set --track production --version-code 42 --status draft
  gplay tracks set --track production --version-code 42 --country JP --country US
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--changes-in-review-behavior` | string | — | CANCEL_IN_REVIEW_AND_SUBMIT or ERROR_IF_IN_REVIEW when a review is already pending |
| `--changes-not-sent-for-review` | bool | `false` | commit without sending changes for review (only for changes that do not require review) |
| `--country` | stringSlice | `[]` | restrict the release to these CLDR country codes, e.g. JP,US |
| `--edit` | string | — | apply to an existing edit id instead of creating one (leaves it open) |
| `--from-json` | string | — | TrackRelease JSON, or @file; other flags override it |
| `--in-app-update-priority` | int64 | `0` | in-app update priority, 0-5 |
| `--include-rest-of-world` | bool | `false` | also target "rest of world" alongside --country |
| `--keep-existing` | bool | `false` | merge into the track's existing releases instead of replacing them |
| `--no-commit` | bool | `false` | create the edit and apply changes but do not commit; prints the edit id |
| `--notes` | stringArray | `[]` | release notes as <language>=<text> or <language>=@file; repeatable |
| `--put` | bool | `false` | send a full track update (PUT) instead of a patch |
| `--release-name` | string | — | release name shown in Play Console |
| `--status` | string | — | draft, inProgress, halted or completed |
| `--track` | string | — | track name: internal, alpha, beta, production or a closed track (required) |
| `--user-fraction` | float64 | `0` | staged rollout fraction, 0 < f < 1 (implies --status inProgress) |
| `--version-code` | int64Slice | `[]` | version code to release; repeat for several (e.g. an APK per ABI) |

## `gplay users`

Developer account users and their permissions

Manages who has access to the developer account. These commands work on the
developer account rather than a single app, so they need the developer account
id: the number in the Play Console URL
(play.google.com/console/u/0/developers/<DEVELOPER_ID>/...). Store it once with
"gplay configure --developer-id ...".

Account-wide permissions go on the user; per-app permissions are grants, one
per package (see "gplay grants").

### `gplay users invite`

Invite a user to the developer account

Sends an invitation. Account-wide permissions are enum values such as
CAN_VIEW_FINANCIAL_DATA_GLOBAL, CAN_MANAGE_PERMISSIONS_GLOBAL or
CAN_REPLY_TO_REVIEWS_GLOBAL; per-app access is granted separately with
"gplay grants create".

Example:

```
gplay users invite --email dev@example.com --permission CAN_REPLY_TO_REVIEWS_GLOBAL
  gplay users invite --email dev@example.com --expires 2026-12-31T00:00:00Z
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--developer-id` | string | — | Play Console developer account id (env: GPLAY_DEVELOPER_ID) |
| `--email` | string | — | user email address (required) |
| `--expires` | string | — | access expiry as an RFC 3339 timestamp |
| `--from-json` | string | — | User JSON, or @file |
| `--permission` | stringArray | `[]` | account-wide permission enum; repeatable |

### `gplay users list`

List the users of the developer account

Example:

```
gplay users list
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--developer-id` | string | — | Play Console developer account id (env: GPLAY_DEVELOPER_ID) |

### `gplay users remove`

Remove a user from the developer account

| flag | type | default | description |
| --- | --- | --- | --- |
| `--developer-id` | string | — | Play Console developer account id (env: GPLAY_DEVELOPER_ID) |
| `--email` | string | — | user email address (required) |

### `gplay users update`

Change a user's account-wide permissions or expiry

Example:

```
gplay users update --email dev@example.com --permission CAN_REPLY_TO_REVIEWS_GLOBAL
```

| flag | type | default | description |
| --- | --- | --- | --- |
| `--developer-id` | string | — | Play Console developer account id (env: GPLAY_DEVELOPER_ID) |
| `--email` | string | — | user email address (required) |
| `--expires` | string | — | access expiry as an RFC 3339 timestamp |
| `--permission` | stringArray | `[]` | account-wide permission enum; repeatable |
