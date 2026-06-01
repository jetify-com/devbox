#!/usr/bin/env python3
"""Generate an asciinema v2 cast of the current devbox experience.

Used to produce the animated terminal demo in the README. The output strings
mirror what the devbox CLI actually prints today, so refresh this script when
those messages change.

To regenerate docs/app/static/img/devbox_demo.svg:

    python3 scripts/gen_cast.py /tmp/devbox.cast
    npx svg-term-cli --in /tmp/devbox.cast \\
        --out docs/app/static/img/devbox_demo.svg \\
        --window --width 84 --height 26
"""
import json
import sys

# ANSI styles
CYAN = "\x1b[36m"
BLUE = "\x1b[94m"
GREEN = "\x1b[32m"
DIM = "\x1b[90m"
YELLOW = "\x1b[33m"
BGREEN = "\x1b[92m"
RESET = "\x1b[0m"

events = []
t = 0.4  # start a touch after the window opens


def out(s):
    events.append([round(t, 3), "o", s])


def emit(s, dt=0.0):
    global t
    t += dt
    out(s)


def prompt(in_shell=False):
    """Print the starship-style two-line prompt."""
    if in_shell:
        emit(f"{CYAN}~{RESET} {DIM}in{RESET} \U0001F4E6 {BLUE}devbox{RESET}\r\n", 0.5)
    else:
        emit(f"{CYAN}~{RESET}\r\n", 0.5)
    emit(f"{GREEN}❯{RESET} ", 0.15)


def type_cmd(cmd):
    """Type a command character by character, first word highlighted."""
    global t
    first, _, rest = cmd.partition(" ")
    emit(f"{GREEN}{first[0]}{RESET}", 0.12)
    for ch in first[1:]:
        emit(f"{GREEN}{ch}{RESET}", 0.05)
    if rest:
        emit(" ", 0.05)
        for ch in rest:
            emit(ch, 0.045)
    emit("\r\n", 0.25)  # press enter


def output(lines, dt_first=0.25, dt=0.12):
    first = True
    for line in lines:
        emit(line + "\r\n", dt_first if first else dt)
        first = False


# 1. go is not available on the host
prompt()
type_cmd("go version")
output([f"{DIM}zsh: command not found: go{RESET}"])

# 2. init a project
prompt()
type_cmd("devbox init")
output([
    "Created devbox.json in .",
    "Run `devbox add <package>` to add packages, or `devbox shell` to start a dev shell.",
])

# 3. add packages with the modern @version syntax
prompt()
type_cmd("devbox add python@3.10 go@1.18")
output([
    f"{YELLOW}Info:{RESET} Adding package 'python@3.10' to devbox.json",
    f"{YELLOW}Info:{RESET} Adding package 'go@1.18' to devbox.json",
    f"{YELLOW}Info:{RESET} Ensuring packages are installed.",
    f"{YELLOW}Info:{RESET} Installing the following packages to the nix store: python@3.10, go@1.18",
], dt_first=0.4, dt=0.5)
emit(f"{BGREEN}✓{RESET} Computed the Devbox environment.\r\n", 1.2)

# 4. run a single command in the environment
prompt()
type_cmd("devbox run python --version")
output(["Python 3.10.14"], dt_first=0.8)

# 5. drop into an interactive shell
prompt()
type_cmd("devbox shell")
output([f"{DIM}Starting a devbox shell...{RESET}"])
emit(f"{BGREEN}✓{RESET} Computed the Devbox environment.\r\n", 1.0)

# 6. tools are now on PATH inside the shell
prompt(in_shell=True)
type_cmd("python --version")
output(["Python 3.10.14"], dt_first=0.5)

prompt(in_shell=True)
type_cmd("go version")
output(["go version go1.18 linux/amd64"], dt_first=0.5)

# settle on a final prompt
prompt(in_shell=True)
emit("", 1.5)

header = {
    "version": 2,
    "width": 84,
    "height": 26,
    "timestamp": 0,
    "env": {"SHELL": "/bin/zsh", "TERM": "xterm-256color"},
}

with open(sys.argv[1], "w") as f:
    f.write(json.dumps(header) + "\n")
    for ev in events:
        f.write(json.dumps(ev) + "\n")
