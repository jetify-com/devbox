# Using Devbox with Nushell

Devbox now supports [nushell](https://github.com/nushell/nushell) through the `--format` flag on the `shellenv` command.

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

This is equivalent to bash's `eval "$(devbox global shellenv)"` and runs on every fresh shell start.

---

## Global Configuration

To use devbox global packages with nushell, you need to load the environment similar to how bash/zsh use `eval "$(devbox global shellenv)"`.

### Dynamic loading with `load-env` - eval equivalent

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

- `--format nushell` - Output in nushell syntax
- `--preserve-path-stack` - Maintain existing PATH order if devbox is already active
- `-r` (recompute) - Always recompute the environment, prevents "out of date" warnings
