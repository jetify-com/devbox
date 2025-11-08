# Using Devbox with Nushell

Devbox supports nushell through the `--format` flag on the `shellenv` command.

## Quick Start

**Add this to `~/.config/nushell/env.nu`:**

```nushell
devbox global shellenv --format nushell --preserve-path-stack -r
  | lines 
  | parse "$env.{name} = \"{value}\""
  | where name != null 
  | transpose -r 
  | into record 
  | load-env
```

This is equivalent to bash's `eval "$(devbox global shellenv)"` and runs fresh on every shell start.

---

## Global Configuration

To use devbox global packages with nushell, you need to load the environment similar to how bash/zsh use `eval "$(devbox global shellenv)"`.

### Option 1 (Recommended): Dynamic loading with `load-env` - True eval equivalent

Add this to `~/.config/nushell/env.nu` to regenerate and load devbox environment fresh every time, just like bash's `eval`:

```nushell
# Load devbox global environment dynamically (equivalent to bash eval)
devbox global shellenv --format nushell --preserve-path-stack -r
  | lines 
  | parse "$env.{name} = \"{value}\""
  | where name != null 
  | transpose -r 
  | into record 
  | load-env
```

**Flags explained:**

- `--format nushell` - Output in nushell syntax
- `--preserve-path-stack` - Maintain existing PATH order if devbox is already active
- `-r` (recompute) - Always recompute the environment, prevents "out of date" warnings

**Advantages:**

- ✅ Regenerates environment every time (like `eval "$(devbox global shellenv)"`)
- ✅ No cached files
- ✅ Always up to date
- ✅ Works in a single shell startup

### Option 2: Cache-based loading (faster startup)

If you prefer faster shell startup, use a cached file approach:

**First time setup:**

```nushell
# Run this once to generate the cache file
devbox global shellenv --format nushell --preserve-path-stack -r | save -f ~/.cache/devbox-global-env.nu --force
```

**Then add to `~/.config/nushell/env.nu`:**

```nushell
# Load devbox from cache (faster, but needs manual refresh)
source ~/.cache/devbox-global-env.nu
```

**Add this refresh command to `~/.config/nushell/config.nu`:**

```nushell
# Refresh devbox environment when needed
def devbox-refresh [] {
    devbox global shellenv --format nushell --preserve-path-stack -r | save -f ~/.cache/devbox-global-env.nu --force
    print "Devbox environment updated! Restart your shell to apply changes."
}
```

**When to refresh:** Run `devbox-refresh` after:

- Adding/removing global packages (`devbox global add/rm`)
- Updating packages (`devbox global update`)

### How It Works (Compared to Bash eval)

**Bash/Zsh:**

```bash
eval "$(devbox global shellenv)"  # Generates and executes every time
```

**Nushell Option 1 (Dynamic - Recommended):**

```nushell
devbox global shellenv --format nushell --preserve-path-stack -r 
  | lines | parse ... | load-env  # Generates and loads every time, just like bash eval
```

**Nushell Option 2 (Cached):**

```nushell
source ~/.cache/devbox-global-env.nu  # Loads from cache, faster but needs manual refresh
```

**Why the extra flags?**

- `--preserve-path-stack` - Prevents PATH conflicts if devbox is already active
- `-r` (recompute) - Forces environment regeneration, eliminates "out of date" warnings

## Project-Specific Shells

For project-specific devbox shells, nushell users have a few options:

### Option 1: Enter a devbox shell normally

```nushell
devbox shell
```

This will start a new shell with your devbox environment activated.

### Option 2: Load environment in current nushell session

```nushell
devbox shellenv --format nushell | save -f /tmp/devbox-env.nu
source /tmp/devbox-env.nu
```

## Differences from Bash/Zsh

### Environment Variable Syntax

Nushell uses a different syntax for environment variables:

**Bash/Zsh:**

```bash
export MY_VAR="value"
```

**Nushell:**

```nushell
$env.MY_VAR = "value"
```

### No eval command

Unlike bash/zsh which use `eval "$(devbox shellenv)"`, nushell doesn't have an `eval` command. Instead, the recommended approach is to:

1. Save the output to a temporary file
2. Source that file

This is why we use the pattern: `devbox global shellenv --format nushell | save -f ~/.cache/devbox-env.nu`

### Refresh command

The refresh alias functionality works differently in nushell. After making changes to your devbox configuration, you can reload the environment with:

```nushell
devbox global shellenv --format nushell | save -f ~/.cache/devbox-env.nu
source ~/.cache/devbox-env.nu
```

## Troubleshooting

### Environment not loading

Make sure your nushell config file is being sourced. You can check by running:

```nushell
echo $env.DEVBOX_PROJECT_ROOT
```

If this returns nothing, devbox hasn't been loaded yet.

### Packages not in PATH

Verify that the devbox environment was properly sourced:

```nushell
echo $env.PATH
```

You should see devbox-related paths in the output.

## More Information

For more information about devbox, visit:

- [Devbox Documentation](https://www.jetify.com/devbox/docs/)
- [GitHub Repository](https://github.com/jetify-com/devbox)
- [Join our Discord](https://discord.gg/jetify)
