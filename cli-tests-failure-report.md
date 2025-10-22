# CLI Tests Failure Report - GitHub API Rate Limiting

## Executive Summary

The cli-tests on the `main` branch have been failing intermittently in GitHub Actions since at least October 7, 2025, due to GitHub API rate limiting when Nix attempts to fetch nixpkgs metadata. Despite having access tokens configured, the tests are still hitting unauthenticated rate limits.

## Problem Description

### Error Message
```
unable to download 'https://api.github.com/repos/NixOS/nixpkgs/commits/nixpkgs-unstable': HTTP error 403
API rate limit exceeded for 13.105.49.133. (But here's the good news: Authenticated requests get a higher rate limit. Check out the documentation for more details.)
```

### Affected Tests
Multiple test files are affected, including:
- `add/add.test.txt:7`
- `run/env.test.txt:7`
- `lockfile/lockfile_tidy.test.txt:5`
- `lockfile/nopaths.txt:4`
- `languages/python_patch_old_glibc.test.txt:6`
- And many more...

### Timeline
- Tests have been failing on main since October 7, 2025 (possibly earlier)
- Last successful run before recent fixes: October 21, 2025 at 20:28 UTC
- Failed runs: October 22 at 08:39 UTC (run ID: 18710425439)
- Successful run after fixes: October 22 at 15:35 UTC (run ID: 18721603294)

## Root Cause Analysis

### Configuration Status
The GitHub Actions workflow (`.github/workflows/cli-tests.yaml`) has the following authentication setup:

1. **Environment Variables** (lines 41-53):
   ```yaml
   env:
     DEVBOX_GITHUB_API_TOKEN: ${{ secrets.GITHUB_TOKEN }}
     GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
     HOMEBREW_GITHUB_API_TOKEN: ${{ secrets.GITHUB_TOKEN }}
     NIX_CONFIG: |
       access-tokens = github.com=${{ secrets.GITHUB_TOKEN }}
   ```

2. **Nix Config File Setup** (lines 181-185):
   ```yaml
   - name: Setup Nix GitHub authentication
     run: |
       mkdir -p ~/.config/nix
       echo "access-tokens = github.com=${{ secrets.GITHUB_TOKEN }}" > ~/.config/nix/nix.conf
   ```

3. **Verified Configuration**:
   - The logs confirm `nix show-config` shows: `access-tokens = github.com=***`
   - The configuration is present in both environment and config file

### The Issue

Despite the access tokens being configured correctly, Nix is still being rate-limited. The key evidence:

1. **IP Address in Error**: The error message shows `13.105.49.133`, which is a GitHub Actions runner IP
2. **Unauthenticated Behavior**: GitHub's response indicates the request is unauthenticated
3. **Token Not Working**: The configured token is not being used for API authentication

### Potential Causes

1. **Nix Daemon vs Direct Execution**:
   - On macOS runners, Nix may use a daemon that doesn't inherit the environment variables
   - The daemon runs as a different user and may not read `~/.config/nix/nix.conf`
   - The daemon reads `/etc/nix/nix.conf` instead

2. **Token Format**:
   - GitHub Actions' `GITHUB_TOKEN` may need special handling for Nix
   - The token might need to be in a specific format or need additional scopes

3. **Intermittent Nature**:
   - Rate limits are shared across the GitHub Actions IP pool
   - Tests may succeed when runners have available quota
   - Tests fail when quota is exhausted

## Evidence

### Test Logs
From failed run 18710425439:
```
time=2025-10-22T08:49:01.781Z level=DEBUG msg="nix command exited"
cmd.args="nix --extra-experimental-features ca-derivations --option experimental-features 'nix-command flakes fetch-closure' flake metadata --json github:NixOS/nixpkgs/nixpkgs-unstable"
cmd.stderr="unable to download 'https://api.github.com/repos/NixOS/nixpkgs/commits/nixpkgs-unstable': HTTP error 403"
cmd.code=1
```

### Configuration Verification
The `nix show-config` output confirms:
```
access-tokens =  github.com=***
```

## Impact

- Tests on `main` branch fail intermittently
- Pull request tests may also be affected
- False negatives reduce confidence in CI
- Developers may merge code thinking tests passed when they didn't actually run successfully

## Proposed Solutions

### Solution 1: Configure Nix Daemon Properly (macOS-specific)
For macOS runners, configure the system-wide Nix configuration:

```yaml
- name: Setup Nix GitHub authentication for daemon
  run: |
    # For macOS, configure system-wide nix.conf
    if [ "$RUNNER_OS" == "macOS" ]; then
      echo "access-tokens = github.com=${{ secrets.GITHUB_TOKEN }}" | sudo tee -a /etc/nix/nix.conf
      # Restart nix daemon to pick up config
      sudo launchctl stop org.nixos.nix-daemon
      sudo launchctl start org.nixos.nix-daemon
    fi
    # For Linux, user config should work
    mkdir -p ~/.config/nix
    echo "access-tokens = github.com=${{ secrets.GITHUB_TOKEN }}" > ~/.config/nix/nix.conf
```

### Solution 2: Pass Token via Command-Line Options
Modify the Nix command execution in `internal/nix/command.go` to pass the access token explicitly:

```go
func init() {
    Default.ExtraArgs = Args{
        "--extra-experimental-features", "ca-derivations",
        "--option", "experimental-features", "nix-command flakes fetch-closure",
    }

    // Add GitHub token if available
    if token := os.Getenv("GITHUB_TOKEN"); token != "" {
        Default.ExtraArgs = append(Default.ExtraArgs,
            "--option", "access-tokens", "github.com="+token)
    }
}
```

### Solution 3: Use netrc for Authentication
Configure Git-style credentials that Nix can use:

```yaml
- name: Setup GitHub netrc authentication
  run: |
    cat > ~/.netrc <<EOF
    machine github.com
    login ${{ github.actor }}
    password ${{ secrets.GITHUB_TOKEN }}
    EOF
    chmod 600 ~/.netrc
```

### Solution 4: Pre-fetch nixpkgs Metadata
Cache the nixpkgs flake metadata to avoid repeated API calls:

```yaml
- name: Pre-fetch nixpkgs flake
  run: |
    nix flake prefetch github:NixOS/nixpkgs/nixpkgs-unstable --refresh
```

## Recommended Approach

Implement **Solution 1** (configure Nix daemon properly) as the primary fix, with **Solution 2** (command-line options) as a backup. This ensures:

1. The Nix daemon on macOS runners has access to the token
2. The token is passed explicitly in case environment/config isn't picked up
3. Both Linux and macOS runners are covered

## Verification Plan

1. Implement the fix on a test branch
2. Create a PR to trigger CI tests
3. Monitor multiple test runs to ensure consistency
4. Check logs to confirm tokens are being used (look for different IP or successful API calls)
5. Verify tests pass consistently over multiple runs

## Additional Notes

- The tests succeeded on October 22 at 15:35 UTC without code changes, suggesting the issue is intermittent and related to shared rate limits
- Recent successful runs may indicate GitHub temporarily increased rate limits or the runner pool had available quota
- Long-term solution should include caching nixpkgs metadata to reduce API calls
