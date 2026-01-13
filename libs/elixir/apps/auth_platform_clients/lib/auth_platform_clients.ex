defmodule AuthPlatformClients do
  @moduledoc """
  Platform service clients for Auth Platform.

  This module provides gRPC clients for platform services:

  - `AuthPlatformClients.Logging` - Logging service client
  - `AuthPlatformClients.Cache` - Cache service client

  All clients include circuit breaker protection and telemetry events.
  """

  @doc """
  Returns the library version.
  """
  @spec version() :: String.t()
  def version, do: "0.1.0"
end
