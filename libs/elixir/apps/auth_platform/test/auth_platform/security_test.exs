defmodule AuthPlatform.SecurityTest do
  use ExUnit.Case, async: true

  alias AuthPlatform.Security

  describe "constant_time_compare/2" do
    test "returns true for equal strings" do
      assert Security.constant_time_compare("secret", "secret")
      assert Security.constant_time_compare("", "")
    end

    test "returns false for different strings" do
      refute Security.constant_time_compare("secret", "other")
      refute Security.constant_time_compare("secret", "secrets")
      refute Security.constant_time_compare("a", "b")
    end

    test "returns false for different lengths" do
      refute Security.constant_time_compare("short", "longer")
    end

    test "returns false for non-binary inputs" do
      refute Security.constant_time_compare(123, "123")
      refute Security.constant_time_compare(nil, nil)
    end
  end

  describe "generate_token/2" do
    test "generates hex token by default" do
      token = Security.generate_token(16)
      assert byte_size(token) == 32
      assert Regex.match?(~r/^[0-9a-f]+$/, token)
    end

    test "generates base64 token" do
      token = Security.generate_token(16, encoding: :base64)
      assert {:ok, _} = Base.decode64(token)
    end

    test "generates url-safe base64 token" do
      token = Security.generate_token(16, encoding: :url_safe_base64)
      refute String.contains?(token, "+")
      refute String.contains?(token, "/")
    end

    test "generates unique tokens" do
      tokens = for _ <- 1..100, do: Security.generate_token(16)
      assert length(Enum.uniq(tokens)) == 100
    end
  end

  describe "mask_sensitive/2" do
    test "masks with default settings" do
      assert Security.mask_sensitive("4111111111111111") == "************1111"
    end

    test "masks with custom visible count" do
      assert Security.mask_sensitive("secret123", visible: 3) == "******123"
    end

    test "masks with custom character" do
      assert Security.mask_sensitive("secret", visible: 2, mask_char: "X") == "XXXXet"
    end

    test "masks entire string when too short" do
      assert Security.mask_sensitive("abc") == "***"
      assert Security.mask_sensitive("abcd") == "****"
    end

    test "handles empty string" do
      assert Security.mask_sensitive("") == ""
    end
  end

  describe "sanitize_html/1" do
    test "escapes HTML special characters" do
      assert Security.sanitize_html("<script>") == "&lt;script&gt;"
      assert Security.sanitize_html("\"quoted\"") == "&quot;quoted&quot;"
      assert Security.sanitize_html("'single'") == "&#39;single&#39;"
      assert Security.sanitize_html("a & b") == "a &amp; b"
    end

    test "handles XSS attempts" do
      input = "<script>alert('xss')</script>"
      output = Security.sanitize_html(input)
      refute String.contains?(output, "<script>")
    end

    test "preserves safe content" do
      assert Security.sanitize_html("Hello World") == "Hello World"
    end
  end

  describe "sanitize_sql/1" do
    test "escapes single quotes" do
      assert Security.sanitize_sql("O'Brien") == "O''Brien"
    end

    test "escapes backslashes" do
      assert Security.sanitize_sql("path\\file") == "path\\\\file"
    end

    test "removes null bytes" do
      assert Security.sanitize_sql("test\x00value") == "testvalue"
    end
  end

  describe "detect_sql_injection/1" do
    test "detects OR 1=1 pattern" do
      assert Security.detect_sql_injection("' OR 1=1 --")
      assert Security.detect_sql_injection("admin' OR '1'='1")
    end

    test "detects UNION SELECT" do
      assert Security.detect_sql_injection("' UNION SELECT * FROM users")
      assert Security.detect_sql_injection("' UNION ALL SELECT password")
    end

    test "detects DROP TABLE" do
      assert Security.detect_sql_injection("'; DROP TABLE users; --")
    end

    test "detects comment injection" do
      assert Security.detect_sql_injection("admin'--")
      assert Security.detect_sql_injection("/* comment */")
    end

    test "returns false for safe input" do
      refute Security.detect_sql_injection("John Doe")
      refute Security.detect_sql_injection("user@example.com")
      refute Security.detect_sql_injection("123-456-7890")
    end
  end

  describe "hash/1" do
    test "returns SHA-256 hash" do
      hash = Security.hash("password")
      assert byte_size(hash) == 64
      assert Regex.match?(~r/^[0-9a-f]+$/, hash)
    end

    test "produces consistent results" do
      hash1 = Security.hash("test")
      hash2 = Security.hash("test")
      assert hash1 == hash2
    end

    test "produces different hashes for different inputs" do
      hash1 = Security.hash("password1")
      hash2 = Security.hash("password2")
      refute hash1 == hash2
    end
  end

  describe "hash_with_salt/2" do
    test "produces different hash than unsalted" do
      unsalted = Security.hash("password")
      salted = Security.hash_with_salt("password", "salt")
      refute unsalted == salted
    end

    test "different salts produce different hashes" do
      hash1 = Security.hash_with_salt("password", "salt1")
      hash2 = Security.hash_with_salt("password", "salt2")
      refute hash1 == hash2
    end
  end
end
