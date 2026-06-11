# Devbox Run Hooks

The hook system allows you to intercept and modify command execution when using `devbox run`. This enables use cases like policy enforcement, instrumentation, command wrapping, and output processing.

## Overview

Hooks are configured in the `shell` section of your `devbox.json` file. There are three types of hooks:

1. **Pre-run hooks** (`pre_run`) - Execute before a command runs
2. **Command wrapper** (`command_wrapper`) - Simple string wrapper for all commands
3. **Post-run hooks** (`post_run`) - Execute after a command finishes

## Configuration

### Pre-Run Hooks

Pre-run hooks execute before a command and can modify execution behavior:

```json
{
  "shell": {
    "pre_run": [
      {
        "command": "echo 'About to execute command'",
        "can_block": true,
        "can_modify_args": true,
        "can_modify_env": true,
        "can_modify_stdin": true
      }
    ]
  }
}
```

**Capability Gates:**
- `can_block` - Allow the hook to block command execution
- `can_modify_args` - Allow the hook to modify command arguments
- `can_modify_env` - Allow the hook to modify environment variables
- `can_modify_stdin` - Allow the hook to modify stdin

All capabilities default to `false` for security. You must explicitly enable each capability.

### Command Wrapper

The command wrapper is a simple string that prefixes all commands:

```json
{
  "shell": {
    "command_wrapper": "rtk exec --"
  }
}
```

This would wrap every command as `rtk exec -- <original command>`.

### Post-Run Hooks

Post-run hooks execute after a command finishes and can modify the result:

```json
{
  "shell": {
    "post_run": [
      {
        "command": "echo 'Command finished'",
        "can_modify_exit": true,
        "can_modify_stdout": true,
        "can_modify_stderr": true
      }
    ]
  }
}
```

**Capability Gates:**
- `can_modify_exit` - Allow the hook to modify the exit code
- `can_modify_stdout` - Allow the hook to modify stdout
- `can_modify_stderr` - Allow the hook to modify stderr

## Hook Output Format

Hooks can return JSON to modify execution behavior:

```json
{
  "success": true,
  "block": false,
  "block_reason": "",
  "modified_args": ["arg1", "arg2"],
  "modified_env": {"KEY": "value"},
  "modified_exit": 0,
  "modified_stdout": "output",
  "modified_stderr": "error"
}
```

**Fields:**
- `success` - Whether the hook executed successfully
- `block` - If `true` and `can_block` is enabled, blocks command execution
- `block_reason` - Reason for blocking (required when `block` is true)
- `modified_args` - Modified command arguments (requires `can_modify_args`)
- `modified_env` - Modified environment variables (requires `can_modify_env`)
- `modified_exit` - Modified exit code (requires `can_modify_exit`, post-run only)
- `modified_stdout` - Modified stdout (requires `can_modify_stdout`, post-run only)
- `modified_stderr` - Modified stderr (requires `can_modify_stderr`, post-run only)

## Hook Context

Hooks receive context about the command being executed via environment variables:

**Pre-run hooks:**
- `DEVBOX_HOOK_COMMAND` - The command name
- `DEVBOX_HOOK_ARGS` - Command arguments (JSON array)
- `DEVBOX_HOOK_ENV` - Environment variables (JSON object)
- `DEVBOX_HOOK_DIR` - Project directory

**Post-run hooks:**
- All pre-run context variables, plus:
- `DEVBOX_HOOK_EXIT_CODE` - Exit code of the command
- `DEVBOX_HOOK_STDOUT` - Stdout from the command
- `DEVBOX_HOOK_STDERR` - Stderr from the command

## Use Cases

### Policy Enforcement

Block certain commands or require approval:

```json
{
  "shell": {
    "pre_run": [
      {
        "command": "check-policy.sh",
        "can_block": true
      }
    ]
  }
}
```

### Instrumentation

Log all commands with timing information:

```json
{
  "shell": {
    "pre_run": [
      {
        "command": "log-command-start.sh"
      }
    ],
    "post_run": [
      {
        "command": "log-command-end.sh"
      }
    ]
  }
}
```

### Command Wrapping

Wrap tools like `rtk exec --` around all commands:

```json
{
  "shell": {
    "command_wrapper": "rtk exec --"
  }
}
```

### Environment Modification

Dynamically set environment variables:

```json
{
  "shell": {
    "pre_run": [
      {
        "command": "set-env.sh",
        "can_modify_env": true
      }
    ]
  }
}
```

### Output Processing

Filter or transform command output:

```json
{
  "shell": {
    "post_run": [
      {
        "command": "process-output.sh",
        "can_modify_stdout": true,
        "can_modify_stderr": true
      }
    ]
  }
}
```

## Security

All capability gates default to `false` for security. You must explicitly enable each capability you need. This prevents hooks from accidentally or maliciously modifying execution behavior.

## Example

See the [hooks example](../examples/hooks/) for a complete working example.

## Migration from Workarounds

The hook system provides a clean solution for common workarounds:

- **Shell hooks** (only for `devbox shell`) → Use `pre_run` hooks
- **PATH shims** (bypassed by Devbox) → Use `command_wrapper`
- **Aliasing** (non-interactive shells) → Use `pre_run` hooks
- **$BASH_ENV** (fragile) → Use `pre_run` hooks

## Design Principles

1. **Explicit capability gates** - Security and clarity through explicit permissions
2. **Symmetric design** - Pre-run and post-run hooks follow similar patterns
3. **Default-deny** - Dangerous capabilities are disabled by default
4. **Shell namespace** - Hooks live under `shell` configuration, following Devbox conventions
