defmodule ElixirHello do
  use Application
  @moduledoc """
  Documentation for `ElixirHello`.
  """

  @doc """
  Hello world.

  ## Examples

      iex> ElixirHello.hello()
      :world

  """
  def start(_type, _args) do
    IO.puts("Hello World!")
    Task.start(fn -> :timer.sleep(1000); IO.puts("Goodbye World"); exit(:shutdown) end)
  end
end
