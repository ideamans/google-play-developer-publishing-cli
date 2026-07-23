package cmd

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The tests in this file exist to prove that endpoints whose paths are built at
// runtime — the {+name} resources and the ":batchX" suffixes — really are wired
// up, since a source grep cannot see them. Each one asserts the exact method and
// path that reaches the API.

func TestUsersAndGrantsHitTheDeveloperPaths(t *testing.T) {
	const dev = "/developers/1234567890"
	const user = dev + "/users/dev@example.com"

	cases := []struct {
		name    string
		args    []string
		stub    string // "METHOD /path" to stub
		want    string // "METHOD /path" that must be recorded
		body    string
		assert  func(t *testing.T, ts *testServer)
		noBody  bool
		queryIn string
	}{
		{
			name: "users list",
			args: []string{"users", "list"},
			stub: "GET " + dev + "/users",
			want: "GET " + dev + "/users",
			body: `{"users":[{"email":"dev@example.com","accessState":"ACCESS_GRANTED"}]}`,
			// The live API rejects any page size other than -1.
			queryIn: "pageSize=-1",
		},
		{
			name: "users invite",
			args: []string{"users", "invite", "--email", "dev@example.com", "--permission", "CAN_REPLY_TO_REVIEWS_GLOBAL"},
			stub: "POST " + dev + "/users",
			want: "POST " + dev + "/users",
			body: `{"email":"dev@example.com"}`,
		},
		{
			name:    "users update",
			args:    []string{"users", "update", "--email", "dev@example.com", "--permission", "CAN_REPLY_TO_REVIEWS_GLOBAL"},
			stub:    "PATCH " + user,
			want:    "PATCH " + user,
			body:    `{"email":"dev@example.com"}`,
			queryIn: "updateMask=developerAccountPermissions",
		},
		{
			name:   "users remove",
			args:   []string{"users", "remove", "--email", "dev@example.com"},
			stub:   "DELETE " + user,
			want:   "DELETE " + user,
			body:   ``,
			noBody: true,
		},
		{
			name: "grants create",
			args: []string{"grants", "create", "--email", "dev@example.com", "--permission", "CAN_MANAGE_PUBLIC_APKS"},
			stub: "POST " + user + "/grants",
			want: "POST " + user + "/grants",
			body: `{"packageName":"com.example.app"}`,
		},
		{
			name:    "grants update",
			args:    []string{"grants", "update", "--email", "dev@example.com", "--permission", "CAN_VIEW_FINANCIAL_DATA"},
			stub:    "PATCH " + user + "/grants/com.example.app",
			want:    "PATCH " + user + "/grants/com.example.app",
			body:    `{}`,
			queryIn: "updateMask=appLevelPermissions",
		},
		{
			name:   "grants delete",
			args:   []string{"grants", "delete", "--email", "dev@example.com"},
			stub:   "DELETE " + user + "/grants/com.example.app",
			want:   "DELETE " + user + "/grants/com.example.app",
			noBody: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := newTestServer(t)
			if tc.noBody {
				ts.handle(tc.stub, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
			} else {
				ts.respond(tc.stub, tc.body)
			}
			if err := runCommand(t, tc.args...); err != nil {
				t.Fatal(err)
			}
			calls := ts.calls(tc.want)
			if len(calls) != 1 {
				t.Fatalf("%s hit %d times, want 1", tc.want, len(calls))
			}
			if tc.queryIn != "" && !strings.Contains(calls[0].Query, tc.queryIn) {
				t.Errorf("query = %q, want it to contain %q", calls[0].Query, tc.queryIn)
			}
		})
	}
}

