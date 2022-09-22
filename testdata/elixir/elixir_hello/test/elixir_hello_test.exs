defmodule ElixirHelloTest do
  use ExUnit.Case
  doctest ElixirHello

  test "greets the world" do
    assert ElixirHello.hello() == "Hello World!"
  end
end
