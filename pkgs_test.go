package devbox

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// needNixPkgs skips the test if Nix or any of the given packages aren't
// installed.
func needNixPkgs(t *testing.T, needPkgs ...string) {
	t.Helper()

	if _, err := exec.LookPath("nix"); err != nil {
		t.Skip("nix command not found in PATH:", os.Getenv("PATH"))
	}
	if _, err := os.Stat("/nix/store"); errors.Is(err, os.ErrNotExist) {
		t.Skip("Nix store not found at /nix/store:", err)
	}

	missing := make([]string, 0, len(needPkgs))
	for _, pkgName := range needPkgs {
		path := filepath.Join("/nix/store", pkgName)
		_, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			missing = append(missing, path)
		}
	}
	if len(missing) > 0 {
		joined := strings.Join(missing, " ")
		t.Skipf("This test requires Nix packages that are missing in /nix/store. "+
			"To install them, run:\nnix-store --realise %s", joined)
	}
}

func TestPackageDirectDependencies(t *testing.T) {
	// Coincidentally, Go is one of the larger packages, which makes it a
	// good test candidate.
	const goPkgName = "nlk4zqsryciwiq8qlbr7fca0031yyv09-go-1.18.4"
	needNixPkgs(t, goPkgName)

	store := LocalNixStore("/nix/store")
	wantPkgNames := []string{
		"1w94mkhazwa3271828qwm17xnqs975gd-perl-5.34.1",
		"1wrxx15a7hkihn77x77yip24q3jbxj44-coreutils-9.1",
		"4sx92yl55cm6f4pnrfkq6d2salm6j13d-apple-framework-Foundation-11.0.0",
		"65qwm1z43rjxm8n7ajq37cp6aw466rcf-mailcap-2.1.53",
		"7k3nqj8bpmpsk5vf8933hvs12q9307vc-bash-5.1-p16",
		"amkqyy3v347mkdd9x1116nawinnc1r3j-libSystem-11.0.0",
		"b9q47dm0qm4ai8cpbkksd8figg06z1m4-iana-etc-20220520",
		"fszwqjcjwjpqrb6hc06i3xnzwap1afm6-tzdata-2022a",
		"griqc100a3gc6b2z1ydx5zh97zmqbnvi-clang-wrapper-11.1.0",
		"j09bsigc968ksp2gng1fkfk2f7zn5hl9-xcodebuild-0.1.2-pre",
		"whd6jpk0xgrysm5snlw3a9319q20nb1v-apple-framework-Security-11.0.0",
	}
	sort.Strings(wantPkgNames)

	pkg, err := store.Package(goPkgName)
	if err != nil {
		t.Fatal("Got store.Package error:", err)
	}
	gotPkgNames := make([]string, len(pkg.DirectDependencies))
	for i, pkg := range pkg.DirectDependencies {
		gotPkgNames[i] = pkg.StoreName
	}
	sort.Strings(gotPkgNames)

	gotStr, wantStr := strings.Join(gotPkgNames, " "), strings.Join(wantPkgNames, " ")
	if gotStr != wantStr {
		t.Errorf("Got wrong dependencies for %q.\ngot:  %s\nwant: %s",
			goPkgName, gotStr, wantStr)
	}
}
