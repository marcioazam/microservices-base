defmodule AuthPlatform.Codec.JSON do
  @moduledoc """
  JSON encoding and decoding utilities.

  Provides a consistent interface for JSON operations with proper error handling.

  ## Usage

      # Encode to JSON
      {:ok, json} = JSON.encode(%{name: "John", age: 30})

      # Decode from JSON
      {:ok, data} = JSON.decode(~s({"name": "John"}))

      # Pretty print
      {:ok, pretty} = JSON.encode_pretty(%{nested: %{data: true}})

  """

  alias AuthPlatform.Functional.Result

  @doc """
  Encodes a term to JSON string.

  Returns `{:ok, json}` on success, `{:error, reason}` on failure.

  ## Examples

      iex> JSON.encode(%{name: "John"})
      {:ok, ~s({"name":"John"})}

      iex> JSON.encode({:tuple, :not_supported})
      {:error, %Protocol.UndefinedError{...}}

  """
  @spec encode(term()) :: Result.t(String.t(), Exception.t())
  def encode(term) do
    {:ok, Jason.encode!(term)}
  rescue
    e -> {:error, e}
  end

  @doc """
  Encodes a term to JSON string, raising on error.

  ## Examples

      iex> JSON.encode!(%{name: "John"})
      ~s({"name":"John"})

  """
  @spec encode!(term()) :: String.t()
  def encode!(term), do: Jason.encode!(term)

  @doc """
  Encodes a term to pretty-printed JSON string.

  ## Examples

      iex> JSON.encode_pretty(%{name: "John"})
      {:ok, ~s({\\n  "name": "John"\\n})}

  """
  @spec encode_pretty(term()) :: Result.t(String.t(), Exception.t())
  def encode_pretty(term) do
    {:ok, Jason.encode!(term, pretty: true)}
  rescue
    e -> {:error, e}
  end

  @doc """
  Decodes a JSON string to a term.

  Returns `{:ok, term}` on success, `{:error, reason}` on failure.

  ## Options

    * `:keys` - How to decode object keys: `:strings` (default) or `:atoms`

  ## Examples

      iex> JSON.decode(~s({"name": "John"}))
      {:ok, %{"name" => "John"}}

      iex> JSON.decode(~s({"name": "John"}), keys: :atoms)
      {:ok, %{name: "John"}}

      iex> JSON.decode("invalid json")
      {:error, %Jason.DecodeError{...}}

  """
  @spec decode(String.t(), keyword()) :: Result.t(term(), Exception.t())
  def decode(json, opts \\ []) when is_binary(json) do
    keys = Keyword.get(opts, :keys, :strings)
    {:ok, Jason.decode!(json, keys: keys)}
  rescue
    e -> {:error, e}
  end

  @doc """
  Decodes a JSON string to a term, raising on error.

  ## Examples

      iex> JSON.decode!(~s({"name": "John"}))
      %{"name" => "John"}

  """
  @spec decode!(String.t(), keyword()) :: term()
  def decode!(json, opts \\ []) when is_binary(json) do
    keys = Keyword.get(opts, :keys, :strings)
    Jason.decode!(json, keys: keys)
  end

  @doc """
  Checks if a string is valid JSON.

  ## Examples

      iex> JSON.valid?(~s({"name": "John"}))
      true

      iex> JSON.valid?("not json")
      false

  """
  @spec valid?(String.t()) :: boolean()
  def valid?(json) when is_binary(json) do
    case decode(json) do
      {:ok, _} -> true
      {:error, _} -> false
    end
  end
end
