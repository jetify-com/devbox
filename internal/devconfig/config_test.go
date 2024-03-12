package devconfig

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tailscale/hujson"
	"go.jetpack.io/devbox/internal/devconfig/configfile"
)

func TestDefault(t *testing.T) {
	path := filepath.Join(t.TempDir())
	cfg := DefaultConfig()
	inBytes := cfg.Root.Bytes()
	if _, err := hujson.Parse(inBytes); err != nil {
		t.Fatalf("default config JSON is invalid: %v\n%s", err, inBytes)
	}
	err := cfg.Root.SaveTo(path)
	if err != nil {
		t.Fatal("got save error:", err)
	}
	out, err := LoadForTest(filepath.Join(path, configfile.DefaultName))
	if err != nil {
		t.Fatal("got load error:", err)
	}
	if diff := cmp.Diff(
		cfg,
		out,
		cmpopts.IgnoreUnexported(configfile.ConfigFile{}, configfile.PackagesMutator{}, Config{}),
		cmpopts.IgnoreFields(configfile.ConfigFile{}, "AbsRootPath"),
	); diff != "" {
		t.Errorf("configs not equal (-in +out):\n%s", diff)
	}

	outBytes := out.Root.Bytes()
	if _, err := hujson.Parse(outBytes); err != nil {
		t.Fatalf("loaded default config JSON is invalid: %v\n%s", err, outBytes)
	}
	if string(inBytes) != string(outBytes) {
		t.Errorf("got different JSON after load/save/load:\ninput:\n%s\noutput:\n%s", inBytes, outBytes)
	}
}
