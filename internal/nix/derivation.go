package nix

import (
	"encoding/json"

	"github.com/pkg/errors"
	"golang.org/x/exp/maps"
)

type DerivationShowOutput struct {
	Name string `json:"name"`
}

func DerivationShow(path string) (*DerivationShowOutput, error) {
	cmd := command("derivation", "show", path, "--impure")
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var output map[string]*DerivationShowOutput
	if err := json.Unmarshal(out, &output); err != nil {
		return nil, err
	}
	if len(output) != 1 {
		return nil, errors.Errorf("expected 1 output, got %d", len(output))
	}
	return maps.Values(output)[0], nil

}
