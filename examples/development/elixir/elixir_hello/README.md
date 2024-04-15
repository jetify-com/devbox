# Elixir

Basic Elixir project using Mix in Devbox.

## Configuration

This project configures Hex and Mix to install packages + dependencies in local project directories. You can modify where these packages are installed by changing the variables in `conf/set-env.sh`

## Installation

To run the project: `mix run`

To create a release: `mix release`

## Elixir Readme

If [available in Hex](https://hex.pm/docs/publish), the package can be installed
by adding `elixir_hello` to your list of dependencies in `mix.exs`:

```elixir
def deps do
  [
    {:elixir_hello, "~> 0.1.0"}
  ]
end
```

Documentation can be generated with [ExDoc](https://github.com/elixir-lang/ex_doc)
and published on [HexDocs](https://hexdocs.pm). Once published, the docs can
be found at `https://hexdocs.pm/elixir_hello`.
