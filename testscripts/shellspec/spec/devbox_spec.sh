Describe 'Check devbox version'
    Skip 'No need to check for non-release tests'
    It 'Checks devbox vesion'
        When call devbox version
        The output should equal '0.3.2'
    End
End

Describe 'devbox basic tests'
    AfterAll '$(rm devbox.json && rm -r .devbox)'
    Path devboxJSON="devbox.json"
    It 'Creates a devbox.json'
        When run devbox init
        The path devboxJSON should be file
        The path devboxJSON should not be empty file
    End
    It 'Adds a package'
        When run devbox add hello
        The stderr should include 'hello'
        The stderr should include 'is now installed.'
    End
    Before 'export DEVBOX_FEATURE_STRICT_RUN=1'
    It 'Runs Hello'
        When run devbox run hello
        The stderr should include 'Ensuring packages are installed.'
        The status should equal '0'
        The stdout should equal 'Hello, world!'
    End
    It 'Removes package'
        When run devbox rm hello
        The stderr should include 'hello'
        The stderr should include 'is now removed.'
    End
End
