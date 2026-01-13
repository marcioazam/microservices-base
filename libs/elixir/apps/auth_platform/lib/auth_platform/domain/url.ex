defmodule AuthPlatform.Domain.URL do
  @moduledoc """
  URL value object with scheme validation.

  Provides validated URL handling with support for http and https schemes.

  ## Usage

      alias AuthPlatform.Domain.URL

      # Creating URLs
      {:ok, url} = URL.new("https://example.com/path?query=1")

      # Accessing parts
      URL.scheme(url)  # :https
      URL.host(url)    # "example.com"
      URL.path(url)    # "/path"

      # String conversion
      to_string(url)  # "https://example.com/path?query=1"

  """

  @type scheme :: :http | :https

  @type t :: %__MODULE__{
          value: String.t(),
          scheme: scheme(),
          host: String.t(),
          port: non_neg_integer() | nil,
          path: String.t(),
          query: String.t() | nil
        }

  @enforce_keys [:value, :scheme, :host]
  defstruct [:value, :scheme, :host, :port, :path, :query]

  @allowed_schemes ["http", "https"]

  @doc """
  Creates a new URL from a string.

  Only http and https schemes are allowed.

  ## Examples

      iex> AuthPlatform.Domain.URL.new("https://example.com")
      {:ok, %AuthPlatform.Domain.URL{value: "https://example.com", scheme: :https, host: "example.com", path: "", port: nil, query: nil}}

      iex> AuthPlatform.Domain.URL.new("ftp://example.com")
      {:error, "unsupported URL scheme: ftp"}

      iex> AuthPlatform.Domain.URL.new("not a url")
      {:error, "invalid URL format"}

  """
  @spec new(String.t()) :: {:ok, t()} | {:error, String.t()}
  def new(value) when is_binary(value) do
    case URI.parse(value) do
      %URI{scheme: nil} ->
        {:error, "invalid URL format"}

      %URI{scheme: scheme} when scheme not in @allowed_schemes ->
        {:error, "unsupported URL scheme: #{scheme}"}

      %URI{host: nil} ->
        {:error, "invalid URL format"}

      %URI{host: ""} ->
        {:error, "invalid URL format"}

      %URI{scheme: scheme, host: host, port: port, path: path, query: query} ->
        {:ok,
         %__MODULE__{
           value: value,
           scheme: String.to_atom(scheme),
           host: host,
           port: port,
           path: path || "",
           query: query
         }}
    end
  end

  def new(_), do: {:error, "URL must be a string"}

  @doc """
  Creates a new URL from a string, raising on invalid input.

  ## Examples

      iex> AuthPlatform.Domain.URL.new!("https://example.com")
      %AuthPlatform.Domain.URL{value: "https://example.com", scheme: :https, host: "example.com", path: "", port: nil, query: nil}

  """
  @spec new!(String.t()) :: t()
  def new!(value) do
    case new(value) do
      {:ok, url} -> url
      {:error, reason} -> raise ArgumentError, reason
    end
  end

  @doc """
  Returns the URL value as a string.
  """
  @spec to_string(t()) :: String.t()
  def to_string(%__MODULE__{value: value}), do: value

  @doc """
  Returns the URL scheme.

  ## Examples

      iex> url = AuthPlatform.Domain.URL.new!("https://example.com")
      iex> AuthPlatform.Domain.URL.scheme(url)
      :https

  """
  @spec scheme(t()) :: scheme()
  def scheme(%__MODULE__{scheme: scheme}), do: scheme

  @doc """
  Returns the URL host.

  ## Examples

      iex> url = AuthPlatform.Domain.URL.new!("https://example.com/path")
      iex> AuthPlatform.Domain.URL.host(url)
      "example.com"

  """
  @spec host(t()) :: String.t()
  def host(%__MODULE__{host: host}), do: host

  @doc """
  Returns the URL port, or the default port for the scheme.

  ## Examples

      iex> url = AuthPlatform.Domain.URL.new!("https://example.com")
      iex> AuthPlatform.Domain.URL.port(url)
      443

      iex> url = AuthPlatform.Domain.URL.new!("http://example.com:8080")
      iex> AuthPlatform.Domain.URL.port(url)
      8080

  """
  @spec port(t()) :: non_neg_integer()
  def port(%__MODULE__{port: nil, scheme: :https}), do: 443
  def port(%__MODULE__{port: nil, scheme: :http}), do: 80
  def port(%__MODULE__{port: port}), do: port

  @doc """
  Returns the URL path.

  ## Examples

      iex> url = AuthPlatform.Domain.URL.new!("https://example.com/api/v1")
      iex> AuthPlatform.Domain.URL.path(url)
      "/api/v1"

  """
  @spec path(t()) :: String.t()
  def path(%__MODULE__{path: path}), do: path

  @doc """
  Returns the URL query string.

  ## Examples

      iex> url = AuthPlatform.Domain.URL.new!("https://example.com?foo=bar")
      iex> AuthPlatform.Domain.URL.query(url)
      "foo=bar"

  """
  @spec query(t()) :: String.t() | nil
  def query(%__MODULE__{query: query}), do: query

  @doc """
  Checks if the URL uses HTTPS.

  ## Examples

      iex> url = AuthPlatform.Domain.URL.new!("https://example.com")
      iex> AuthPlatform.Domain.URL.secure?(url)
      true

      iex> url = AuthPlatform.Domain.URL.new!("http://example.com")
      iex> AuthPlatform.Domain.URL.secure?(url)
      false

  """
  @spec secure?(t()) :: boolean()
  def secure?(%__MODULE__{scheme: :https}), do: true
  def secure?(%__MODULE__{}), do: false

  @doc """
  Returns the origin (scheme + host + port) of the URL.

  ## Examples

      iex> url = AuthPlatform.Domain.URL.new!("https://example.com/path")
      iex> AuthPlatform.Domain.URL.origin(url)
      "https://example.com"

      iex> url = AuthPlatform.Domain.URL.new!("http://example.com:8080/path")
      iex> AuthPlatform.Domain.URL.origin(url)
      "http://example.com:8080"

  """
  @spec origin(t()) :: String.t()
  def origin(%__MODULE__{scheme: scheme, host: host, port: port_val}) do
    default_port = if scheme == :https, do: 443, else: 80

    if port_val == nil or port_val == default_port do
      "#{scheme}://#{host}"
    else
      "#{scheme}://#{host}:#{port_val}"
    end
  end

  defimpl String.Chars do
    def to_string(%{value: value}), do: value
  end

  defimpl Jason.Encoder do
    def encode(%{value: value}, opts), do: Jason.Encode.string(value, opts)
  end
end
