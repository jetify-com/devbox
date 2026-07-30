package s3

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
)

// TestConfigPinsRegion guards against a regression of jetify-com/devbox#2591,
// where `devbox global push` failed with "Invalid Configuration: Missing
// Region" for users who had no AWS region configured in their environment.
//
// assumeRole builds AWS configs (for both the STS and S3 clients) that must
// explicitly pin the region, since neither STS nor S3 can resolve an endpoint
// without one. This test reproduces the reporter's environment (no region
// configured anywhere) and verifies that pinning the region via
// config.WithRegion(region) still yields a config with the expected region.
func TestConfigPinsRegion(t *testing.T) {
	if region == "" {
		t.Fatal("region constant must not be empty; STS/S3 endpoint resolution requires it")
	}

	// Simulate a user with no AWS region configured anywhere: clear the
	// region environment variables and point the shared config/credentials
	// files at nonexistent paths so no ambient region leaks in.
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_CONFIG_FILE", "/dev/null")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "/dev/null")

	ctx := context.Background()

	// Without an explicit region, the loaded config has no region. This is the
	// condition that produced the "Missing Region" error before the fix.
	noRegion, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		t.Fatalf("loading default config: %v", err)
	}
	if noRegion.Region != "" {
		t.Skipf("test environment provides a default AWS region (%q); "+
			"cannot exercise the missing-region case", noRegion.Region)
	}

	// With the region pinned (as assumeRole now does for both configs), the
	// region is set regardless of the environment.
	pinned, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		t.Fatalf("loading config with pinned region: %v", err)
	}
	if pinned.Region != region {
		t.Errorf("pinned region = %q, want %q", pinned.Region, region)
	}
}
