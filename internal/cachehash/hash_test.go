//nolint:varnamelen
package cachehash

import (
	"os"
	"path/filepath"
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
