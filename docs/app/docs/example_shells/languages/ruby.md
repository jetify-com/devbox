---
title: Ruby
---

Ruby projects can be managed by installing gems locally using Bundler. If you want to install global gems to use from the CLI (like `rails`), this can be done by configuring bundler/gems to install them in a local directory, and then adding this directory to your path.

## Adding Ruby to your shell

Run `devbox add ruby bundler`, or add the following to your `devbox.json`

```json
    "packages": [
        "ruby_3_1",
        "bundler"
    ]
```

This will install Ruby 3.1 to your shell. 

Other versions available include:  

* `ruby` (Ruby 2.7)
* `ruby_3_0` (Ruby 3.0)

## Using Global Gems

To install gems that you want to use from the command line (like `rails`), you will need to configure `gem` within your shell to install to a local folder in your project. 

Adding the following to the `init_hook` in your `devbox.json` will ensure those gems are installed locally

```json
"init_hook": [
    "export GEMRC=$PWD/conf/ruby/.gemrc",
    "export GEM_HOME=$PWD/conf/ruby/gems",
    "export PATH=$GEM_HOME/bin:$PATH"
]
```

You can now install rails as normal using `gem install rails`