package nix

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/lock"
)

var ErrPackageNotFound = errors.New("package not found")
var ErrPackageNotInstalled = errors.New("package not installed")

func PkgExists(pkg string, lock *lock.File) (bool, error) {
	return InputFromString(pkg, lock).validateExists()
}

type Info struct {
	// attribute key is different in flakes vs legacy so we should only use it
	// if we know exactly which version we are using
	attributeKey string
	PName        string
	Version      string
}

func (i *Info) String() string {
	return fmt.Sprintf("%s-%s", i.PName, i.Version)
}

func PkgInfo(nixpkgsCommit, pkg string) *Info {
	exactPackage := fmt.Sprintf("%s#%s", FlakeNixpkgs(nixpkgsCommit), pkg)
	if nixpkgsCommit == "" {
		exactPackage = fmt.Sprintf("nixpkgs#%s", pkg)
	}

	results := search(exactPackage)
	if len(results) == 0 {
		return nil
	}
	// we should only have one result
	return lo.Values(results)[0]
}

func search(url string) map[string]*Info {
	return searchSystem(url, "")
}

func parseSearchResults(data []byte) map[string]*Info {
	var results map[string]map[string]any
	err := json.Unmarshal(data, &results)
	if err != nil {
		panic(err)
	}
	infos := map[string]*Info{}
	for key, result := range results {
		infos[key] = &Info{
			attributeKey: key,
			PName:        result["pname"].(string),
			Version:      result["version"].(string),
		}

	}
	return infos
}

// pkgExistsForAnySystem is a bit slow (~600ms). Only use it if there's already
// been an error and we want to provide a better error message.
func pkgExistsForAnySystem(pkg string) bool {
	systems := []string{
		// Check most common systems first.
		"x86_64-linux",
		"x86_64-darwin",
		"aarch64-linux",
		"aarch64-darwin",

		"armv5tel-linux",
		"armv6l-linux",
		"armv7l-linux",
		"i686-linux",
		"mipsel-linux",
		"powerpc64le-linux",
		"riscv64-linux",
	}
	for _, system := range systems {
		if len(searchSystem(pkg, system)) > 0 {
			return true
		}
	}
	return false
}

func searchSystem(url string, system string) map[string]*Info {
	cmd := exec.Command("nix", "search", "--json", url)
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)
	if system != "" {
		cmd.Args = append(cmd.Args, "--system", system)
	}
	cmd.Stderr = os.Stderr
	debug.Log("running command: %s\n", cmd)
	out, err := cmd.Output()
	if err != nil {
		// for now, assume all errors are invalid packages.
		return nil
	}
	return parseSearchResults(out)
}
