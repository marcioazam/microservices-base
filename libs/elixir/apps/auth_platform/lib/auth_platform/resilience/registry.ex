defmodule AuthPlatform.Resilience.Registry do
  @moduledoc """
  Registry for resilience components (Circuit Breakers, Rate Limiters, Bulkheads).

  This module provides a centralized registry using Elixir's built-in Registry
  for managing named resilience components. Each component is registered with
  a unique name and can be looked up efficiently.

  ## Usage

      # Components register themselves via the registry
      {:via, Registry, {AuthPlatform.Resilience.Registry, :my_circuit_breaker}}

      # Lookup a component
      AuthPlatform.Resilience.Registry.lookup(:my_circuit_breaker)

  """

  @registry_name __MODULE__

  @doc """
  Returns the registry name for use in child specs.
  """
  @spec name() :: atom()
  def name, do: @registry_name

  @doc """
  Returns a via tuple for registering a process with the given name.

  ## Examples

      iex> AuthPlatform.Resilience.Registry.via_tuple(:my_breaker)
      {:via, Registry, {AuthPlatform.Resilience.Registry, :my_breaker}}

  """
  @spec via_tuple(atom()) :: {:via, Registry, {atom(), atom()}}
  def via_tuple(name) when is_atom(name) do
    {:via, Registry, {@registry_name, name}}
  end

  @doc """
  Looks up a process by name in the registry.

  Returns `{:ok, pid}` if found, `{:error, :not_found}` otherwise.

  ## Examples

      iex> AuthPlatform.Resilience.Registry.lookup(:my_breaker)
      {:ok, #PID<0.123.0>}

      iex> AuthPlatform.Resilience.Registry.lookup(:unknown)
      {:error, :not_found}

  """
  @spec lookup(atom()) :: {:ok, pid()} | {:error, :not_found}
  def lookup(name) when is_atom(name) do
    case Registry.lookup(@registry_name, name) do
      [{pid, _}] -> {:ok, pid}
      [] -> {:error, :not_found}
    end
  end

  @doc """
  Checks if a component with the given name is registered.

  ## Examples

      iex> AuthPlatform.Resilience.Registry.registered?(:my_breaker)
      true

  """
  @spec registered?(atom()) :: boolean()
  def registered?(name) when is_atom(name) do
    case lookup(name) do
      {:ok, _} -> true
      {:error, :not_found} -> false
    end
  end

  @doc """
  Returns all registered component names.

  ## Examples

      iex> AuthPlatform.Resilience.Registry.all_names()
      [:breaker_1, :breaker_2, :rate_limiter_1]

  """
  @spec all_names() :: [atom()]
  def all_names do
    Registry.select(@registry_name, [{{:"$1", :_, :_}, [], [:"$1"]}])
  end

  @doc """
  Returns the count of registered components.
  """
  @spec count() :: non_neg_integer()
  def count do
    length(all_names())
  end

  @doc """
  Returns the child spec for starting the registry under a supervisor.
  """
  @spec child_spec(keyword()) :: Supervisor.child_spec()
  def child_spec(_opts) do
    %{
      id: @registry_name,
      start: {Registry, :start_link, [[keys: :unique, name: @registry_name]]},
      type: :supervisor
    }
  end
end
