# Change Log

All notable changes to the "devbox" extension will be documented in this file.

Check [Keep a Changelog](http://keepachangelog.com/) for recommendations on how to structure this file.

## [0.0.3]

- Small fix for DevContainers and Github CodeSpaces compatibility.

## [0.0.2]

- Added ability to run devbox commands from VSCode command palette
- Added VSCode command to generate DevContainer files to run VSCode in local container or Github CodeSpaces.
- Added customization in settings to turn on/off automatically running `devbox shell` when a terminal window is opened.

## [0.0.1]

- Initial release
- When VScode Terminal is opened on a devbox project, this extension detects `devbox.json` and runs `devbox shell` so terminal is automatically in devbox shell environment.
