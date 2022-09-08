# Test Suite for many different languages

`testdata/` contains a test-suite of projects using different programming
languages and frameworks. `devbox_test.go` automatically loops through every
project in here that has a `devbox.json` and runs `devbox plan`.

It then checks that `DevPackages` and `RuntimePackages` match the stated values
in `plan.json`.

To add a new test:
+ Create a new example
+ Initialize it with a `devbox.json`. If you are trying to test that planners are
  automatically adding the right packages, your `devbox.json` should probably be
  empty. If you are trying to test the interaction between what a planner does, and
  the packages a user might have declared, then your `devbox.json` should include
  the user packages.
+ Add a `plan.json` containing the expected values of `DevPackages` and `RuntimePackages`.