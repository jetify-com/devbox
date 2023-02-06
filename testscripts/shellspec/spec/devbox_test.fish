#!/usr/bin/env fish

#### Creates a devbox.json ####
devbox init
# check if devbox.json is created
ls devbox.json >>/dev/null
# check if devbox.json is empty
if test $(du -s devbox.json | cut -f1) -eq 0
    echo "faulty devbox file"
    exit 1
end

#### Adds a package ####
devbox add hello >/dev/null 2>output.tmp
# check output (stderr)
grep hello output.tmp >/dev/null
grep 'is now installed.' output.tmp >/dev/null
rm output.tmp

#### Runs Hello ####
DEVBOX_FEATURE_STRICT_RUN=1 devbox run hello >>output.tmp 2>&1
# check output (stderr and stdout)
grep 'Ensuring packages are installed.' output.tmp >/dev/null
grep 'Hello, world!' output.tmp >/dev/null
rm output.tmp

#### Removes package ####
devbox rm hello >/dev/null 2>output.tmp
grep hello output.tmp >/dev/null
grep 'is now removed.' output.tmp >/dev/null
rm output.tmp
echo done
