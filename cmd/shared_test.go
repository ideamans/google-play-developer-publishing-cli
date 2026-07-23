package cmd

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestValueOrFile(t *testing.T) {
	if got, err := valueOrFile("plain"); err != nil || got != "plain" {
		t.Fatalf("valueOrFile(plain) = %q, %v", got, err)
	}
	path := filepath.Join(t.TempDir(), "desc.txt")
	if err := os.WriteFile(path, []byte("from file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := valueOrFile("@" + path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "from file" {
		t.Fatalf("valueOrFile(@file) = %q, want %q (trailing newline trimmed)", got, "from file")
	}
}

func TestParseLocalized(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes-ja.txt")
	if err := os.WriteFile(path, []byte("不具合を修正しました\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := parseLocalized([]string{"en-US=Bug fixes", "ja=@" + path})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0]["language"] != "en-US" || got[0]["text"] != "Bug fixes" {
		t.Fatalf("parseLocalized = %v", got)
	}
	if got[1]["text"] != "不具合を修正しました" {
		t.Fatalf("file-backed note = %q", got[1]["text"])
	}
	if _, err := parseLocalized([]string{"missing-separator"}); err == nil {
		t.Fatal("expected an error for a value without =")
	}
}

func TestParsePrice(t *testing.T) {
	cases := map[string][2]string{
		"JPY:480":  {"JPY", "480000000"},
		"USD:1.99": {"USD", "1990000"},
		"usd:0":    {"USD", "0"},
	}
	for in, want := range cases {
		currency, micros, err := parsePrice(in)
		if err != nil {
			t.Fatalf("parsePrice(%q): %v", in, err)
		}
		if currency != want[0] || micros != want[1] {
			t.Errorf("parsePrice(%q) = %s %s, want %s %s", in, currency, micros, want[0], want[1])
		}
	}
	if _, _, err := parsePrice("480"); err == nil {
		t.Error("expected an error for a price without a currency")
	}
}

func TestFormatPrice(t *testing.T) {
	if got := formatPrice("JPY", "480000000"); got != "JPY 480" {
		t.Errorf("formatPrice = %q, want \"JPY 480\"", got)
	}
	if got := formatPrice("USD", "1990000"); got != "USD 1.99" {
		t.Errorf("formatPrice = %q, want \"USD 1.99\"", got)
	}
	if got := formatPrice("", ""); got != "-" {
		t.Errorf("formatPrice of nothing = %q, want \"-\"", got)
	}
}

func TestEpochMillis(t *testing.T) {
	if got, err := epochMillis("1750000000000"); err != nil || got != "1750000000000" {
		t.Fatalf("epochMillis(millis) = %q, %v", got, err)
	}
	got, err := epochMillis("2026-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if got != "1767225600000" {
		t.Fatalf("epochMillis(RFC3339) = %q, want 1767225600000", got)
	}
	if _, err := epochMillis("not a time"); err == nil {
		t.Fatal("expected an error for an unparseable timestamp")
	}
}

func TestAppPathEscapesSegments(t *testing.T) {
	if got := appPath("com.example.app", "/edits/%s", esc("a/b")); got != "/applications/com.example.app/edits/a%2Fb" {
		t.Errorf("appPath = %q", got)
	}
}

// writePNG writes a solid PNG of the given size, for image validation tests.
func writePNG(t *testing.T, path string, width, height int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.Set(0, 0, color.White)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}
