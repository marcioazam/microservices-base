defmodule SessionIdentityCore.Repo do
  use Ecto.Repo,
    otp_app: :session_identity_core,
    adapter: Ecto.Adapters.Postgres
end
