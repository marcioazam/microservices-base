defmodule SessionIdentityCore.Crypto.FeatureToggle do
  @moduledoc """
  Feature toggle for crypto integration.
  
  Allows enabling/disabling crypto-service integration at runtime.
  When disabled, uses local implementations.
  """

  use GenServer
  require Logger

  @table_name :crypto_feature_toggle

  # Client API

  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @doc """
  Checks if crypto integration is enabled.
  """
  @spec enabled?() :: boolean()
  def enabled? do
    case :ets.lookup(@table_name, :enabled) do
      [{:enabled, value}] -> value
      [] -> true  # Default enabled
    end
  end

  @doc """
  Enables crypto integration.
  """
  @spec enable() :: :ok
  def enable do
    GenServer.call(__MODULE__, :enable)
  end

  @doc """
  Disables crypto integration.
  """
  @spec disable() :: :ok
  def disable do
    GenServer.call(__MODULE__, :disable)
  end

  @doc """
  Sets crypto integration state.
  """
  @spec set_enabled(boolean()) :: :ok
  def set_enabled(enabled) when is_boolean(enabled) do
    GenServer.call(__MODULE__, {:set_enabled, enabled})
  end

  @doc """
  Executes function only if crypto is enabled, otherwise uses fallback.
  """
  @spec when_enabled((() -> term()), (() -> term())) :: term()
  def when_enabled(enabled_fn, disabled_fn) do
    if enabled?() do
      enabled_fn.()
    else
      disabled_fn.()
    end
  end

  # Server Callbacks

  @impl true
  def init(_opts) do
    :ets.new(@table_name, [:named_table, :set, :public, read_concurrency: true])
    
    # Initialize from config
    initial_state = Application.get_env(:session_identity_core, :crypto_enabled, true)
    :ets.insert(@table_name, {:enabled, initial_state})
    
    Logger.info("Crypto feature toggle initialized", enabled: initial_state)
    {:ok, %{}}
  end

  @impl true
  def handle_call(:enable, _from, state) do
    :ets.insert(@table_name, {:enabled, true})
    Logger.info("Crypto integration enabled")
    emit_toggle_event(true)
    {:reply, :ok, state}
  end

  @impl true
  def handle_call(:disable, _from, state) do
    :ets.insert(@table_name, {:enabled, false})
    Logger.warning("Crypto integration disabled")
    emit_toggle_event(false)
    {:reply, :ok, state}
  end

  @impl true
  def handle_call({:set_enabled, enabled}, _from, state) do
    :ets.insert(@table_name, {:enabled, enabled})
    Logger.info("Crypto integration toggled", enabled: enabled)
    emit_toggle_event(enabled)
    {:reply, :ok, state}
  end

  defp emit_toggle_event(enabled) do
    :telemetry.execute(
      [:session_identity, :crypto, :feature_toggle],
      %{enabled: if(enabled, do: 1, else: 0)},
      %{}
    )
  end
end
