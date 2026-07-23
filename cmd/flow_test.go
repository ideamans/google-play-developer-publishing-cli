package cmd

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseUploadsBundleAndWritesTrack(t *testing.T) {
	ts := newTestServer(t)
	stubEdit(ts, "EDIT1")
	ts.respond("POST /applications/com.example.app/edits/EDIT1/bundles", `{"versionCode":42,"sha256":"abc"}`)
	ts.respond("PATCH /applications/com.example.app/edits/EDIT1/tracks/internal", `{"track":"internal"}`)

	aab := filepath.Join(t.TempDir(), "app.aab")
	if err := os.WriteFile(aab, []byte("not really a bundle"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := runCommand(t, "release", "--track", "internal", "--aab", aab, "--notes", "ja=リリースノート"); err != nil {
		t.Fatal(err)
	}

	if got := len(ts.calls("POST /applications/com.example.app/edits/EDIT1/bundles")); got != 1 {
		t.Fatalf("bundle uploads = %d, want 1", got)
	}
	body := ts.lastBody("PATCH /applications/com.example.app/edits/EDIT1/tracks/internal")
	if got := dig(t, body, "releases", 0, "versionCodes", 0); got != "42" {
		t.Errorf("versionCodes[0] = %v, want \"42\" (the version code from the upload)", got)
	}
	if got := dig(t, body, "releases", 0, "status"); got != "completed" {
		t.Errorf("status = %v, want completed (the default without --user-fraction)", got)
	}
	if got := dig(t, body, "releases", 0, "releaseNotes", 0, "text"); got != "リリースノート" {
		t.Errorf("releaseNotes[0].text = %v", got)
	}
	if got := len(ts.calls("POST /applications/com.example.app/edits/EDIT1:commit")); got != 1 {
		t.Fatalf("commits = %d, want 1", got)
	}
}

func TestReleaseUserFractionImpliesInProgress(t *testing.T) {
	ts := newTestServer(t)
	stubEdit(ts, "EDIT1")
	ts.respond("POST /applications/com.example.app/edits/EDIT1/bundles", `{"versionCode":7}`)
	ts.respond("PATCH /applications/com.example.app/edits/EDIT1/tracks/production", `{"track":"production"}`)

	aab := filepath.Join(t.TempDir(), "app.aab")
	if err := os.WriteFile(aab, []byte("bundle"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runCommand(t, "release", "--track", "production", "--aab", aab, "--user-fraction", "0.1"); err != nil {
		t.Fatal(err)
	}
	body := ts.lastBody("PATCH /applications/com.example.app/edits/EDIT1/tracks/production")
	if got := dig(t, body, "releases", 0, "status"); got != "inProgress" {
		t.Errorf("status = %v, want inProgress", got)
	}
	if got := dig(t, body, "releases", 0, "userFraction"); got != 0.1 {
		t.Errorf("userFraction = %v, want 0.1", got)
	}
}

func TestReleaseRejectsFullUserFraction(t *testing.T) {
	ts := newTestServer(t)
	stubEdit(ts, "EDIT1")
	ts.respond("POST /applications/com.example.app/edits/EDIT1/bundles", `{"versionCode":7}`)

	aab := filepath.Join(t.TempDir(), "app.aab")
	if err := os.WriteFile(aab, []byte("bundle"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := runCommand(t, "release", "--track", "production", "--aab", aab, "--user-fraction", "1")
	if err == nil || !strings.Contains(err.Error(), "greater than 0 and less than 1") {
		t.Fatalf("error = %v, want a complaint about the user fraction range", err)
	}
	// The edit must not be left behind when a command fails.
	if got := len(ts.calls("DELETE /applications/com.example.app/edits/EDIT1")); got != 1 {
		t.Errorf("edit deletions = %d, want 1 (the failed edit should be abandoned)", got)
	}
}

func TestEditFlagLeavesEditOpen(t *testing.T) {
	ts := newTestServer(t)
	ts.respond("PATCH /applications/com.example.app/edits/GIVEN/listings/ja", `{"language":"ja"}`)

	if err := runCommand(t, "listings", "set", "--edit", "GIVEN", "--language", "ja", "--title", "レシート"); err != nil {
		t.Fatal(err)
	}
	if got := len(ts.calls("POST /applications/com.example.app/edits")); got != 0 {
		t.Errorf("edits.insert calls = %d, want 0 when --edit is given", got)
	}
	if got := len(ts.calls("POST /applications/com.example.app/edits/GIVEN:commit")); got != 0 {
		t.Errorf("commits = %d, want 0 when --edit is given", got)
	}
	body := ts.lastBody("PATCH /applications/com.example.app/edits/GIVEN/listings/ja")
	if body["title"] != "レシート" {
		t.Errorf("title = %v", body["title"])
	}
}

func TestListingsSetEnforcesTitleLimit(t *testing.T) {
	newTestServer(t)
	err := runCommand(t, "listings", "set", "--edit", "E", "--language", "ja", "--title", strings.Repeat("あ", 31))
	if err == nil || !strings.Contains(err.Error(), "at most 30") {
		t.Fatalf("error = %v, want the 30-character title limit", err)
	}
}

func TestReadCommandDiscardsItsEdit(t *testing.T) {
	ts := newTestServer(t)
	stubEdit(ts, "EDIT1")
	ts.respond("GET /applications/com.example.app/edits/EDIT1/tracks", `{"tracks":[{"track":"internal","releases":[{"status":"completed","versionCodes":["42"]}]}]}`)

	if err := runCommand(t, "tracks", "list"); err != nil {
		t.Fatal(err)
	}
	if got := len(ts.calls("POST /applications/com.example.app/edits/EDIT1:commit")); got != 0 {
		t.Errorf("commits = %d, want 0 for a read-only command", got)
	}
	if got := len(ts.calls("DELETE /applications/com.example.app/edits/EDIT1")); got != 1 {
		t.Errorf("edit deletions = %d, want 1", got)
	}
}

func TestTracksRolloutUpdatesTheInProgressRelease(t *testing.T) {
	ts := newTestServer(t)
	stubEdit(ts, "EDIT1")
	ts.respond("GET /applications/com.example.app/edits/EDIT1/tracks/production", `{"track":"production","releases":[
		{"status":"completed","versionCodes":["41"]},
		{"status":"inProgress","versionCodes":["42"],"userFraction":0.1}
	]}`)
	ts.respond("PATCH /applications/com.example.app/edits/EDIT1/tracks/production", `{"track":"production"}`)

	if err := runCommand(t, "tracks", "rollout", "--track", "production", "--user-fraction", "0.5"); err != nil {
		t.Fatal(err)
	}
	body := ts.lastBody("PATCH /applications/com.example.app/edits/EDIT1/tracks/production")
	releases, ok := body["releases"].([]any)
	if !ok || len(releases) != 2 {
		t.Fatalf("releases = %v, want both releases preserved", body["releases"])
	}
	if got := dig(t, body, "releases", 0, "status"); got != "completed" {
		t.Errorf("the completed release was modified: %v", got)
	}
	if got := dig(t, body, "releases", 1, "userFraction"); got != 0.5 {
		t.Errorf("userFraction = %v, want 0.5", got)
	}
}

func TestTracksCompleteDropsUserFraction(t *testing.T) {
	ts := newTestServer(t)
	stubEdit(ts, "EDIT1")
	ts.respond("GET /applications/com.example.app/edits/EDIT1/tracks/production",
		`{"track":"production","releases":[{"status":"inProgress","versionCodes":["42"],"userFraction":0.5}]}`)
	ts.respond("PATCH /applications/com.example.app/edits/EDIT1/tracks/production", `{"track":"production"}`)

	if err := runCommand(t, "tracks", "complete", "--track", "production"); err != nil {
		t.Fatal(err)
	}
	body := ts.lastBody("PATCH /applications/com.example.app/edits/EDIT1/tracks/production")
	release, _ := dig(t, body, "releases", 0).(map[string]any)
	if release["status"] != "completed" {
		t.Errorf("status = %v, want completed", release["status"])
	}
	if _, ok := release["userFraction"]; ok {
		t.Errorf("userFraction is still present: %v", release["userFraction"])
	}
}

func TestTracksPromoteClearsTheSourceTrack(t *testing.T) {
	ts := newTestServer(t)
	stubEdit(ts, "EDIT1")
	ts.respond("GET /applications/com.example.app/edits/EDIT1/tracks/beta",
		`{"track":"beta","releases":[{"status":"completed","versionCodes":["42"],"name":"1.2.0","releaseNotes":[{"language":"ja","text":"改善"}]}]}`)
	ts.respond("PATCH /applications/com.example.app/edits/EDIT1/tracks/production", `{"track":"production"}`)
	ts.respond("PATCH /applications/com.example.app/edits/EDIT1/tracks/beta", `{"track":"beta"}`)

	if err := runCommand(t, "tracks", "promote", "--from", "beta", "--to", "production"); err != nil {
		t.Fatal(err)
	}
	target := ts.lastBody("PATCH /applications/com.example.app/edits/EDIT1/tracks/production")
	if got := dig(t, target, "releases", 0, "versionCodes", 0); got != "42" {
		t.Errorf("promoted version code = %v, want 42 (carried over from the source track)", got)
	}
	if got := dig(t, target, "releases", 0, "releaseNotes", 0, "text"); got != "改善" {
		t.Errorf("release notes were not carried over: %v", got)
	}
	source := ts.lastBody("PATCH /applications/com.example.app/edits/EDIT1/tracks/beta")
	if releases, _ := source["releases"].([]any); len(releases) != 0 {
		t.Errorf("source track releases = %v, want empty", releases)
	}
}

func TestCommitPassesChangesNotSentForReview(t *testing.T) {
	ts := newTestServer(t)
	ts.respond("POST /applications/com.example.app/edits/E9:commit", `{"id":"E9"}`)

	if err := runCommand(t, "edits", "commit", "E9", "--changes-not-sent-for-review"); err != nil {
		t.Fatal(err)
	}
	calls := ts.calls("POST /applications/com.example.app/edits/E9:commit")
	if len(calls) != 1 {
		t.Fatalf("commits = %d, want 1", len(calls))
	}
	if !strings.Contains(calls[0].Query, "changesNotSentForReview=true") {
		t.Errorf("query = %q, want changesNotSentForReview=true", calls[0].Query)
	}
}

func TestDryRunSendsNoMutation(t *testing.T) {
	ts := newTestServer(t)

	if err := runCommand(t, "listings", "set", "--dry-run", "--language", "ja", "--title", "テスト"); err != nil {
		t.Fatal(err)
	}
	if got := len(ts.calls("POST /applications/com.example.app/edits")); got != 0 {
		t.Errorf("edits.insert calls = %d, want 0 under --dry-run", got)
	}
	if got := len(ts.calls("PATCH /applications/com.example.app/edits/DRY-RUN-EDIT/listings/ja")); got != 0 {
		t.Errorf("listing patches = %d, want 0 under --dry-run", got)
	}
}

func TestDryRunStillReadsThroughARealEdit(t *testing.T) {
	ts := newTestServer(t)
	stubEdit(ts, "EDIT1")
	ts.respond("GET /applications/com.example.app/edits/EDIT1/tracks", `{"tracks":[]}`)

	// A read has nothing to preview, so --dry-run must not break it.
	if err := runCommand(t, "tracks", "list", "--dry-run"); err != nil {
		t.Fatal(err)
	}
	if got := len(ts.calls("GET /applications/com.example.app/edits/EDIT1/tracks")); got != 1 {
		t.Errorf("track reads = %d, want 1 even under --dry-run", got)
	}
	if got := len(ts.calls("DELETE /applications/com.example.app/edits/EDIT1")); got != 1 {
		t.Errorf("edit deletions = %d, want 1; the throwaway edit must not be left behind", got)
	}
}

func TestProductsCreateConvertsPricesToMicros(t *testing.T) {
	ts := newTestServer(t)
	ts.respond("POST /applications/com.example.app/inappproducts", `{"sku":"premium"}`)

	err := runCommand(t, "products", "create", "--sku", "premium",
		"--default-language", "ja", "--title", "ja=プレミアム", "--description", "ja=広告非表示",
		"--default-price", "JPY:480", "--price", "us=USD:3.99", "--auto-convert-prices")
	if err != nil {
		t.Fatal(err)
	}
	body := ts.lastBody("POST /applications/com.example.app/inappproducts")
	if got := dig(t, body, "defaultPrice", "priceMicros"); got != "480000000" {
		t.Errorf("defaultPrice.priceMicros = %v, want 480000000", got)
	}
	if got := dig(t, body, "prices", "US", "priceMicros"); got != "3990000" {
		t.Errorf("prices.US.priceMicros = %v, want 3990000", got)
	}
	if got := dig(t, body, "listings", "ja", "title"); got != "プレミアム" {
		t.Errorf("listings.ja.title = %v", got)
	}
	if got := dig(t, body, "purchaseType"); got != "managedUser" {
		t.Errorf("purchaseType = %v, want managedUser by default", got)
	}
	calls := ts.calls("POST /applications/com.example.app/inappproducts")
	if !strings.Contains(calls[0].Query, "autoConvertMissingPrices=true") {
		t.Errorf("query = %q, want autoConvertMissingPrices=true", calls[0].Query)
	}
}

func TestReviewsReplyRejectsLongText(t *testing.T) {
	newTestServer(t)
	err := runCommand(t, "reviews", "reply", "--review", "R1", "--text", strings.Repeat("あ", 351))
	if err == nil || !strings.Contains(err.Error(), "at most 350") {
		t.Fatalf("error = %v, want the 350-character reply limit", err)
	}
}

func TestImagesReplaceDeletesThenUploads(t *testing.T) {
	ts := newTestServer(t)
	stubEdit(ts, "EDIT1")
	ts.handle("DELETE /applications/com.example.app/edits/EDIT1/listings/ja/phoneScreenshots",
		func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	ts.respond("POST /applications/com.example.app/edits/EDIT1/listings/ja/phoneScreenshots",
		`{"image":{"id":"img1"}}`)

	shot := filepath.Join(t.TempDir(), "01.png")
	writePNG(t, shot, 1080, 1920)

	if err := runCommand(t, "images", "replace", "--language", "ja", "--type", "phoneScreenshots", "--file", shot); err != nil {
		t.Fatal(err)
	}
	if got := len(ts.calls("DELETE /applications/com.example.app/edits/EDIT1/listings/ja/phoneScreenshots")); got != 1 {
		t.Errorf("delete-all calls = %d, want 1", got)
	}
	if got := len(ts.calls("POST /applications/com.example.app/edits/EDIT1/listings/ja/phoneScreenshots")); got != 1 {
		t.Errorf("uploads = %d, want 1", got)
	}
}

func TestImagesUploadRejectsWrongIconSize(t *testing.T) {
	newTestServer(t)
	icon := filepath.Join(t.TempDir(), "icon.png")
	writePNG(t, icon, 256, 256)

	err := runCommand(t, "images", "upload", "--edit", "E", "--language", "ja", "--type", "icon", "--file", icon)
	if err == nil || !strings.Contains(err.Error(), "512x512") {
		t.Fatalf("error = %v, want the icon dimension requirement", err)
	}
}
