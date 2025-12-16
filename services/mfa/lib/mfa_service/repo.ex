defmodule MfaService.Repo do
  use Ecto.Repo,
    otp_app: :mfa_service,
    adapter: Ecto.Adapters.Postgres
end
