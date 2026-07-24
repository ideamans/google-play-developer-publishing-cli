# Known pitfalls (Google Play specific)

Learned from real releases. None of these are inferable from the API reference
or the command list — read this chapter before driving a release end to end.

- **An app must have had at least one APK/AAB uploaded through the Play Console
  UI before the API can touch it.** Until then every call 404s on the package.
- **Version codes must increase.** Re-uploading the same version code fails;
  the version code, not the version name, is what Play orders releases by.
- **A track update replaces that track's release list.** "gplay tracks set"
  sends exactly one release by design; use --keep-existing to merge instead.
- **userFraction only applies to inProgress/halted releases** and must satisfy
  0 < f < 1. gplay drops it for draft/completed releases and rejects 0 or 1
  (use "gplay tracks complete" for a full rollout).
- **Promotion is a move, not a copy.** "gplay tracks promote" writes the
  release into the target track and clears the source track, which is what the
  Play Console does.
- **--changes-not-sent-for-review is not a shortcut for review.** It only works
  for changes that need no review (a rollout percentage). Google rejects the
  commit otherwise, and changes held back this way stay pending.
- **Store listing limits are enforced before upload**: title 30 characters,
  short description 80, full description 4000.
- **Image dimensions are exact for icon (512x512), feature graphic (1024x500)
  and TV banner (1280x720)**; screenshots must be 320-3840 px per side. gplay
  validates locally so a failure names the file rather than the whole edit.
- **Screenshots accumulate.** "gplay images upload" adds; "gplay images
  replace" is the idempotent one.
- **Subscriptions are not edit-scoped and take an updateMask.** Base plans and
  offers are nested inside the subscription resource; only their state changes
  (activate/deactivate/delete/migrate prices) have dedicated endpoints.
- **Purchases must be acknowledged within three days** or Google refunds them
  automatically. Prefer the v2 endpoints for new integrations.
- **There is no endpoint that lists apps.** "gplay apps list" borrows the Play
  Developer Reporting API for that and needs it enabled separately.
- **Statistics, vitals and financial reports are not in this API.** They live in
  the Play Developer Reporting API and in the Cloud Storage report bucket.
- **Prices come in two encodings.** inappproducts uses priceMicros strings
  ("gplay products" takes CURRENCY:AMOUNT and converts); the monetization
  endpoints use Money with units and nanos ("gplay pricing convert" emits that
  shape). Do not copy one into the other.
- **Bulk endpoints exist for the monetization resources** and take a
  {"requests":[...]} body: "subscriptions batch-update",
  "subscriptions base-plans batch-set-states / batch-migrate-prices",
  "subscriptions offers batch-get / batch-update / batch-set-states", and the
  same set under "one-time-products offers". Prefer them over loops when
  changing many products at once — they are atomic and cheaper on quota.
- **"gplay app-store" is not for app developers.** It is the surface for
  operators of a competing Android app store: registering the apps their store
  hosts for Google's policy review, and reading the Play catalog export. Google
  gates it to approved partners, so other accounts get 403.

### Things only a human can do in the Play Console

- Create the app entry and publish the very first binary.
- Answer the content rating questionnaire, the target audience and ads
  declarations, and app access instructions for review.
- Accept the developer distribution agreement, set up the payments profile and
  tax information.
- Manage individual (non-group) testers on closed tracks; the API only handles
  Google Groups.
- Provide the Data safety answers interactively — the API only accepts the CSV
  that the console exports and imports ("gplay data-safety --file ...").
