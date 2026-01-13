defmodule AuthPlatform.Testing.Generators do
  @moduledoc """
  StreamData generators for Auth Platform domain types.

  Provides generators for creating valid test data for property-based testing.

  ## Usage

      use ExUnitProperties

      property "emails are valid" do
        check all email <- Generators.email_generator() do
          assert {:ok, _} = Email.new(email)
        end
      end

  """
  use ExUnitProperties

  @doc """
  Generates valid RFC 5322 email addresses.

  ## Examples

      iex> email = Generators.email_generator() |> Enum.take(1) |> hd()
      "user123@example.com"

  """
  @spec email_generator() :: StreamData.t(String.t())
  def email_generator do
    gen all(
          local <- local_part_generator(),
          domain <- domain_generator()
        ) do
      "#{local}@#{domain}"
    end
  end

  @doc """
  Generates valid UUID v4 strings.

  ## Examples

      iex> uuid = Generators.uuid_generator() |> Enum.take(1) |> hd()
      "550e8400-e29b-41d4-a716-446655440000"

  """
  @spec uuid_generator() :: StreamData.t(String.t())
  def uuid_generator do
    gen all(
          a <- hex_string(8),
          b <- hex_string(4),
          c <- hex_string(3),
          d <- hex_string(3),
          e <- hex_string(12)
        ) do
      "#{a}-#{b}-4#{c}-#{variant_char()}#{d}-#{e}"
    end
  end

  @doc """
  Generates valid ULID strings.

  ## Examples

      iex> ulid = Generators.ulid_generator() |> Enum.take(1) |> hd()
      "01ARZ3NDEKTSV4RRFFQ69G5FAV"

  """
  @spec ulid_generator() :: StreamData.t(String.t())
  def ulid_generator do
    # ULID uses Crockford's Base32 alphabet
    alphabet = ~c"0123456789ABCDEFGHJKMNPQRSTVWXYZ"

    gen all(chars <- list_of(member_of(alphabet), length: 26)) do
      List.to_string(chars)
    end
  end

  @doc """
  Generates valid Money structs.

  ## Examples

      iex> money = Generators.money_generator() |> Enum.take(1) |> hd()
      %{amount: 1234, currency: :USD}

  """
  @spec money_generator() :: StreamData.t(map())
  def money_generator do
    currencies = [:USD, :EUR, :GBP, :BRL, :JPY]

    gen all(
          amount <- integer(0..1_000_000_00),
          currency <- member_of(currencies)
        ) do
      %{amount: amount, currency: currency}
    end
  end

  @doc """
  Generates valid E.164 phone numbers.

  ## Examples

      iex> phone = Generators.phone_number_generator() |> Enum.take(1) |> hd()
      "+14155551234"

  """
  @spec phone_number_generator() :: StreamData.t(String.t())
  def phone_number_generator do
    country_codes = ["1", "44", "55", "49", "33", "81", "86"]

    gen all(
          country <- member_of(country_codes),
          number <- string(?0..?9, length: 10)
        ) do
      "+#{country}#{number}"
    end
  end

  @doc """
  Generates valid URLs.

  ## Examples

      iex> url = Generators.url_generator() |> Enum.take(1) |> hd()
      "https://example.com/path"

  """
  @spec url_generator() :: StreamData.t(String.t())
  def url_generator do
    schemes = ["http", "https"]

    gen all(
          scheme <- member_of(schemes),
          domain <- domain_generator(),
          path <- path_generator()
        ) do
      "#{scheme}://#{domain}#{path}"
    end
  end

  @doc """
  Generates valid correlation IDs.
  """
  @spec correlation_id_generator() :: StreamData.t(String.t())
  def correlation_id_generator do
    hex_string(16)
  end

  @doc """
  Generates AppError structs.
  """
  @spec app_error_generator() :: StreamData.t(map())
  def app_error_generator do
    codes = [:not_found, :validation, :unauthorized, :internal, :rate_limited, :timeout, :unavailable]

    gen all(
          code <- member_of(codes),
          message <- string(:alphanumeric, min_length: 5, max_length: 100),
          correlation_id <- one_of([constant(nil), correlation_id_generator()])
        ) do
      %{
        code: code,
        message: message,
        correlation_id: correlation_id,
        retryable: code in [:rate_limited, :timeout, :unavailable]
      }
    end
  end

  # Private generators

  defp local_part_generator do
    gen all(
          first <- string(:alphanumeric, min_length: 1, max_length: 10),
          rest <- string([?a..?z, ?0..?9, ?., ?_, ?-], max_length: 20)
        ) do
      first <> rest
    end
  end

  defp domain_generator do
    tlds = ["com", "org", "net", "io", "dev", "app"]

    gen all(
          name <- string(:alphanumeric, min_length: 3, max_length: 15),
          tld <- member_of(tlds)
        ) do
      String.downcase(name) <> "." <> tld
    end
  end

  defp path_generator do
    gen all(
          segments <- list_of(string(:alphanumeric, min_length: 1, max_length: 10), max_length: 3)
        ) do
      case segments do
        [] -> ""
        _ -> "/" <> Enum.join(segments, "/")
      end
    end
  end

  defp hex_string(length) do
    gen all(chars <- list_of(member_of(~c"0123456789abcdef"), length: length)) do
      List.to_string(chars)
    end
  end

  defp variant_char do
    Enum.random(~c"89ab")
  end
end
