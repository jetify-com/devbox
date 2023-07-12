package nix

import (
	"encoding/json"
	"os"
	"strconv"
)

func EvalPackageName(path string) (string, error) {
	cmd := command("eval", "--raw", path+".name")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// PackageIsInsecure is a fun little nix eval that maybe works.
func PackageIsInsecure(path string) bool {
	cmd := command("eval", path+".meta.insecure")
	out, err := cmd.Output()
	if err != nil {
		// We can't know for sure, but probably not.
		return false
	}
	var insecure bool
	if err := json.Unmarshal(out, &insecure); err != nil {
		// We can't know for sure, but probably not.
		return false
	}
	return insecure
}

func PackageKnownVulnerabilities(path string) []string {
	cmd := command("eval", path+".meta.knownVulnerabilities")
	out, err := cmd.Output()
	if err != nil {
		// We can't know for sure, but probably not.
		return nil
	}
	var vulnerabilities []string
	if err := json.Unmarshal(out, &vulnerabilities); err != nil {
		// We can't know for sure, but probably not.
		return nil
	}
	return vulnerabilities
}

func AllowInsecurePackages() {
	os.Setenv("NIXPKGS_ALLOW_INSECURE", "1")
}

func IsInsecureAllowed() bool {
	allowed, _ := strconv.ParseBool(os.Getenv("NIXPKGS_ALLOW_INSECURE"))
	return allowed
}