func TestExternalTransactionsHitTheNamePaths(t *testing.T) {
	const base = "/applications/com.example.app/externalTransactions"

	t.Run("get", func(t *testing.T) {
		ts := newTestServer(t)
		ts.respond("GET "+base+"/order-123", `{"externalTransactionId":"order-123"}`)
		if err := runCommand(t, "external-transactions", "get", "--id", "order-123"); err != nil {
			t.Fatal(err)
		}
		if got := len(ts.calls("GET " + base + "/order-123")); got != 1 {
			t.Fatalf("hits = %d, want 1", got)
		}
	})

	t.Run("create", func(t *testing.T) {
		ts := newTestServer(t)
		ts.respond("POST "+base, `{"externalTransactionId":"order-123"}`)
		err := runCommand(t, "external-transactions", "create", "--id", "order-123",
			"--from-json", `{"originalPreTaxAmount":{"currency":"JPY","priceMicros":"480000000"}}`)
		if err != nil {
			t.Fatal(err)
		}
		calls := ts.calls("POST " + base)
		if len(calls) != 1 {
			t.Fatalf("hits = %d, want 1", len(calls))
		}
		if !strings.Contains(calls[0].Query, "externalTransactionId=order-123") {
			t.Errorf("query = %q, want the transaction id", calls[0].Query)
		}
	})

	t.Run("refund", func(t *testing.T) {
		ts := newTestServer(t)
		ts.respond("POST "+base+"/order-123:refund", `{"externalTransactionId":"order-123"}`)
		if err := runCommand(t, "external-transactions", "refund", "--id", "order-123"); err != nil {
			t.Fatal(err)
		}
		if got := len(ts.calls("POST " + base + "/order-123:refund")); got != 1 {
			t.Fatalf("hits = %d, want 1", got)
		}
		body := ts.lastBody("POST " + base + "/order-123:refund")
		if _, ok := body["fullRefund"]; !ok {
			t.Errorf("body = %v, want a fullRefund by default", body)
		}
	})
}

func TestOneTimeProductOfferBatchPaths(t *testing.T) {
	const base = "/applications/com.example.app/oneTimeProducts/rental/purchaseOptions/standard/offers"

	t.Run("batch-delete", func(t *testing.T) {
		ts := newTestServer(t)
		ts.respond("POST "+base+":batchDelete", `{}`)
		err := runCommand(t, "one-time-products", "offers", "batch-delete",
			"--product", "rental", "--purchase-option", "standard", "--offer", "old-promo")
		if err != nil {
			t.Fatal(err)
		}
		body := ts.lastBody("POST " + base + ":batchDelete")
		if got := dig(t, body, "requests", 0, "offerId"); got != "old-promo" {
			t.Errorf("requests[0].offerId = %v", got)
		}
		if got := dig(t, body, "requests", 0, "purchaseOptionId"); got != "standard" {
			t.Errorf("requests[0].purchaseOptionId = %v", got)
		}
	})

	t.Run("batch-get", func(t *testing.T) {
		ts := newTestServer(t)
		ts.respond("POST "+base+":batchGet", `{"oneTimeProductOffers":[]}`)
		err := runCommand(t, "one-time-products", "offers", "batch-get",
			"--product", "rental", "--purchase-option", "standard", "--offer", "launch")
		if err != nil {
			t.Fatal(err)
		}
		if got := len(ts.calls("POST " + base + ":batchGet")); got != 1 {
			t.Fatalf("hits = %d, want 1", got)
		}
	})

	t.Run("batch-set-states", func(t *testing.T) {
		ts := newTestServer(t)
		ts.respond("POST "+base+":batchUpdateStates", `{}`)
		err := runCommand(t, "one-time-products", "offers", "batch-set-states",
			"--product", "rental", "--purchase-option", "standard",
			"--activate", "launch", "--cancel", "old-promo")
		if err != nil {
			t.Fatal(err)
		}
		body := ts.lastBody("POST " + base + ":batchUpdateStates")
		requests, _ := body["requests"].([]any)
		if len(requests) != 2 {
			t.Fatalf("requests = %v, want one activate and one cancel", requests)
		}
		if got := dig(t, body, "requests", 0, "activateOneTimeProductOfferRequest", "offerId"); got != "launch" {
			t.Errorf("activate request = %v", got)
		}
		if got := dig(t, body, "requests", 1, "cancelOneTimeProductOfferRequest", "offerId"); got != "old-promo" {
			t.Errorf("cancel request = %v", got)
		}
	})
}

