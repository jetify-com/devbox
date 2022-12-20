package rust

import (
	"path/filepath"
	"strings"

	"go.jetpack.io/devbox/internal/pkgsuggest/suggestors"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

// `cargo new` generates a file with uppercase Cargo.toml
const cargoToml = "Cargo.toml"

type Suggestor struct{}

// implements interface Suggestor (compile-time check)
var _ suggestors.Suggestor = (*Suggestor)(nil)

func (s *Suggestor) IsRelevant(srcDir string) bool {
	return cargoTomlPath(srcDir) != ""
}

func (s *Suggestor) Packages(_ string) []string {
	return []string{"rustup"}
}

// Tries to find Cargo.toml or cargo.toml. Returns the path with srcDir if found
// and empty-string if not found.
//
// NOTE: `cargo build` succeeded with lowercase cargo.toml, but `cargo build --release`
// will insist on `Cargo.toml`. We are lenient and tolerate both.
func cargoTomlPath(srcDir string) string {

	cargoTomlPath := filepath.Join(srcDir, cargoToml)
	if plansdk.FileExists(cargoTomlPath) {
		return cargoTomlPath
	}

	lowerCargoTomlPath := filepath.Join(srcDir, strings.ToLower(cargoToml))
	if plansdk.FileExists(lowerCargoTomlPath) {
		return lowerCargoTomlPath
	}
	return ""
}
