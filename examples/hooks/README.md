# Devbox Hooks Example

This example demonstrates the hook system for `devbox run` commands.

## Overview

The hook system allows you to intercept and modify command execution in devbox. There are three types of hooks:

### 1. Pre-Run Hooks (`pre_run`)

Pre-run hooks execute before a command runs. They can:
- Block execution (with `can_block: true`)
- Modify command arguments (with `can_modify_args: true`)
- Modify environment variables (with `can_modify_env: true`)
- Modify stdin (with `can_modify_stdin: true`)

### 2. Command Wrapper (`command_wrapper`)

A simple string wrapper that prefixes all commands. For example:
```json
"command_wrapper": "rtk exec --"
```
This would wrap every command as `rtk exec -- <original command>`.

### 3. Post-Run Hooks (`post_run`)

Post-run hooks execute after a command finishes. They can:
- Modify the exit code (with `can_modify_exit: true`)
- Modify stdout (with `can_modify_stdout: true`)
- Modify stderr (with `can_modify_stderr: true`)

## Configuration

Hooks are configured in the `shell` section of `devbox.json`:

```json
{
  "shell": {
    "pre_run": [
      {
        "command": "echo 'About to run command'",
        "can_block": true,
        "can_modify_args": true,
        "can_modify_env": true,
        "can_modify_stdin": true
      }
    ],
    "command_wrapper": "rtk exec --",
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

## Security

All capability gates default to `false` for security. You must explicitly enable each capability you need.

## Use Cases

- **Policy enforcement**: Block certain commands or require approval
- **Instrumentation**: Log all commands with timing information
- **Command wrapping**: Wrap tools like `rtk exec --` around all commands
- **Environment modification**: Dynamically set environment variables
- **Output processing**: Filter or transform command output

## Testing

Try running:
```bash
devbox run hello
devbox run test
```

You should see the pre-run and post-run hooks executing around the commands.