func TestSubscriptionBatchPaths(t *testing.T) {
	t.Run("subscriptions batch-update", func(t *testing.T) {
		ts := newTestServer(t)
		ts.respond("POST /applications/com.example.app/subscriptions:batchUpdate", `{"subscriptions":[]}`)
		err := runCommand(t, "subscriptions", "batch-update", "--from-json", `{"requests":[]}`)
		if err != nil {
			t.Fatal(err)
		}
		if got := len(ts.calls("POST /applications/com.example.app/subscriptions:batchUpdate")); got != 1 {
			t.Fatalf("hits = %d, want 1", got)
		}
	})

	t.Run("base-plans batch-set-states", func(t *testing.T) {
		ts := newTestServer(t)
		const path = "POST /applications/com.example.app/subscriptions/premium/basePlans:batchUpdateStates"
		ts.respond(path, `{"subscriptions":[]}`)
		err := runCommand(t, "subscriptions", "base-plans", "batch-set-states",
			"--product", "premium", "--activate", "monthly", "--deactivate", "legacy")
		if err != nil {
			t.Fatal(err)
		}
		body := ts.lastBody(path)
		if got := dig(t, body, "requests", 0, "activateBasePlanRequest", "basePlanId"); got != "monthly" {
			t.Errorf("activate request = %v", got)
		}
		if got := dig(t, body, "requests", 1, "deactivateBasePlanRequest", "basePlanId"); got != "legacy" {
			t.Errorf("deactivate request = %v", got)
		}
	})

	t.Run("base-plans batch-migrate-prices", func(t *testing.T) {
		ts := newTestServer(t)
		const path = "POST /applications/com.example.app/subscriptions/premium/basePlans:batchMigratePrices"
		ts.respond(path, `{}`)
		err := runCommand(t, "subscriptions", "base-plans", "batch-migrate-prices",
			"--product", "premium", "--from-json", `{"requests":[]}`)
		if err != nil {
			t.Fatal(err)
		}
		if got := len(ts.calls(path)); got != 1 {
			t.Fatalf("hits = %d, want 1", got)
		}
	})

	t.Run("offers batch-get and batch-set-states", func(t *testing.T) {
		ts := newTestServer(t)
		const base = "/applications/com.example.app/subscriptions/premium/basePlans/monthly/offers"
		ts.respond("POST "+base+":batchGet", `{"subscriptionOffers":[]}`)
		ts.respond("POST "+base+":batchUpdateStates", `{}`)

		err := runCommand(t, "subscriptions", "offers", "batch-get",
			"--product", "premium", "--base-plan", "monthly", "--offer", "intro")
		if err != nil {
			t.Fatal(err)
		}
		body := ts.lastBody("POST " + base + ":batchGet")
		if got := dig(t, body, "requests", 0, "offerId"); got != "intro" {
			t.Errorf("batch-get request = %v", got)
		}

		err = runCommand(t, "subscriptions", "offers", "batch-set-states",
			"--product", "premium", "--base-plan", "monthly", "--activate", "intro")
		if err != nil {
			t.Fatal(err)
		}
		states := ts.lastBody("POST " + base + ":batchUpdateStates")
		if got := dig(t, states, "requests", 0, "activateSubscriptionOfferRequest", "basePlanId"); got != "monthly" {
			t.Errorf("activate request = %v", got)
		}
	})
}

func TestPricingConvertSendsMoney(t *testing.T) {
	ts := newTestServer(t)
	const path = "POST /applications/com.example.app/pricing:convertRegionPrices"
	ts.respond(path, `{"convertedRegionPrices":{"JP":{"price":{"currencyCode":"JPY","units":"700","nanos":0}}}}`)

	if err := runCommand(t, "pricing", "convert", "--price", "USD:4.99"); err != nil {
		t.Fatal(err)
	}
	body := ts.lastBody(path)
	if got := dig(t, body, "price", "currencyCode"); got != "USD" {
		t.Errorf("currencyCode = %v", got)
	}
	if got := dig(t, body, "price", "units"); got != "4" {
		t.Errorf("units = %v, want \"4\"", got)
	}
	if got := dig(t, body, "price", "nanos"); got != float64(990000000) {
		t.Errorf("nanos = %v, want 990000000", got)
	}
}

