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
	ab := struct{ A, B string }{"a", "b"}
	abHash, err := JSON(ab)
	if err != nil {
		t.Errorf("got JSON(%#q) error: %v", ab, err)
	}
	if abHash == "" {
		t.Errorf(`got JSON(%#q) == ""`, ab)
	}

	ba := struct{ B, A string }{"b", "a"}
	bHash, err := JSON(ba)
	if err != nil {
		t.Errorf("got JSON(%#q) error: %v", ba, err)
	}
	if bHash == "" {
		t.Errorf(`got JSON(%#q) == ""`, ba)
	}
	if abHash != bHash {
		t.Errorf("got (JSON(%#q) = %q) != (JSON(%#q) = %q), want equal hashes", ab, abHash, ba, bHash)
	}
}

func TestJSONUnsupportedType(t *testing.T) {
	j := struct{ C chan int }{}
	_, err := JSON(j)
	if err == nil {
		t.Error("got nil error for struct with channel field")
	}
}

func TestJSONBytesInvalid(t *testing.T) {
	b := []byte("bad")
	_, err := JSONBytes(b)
	if err == nil {
		t.Error("got nil error for invalid JSON")
	}
}

func TestJSONFile(t *testing.T) {
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

	abHash, err := JSONFile(ab)
	if err != nil {
		t.Errorf("got JSONFile(ab) error: %v", err)
	}
	baHash, err := JSONFile(ba)
	if err != nil {
		t.Errorf("got JSONFile(ba) error: %v", err)
	}
	if abHash != baHash {
		t.Errorf("got (JSONFile(%q) = %q) != (JSONFile(%q) = %q), want equal hashes", ab, abHash, ba, baHash)
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
