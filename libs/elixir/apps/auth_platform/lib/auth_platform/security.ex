defmodule AuthPlatform.Security do
  @moduledoc """
  Security utilities for the Auth Platform.

  Provides cryptographic operations, input sanitization, and security helpers.

  ## Features

  - Constant-time comparison for secrets
  - Secure token generation
  - Sensitive data masking
  - HTML/SQL sanitization
  - SQL injection detection

  ## Usage

      # Constant-time comparison
      Security.constant_time_compare(user_token, stored_token)

      # Generate secure token
      token = Security.generate_token(32)

      # Mask sensitive data
      masked = Security.mask_sensitive("4111111111111111", visible: 4)
      #=> "************1111"

  """

  @doc """
  Compares two strings in constant time to prevent timing attacks.

  Returns `true` if the strings are equal, `false` otherwise.

  ## Examples

      iex> Security.constant_time_compare("secret", "secret")
      true

      iex> Security.constant_time_compare("secret", "other")
      false

  """
  @spec constant_time_compare(String.t(), String.t()) :: boolean()
  def constant_time_compare(a, b) when is_binary(a) and is_binary(b) do
    byte_size(a) == byte_size(b) and :crypto.hash_equals(a, b)
  end

  def constant_time_compare(_, _), do: false

  @doc """
  Generates a cryptographically secure random token.

  ## Options

    * `:encoding` - Output encoding: `:hex` (default), `:base64`, `:url_safe_base64`

  ## Examples

      iex> token = Security.generate_token(32)
      iex> byte_size(token)
      64  # hex encoding doubles the size

      iex> Security.generate_token(16, encoding: :base64)
      "..." # 24 character base64 string

  """
  @spec generate_token(pos_integer(), keyword()) :: String.t()
  def generate_token(bytes, opts \\ []) when is_integer(bytes) and bytes > 0 do
    encoding = Keyword.get(opts, :encoding, :hex)
    random_bytes = :crypto.strong_rand_bytes(bytes)

    case encoding do
      :hex -> Base.encode16(random_bytes, case: :lower)
      :base64 -> Base.encode64(random_bytes)
      :url_safe_base64 -> Base.url_encode64(random_bytes, padding: false)
    end
  end

  @doc """
  Masks sensitive data, showing only the last N characters.

  ## Options

    * `:visible` - Number of visible characters at the end (default: 4)
    * `:mask_char` - Character to use for masking (default: "*")
    * `:min_masked` - Minimum characters to mask (default: 4)

  ## Examples

      iex> Security.mask_sensitive("4111111111111111")
      "************1111"

      iex> Security.mask_sensitive("secret", visible: 2)
      "****et"

      iex> Security.mask_sensitive("abc", visible: 4)
      "***"  # All masked when too short

  """
  @spec mask_sensitive(String.t(), keyword()) :: String.t()
  def mask_sensitive(value, opts \\ []) when is_binary(value) do
    visible = Keyword.get(opts, :visible, 4)
    mask_char = Keyword.get(opts, :mask_char, "*")
    min_masked = Keyword.get(opts, :min_masked, 4)

    len = String.length(value)

    cond do
      len <= min_masked ->
        String.duplicate(mask_char, len)

      len <= visible + min_masked ->
        String.duplicate(mask_char, len)

      true ->
        masked_count = len - visible
        String.duplicate(mask_char, masked_count) <> String.slice(value, -visible, visible)
    end
  end

  @doc """
  Sanitizes a string for safe HTML output by encoding special characters.

  ## Examples

      iex> Security.sanitize_html("<script>alert('xss')</script>")
      "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"

  """
  @spec sanitize_html(String.t()) :: String.t()
  def sanitize_html(input) when is_binary(input) do
    input
    |> String.replace("&", "&amp;")
    |> String.replace("<", "&lt;")
    |> String.replace(">", "&gt;")
    |> String.replace("\"", "&quot;")
    |> String.replace("'", "&#39;")
  end

  @doc """
  Escapes special characters for SQL strings.

  Note: This is a basic escape function. Always prefer parameterized queries.

  ## Examples

      iex> Security.sanitize_sql("O'Brien")
      "O''Brien"

  """
  @spec sanitize_sql(String.t()) :: String.t()
  def sanitize_sql(input) when is_binary(input) do
    input
    |> String.replace("'", "''")
    |> String.replace("\\", "\\\\")
    |> String.replace("\x00", "")
  end

  @doc """
  Detects potential SQL injection patterns in input.

  Returns `true` if suspicious patterns are detected.

  ## Examples

      iex> Security.detect_sql_injection("normal input")
      false

      iex> Security.detect_sql_injection("'; DROP TABLE users; --")
      true

  """
  @spec detect_sql_injection(String.t()) :: boolean()
  def detect_sql_injection(input) when is_binary(input) do
    patterns = [
      ~r/(\s|^)(OR|AND)\s+\d+\s*=\s*\d+/i,
      ~r/(\s|^)(OR|AND)\s+['"]?\w+['"]?\s*=\s*['"]?\w+['"]?/i,
      ~r/;\s*(DROP|DELETE|UPDATE|INSERT|ALTER|CREATE|TRUNCATE)/i,
      ~r/--\s*$/,
      ~r/\/\*.*\*\//,
      ~r/UNION\s+(ALL\s+)?SELECT/i,
      ~r/'\s*(OR|AND)\s*'/i,
      ~r/\bEXEC\s*\(/i,
      ~r/\bXP_/i
    ]

    Enum.any?(patterns, &Regex.match?(&1, input))
  end

  @doc """
  Generates a secure hash of the input using SHA-256.

  ## Examples

      iex> Security.hash("password")
      "5e884898da28047d..."

  """
  @spec hash(String.t()) :: String.t()
  def hash(input) when is_binary(input) do
    :crypto.hash(:sha256, input)
    |> Base.encode16(case: :lower)
  end

  @doc """
  Generates a secure hash with a salt.

  ## Examples

      iex> Security.hash_with_salt("password", "random_salt")
      "..."

  """
  @spec hash_with_salt(String.t(), String.t()) :: String.t()
  def hash_with_salt(input, salt) when is_binary(input) and is_binary(salt) do
    hash(salt <> input <> salt)
  end
end