func TestAppStoreHostedAppPaths(t *testing.T) {
	const store = "/appstore/com.example.appstore"

	t.Run("create", func(t *testing.T) {
		ts := newTestServer(t)
		ts.respond("POST "+store+"/apps:create", `{}`)
		err := runCommand(t, "app-store", "apps", "create",
			"--store-package", "com.example.appstore", "--app-package", "com.example.hosted")
		if err != nil {
			t.Fatal(err)
		}
		body := ts.lastBody("POST " + store + "/apps:create")
		if body["packageName"] != "com.example.hosted" {
			t.Errorf("packageName = %v", body["packageName"])
		}
	})

	t.Run("publish-status expands the enum", func(t *testing.T) {
		ts := newTestServer(t)
		const path = "POST " + store + "/apps/com.example.hosted:updateAppStoreHostedAppPublishStatus"
		ts.respond(path, `{}`)
		err := runCommand(t, "app-store", "apps", "publish-status",
			"--store-package", "com.example.appstore", "--app-package", "com.example.hosted",
			"--state", "UNPUBLISHED")
		if err != nil {
			t.Fatal(err)
		}
		body := ts.lastBody(path)
		if body["publishState"] != "APP_STORE_APP_PUBLISH_STATE_UNPUBLISHED" {
			t.Errorf("publishState = %v", body["publishState"])
		}
	})

	t.Run("upload-policy-file uses multipart", func(t *testing.T) {
		ts := newTestServer(t)
		const path = "POST " + store + "/apps/com.example.hosted/policyDeclarationFiles:upload"
		ts.respond(path, `{"policyDeclarationFileId":"f1"}`)

		file := filepath.Join(t.TempDir(), "declaration.pdf")
		if err := os.WriteFile(file, []byte("%PDF-1.4 fake"), 0o600); err != nil {
			t.Fatal(err)
		}
		err := runCommand(t, "app-store", "apps", "upload-policy-file",
			"--store-package", "com.example.appstore", "--app-package", "com.example.hosted", "--file", file)
		if err != nil {
			t.Fatal(err)
		}
		calls := ts.calls(path)
		if len(calls) != 1 {
			t.Fatalf("hits = %d, want 1", len(calls))
		}
		if !strings.Contains(calls[0].Query, "uploadType=multipart") {
			t.Errorf("query = %q, want uploadType=multipart", calls[0].Query)
		}
		payload := string(calls[0].Body)
		if !strings.Contains(payload, `"fileType":"DECLARATION_FILE_TYPE_DOCUMENT"`) {
			t.Errorf("multipart body is missing the fileType metadata:\n%s", payload)
		}
		if !strings.Contains(payload, "%PDF-1.4 fake") {
			t.Errorf("multipart body is missing the media part:\n%s", payload)
		}
	})

	t.Run("upload-apk", func(t *testing.T) {
		ts := newTestServer(t)
		const path = "POST " + store + "/apps/com.example.hosted/apks:upload"
		ts.respond(path, `{"apkId":"apk-1"}`)
		file := filepath.Join(t.TempDir(), "app.apk")
		if err := os.WriteFile(file, []byte("apk"), 0o600); err != nil {
			t.Fatal(err)
		}
		err := runCommand(t, "app-store", "apps", "upload-apk",
			"--store-package", "com.example.appstore", "--app-package", "com.example.hosted", "--file", file)
		if err != nil {
			t.Fatal(err)
		}
		if got := len(ts.calls(path)); got != 1 {
			t.Fatalf("hits = %d, want 1", got)
		}
	})
}

