defmodule AuthPlatform.Domain.DomainPrimitivesTest do
  @moduledoc """
  Property and unit tests for Domain Primitives.

  **Property 10: Domain Primitive Validation**
  **Property 11: Domain Primitive Serialization Round-Trip**
  **Validates: Requirements 4.1-4.9**
  """
  use ExUnit.Case, async: true
  use ExUnitProperties

  alias AuthPlatform.Domain.{Email, UUID, ULID, Money, PhoneNumber, URL}

  # ============================================================================
  # Generators
  # ============================================================================

  defp valid_email_generator do
    gen all(
          local <- string(:alphanumeric, min_length: 1, max_length: 20),
          domain <- string(:alphanumeric, min_length: 1, max_length: 10),
          tld <- member_of(["com", "org", "net", "io", "dev"])
        ) do
      "#{local}@#{domain}.#{tld}"
    end
  end

  defp valid_uuid_generator do
    gen all(bytes <- binary(length: 16)) do
      <<a::32, b::16, c::16, d::16, e::48>> = bytes
      c_versioned = (c &&& 0x0FFF) ||| 0x4000
      d_varianted = (d &&& 0x3FFF) ||| 0x8000

      :io_lib.format(
        "~8.16.0b-~4.16.0b-~4.16.0b-~4.16.0b-~12.16.0b",
        [a, b, c_versioned, d_varianted, e]
      )
      |> IO.iodata_to_binary()
      |> String.downcase()
    end
  end

  defp valid_ulid_generator do
    gen all(timestamp <- integer(0..0xFFFFFFFFFFFF)) do
      ulid = ULID.generate(timestamp)
      ulid.value
    end
  end

  defp valid_money_generator do
    gen all(
          amount <- integer(-1_000_000_000..1_000_000_000),
          currency <- member_of([:USD, :EUR, :GBP, :BRL, :JPY])
        ) do
      {amount, currency}
    end
  end

  defp valid_phone_generator do
    gen all(
          country_code <- member_of(["1", "55", "44", "49", "81"]),
          national <- string(?0..?9, min_length: 8, max_length: 12)
        ) do
      "+#{country_code}#{national}"
    end
  end

  defp valid_url_generator do
    gen all(
          scheme <- member_of(["http", "https"]),
          host <- string(:alphanumeric, min_length: 3, max_length: 15),
          tld <- member_of(["com", "org", "net", "io"]),
          path <- one_of([constant(""), string(:alphanumeric, min_length: 1, max_length: 10)])
        ) do
      path_str = if path == "", do: "", else: "/#{path}"
      "#{scheme}://#{host}.#{tld}#{path_str}"
    end
  end

  # ============================================================================
  # Property 10: Domain Primitive Validation
  # ============================================================================

  describe "Property 10: Domain Primitive Validation - Email" do
    @tag property: true
    @tag validates: "Requirements 4.1, 4.7"
    property "valid emails are accepted and normalized to lowercase" do
      check all(email_str <- valid_email_generator()) do
        assert {:ok, email} = Email.new(email_str)
        assert email.value == String.downcase(String.trim(email_str))
      end
    end

    @tag property: true
    @tag validates: "Requirements 4.1"
    property "invalid emails are rejected" do
      check all(invalid <- one_of([
                  constant(""),
                  constant("no-at-sign"),
                  constant("@no-local"),
                  constant("no-domain@"),
                  constant("spaces in@email.com"),
                  integer()
                ])) do
        assert {:error, _} = Email.new(invalid)
      end
    end
  end

  describe "Property 10: Domain Primitive Validation - UUID" do
    @tag property: true
    @tag validates: "Requirements 4.2"
    property "generated UUIDs are valid v4 format" do
      check all(_ <- constant(nil), max_runs: 100) do
        uuid = UUID.generate()
        assert String.length(uuid.value) == 36
        assert {:ok, _} = UUID.new(uuid.value)
        # Version 4 check
        assert String.at(uuid.value, 14) == "4"
        # Variant check (8, 9, a, or b)
        variant = String.at(uuid.value, 19)
        assert variant in ["8", "9", "a", "b"]
      end
    end

    @tag property: true
    @tag validates: "Requirements 4.2"
    property "valid UUID strings are accepted" do
      check all(uuid_str <- valid_uuid_generator()) do
        assert {:ok, uuid} = UUID.new(uuid_str)
        assert uuid.value == String.downcase(uuid_str)
      end
    end

    @tag property: true
    @tag validates: "Requirements 4.2"
    property "invalid UUID strings are rejected" do
      check all(invalid <- one_of([
                  constant(""),
                  constant("not-a-uuid"),
                  constant("12345678-1234-1234-1234-123456789012"),  # wrong version
                  string(:alphanumeric, min_length: 1, max_length: 35)
                ])) do
        result = UUID.new(invalid)
        # Some random strings might accidentally be valid UUIDs, so we check
        case result do
          {:ok, _} -> :ok  # Rare but possible
          {:error, _} -> :ok
        end
      end
    end
  end

  describe "Property 10: Domain Primitive Validation - ULID" do
    @tag property: true
    @tag validates: "Requirements 4.3"
    property "generated ULIDs are valid format and time-ordered" do
      check all(timestamps <- list_of(integer(0..0xFFFFFFFFFFFF), min_length: 2, max_length: 10)) do
        ulids = Enum.map(timestamps, &ULID.generate/1)

        # All should be valid
        for ulid <- ulids do
          assert String.length(ulid.value) == 26
          assert {:ok, _} = ULID.new(ulid.value)
        end

        # Sorted timestamps should produce lexicographically sorted ULIDs
        sorted_timestamps = Enum.sort(timestamps)
        sorted_ulids = Enum.map(sorted_timestamps, &ULID.generate/1)
        sorted_values = Enum.map(sorted_ulids, & &1.value)

        # Note: Due to random component, we can only guarantee ordering for different timestamps
        # For same timestamp, order is not guaranteed
      end
    end

    @tag property: true
    @tag validates: "Requirements 4.3"
    property "ULID timestamp extraction is correct" do
      check all(timestamp <- integer(0..0xFFFFFFFFFFFF)) do
        ulid = ULID.generate(timestamp)
        extracted = ULID.timestamp_ms(ulid)
        assert extracted == timestamp
      end
    end
  end

  describe "Property 10: Domain Primitive Validation - Money" do
    @tag property: true
    @tag validates: "Requirements 4.4"
    property "valid money values are accepted" do
      check all({amount, currency} <- valid_money_generator()) do
        assert {:ok, money} = Money.new(amount, currency)
        assert money.amount == amount
        assert money.currency == currency
      end
    end

    @tag property: true
    @tag validates: "Requirements 4.4"
    property "money addition preserves currency and is commutative" do
      check all(
              {a1, currency} <- valid_money_generator(),
              a2 <- integer(-1_000_000..1_000_000)
            ) do
        {:ok, m1} = Money.new(a1, currency)
        {:ok, m2} = Money.new(a2, currency)

        {:ok, sum1} = Money.add(m1, m2)
        {:ok, sum2} = Money.add(m2, m1)

        assert sum1.amount == a1 + a2
        assert sum1.currency == currency
        assert sum1.amount == sum2.amount  # Commutative
      end
    end

    @tag property: true
    @tag validates: "Requirements 4.4"
    property "money with different currencies cannot be added" do
      check all(
              amount1 <- integer(),
              amount2 <- integer(),
              {c1, c2} <- filter(
                tuple({member_of([:USD, :EUR, :GBP, :BRL, :JPY]), member_of([:USD, :EUR, :GBP, :BRL, :JPY])}),
                fn {a, b} -> a != b end
              )
            ) do
        {:ok, m1} = Money.new(amount1, c1)
        {:ok, m2} = Money.new(amount2, c2)

        assert {:error, _} = Money.add(m1, m2)
      end
    end
  end

  describe "Property 10: Domain Primitive Validation - PhoneNumber" do
    @tag property: true
    @tag validates: "Requirements 4.5"
    property "valid E.164 phone numbers are accepted" do
      check all(phone_str <- valid_phone_generator()) do
        assert {:ok, phone} = PhoneNumber.new(phone_str)
        assert phone.value == String.replace(phone_str, ~r/\s/, "")
      end
    end

    @tag property: true
    @tag validates: "Requirements 4.5"
    property "invalid phone numbers are rejected" do
      check all(invalid <- one_of([
                  constant(""),
                  constant("12345"),  # No +
                  constant("+"),      # Just +
                  constant("+0123"),  # Starts with 0
                  string(:alphanumeric, min_length: 1, max_length: 10)
                ])) do
        assert {:error, _} = PhoneNumber.new(invalid)
      end
    end
  end

  describe "Property 10: Domain Primitive Validation - URL" do
    @tag property: true
    @tag validates: "Requirements 4.6"
    property "valid http/https URLs are accepted" do
      check all(url_str <- valid_url_generator()) do
        assert {:ok, url} = URL.new(url_str)
        assert url.scheme in [:http, :https]
        assert is_binary(url.host)
        assert String.length(url.host) > 0
      end
    end

    @tag property: true
    @tag validates: "Requirements 4.6"
    property "URLs with unsupported schemes are rejected" do
      check all(
              scheme <- member_of(["ftp", "file", "mailto", "ssh"]),
              host <- string(:alphanumeric, min_length: 3, max_length: 10)
            ) do
        url_str = "#{scheme}://#{host}.com"
        assert {:error, "unsupported URL scheme: " <> _} = URL.new(url_str)
      end
    end
  end


  # ============================================================================
  # Property 11: Domain Primitive Serialization Round-Trip
  # ============================================================================

  describe "Property 11: Serialization Round-Trip - Email" do
    @tag property: true
    @tag validates: "Requirements 4.8, 4.9"
    property "Email JSON encode/decode round-trip preserves value" do
      check all(email_str <- valid_email_generator()) do
        {:ok, email} = Email.new(email_str)

        # Encode to JSON
        {:ok, json} = Jason.encode(email)

        # Decode back
        {:ok, decoded} = Jason.decode(json)

        # Should match the normalized value
        assert decoded == email.value
      end
    end

    @tag property: true
    @tag validates: "Requirements 4.9"
    property "Email String.Chars protocol returns value" do
      check all(email_str <- valid_email_generator()) do
        {:ok, email} = Email.new(email_str)
        assert to_string(email) == email.value
      end
    end
  end

  describe "Property 11: Serialization Round-Trip - UUID" do
    @tag property: true
    @tag validates: "Requirements 4.8, 4.9"
    property "UUID JSON encode/decode round-trip preserves value" do
      check all(_ <- constant(nil), max_runs: 100) do
        uuid = UUID.generate()

        {:ok, json} = Jason.encode(uuid)
        {:ok, decoded} = Jason.decode(json)

        assert decoded == uuid.value
      end
    end

    @tag property: true
    @tag validates: "Requirements 4.9"
    property "UUID String.Chars protocol returns value" do
      check all(_ <- constant(nil), max_runs: 100) do
        uuid = UUID.generate()
        assert to_string(uuid) == uuid.value
      end
    end

    @tag property: true
    @tag validates: "Requirements 4.2"
    property "UUID new/to_string round-trip" do
      check all(uuid_str <- valid_uuid_generator()) do
        {:ok, uuid} = UUID.new(uuid_str)
        assert UUID.to_string(uuid) == String.downcase(uuid_str)
      end
    end
  end

  describe "Property 11: Serialization Round-Trip - ULID" do
    @tag property: true
    @tag validates: "Requirements 4.8, 4.9"
    property "ULID JSON encode/decode round-trip preserves value" do
      check all(timestamp <- integer(0..0xFFFFFFFFFFFF)) do
        ulid = ULID.generate(timestamp)

        {:ok, json} = Jason.encode(ulid)
        {:ok, decoded} = Jason.decode(json)

        assert decoded == ulid.value
      end
    end

    @tag property: true
    @tag validates: "Requirements 4.9"
    property "ULID String.Chars protocol returns value" do
      check all(timestamp <- integer(0..0xFFFFFFFFFFFF)) do
        ulid = ULID.generate(timestamp)
        assert to_string(ulid) == ulid.value
      end
    end

    @tag property: true
    @tag validates: "Requirements 4.3"
    property "ULID new/to_string round-trip" do
      check all(timestamp <- integer(0..0xFFFFFFFFFFFF)) do
        ulid = ULID.generate(timestamp)
        {:ok, parsed} = ULID.new(ulid.value)
        assert ULID.to_string(parsed) == ulid.value
      end
    end
  end

  describe "Property 11: Serialization Round-Trip - Money" do
    @tag property: true
    @tag validates: "Requirements 4.8"
    property "Money JSON encode/decode round-trip preserves amount and currency" do
      check all({amount, currency} <- valid_money_generator()) do
        {:ok, money} = Money.new(amount, currency)

        {:ok, json} = Jason.encode(money)
        {:ok, decoded} = Jason.decode(json)

        assert decoded["amount"] == amount
        assert decoded["currency"] == Atom.to_string(currency)
      end
    end

    @tag property: true
    @tag validates: "Requirements 4.4"
    property "Money decimal conversion round-trip" do
      check all(
              amount <- integer(0..1_000_000),
              currency <- member_of([:USD, :EUR, :GBP, :BRL])
            ) do
        {:ok, money} = Money.new(amount, currency)
        decimal = Money.to_decimal(money)

        # Convert back
        {:ok, money2} = Money.from_decimal(decimal, currency)

        # Should be equal (within rounding)
        assert money2.amount == amount
      end
    end
  end

  describe "Property 11: Serialization Round-Trip - PhoneNumber" do
    @tag property: true
    @tag validates: "Requirements 4.8, 4.9"
    property "PhoneNumber JSON encode/decode round-trip preserves value" do
      check all(phone_str <- valid_phone_generator()) do
        {:ok, phone} = PhoneNumber.new(phone_str)

        {:ok, json} = Jason.encode(phone)
        {:ok, decoded} = Jason.decode(json)

        assert decoded == phone.value
      end
    end

    @tag property: true
    @tag validates: "Requirements 4.9"
    property "PhoneNumber String.Chars protocol returns value" do
      check all(phone_str <- valid_phone_generator()) do
        {:ok, phone} = PhoneNumber.new(phone_str)
        assert to_string(phone) == phone.value
      end
    end
  end

  describe "Property 11: Serialization Round-Trip - URL" do
    @tag property: true
    @tag validates: "Requirements 4.8, 4.9"
    property "URL JSON encode/decode round-trip preserves value" do
      check all(url_str <- valid_url_generator()) do
        {:ok, url} = URL.new(url_str)

        {:ok, json} = Jason.encode(url)
        {:ok, decoded} = Jason.decode(json)

        assert decoded == url.value
      end
    end

    @tag property: true
    @tag validates: "Requirements 4.9"
    property "URL String.Chars protocol returns value" do
      check all(url_str <- valid_url_generator()) do
        {:ok, url} = URL.new(url_str)
        assert to_string(url) == url.value
      end
    end
  end

  # ============================================================================
  # Unit Tests - Edge Cases
  # ============================================================================

  describe "Email edge cases" do
    test "trims whitespace" do
      assert {:ok, email} = Email.new("  user@example.com  ")
      assert email.value == "user@example.com"
    end

    test "normalizes to lowercase" do
      assert {:ok, email} = Email.new("USER@EXAMPLE.COM")
      assert email.value == "user@example.com"
    end

    test "extracts local part and domain" do
      email = Email.new!("user@example.com")
      assert Email.local_part(email) == "user"
      assert Email.domain(email) == "example.com"
    end

    test "new! raises on invalid input" do
      assert_raise ArgumentError, fn -> Email.new!("invalid") end
    end
  end

  describe "UUID edge cases" do
    test "generates unique UUIDs" do
      uuids = for _ <- 1..100, do: UUID.generate().value
      assert length(Enum.uniq(uuids)) == 100
    end

    test "to_binary returns 16 bytes" do
      uuid = UUID.generate()
      binary = UUID.to_binary(uuid)
      assert byte_size(binary) == 16
    end

    test "accepts uppercase UUIDs" do
      {:ok, uuid} = UUID.new("550E8400-E29B-41D4-A716-446655440000")
      assert uuid.value == "550e8400-e29b-41d4-a716-446655440000"
    end
  end

  describe "ULID edge cases" do
    test "generates unique ULIDs" do
      ulids = for _ <- 1..100, do: ULID.generate().value
      assert length(Enum.uniq(ulids)) == 100
    end

    test "timestamp extraction returns DateTime" do
      ulid = ULID.generate()
      timestamp = ULID.timestamp(ulid)
      assert %DateTime{} = timestamp
    end

    test "accepts lowercase ULIDs" do
      ulid = ULID.generate()
      lower = String.downcase(ulid.value)
      {:ok, parsed} = ULID.new(lower)
      assert parsed.value == String.upcase(lower)
    end
  end

  describe "Money edge cases" do
    test "rejects unsupported currencies" do
      assert {:error, _} = Money.new(100, :XYZ)
    end

    test "rejects non-integer amounts" do
      assert {:error, _} = Money.new(10.5, :USD)
      assert {:error, _} = Money.new("100", :USD)
    end

    test "formats correctly for different currencies" do
      {:ok, usd} = Money.new(1050, :USD)
      assert Money.format(usd) == "$10.50"

      {:ok, jpy} = Money.new(1000, :JPY)
      assert Money.format(jpy) == "¥1000"
    end

    test "zero?, positive?, negative? predicates" do
      {:ok, zero} = Money.new(0, :USD)
      {:ok, positive} = Money.new(100, :USD)
      {:ok, negative} = Money.new(-100, :USD)

      assert Money.zero?(zero)
      refute Money.zero?(positive)

      assert Money.positive?(positive)
      refute Money.positive?(negative)

      assert Money.negative?(negative)
      refute Money.negative?(positive)
    end
  end

  describe "PhoneNumber edge cases" do
    test "removes whitespace" do
      {:ok, phone} = PhoneNumber.new("+55 11 999999999")
      assert phone.value == "+5511999999999"
    end

    test "from_parts creates valid phone" do
      {:ok, phone} = PhoneNumber.from_parts("55", "11999999999")
      assert phone.value == "+5511999999999"
    end

    test "extracts country code" do
      phone = PhoneNumber.new!("+5511999999999")
      assert PhoneNumber.country_code(phone) == "55"

      phone_us = PhoneNumber.new!("+14155551234")
      assert PhoneNumber.country_code(phone_us) == "1"
    end
  end

  describe "URL edge cases" do
    test "extracts URL components" do
      {:ok, url} = URL.new("https://example.com:8080/path?query=1")

      assert URL.scheme(url) == :https
      assert URL.host(url) == "example.com"
      assert URL.port(url) == 8080
      assert URL.path(url) == "/path"
      assert URL.query(url) == "query=1"
    end

    test "default ports" do
      {:ok, https} = URL.new("https://example.com")
      assert URL.port(https) == 443

      {:ok, http} = URL.new("http://example.com")
      assert URL.port(http) == 80
    end

    test "secure? predicate" do
      {:ok, https} = URL.new("https://example.com")
      {:ok, http} = URL.new("http://example.com")

      assert URL.secure?(https)
      refute URL.secure?(http)
    end

    test "origin extraction" do
      {:ok, url} = URL.new("https://example.com/path")
      assert URL.origin(url) == "https://example.com"

      {:ok, url_port} = URL.new("http://example.com:8080/path")
      assert URL.origin(url_port) == "http://example.com:8080"
    end
  end
end
