package nix

import (
	"context"
	"os/user"
	"testing"
)

//nolint:revive
func TestConfigIsUserTrusted(t *testing.T) {
	curUser, err := user.Current()
	if err != nil {
		t.Fatal("lookup current user:", err)
	}

	t.Run("UsernameInList", func(t *testing.T) {
		t.Setenv("NIX_CONFIG", "trusted-users = "+curUser.Username)

		ctx := context.Background()
		cfg, err := CurrentConfig(ctx)
		if err != nil {
			t.Fatal(err)
		}

		trusted, err := cfg.IsUserTrusted(ctx, curUser.Username)
		if err != nil {
			t.Fatal(err)
		}
		if !trusted {
			t.Error("got trusted = false, want true")
		}
	})
	t.Run("UserGroupInList", func(t *testing.T) {
		g, err := user.LookupGroupId(curUser.Gid)
		if err != nil {
			t.Fatal(err)
		}
		t.Setenv("NIX_CONFIG", "trusted-users = @"+g.Name)

		ctx := context.Background()
		cfg, err := CurrentConfig(ctx)
		if err != nil {
			t.Fatal(err)
		}

		trusted, err := cfg.IsUserTrusted(ctx, curUser.Username)
		if err != nil {
			t.Fatal(err)
		}
		if !trusted {
			t.Error("got trusted = false, want true")
		}
	})
	t.Run("NotInList", func(t *testing.T) {
		t.Setenv("NIX_CONFIG", "trusted-users = root")

		ctx := context.Background()
		cfg, err := CurrentConfig(ctx)
		if err != nil {
			t.Fatal(err)
		}

		trusted, err := cfg.IsUserTrusted(ctx, curUser.Username)
		if err != nil {
			t.Fatal(err)
		}
		if trusted {
			t.Error("got trusted = true, want false")
		}
	})
	t.Run("EmptyList", func(t *testing.T) {
		t.Setenv("NIX_CONFIG", "trusted-users =")

		ctx := context.Background()
		cfg, err := CurrentConfig(ctx)
		if err != nil {
			t.Fatal(err)
		}

		trusted, err := cfg.IsUserTrusted(ctx, curUser.Username)
		if err != nil {
			t.Fatal(err)
		}
		if trusted {
			t.Error("got trusted = true, want false")
		}
	})
	t.Run("UnknownGroup", func(t *testing.T) {
		t.Setenv("NIX_CONFIG", "trusted-users = @dummygroup")

		ctx := context.Background()
		cfg, err := CurrentConfig(ctx)
		if err != nil {
			t.Fatal(err)
		}

		trusted, err := cfg.IsUserTrusted(ctx, curUser.Username)
		if err != nil {
			t.Fatal(err)
		}
		if trusted {
			t.Error("got trusted = true, want false")
		}
	})
}