func TestAppStoreCatalogPaths(t *testing.T) {
	const catalog = "/appstorecatalog/com.example.appstore"

	t.Run("app view", func(t *testing.T) {
		ts := newTestServer(t)
		ts.respond("GET "+catalog+"/recentAppViews/com.example.playapp", `{"appView":{}}`)
		err := runCommand(t, "app-store", "catalog", "app",
			"--store-package", "com.example.appstore", "--app-package", "com.example.playapp")
		if err != nil {
			t.Fatal(err)
		}
		if got := len(ts.calls("GET " + catalog + "/recentAppViews/com.example.playapp")); got != 1 {
			t.Fatalf("hits = %d, want 1", got)
		}
	})

	t.Run("update events", func(t *testing.T) {
		ts := newTestServer(t)
		ts.respond("GET "+catalog+"/recentUpdateEvents",
			`{"recentUpdateEvents":[{"eventTime":"2026-07-01T00:00:00Z","updateType":"MODIFICATION","playAppPackageName":"com.x"}]}`)
		err := runCommand(t, "app-store", "catalog", "updates",
			"--store-package", "com.example.appstore", "--start", "2026-07-01T00:00:00Z")
		if err != nil {
			t.Fatal(err)
		}
		calls := ts.calls("GET " + catalog + "/recentUpdateEvents")
		if len(calls) != 1 {
			t.Fatalf("hits = %d, want 1", len(calls))
		}
		if !strings.Contains(calls[0].Query, "startTime=") {
			t.Errorf("query = %q, want startTime", calls[0].Query)
		}
	})
}

func TestPutVariantsAreReachable(t *testing.T) {
	t.Run("details --replace", func(t *testing.T) {
		ts := newTestServer(t)
		stubEdit(ts, "E1")
		ts.respond("PUT /applications/com.example.app/edits/E1/details", `{}`)
		err := runCommand(t, "details", "set", "--replace", "--contact-email", "s@example.com")
		if err != nil {
			t.Fatal(err)
		}
		if got := len(ts.calls("PUT /applications/com.example.app/edits/E1/details")); got != 1 {
			t.Fatalf("PUT hits = %d, want 1", got)
		}
	})

	t.Run("testers --replace", func(t *testing.T) {
		ts := newTestServer(t)
		stubEdit(ts, "E1")
		ts.respond("PUT /applications/com.example.app/edits/E1/testers/alpha", `{}`)
		err := runCommand(t, "testers", "set", "--track", "alpha", "--group", "qa@example.com", "--replace")
		if err != nil {
			t.Fatal(err)
		}
		if got := len(ts.calls("PUT /applications/com.example.app/edits/E1/testers/alpha")); got != 1 {
			t.Fatalf("PUT hits = %d, want 1", got)
		}
	})

	t.Run("tracks --put", func(t *testing.T) {
		ts := newTestServer(t)
		stubEdit(ts, "E1")
		ts.respond("PUT /applications/com.example.app/edits/E1/tracks/internal", `{}`)
		err := runCommand(t, "tracks", "set", "--track", "internal", "--version-code", "42", "--put")
		if err != nil {
			t.Fatal(err)
		}
		if got := len(ts.calls("PUT /applications/com.example.app/edits/E1/tracks/internal")); got != 1 {
			t.Fatalf("PUT hits = %d, want 1", got)
		}
	})

	t.Run("expansion --replace", func(t *testing.T) {
		ts := newTestServer(t)
		stubEdit(ts, "E1")
		ts.respond("PUT /applications/com.example.app/edits/E1/apks/43/expansionFiles/main", `{}`)
		err := runCommand(t, "expansion", "reuse", "--version-code", "43",
			"--references-version", "42", "--replace")
		if err != nil {
			t.Fatal(err)
		}
		if got := len(ts.calls("PUT /applications/com.example.app/edits/E1/apks/43/expansionFiles/main")); got != 1 {
			t.Fatalf("PUT hits = %d, want 1", got)
		}
	})

	t.Run("products --allow-missing upserts with PUT", func(t *testing.T) {
		ts := newTestServer(t)
		ts.respond("PUT /applications/com.example.app/inappproducts/premium", `{"sku":"premium"}`)
		err := runCommand(t, "products", "update", "--sku", "premium",
			"--default-price", "JPY:480", "--allow-missing")
		if err != nil {
			t.Fatal(err)
		}
		calls := ts.calls("PUT /applications/com.example.app/inappproducts/premium")
		if len(calls) != 1 {
			t.Fatalf("PUT hits = %d, want 1", len(calls))
		}
		if !strings.Contains(calls[0].Query, "allowMissing=true") {
			t.Errorf("query = %q, want allowMissing=true", calls[0].Query)
		}
	})
}
