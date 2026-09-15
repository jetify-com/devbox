# Verify `devbox generate readme` includes a Services section listing the
# services defined for the project. See issue #2626.

exec devbox init
exists devbox.json

# `devbox generate readme` should discover the services defined in the
# project's process-compose.yaml (see the embedded file below).
exec devbox generate readme
exists README.md

# The generated README should contain a Services section that lists each
# service and documents how to start/stop them.
grep '## Services' README.md
grep '\* web' README.md
grep '\* worker' README.md
grep 'devbox services up' README.md
grep 'devbox services stop' README.md

-- process-compose.yaml --
version: "0.5"
processes:
  web:
    command: "echo web"
  worker:
    command: "echo worker"
