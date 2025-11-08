# Using Devbox with Nushell

Devbox supports nushell through the `--format` flag on the `shellenv` command.

## Global Configuration

To use devbox global packages with nushell, add the following to your nushell configuration file:

### Option 1: Add to `~/.config/nushell/env.nu` (recommended)

```nushell
# Load devbox global environment
devbox global shellenv --format nushell | save -f ~/.cache/devbox-env.nu
source ~/.cache/devbox-env.nu
```

### Option 2: Add to `~/.config/nushell/config.nu`

```nushell
# Load devbox global environment
devbox global shellenv --format nushell | save -f ~/.cache/devbox-env.nu
source ~/.cache/devbox-env.nu
```

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
