# Saving the default readme template doesn't need a devbox.json.
! exists devbox.json
exec devbox generate readme --save-template template.md
exists template.md
! exists devbox.json
