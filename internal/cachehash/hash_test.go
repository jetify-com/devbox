//nolint:varnamelen
package cachehash

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFile(t *testing.T) {
	dir := t.TempDir()

	ab := filepath.Join(dir, "ab.json")
	err := os.WriteFile(ab, []byte(`{"a":"\n","b":"\u000A"}`), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	ba := filepath.Join(dir, "ba.json")
	err = os.WriteFile(ba, []byte(`{"b":"\n","a":"\u000A"}`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	abHash, err := File(ab)
	if err != nil {
		t.Errorf("got File(ab) error: %v", err)
	}
	baHash, err := File(ba)
	if err != nil {
		t.Errorf("got File(ba) error: %v", err)
	}
	if abHash == baHash {
		t.Errorf("got (File(%q) = %q) == (File(%q) = %q), want different hashes", ab, abHash, ba, baHash)
	}
}

func TestFileNotExist(t *testing.T) {
	t.TempDir()
	hash, err := File(t.TempDir() + "/notafile")
	if err != nil {
		t.Errorf("got error: %v", err)
	}
	if hash != "" {
		t.Errorf("got non-empty hash %q", hash)
	}
}

func TestJSON(t *testing.T) {
	a := struct{ A, B string }{"a", "b"}
	aHash, err := JSON(a)
	if err != nil {
		t.Errorf("got JSON(%#q) error: %v", a, err)
	}
	if aHash == "" {
		t.Errorf(`got JSON(%#q) == ""`, a)
	}

	b := map[string]string{"A": "a", "B": "b"}
	bHash, err := JSON(b)
	if err != nil {
		t.Errorf("got JSON(%#q) error: %v", b, err)
	}
	if bHash == "" {
		t.Errorf(`got JSON(%#q) == ""`, b)
	}
	if aHash != bHash {
		t.Errorf("got (JSON(%#q) = %q) != (JSON(%#q) = %q), want equal hashes", a, aHash, b, bHash)
	}
}

func TestJSONUnsupportedType(t *testing.T) {
	j := struct{ C chan int }{}
	_, err := JSON(j)
	if err == nil {
		t.Error("got nil error for struct with channel field")
	}
}

func TestJSONFile(t *testing.T) {
	dir := t.TempDir()

	compact := filepath.Join(dir, "compact.json")
	err := os.WriteFile(compact, []byte(`{"key":"value"}`), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	space := filepath.Join(dir, "space.json")
	err = os.WriteFile(space, []byte(`{ "key": "value" }`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	compactHash, err := JSONFile(compact)
	if err != nil {
		t.Errorf("got JSONFile(ab) error: %v", err)
	}
	spaceHash, err := JSONFile(space)
	if err != nil {
		t.Errorf("got JSONFile(ba) error: %v", err)
	}
	if compactHash != spaceHash {
		t.Errorf("got (JSONFile(%q) = %q) != (JSONFile(%q) = %q), want equal hashes", compact, compactHash, space, spaceHash)
	}
}

func TestJSONFileNotExist(t *testing.T) {
	t.TempDir()
	hash, err := JSONFile(t.TempDir() + "/notafile")
	if err != nil {
		t.Errorf("got error: %v", err)
	}
	if hash != "" {
		t.Errorf("got non-empty hash %q", hash)
	}
}

func TestSlug(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{"basic", "HelloWorld", "helloworld-872e4e"},
		{"special chars", "Hello, World!", "hello-world-dffd60"},
		{"empty string", "", "e3b0c4"},
		{"leading special char", "@hello", "athello-8f8a2f"},
		{
			"url",
			"https://example.com/path?query=foo.bar",
			"https-example-com-path-query-foo-bar-3e60ff",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Slug(c.input)
			if got != c.expected {
				t.Errorf("Slug(%q) == %q, want %q", c.input, got, c.expected)
			}
		})
	}

	// test that 2 similar strings have different slugs
	s1 := "Hello, World!"
	s2 := "Hello, World"
	if Slug(s1) == Slug(s2) {
		t.Errorf("Slug(%q) == Slug(%q), want different slugs", s1, s2)
	}

	// Test that 2 super long truncated strings have the same slug
	s3 := "	" + strings.Repeat("a", 1000)
	s4 := "	" + strings.Repeat("a", 1000)
	if Slug(s3) != Slug(s4) {
		t.Errorf("Slug(%q) != Slug(%q), want same slug", s3, s4)
	}

	// Test that 2 super long strings have different slugs
	s5 := strings.Repeat("a", 1000)
	s6 := strings.Repeat("a", 1000) + "b"
	if Slug(s5) == Slug(s6) {
		t.Errorf("Slug(%q) == Slug(%q), want different slugs", s5, s6)
	}
}
