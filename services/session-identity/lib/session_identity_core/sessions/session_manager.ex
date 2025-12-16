defmodule SessionIdentityCore.Sessions.SessionManager do
  @moduledoc """
  GenServer for managing session state and broadcasting events.
  """

  use GenServer
  alias SessionIdentityCore.Sessions.{Session, SessionStore}
  alias SessionIdentityCore.Identity.RiskScorer
  alias Phoenix.PubSub

  @pubsub SessionIdentityCore.PubSub

  def start_link(_opts) do
    GenServer.start_link(__MODULE__, %{}, name: __MODULE__)
  end

  # Client API

  def create_session(attrs) do
    GenServer.call(__MODULE__, {:create_session, attrs})
  end

  def get_session(session_id) do
    GenServer.call(__MODULE__, {:get_session, session_id})
  end

  def list_user_sessions(user_id) do
    GenServer.call(__MODULE__, {:list_user_sessions, user_id})
  end

  def terminate_session(session_id, user_id, reason \\ "user_request") do
    GenServer.call(__MODULE__, {:terminate_session, session_id, user_id, reason})
  end

  def update_risk_score(session_id, context) do
    GenServer.call(__MODULE__, {:update_risk_score, session_id, context})
  end

  # Server Callbacks

  @impl true
  def init(state) do
    {:ok, state}
  end

  @impl true
  def handle_call({:create_session, attrs}, _from, state) do
    changeset = Session.changeset(%Session{}, attrs)

    case Ecto.Changeset.apply_action(changeset, :insert) do
      {:ok, session} ->
        session = %{session | id: Ecto.UUID.generate()}
        
        case SessionStore.store_session(session) do
          {:ok, _} ->
            broadcast_session_event(session.user_id, :session_created, session)
            {:reply, {:ok, session}, state}

          error ->
            {:reply, error, state}
        end

      {:error, changeset} ->
        {:reply, {:error, changeset}, state}
    end
  end

  @impl true
  def handle_call({:get_session, session_id}, _from, state) do
    result = SessionStore.get_session(session_id)
    {:reply, result, state}
  end

  @impl true
  def handle_call({:list_user_sessions, user_id}, _from, state) do
    result = SessionStore.get_user_sessions(user_id)
    {:reply, result, state}
  end

  @impl true
  def handle_call({:terminate_session, session_id, user_id, reason}, _from, state) do
    case SessionStore.delete_session(session_id, user_id) do
      :ok ->
        broadcast_session_event(user_id, :session_terminated, %{
          session_id: session_id,
          reason: reason
        })
        
        # Notify token service to revoke tokens
        notify_token_revocation(session_id)
        
        {:reply, :ok, state}

      error ->
        {:reply, error, state}
    end
  end

  @impl true
  def handle_call({:update_risk_score, session_id, context}, _from, state) do
    case SessionStore.get_session(session_id) do
      {:ok, session} ->
        risk_score = RiskScorer.calculate_risk(session, context)
        step_up_required = RiskScorer.requires_step_up?(risk_score)
        required_factors = RiskScorer.get_required_factors(risk_score)

        {:reply, {:ok, %{
          risk_score: risk_score,
          step_up_required: step_up_required,
          required_factors: required_factors
        }}, state}

      error ->
        {:reply, error, state}
    end
  end

  # Private functions

  defp broadcast_session_event(user_id, event_type, payload) do
    PubSub.broadcast(@pubsub, "user:#{user_id}", {event_type, payload})
  end

  defp notify_token_revocation(session_id) do
    # In production, this would call the Token Service via gRPC
    # to revoke all tokens associated with this session
    :ok
  end
end
