---
title: Ruby
---

### Detection

Devbox will automatically create a Ruby project plan whenever a `Gemfile` file is detected in the project's root directory.

### Supported Versions

Devbox will attempt to install the Ruby version specified in the `Gemfile`, but it is limited to the following versions: `3.1.2`, `3.0.4`, and `2.7.6`.

### Included Nix Packages

Install and Build Stage Image:
* Ruby (either `ruby_3_1`, `ruby_3_0`, or `ruby`)
* `gcc`
* `gnumake`

GCC and Make are included in case certain gems require them, like `rails`.

Start Stage Image:
* Ruby (either `ruby_3_1`, `ruby_3_0`, or `ruby`)

### Default Stages
These stages can be customized by adding them to your `devbox.json`. See the [Configuration Guide](../configuration.md) for more details.

#### Install Stage

```bash
bundle config set --local deployment 'true' && bundle install
```

#### Build Stage

Skipped: _This stage is skipped for Ruby projects_

#### Start Stage

If `rails` is detected in the `Gemfile`, then:
```bash
./bin/rails server -b 0.0.0.0 -e production
```

You can then `docker run -p 3000:3000 devbox` and access your project at `0.0.0.0:3000`.

Else (not a Rails project):
```bash
bundle exec ruby app.rb
```
