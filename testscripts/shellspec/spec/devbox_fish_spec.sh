Describe 'Devbox basic test in fish'
    AfterAll '$(rm devbox.json)'
    It 'Runs devbox test script written in fish'
        When run fish spec/devbox_test.fish
        The status should be success
        The stdout should equal 'done'
    End
End