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
        "can_modify_stdin": true,
        "can_read_stdin": true
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
- `can_read_stdin` - Allow the hook to read from stdin

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
        "can_modify_stderr": true,
        "can_read_stdin": true,
        "can_read_stdout": true,
        "can_read_stderr": true
      }
    ]
  }
}
```

**Capability Gates:**
- `can_modify_exit` - Allow the hook to modify the exit code
- `can_modify_stdout` - Allow the hook to modify stdout
- `can_modify_stderr` - Allow the hook to modify stderr
- `can_read_stdin` - Allow the hook to read from stdin
- `can_read_stdout` - Allow the hook to read from stdout
- `can_read_stderr` - Allow the hook to read from stderr

## Read Capability Gates

Read capability gates control whether a hook can access stream data (stdin, stdout, stderr). These are separate from modify capabilities to provide fine-grained access control.

**Pre-run hooks:**
- `can_read_stdin` - Allow the hook to read from stdin (default: false)

**Post-run hooks:**
- `can_read_stdin` - Allow the hook to read from stdin (default: false)
- `can_read_stdout` - Allow the hook to read from stdout (default: false)
- `can_read_stderr` - Allow the hook to read from stderr (default: false)

**Important notes:**
- Read capabilities are independent of modify capabilities - you can have `can_read_stdin: true` without `can_modify_stdin: true`
- When a hook doesn't have a read capability, it receives a closed reader (immediate EOF) instead of the actual stream
- This allows multiple hooks in a pipeline to have different access levels - one hook can read stdin while another cannot
- The command wrapper always has full access to stdin/stdout/stderr regardless of hook capabilities

**Example: Selective read access**

```json
{
  "shell": {
    "pre_run": [
      {
        "command": "audit-input.sh",
        "can_read_stdin": true
      },
      {
        "command": "check-policy.sh",
        "can_block": true
      }
    ]
  }
}
```

In this example, `audit-input.sh` can read stdin to log it, but `check-policy.sh` cannot read stdin - it only receives the command context via environment variables.

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

## Current Limitations

### Memory Usage
The current implementation captures stdout and stderr entirely in memory when post-run hooks have `can_modify_stdout` or `can_modify_stderr` capabilities. This means:
- Commands that output large amounts of data (gigabytes) may cause memory exhaustion
- There are no size limits or streaming mechanisms
- This is not suitable for processing large outputs or binary data

### Pipeline Handling
The current implementation does not fully support stdin/stdout/stderr pipelines:
- **stdin**: Not currently captured or passed to hooks (even with `can_modify_stdin`)
- **stdout/stderr**: Captured in memory, not streamed incrementally
- Hooks cannot process data in chunks like Linux pipes

### Read Access Control
The current implementation now provides read capability gates:
- Hooks can be granted or denied access to stdin/stdout/stderr via `can_read_stdin`, `can_read_stdout`, `can_read_stderr`
- When a hook doesn't have a read capability, it receives a closed reader (immediate EOF) instead of the actual stream
- This allows multiple hooks in a pipeline to have different access levels
- Note: Hooks run with user permissions and can still access system resources directly - read capability gates only control structured stream access

## Streaming Support

The hook system now supports streaming for hooks that have stdin/stdout/stderr modification capabilities. This enables efficient processing of large outputs without memory exhaustion.

### Streaming Pipeline

When hooks have `can_modify_stdin`, `can_modify_stdout`, or `can_modify_stderr` capabilities, or when a `command_wrapper` is configured, Devbox uses a streaming pipeline:

```
stdin -> [pre_run hooks] -> [command_wrapper] -> [actual command] -> [post_run hooks] -> stdout
```

Each stage is connected via pipes, allowing data to flow incrementally without loading everything into memory.

### Streaming Behavior

- **Pre-run hooks with `can_modify_stdin`**: Can read from stdin and write to stdout, which becomes the input to the next stage
- **Command wrapper**: Receives stdin from pre-run hooks (or original stdin) and its stdout goes to the actual command
- **Post-run hooks with `can_modify_stdout` or `can_modify_stderr`**: Receive streaming stdin from the previous stage and can process it incrementally
- **Memory efficiency**: Large outputs are streamed through pipes rather than captured entirely in memory

### When Streaming is Used

Streaming is automatically enabled when:
- Any pre-run hook has `can_modify_stdin: true`
- Any post-run hook has `can_modify_stdout: true` or `can_modify_stderr: true`
- A `command_wrapper` is configured

For backward compatibility, hooks without these capabilities use the original non-streaming implementation.

### Example Streaming Hook

A streaming hook that processes output line by line:

```json
{
  "shell": {
    "post_run": [{
      "command": "while read line; do echo \"PROCESSED: $line\"; done",
      "can_modify_stdout": true
    }]
  }
}
```

This hook processes each line of command output as it arrives, rather than waiting for the entire output to complete.

## Future Enhancements

Planned improvements to address remaining limitations:

1. **Incremental Processing** - Allow hooks to process data in chunks rather than requiring full in-memory capture
2. **Stdin Support** - Add stdin capture and passing to hooks for pre-run and post-run processing
3. **Configurable Size Limits** - Make the 1MB JSON output limit configurable for different use cases

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
