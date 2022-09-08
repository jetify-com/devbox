# Contributing

When contributing to this repository, please describe the change you wish to make via a related issue, or a pull request.

Please note we have a [code of conduct](CODE_OF_CONDUCT.md), please follow it in all your interactions with the project.

## Setting Up Development Environment
Before making any changes to the source code (documentation excluded) make sure you have installed all the required tools. 
### Prerequisites
* Install [Nix Package Manager](https://nixos.org/download.html).
* Install [Golang](https://go.dev/doc/install) (current version: 1.19)
* Clone this repository: 
    * ```bash
        git clone github.com/jetpack/devbox go.jetpack.io
        ```
* Setup you `GOPATH` env variable to the parent directory of `go.jetpack.io/` 
    * Example: If the cloned repository is at `/Users/johndoe/projects/go.jetpack.io/`:
        ```bash
        export GOPATH=/Users/johndoe/projects/
        ```

## Building and Testing
Devbox is setup like a typical Go project. After installing the required tools and setting up your environment. You can make changes in the source code, build, and test your changes by following these steps:

1. Install dependencies:
    ```bash
    go install
    ```
2. Build Devbox:
    ```bash
    go build -o ./devbox cmd/devbox/main.go
    ```
    This will build an executable file.
3. Run and test Devbox:
    ```bash
    ./devbox <your_test_command>
    ```


## Pull Request Process

1. Ensure any new feature or functionality also includes tests to verify its correctness.

2. Ensure any new dependency is also included in [go.mod](go.mod) file

2. Ensure any binary file as a result of build (e.g., `./devbox`) are removed and/or excluded from tracking in git.

3. Update the [README.md](README.md) and/or docs with details of changes to the interface, this includes new environment 
   variables, new commands, new flags, and useful file locations.

4. You may merge the Pull Request in once you have the sign-off of developers/maintainers, or if you 
   do not have permission to do that, you may request the maintainers to merge it for you.
