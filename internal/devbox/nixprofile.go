package devbox

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"slices"

	"go.jetpack.io/devbox/internal/nix"
)

func syncFlakeToProfile(ctx context.Context, flakePath, profilePath string) error {
	cmd := exec.CommandContext(ctx, "nix", "eval", "--json", flakePath+"#devShells."+nix.System()+".default.buildInputs")
	b, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("nix eval devShells: %v", err)
	}
	storePaths := []string{}
	if err := json.Unmarshal(b, &storePaths); err != nil {
		return fmt.Errorf("unmarshal store paths: %s: %v", b, err)
	}

	listCmd := exec.CommandContext(ctx, "nix", "profile", "list", "--json", "--profile", profilePath)
	b, err = listCmd.Output()
	if err != nil {
		return err
	}
	var profile struct {
		Elements []struct {
			StorePaths []string
		}
	}
	if err := json.Unmarshal(b, &profile); err != nil {
		return fmt.Errorf("unmarshal profile: %v", err)
	}
	got := make([]string, 0, len(profile.Elements))
	for _, e := range profile.Elements {
		got = append(got, e.StorePaths...)
	}

	add, remove := diffStorePaths(got, storePaths)
	if len(remove) > 0 {
		removeCmd := exec.CommandContext(ctx, "nix", "profile", "remove", "--profile", profilePath)
		removeCmd.Args = append(removeCmd.Args, remove...)
		if err := removeCmd.Run(); err != nil {
			return err
		}
	}
	if len(add) > 0 {
		addCmd := exec.CommandContext(ctx, "nix", "profile", "install", "--profile", profilePath)
		addCmd.Args = append(addCmd.Args, add...)
		if err := addCmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

func diffStorePaths(got, want []string) (add, remove []string) {
	slices.Sort(got)
	slices.Sort(want)

	var g, w int
	for {
		if g >= len(got) {
			add = append(add, want[w:]...)
			break
		}
		if w >= len(want) {
			remove = append(remove, got[g:]...)
			break
		}

		switch {
		case got[g] == want[w]:
			g++
			w++
		case got[g] < want[w]:
			remove = append(remove, got[g])
			g++
		case got[g] > want[w]:
			add = append(add, want[w])
			w++
		}
	}
	return add, remove
}
