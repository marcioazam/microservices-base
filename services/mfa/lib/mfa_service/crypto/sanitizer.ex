defmodule MfaService.Crypto.Sanitizer do
  @moduledoc """
  Sanitizes sensitive data from logs and error messages.
  Ensures no plaintext secrets, encryption keys, or internal details are exposed.
  """

  @sensitive_patterns [
    # Base64 encoded secrets (32+ chars)
    ~r/[A-Za-z0-9+\/]{32,}={0,2}/,
    # Hex encoded keys
    ~r/[0-9a-fA-F]{64,}/,
    # TOTP secrets (base32)
    ~r/[A-Z2-7]{16,}/
  ]

  @sensitive_keys [
    :secret, :plaintext, :key, :password, :token, :ciphertext,
    :encryption_key, :decryption_key, :private_key, :api_key,
    :totp_secret, :mfa_secret, :auth_token
  ]

  @internal_error_patterns [
    ~r/stack trace/i,
    ~r/at line \d+/i,
    ~r/\*\* \(.*\)/,
    ~r/lib\/.*\.ex:\d+/,
    ~r/erlang error/i
  ]

  @doc """
  Sanitizes a string by redacting sensitive patterns.
  """
  @spec sanitize_string(String.t()) :: String.t()
  def sanitize_string(nil), do: nil
  def sanitize_string(str) when is_binary(str) do
    Enum.reduce(@sensitive_patterns, str, fn pattern, acc ->
      Regex.replace(pattern, acc, "[REDACTED]")
    end)
  end

  @doc """
  Sanitizes a map by redacting sensitive keys and values.
  """
  @spec sanitize_map(map()) :: map()
  def sanitize_map(map) when is_map(map) do
    Map.new(map, fn {key, value} ->
      atom_key = normalize_key(key)
      
      cond do
        atom_key in @sensitive_keys ->
          {key, "[REDACTED]"}
        
        is_map(value) ->
          {key, sanitize_map(value)}
        
        is_binary(value) ->
          {key, sanitize_string(value)}
        
        is_list(value) ->
          {key, Enum.map(value, &sanitize_value/1)}
        
        true ->
          {key, value}
      end
    end)
  end

  @doc """
  Sanitizes a keyword list.
  """
  @spec sanitize_keyword(keyword()) :: keyword()
  def sanitize_keyword(kw) when is_list(kw) do
    Enum.map(kw, fn {key, value} ->
      if key in @sensitive_keys do
        {key, "[REDACTED]"}
      else
        {key, sanitize_value(value)}
      end
    end)
  end

  @doc """
  Sanitizes an error message for user display.
  Removes internal details like stack traces and file paths.
  """
  @spec sanitize_error_message(String.t()) :: String.t()
  def sanitize_error_message(nil), do: "An error occurred"
  def sanitize_error_message(message) when is_binary(message) do
    sanitized = Enum.reduce(@internal_error_patterns, message, fn pattern, acc ->
      Regex.replace(pattern, acc, "[internal]")
    end)
    
    # Also sanitize any sensitive data
    sanitize_string(sanitized)
  end

  @doc """
  Sanitizes an error struct for logging.
  """
  @spec sanitize_error(term()) :: term()
  def sanitize_error(%{__struct__: _} = error) do
    error
    |> Map.from_struct()
    |> sanitize_map()
  end

  def sanitize_error(error) when is_map(error) do
    sanitize_map(error)
  end

  def sanitize_error(error) when is_binary(error) do
    sanitize_error_message(error)
  end

  def sanitize_error(error), do: inspect(error) |> sanitize_string()

  @doc """
  Validates that a response has all required fields.
  """
  @spec validate_response(map(), list(atom())) :: :ok | {:error, :invalid_response}
  def validate_response(response, required_fields) when is_map(response) do
    missing = Enum.filter(required_fields, fn field ->
      not Map.has_key?(response, field) or is_nil(Map.get(response, field))
    end)
    
    if Enum.empty?(missing) do
      :ok
    else
      {:error, :invalid_response}
    end
  end

  def validate_response(_, _), do: {:error, :invalid_response}

  @doc """
  Validates response field types.
  """
  @spec validate_response_types(map(), keyword()) :: :ok | {:error, :invalid_response}
  def validate_response_types(response, type_specs) when is_map(response) do
    invalid = Enum.filter(type_specs, fn {field, type} ->
      value = Map.get(response, field)
      not valid_type?(value, type)
    end)
    
    if Enum.empty?(invalid) do
      :ok
    else
      {:error, :invalid_response}
    end
  end

  # Private functions

  defp sanitize_value(value) when is_binary(value), do: sanitize_string(value)
  defp sanitize_value(value) when is_map(value), do: sanitize_map(value)
  defp sanitize_value(value) when is_list(value), do: Enum.map(value, &sanitize_value/1)
  defp sanitize_value(value), do: value

  # SECURITY FIX: Do NOT convert arbitrary strings to atoms to prevent atom exhaustion
  # Keys from external sources (HTTP headers, JSON, etc.) should remain as strings
  defp normalize_key(key) when is_atom(key), do: key
  defp normalize_key(key) when is_binary(key) do
    # Return normalized string key instead of atom
    # This prevents atom table exhaustion from untrusted input
    key
    |> String.downcase()
    |> String.replace("-", "_")
  end
  defp normalize_key(_), do: "unknown"

  defp valid_type?(nil, _), do: false
  defp valid_type?(value, :binary) when is_binary(value), do: true
  defp valid_type?(value, :string) when is_binary(value), do: true
  defp valid_type?(value, :integer) when is_integer(value), do: true
  defp valid_type?(value, :map) when is_map(value), do: true
  defp valid_type?(value, :list) when is_list(value), do: true
  defp valid_type?(value, :boolean) when is_boolean(value), do: true
  defp valid_type?(_, _), do: false
end
