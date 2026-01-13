# Design Document: Auth Platform Elixir Library

## Overview

This document describes the technical design for the Auth Platform Elixir shared library (`libs/elixir`). The library provides production-ready modules for building resilient microservices, following functional programming principles and leveraging Elixir's strengths: immutability, pattern matching, behaviours (as generics), protocols, and the actor model via GenServer.

The design mirrors the patterns established in the Go and Rust libraries while embracing Elixir idioms:
- **Behaviours** replace Go generics for polymorphism
- **Protocols** provide type-class-like dispatch
- **GenServer** provides thread-safe stateful components
- **Telemetry** provides observability hooks
- **StreamData** provides property-based testing

## Architecture

```
libs/elixir/
├── mix.exs                    # Umbrella project config
├── README.md
├── CHANGELOG.md
├── apps/
│   ├── auth_platform/         # Core library (main app)
│   │   ├── lib/
│   │   │   ├── auth_platform.ex
│   │   │   ├── functional/    # Result, Option types
│   │   │   ├── errors/        # AppError, error types
│   │   │   ├── validation/    # Composable validators
│   │   │   ├── domain/        # Domain primitives
│   │   │   ├── resilience/    # Circuit breaker, retry, etc.
│   │   │   ├── codec/         # JSON, Base64 codecs
│   │   │   ├── observability/ # Logging, tracing
│   │   │   └── security/      # Security utilities
│   │   └── test/
│   ├── auth_platform_clients/ # Platform service clients
│   │   ├── lib/
│   │   │   ├── logging/       # Logging service client
│   │   │   └── cache/         # Cache service client
│   │   └── test/
│   └── auth_platform_testing/ # Test utilities
│       ├── lib/
│       │   └── generators/    # StreamData generators
│       └── test/
└── config/
    ├── config.exs
    ├── dev.exs
    ├── test.exs
    └── prod.exs
```


## Components and Interfaces

### 1. Functional Types Module (`AuthPlatform.Functional`)

#### Result Type

```elixir
defmodule AuthPlatform.Functional.Result do
  @moduledoc "Type-safe result handling for operations that may fail."

  @type t(ok, err) :: {:ok, ok} | {:error, err}
  @type t(ok) :: t(ok, any())

  @spec ok(value) :: {:ok, value} when value: any()
  def ok(value), do: {:ok, value}

  @spec error(reason) :: {:error, reason} when reason: any()
  def error(reason), do: {:error, reason}

  @spec map(t(a, e), (a -> b)) :: t(b, e) when a: any(), b: any(), e: any()
  def map({:ok, value}, fun), do: {:ok, fun.(value)}
  def map({:error, _} = err, _fun), do: err

  @spec flat_map(t(a, e), (a -> t(b, e))) :: t(b, e) when a: any(), b: any(), e: any()
  def flat_map({:ok, value}, fun), do: fun.(value)
  def flat_map({:error, _} = err, _fun), do: err

  @spec unwrap!(t(a, any())) :: a when a: any()
  def unwrap!({:ok, value}), do: value
  def unwrap!({:error, reason}), do: raise "Unwrap on error: #{inspect(reason)}"

  @spec unwrap_or(t(a, any()), a) :: a when a: any()
  def unwrap_or({:ok, value}, _default), do: value
  def unwrap_or({:error, _}, default), do: default

  @spec match(t(a, e), (a -> b), (e -> b)) :: b when a: any(), b: any(), e: any()
  def match({:ok, value}, on_ok, _on_error), do: on_ok.(value)
  def match({:error, reason}, _on_ok, on_error), do: on_error.(reason)

  @spec is_ok?(t(any(), any())) :: boolean()
  def is_ok?({:ok, _}), do: true
  def is_ok?({:error, _}), do: false

  @spec is_error?(t(any(), any())) :: boolean()
  def is_error?(result), do: not is_ok?(result)

  defmacro try_result(do: block) do
    quote do
      try do
        {:ok, unquote(block)}
      rescue
        e -> {:error, e}
      end
    end
  end
end
```

#### Option Type

```elixir
defmodule AuthPlatform.Functional.Option do
  @moduledoc "Type-safe optional value handling."

  @type t(a) :: {:some, a} | :none

  @spec some(value) :: {:some, value} when value: any()
  def some(value), do: {:some, value}

  @spec none() :: :none
  def none(), do: :none

  @spec from_nullable(value | nil) :: t(value) when value: any()
  def from_nullable(nil), do: :none
  def from_nullable(value), do: {:some, value}

  @spec map(t(a), (a -> b)) :: t(b) when a: any(), b: any()
  def map({:some, value}, fun), do: {:some, fun.(value)}
  def map(:none, _fun), do: :none

  @spec flat_map(t(a), (a -> t(b))) :: t(b) when a: any(), b: any()
  def flat_map({:some, value}, fun), do: fun.(value)
  def flat_map(:none, _fun), do: :none

  @spec unwrap!(t(a)) :: a when a: any()
  def unwrap!({:some, value}), do: value
  def unwrap!(:none), do: raise "Unwrap on none"

  @spec unwrap_or(t(a), a) :: a when a: any()
  def unwrap_or({:some, value}, _default), do: value
  def unwrap_or(:none, default), do: default

  @spec is_some?(t(any())) :: boolean()
  def is_some?({:some, _}), do: true
  def is_some?(:none), do: false

  @spec is_none?(t(any())) :: boolean()
  def is_none?(opt), do: not is_some?(opt)
end
```


### 2. Error Handling Module (`AuthPlatform.Errors`)

```elixir
defmodule AuthPlatform.Errors.AppError do
  @moduledoc "Structured application error with HTTP/gRPC mapping."

  @type error_code :: :not_found | :validation | :unauthorized | :forbidden |
                      :internal | :rate_limited | :timeout | :unavailable |
                      :conflict | :bad_request

  @type t :: %__MODULE__{
    code: error_code(),
    message: String.t(),
    details: map(),
    correlation_id: String.t() | nil,
    cause: Exception.t() | nil,
    retryable: boolean()
  }

  defstruct [:code, :message, :correlation_id, :cause, details: %{}, retryable: false]

  @http_status_map %{
    not_found: 404,
    validation: 400,
    bad_request: 400,
    unauthorized: 401,
    forbidden: 403,
    internal: 500,
    rate_limited: 429,
    timeout: 504,
    unavailable: 503,
    conflict: 409
  }

  @grpc_code_map %{
    not_found: 5,      # NOT_FOUND
    validation: 3,     # INVALID_ARGUMENT
    bad_request: 3,    # INVALID_ARGUMENT
    unauthorized: 16,  # UNAUTHENTICATED
    forbidden: 7,      # PERMISSION_DENIED
    internal: 13,      # INTERNAL
    rate_limited: 8,   # RESOURCE_EXHAUSTED
    timeout: 4,        # DEADLINE_EXCEEDED
    unavailable: 14,   # UNAVAILABLE
    conflict: 6        # ALREADY_EXISTS
  }

  @spec not_found(String.t()) :: t()
  def not_found(resource), do: %__MODULE__{code: :not_found, message: "#{resource} not found"}

  @spec validation(String.t()) :: t()
  def validation(message), do: %__MODULE__{code: :validation, message: message}

  @spec unauthorized(String.t()) :: t()
  def unauthorized(message), do: %__MODULE__{code: :unauthorized, message: message}

  @spec internal(String.t()) :: t()
  def internal(message), do: %__MODULE__{code: :internal, message: message}

  @spec rate_limited() :: t()
  def rate_limited(), do: %__MODULE__{code: :rate_limited, message: "Rate limit exceeded", retryable: true}

  @spec timeout(String.t()) :: t()
  def timeout(operation), do: %__MODULE__{code: :timeout, message: "#{operation} timed out", retryable: true}

  @spec unavailable(String.t()) :: t()
  def unavailable(service), do: %__MODULE__{code: :unavailable, message: "#{service} unavailable", retryable: true}

  @spec with_details(t(), map()) :: t()
  def with_details(%__MODULE__{} = error, details), do: %{error | details: Map.merge(error.details, details)}

  @spec with_correlation_id(t(), String.t()) :: t()
  def with_correlation_id(%__MODULE__{} = error, id), do: %{error | correlation_id: id}

  @spec http_status(t()) :: pos_integer()
  def http_status(%__MODULE__{code: code}), do: Map.get(@http_status_map, code, 500)

  @spec grpc_code(t()) :: non_neg_integer()
  def grpc_code(%__MODULE__{code: code}), do: Map.get(@grpc_code_map, code, 13)

  @spec is_retryable?(t()) :: boolean()
  def is_retryable?(%__MODULE__{retryable: retryable}), do: retryable

  @spec to_api_response(t()) :: map()
  def to_api_response(%__MODULE__{} = error) do
    %{
      error: %{
        code: error.code,
        message: error.message,
        correlation_id: error.correlation_id
      }
    }
  end
end
```


### 3. Validation Module (`AuthPlatform.Validation`)

```elixir
defmodule AuthPlatform.Validation do
  @moduledoc "Composable validation with error accumulation."

  @type validation_error :: {String.t(), String.t()}  # {field, message}
  @type validation_result(a) :: {:ok, a} | {:errors, [validation_error()]}

  @type validator(a) :: (a -> validation_result(a))

  @spec validate_all([validation_result(any())]) :: validation_result([any()])
  def validate_all(results) do
    {values, errors} = Enum.reduce(results, {[], []}, fn
      {:ok, value}, {vals, errs} -> {[value | vals], errs}
      {:errors, errs}, {vals, acc_errs} -> {vals, errs ++ acc_errs}
    end)

    case errors do
      [] -> {:ok, Enum.reverse(values)}
      _ -> {:errors, errors}
    end
  end

  @spec validate_field(String.t(), any(), [validator(any())]) :: validation_result(any())
  def validate_field(field, value, validators) do
    errors = validators
    |> Enum.flat_map(fn validator ->
      case validator.(value) do
        {:ok, _} -> []
        {:errors, errs} -> Enum.map(errs, fn {_, msg} -> {field, msg} end)
      end
    end)

    case errors do
      [] -> {:ok, value}
      _ -> {:errors, errors}
    end
  end

  # String validators
  @spec required() :: validator(String.t())
  def required do
    fn
      nil -> {:errors, [{"", "is required"}]}
      "" -> {:errors, [{"", "is required"}]}
      value when is_binary(value) -> {:ok, value}
      _ -> {:errors, [{"", "must be a string"}]}
    end
  end

  @spec min_length(pos_integer()) :: validator(String.t())
  def min_length(min) do
    fn value when is_binary(value) ->
      if String.length(value) >= min,
        do: {:ok, value},
        else: {:errors, [{"", "must be at least #{min} characters"}]}
    end
  end

  @spec max_length(pos_integer()) :: validator(String.t())
  def max_length(max) do
    fn value when is_binary(value) ->
      if String.length(value) <= max,
        do: {:ok, value},
        else: {:errors, [{"", "must be at most #{max} characters"}]}
    end
  end

  @spec matches_regex(Regex.t()) :: validator(String.t())
  def matches_regex(regex) do
    fn value when is_binary(value) ->
      if Regex.match?(regex, value),
        do: {:ok, value},
        else: {:errors, [{"", "does not match required format"}]}
    end
  end

  # Numeric validators
  @spec positive() :: validator(number())
  def positive do
    fn value when is_number(value) ->
      if value > 0, do: {:ok, value}, else: {:errors, [{"", "must be positive"}]}
    end
  end

  @spec in_range(number(), number()) :: validator(number())
  def in_range(min, max) do
    fn value when is_number(value) ->
      if value >= min and value <= max,
        do: {:ok, value},
        else: {:errors, [{"", "must be between #{min} and #{max}"}]}
    end
  end

  # Composition
  @spec all([validator(a)]) :: validator(a) when a: any()
  def all(validators) do
    fn value ->
      errors = validators
      |> Enum.flat_map(fn v ->
        case v.(value) do
          {:ok, _} -> []
          {:errors, errs} -> errs
        end
      end)

      case errors do
        [] -> {:ok, value}
        _ -> {:errors, errors}
      end
    end
  end
end
```


### 4. Domain Primitives Module (`AuthPlatform.Domain`)

```elixir
defmodule AuthPlatform.Domain.Email do
  @moduledoc "RFC 5322 compliant email address."

  @type t :: %__MODULE__{value: String.t()}
  defstruct [:value]

  @email_regex ~r/^[a-zA-Z0-9.!#$%&'*+\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$/

  @spec new(String.t()) :: {:ok, t()} | {:error, String.t()}
  def new(value) when is_binary(value) do
    if Regex.match?(@email_regex, value),
      do: {:ok, %__MODULE__{value: String.downcase(value)}},
      else: {:error, "invalid email format"}
  end
  def new(_), do: {:error, "email must be a string"}

  @spec new!(String.t()) :: t()
  def new!(value) do
    case new(value) do
      {:ok, email} -> email
      {:error, reason} -> raise ArgumentError, reason
    end
  end

  @spec to_string(t()) :: String.t()
  def to_string(%__MODULE__{value: value}), do: value

  defimpl String.Chars do
    def to_string(%{value: value}), do: value
  end

  defimpl Jason.Encoder do
    def encode(%{value: value}, opts), do: Jason.Encode.string(value, opts)
  end
end

defmodule AuthPlatform.Domain.UUID do
  @moduledoc "RFC 4122 UUID v4 identifier."

  @type t :: %__MODULE__{value: String.t()}
  defstruct [:value]

  @uuid_regex ~r/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i

  @spec generate() :: t()
  def generate do
    value = :crypto.strong_rand_bytes(16)
    |> set_version_4()
    |> set_variant()
    |> format_uuid()
    %__MODULE__{value: value}
  end

  @spec new(String.t()) :: {:ok, t()} | {:error, String.t()}
  def new(value) when is_binary(value) do
    if Regex.match?(@uuid_regex, value),
      do: {:ok, %__MODULE__{value: String.downcase(value)}},
      else: {:error, "invalid UUID format"}
  end

  defp set_version_4(<<a::48, _::4, b::12, _::2, c::62>>), do: <<a::48, 4::4, b::12, 2::2, c::62>>
  defp set_variant(bytes), do: bytes

  defp format_uuid(<<a::32, b::16, c::16, d::16, e::48>>) do
    [a, b, c, d, e]
    |> Enum.map(&Integer.to_string(&1, 16))
    |> Enum.map(&String.pad_leading(&1, 8, "0"))
    |> Enum.join("-")
    |> String.downcase()
  end

  defimpl String.Chars do
    def to_string(%{value: value}), do: value
  end

  defimpl Jason.Encoder do
    def encode(%{value: value}, opts), do: Jason.Encode.string(value, opts)
  end
end

defmodule AuthPlatform.Domain.Money do
  @moduledoc "Monetary value with currency."

  @type currency :: :USD | :EUR | :GBP | :BRL | :JPY
  @type t :: %__MODULE__{amount: integer(), currency: currency()}
  defstruct [:amount, :currency]

  @currencies [:USD, :EUR, :GBP, :BRL, :JPY]

  @spec new(integer(), currency()) :: {:ok, t()} | {:error, String.t()}
  def new(amount, currency) when is_integer(amount) and currency in @currencies do
    {:ok, %__MODULE__{amount: amount, currency: currency}}
  end
  def new(_, currency) when currency not in @currencies, do: {:error, "unsupported currency"}
  def new(_, _), do: {:error, "amount must be an integer (cents)"}

  @spec add(t(), t()) :: {:ok, t()} | {:error, String.t()}
  def add(%__MODULE__{currency: c} = a, %__MODULE__{currency: c} = b) do
    {:ok, %__MODULE__{amount: a.amount + b.amount, currency: c}}
  end
  def add(_, _), do: {:error, "currency mismatch"}

  defimpl Jason.Encoder do
    def encode(%{amount: amount, currency: currency}, opts) do
      Jason.Encode.map(%{amount: amount, currency: currency}, opts)
    end
  end
end
```


### 5. Circuit Breaker Module (`AuthPlatform.Resilience.CircuitBreaker`)

```elixir
defmodule AuthPlatform.Resilience.CircuitBreaker do
  @moduledoc "Circuit breaker pattern implementation using GenServer."
  use GenServer

  @type state :: :closed | :open | :half_open
  @type config :: %{
    failure_threshold: pos_integer(),
    success_threshold: pos_integer(),
    timeout_ms: pos_integer(),
    half_open_max_requests: pos_integer()
  }

  defstruct [
    :name,
    :config,
    state: :closed,
    failures: 0,
    successes: 0,
    last_failure_at: nil,
    half_open_requests: 0
  ]

  @default_config %{
    failure_threshold: 5,
    success_threshold: 2,
    timeout_ms: 30_000,
    half_open_max_requests: 3
  }

  # Client API

  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts) do
    name = Keyword.fetch!(opts, :name)
    config = Keyword.get(opts, :config, @default_config)
    GenServer.start_link(__MODULE__, {name, config}, name: via_tuple(name))
  end

  @spec allow_request?(atom()) :: boolean()
  def allow_request?(name), do: GenServer.call(via_tuple(name), :allow_request?)

  @spec record_success(atom()) :: :ok
  def record_success(name), do: GenServer.cast(via_tuple(name), :record_success)

  @spec record_failure(atom()) :: :ok
  def record_failure(name), do: GenServer.cast(via_tuple(name), :record_failure)

  @spec get_state(atom()) :: state()
  def get_state(name), do: GenServer.call(via_tuple(name), :get_state)

  @spec reset(atom()) :: :ok
  def reset(name), do: GenServer.cast(via_tuple(name), :reset)

  @spec execute(atom(), (() -> {:ok, a} | {:error, any()})) :: {:ok, a} | {:error, any()} when a: any()
  def execute(name, fun) do
    if allow_request?(name) do
      case fun.() do
        {:ok, result} ->
          record_success(name)
          {:ok, result}
        {:error, reason} ->
          record_failure(name)
          {:error, reason}
      end
    else
      {:error, AuthPlatform.Errors.AppError.unavailable("circuit breaker open")}
    end
  end

  # Server callbacks

  @impl true
  def init({name, config}) do
    state = %__MODULE__{name: name, config: Map.merge(@default_config, config)}
    {:ok, state}
  end

  @impl true
  def handle_call(:allow_request?, _from, state) do
    {allowed, new_state} = check_and_transition(state)
    {:reply, allowed, new_state}
  end

  def handle_call(:get_state, _from, state), do: {:reply, state.state, state}

  @impl true
  def handle_cast(:record_success, state), do: {:noreply, handle_success(state)}
  def handle_cast(:record_failure, state), do: {:noreply, handle_failure(state)}
  def handle_cast(:reset, state), do: {:noreply, reset_state(state)}

  # Private functions

  defp via_tuple(name), do: {:via, Registry, {AuthPlatform.Resilience.Registry, name}}

  defp check_and_transition(%{state: :closed} = state), do: {true, state}

  defp check_and_transition(%{state: :open, last_failure_at: last, config: config} = state) do
    if System.monotonic_time(:millisecond) - last >= config.timeout_ms do
      emit_telemetry(:state_change, state.name, :open, :half_open)
      {true, %{state | state: :half_open, half_open_requests: 0, successes: 0}}
    else
      {false, state}
    end
  end

  defp check_and_transition(%{state: :half_open, half_open_requests: r, config: config} = state) do
    if r < config.half_open_max_requests do
      {true, %{state | half_open_requests: r + 1}}
    else
      {false, state}
    end
  end

  defp handle_success(%{state: :half_open, successes: s, config: config} = state) do
    new_successes = s + 1
    if new_successes >= config.success_threshold do
      emit_telemetry(:state_change, state.name, :half_open, :closed)
      %{state | state: :closed, failures: 0, successes: 0}
    else
      %{state | successes: new_successes}
    end
  end

  defp handle_success(%{state: :closed} = state), do: %{state | failures: 0}
  defp handle_success(state), do: state

  defp handle_failure(%{failures: f, config: config} = state) do
    new_failures = f + 1
    new_state = %{state | failures: new_failures, last_failure_at: System.monotonic_time(:millisecond)}

    if new_failures >= config.failure_threshold and state.state != :open do
      emit_telemetry(:state_change, state.name, state.state, :open)
      %{new_state | state: :open, successes: 0}
    else
      new_state
    end
  end

  defp reset_state(state) do
    emit_telemetry(:reset, state.name, state.state, :closed)
    %{state | state: :closed, failures: 0, successes: 0, half_open_requests: 0}
  end

  defp emit_telemetry(event, name, from, to) do
    :telemetry.execute(
      [:auth_platform, :circuit_breaker, event],
      %{},
      %{name: name, from_state: from, to_state: to}
    )
  end
end
```


### 6. Retry Policy Module (`AuthPlatform.Resilience.Retry`)

```elixir
defmodule AuthPlatform.Resilience.Retry do
  @moduledoc "Retry policy with exponential backoff and jitter."

  @type config :: %{
    max_retries: pos_integer(),
    initial_delay_ms: pos_integer(),
    max_delay_ms: pos_integer(),
    multiplier: float(),
    jitter: boolean()
  }

  @default_config %{
    max_retries: 3,
    initial_delay_ms: 100,
    max_delay_ms: 10_000,
    multiplier: 2.0,
    jitter: true
  }

  @spec default_config() :: config()
  def default_config, do: @default_config

  @spec delay_for_attempt(config(), non_neg_integer()) :: pos_integer()
  def delay_for_attempt(config, attempt) do
    base_delay = config.initial_delay_ms * :math.pow(config.multiplier, attempt)
    capped_delay = min(base_delay, config.max_delay_ms)

    if config.jitter do
      jitter_factor = 1.0 + :rand.uniform() * 0.25
      round(capped_delay * jitter_factor)
    else
      round(capped_delay)
    end
  end

  @spec should_retry?(config(), any(), non_neg_integer()) :: boolean()
  def should_retry?(config, error, attempt) do
    attempt < config.max_retries and is_retryable?(error)
  end

  @spec execute(config(), (() -> {:ok, a} | {:error, any()})) :: {:ok, a} | {:error, any()} when a: any()
  def execute(config \\ @default_config, fun) do
    do_execute(config, fun, 0)
  end

  defp do_execute(config, fun, attempt) do
    case fun.() do
      {:ok, result} ->
        {:ok, result}

      {:error, reason} = error ->
        if should_retry?(config, reason, attempt) do
          delay = delay_for_attempt(config, attempt)
          emit_retry_telemetry(attempt + 1, delay, reason)
          Process.sleep(delay)
          do_execute(config, fun, attempt + 1)
        else
          error
        end
    end
  end

  defp is_retryable?(%AuthPlatform.Errors.AppError{retryable: true}), do: true
  defp is_retryable?(%AuthPlatform.Errors.AppError{}), do: false
  defp is_retryable?({:timeout, _}), do: true
  defp is_retryable?(:timeout), do: true
  defp is_retryable?(_), do: false

  defp emit_retry_telemetry(attempt, delay, reason) do
    :telemetry.execute(
      [:auth_platform, :retry, :attempt],
      %{delay_ms: delay},
      %{attempt: attempt, reason: reason}
    )
  end
end
```

### 7. Rate Limiter Module (`AuthPlatform.Resilience.RateLimiter`)

```elixir
defmodule AuthPlatform.Resilience.RateLimiter do
  @moduledoc "Token bucket rate limiter using GenServer."
  use GenServer

  @type config :: %{
    rate: pos_integer(),      # tokens per second
    burst_size: pos_integer() # max tokens
  }

  defstruct [:name, :config, :tokens, :last_refill]

  @default_config %{rate: 100, burst_size: 100}

  # Client API

  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts) do
    name = Keyword.fetch!(opts, :name)
    config = Keyword.get(opts, :config, @default_config)
    GenServer.start_link(__MODULE__, {name, config}, name: via_tuple(name))
  end

  @spec allow?(atom()) :: boolean()
  def allow?(name), do: GenServer.call(via_tuple(name), :allow?)

  @spec acquire(atom(), pos_integer()) :: :ok | {:error, :rate_limited}
  def acquire(name, timeout_ms \\ 5000) do
    GenServer.call(via_tuple(name), {:acquire, timeout_ms}, timeout_ms + 1000)
  end

  # Server callbacks

  @impl true
  def init({name, config}) do
    state = %__MODULE__{
      name: name,
      config: config,
      tokens: config.burst_size,
      last_refill: System.monotonic_time(:millisecond)
    }
    {:ok, state}
  end

  @impl true
  def handle_call(:allow?, _from, state) do
    state = refill_tokens(state)
    if state.tokens > 0 do
      {:reply, true, %{state | tokens: state.tokens - 1}}
    else
      emit_rate_limited(state.name)
      {:reply, false, state}
    end
  end

  def handle_call({:acquire, timeout_ms}, from, state) do
    state = refill_tokens(state)
    if state.tokens > 0 do
      {:reply, :ok, %{state | tokens: state.tokens - 1}}
    else
      # Schedule retry
      Process.send_after(self(), {:retry_acquire, from, timeout_ms, System.monotonic_time(:millisecond)}, 10)
      {:noreply, state}
    end
  end

  @impl true
  def handle_info({:retry_acquire, from, timeout_ms, start_time}, state) do
    elapsed = System.monotonic_time(:millisecond) - start_time
    state = refill_tokens(state)

    cond do
      state.tokens > 0 ->
        GenServer.reply(from, :ok)
        {:noreply, %{state | tokens: state.tokens - 1}}

      elapsed >= timeout_ms ->
        emit_rate_limited(state.name)
        GenServer.reply(from, {:error, :rate_limited})
        {:noreply, state}

      true ->
        Process.send_after(self(), {:retry_acquire, from, timeout_ms, start_time}, 10)
        {:noreply, state}
    end
  end

  defp refill_tokens(%{last_refill: last, config: config, tokens: tokens} = state) do
    now = System.monotonic_time(:millisecond)
    elapsed_seconds = (now - last) / 1000.0
    new_tokens = min(tokens + round(elapsed_seconds * config.rate), config.burst_size)
    %{state | tokens: new_tokens, last_refill: now}
  end

  defp via_tuple(name), do: {:via, Registry, {AuthPlatform.Resilience.Registry, name}}

  defp emit_rate_limited(name) do
    :telemetry.execute([:auth_platform, :rate_limiter, :rejected], %{}, %{name: name})
  end
end
```


### 8. Bulkhead Module (`AuthPlatform.Resilience.Bulkhead`)

```elixir
defmodule AuthPlatform.Resilience.Bulkhead do
  @moduledoc "Bulkhead pattern for concurrency isolation."
  use GenServer

  @type config :: %{
    max_concurrent: pos_integer(),
    max_queue: pos_integer(),
    queue_timeout_ms: pos_integer()
  }

  defstruct [:name, :config, :active, :queue]

  @default_config %{max_concurrent: 10, max_queue: 100, queue_timeout_ms: 5000}

  # Client API

  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts) do
    name = Keyword.fetch!(opts, :name)
    config = Keyword.get(opts, :config, @default_config)
    GenServer.start_link(__MODULE__, {name, config}, name: via_tuple(name))
  end

  @spec execute(atom(), (() -> a)) :: {:ok, a} | {:error, :rejected | :timeout} when a: any()
  def execute(name, fun) do
    case GenServer.call(via_tuple(name), :acquire, :infinity) do
      :ok ->
        try do
          {:ok, fun.()}
        after
          GenServer.cast(via_tuple(name), :release)
        end

      {:error, _} = error ->
        error
    end
  end

  @spec available_permits(atom()) :: non_neg_integer()
  def available_permits(name), do: GenServer.call(via_tuple(name), :available_permits)

  # Server callbacks

  @impl true
  def init({name, config}) do
    state = %__MODULE__{
      name: name,
      config: config,
      active: 0,
      queue: :queue.new()
    }
    {:ok, state}
  end

  @impl true
  def handle_call(:acquire, from, %{active: active, config: config, queue: queue} = state) do
    cond do
      active < config.max_concurrent ->
        {:reply, :ok, %{state | active: active + 1}}

      :queue.len(queue) < config.max_queue ->
        timer_ref = Process.send_after(self(), {:queue_timeout, from}, config.queue_timeout_ms)
        new_queue = :queue.in({from, timer_ref}, queue)
        {:noreply, %{state | queue: new_queue}}

      true ->
        emit_rejected(state.name)
        {:reply, {:error, :rejected}, state}
    end
  end

  def handle_call(:available_permits, _from, %{active: active, config: config} = state) do
    {:reply, max(0, config.max_concurrent - active), state}
  end

  @impl true
  def handle_cast(:release, %{active: active, queue: queue} = state) do
    case :queue.out(queue) do
      {{:value, {from, timer_ref}}, new_queue} ->
        Process.cancel_timer(timer_ref)
        GenServer.reply(from, :ok)
        {:noreply, %{state | queue: new_queue}}

      {:empty, _} ->
        {:noreply, %{state | active: max(0, active - 1)}}
    end
  end

  @impl true
  def handle_info({:queue_timeout, from}, %{queue: queue} = state) do
    new_queue = :queue.filter(fn {f, _} -> f != from end, queue)
    GenServer.reply(from, {:error, :timeout})
    {:noreply, %{state | queue: new_queue}}
  end

  defp via_tuple(name), do: {:via, Registry, {AuthPlatform.Resilience.Registry, name}}

  defp emit_rejected(name) do
    :telemetry.execute([:auth_platform, :bulkhead, :rejected], %{}, %{name: name})
  end
end
```


### 9. Codec Module (`AuthPlatform.Codec`)

```elixir
defmodule AuthPlatform.Codec.JSON do
  @moduledoc "JSON encoding/decoding utilities."

  @spec encode(term()) :: {:ok, String.t()} | {:error, String.t()}
  def encode(term) do
    case Jason.encode(term) do
      {:ok, json} -> {:ok, json}
      {:error, %Jason.EncodeError{message: msg}} -> {:error, "JSON encode error: #{msg}"}
    end
  end

  @spec encode!(term()) :: String.t()
  def encode!(term), do: Jason.encode!(term)

  @spec encode_pretty(term()) :: {:ok, String.t()} | {:error, String.t()}
  def encode_pretty(term) do
    case Jason.encode(term, pretty: true) do
      {:ok, json} -> {:ok, json}
      {:error, %Jason.EncodeError{message: msg}} -> {:error, "JSON encode error: #{msg}"}
    end
  end

  @spec decode(String.t()) :: {:ok, term()} | {:error, String.t()}
  def decode(json) when is_binary(json) do
    case Jason.decode(json) do
      {:ok, term} -> {:ok, term}
      {:error, %Jason.DecodeError{position: pos}} -> {:error, "JSON decode error at position #{pos}"}
    end
  end

  @spec decode!(String.t()) :: term()
  def decode!(json), do: Jason.decode!(json)
end

defmodule AuthPlatform.Codec.Base64 do
  @moduledoc "Base64 encoding/decoding utilities."

  @spec encode(binary()) :: String.t()
  def encode(data) when is_binary(data), do: Base.encode64(data)

  @spec encode_url_safe(binary()) :: String.t()
  def encode_url_safe(data) when is_binary(data), do: Base.url_encode64(data, padding: false)

  @spec decode(String.t()) :: {:ok, binary()} | {:error, String.t()}
  def decode(encoded) when is_binary(encoded) do
    case Base.decode64(encoded) do
      {:ok, data} -> {:ok, data}
      :error -> {:error, "invalid base64 encoding"}
    end
  end

  @spec decode!(String.t()) :: binary()
  def decode!(encoded), do: Base.decode64!(encoded)

  @spec decode_url_safe(String.t()) :: {:ok, binary()} | {:error, String.t()}
  def decode_url_safe(encoded) when is_binary(encoded) do
    case Base.url_decode64(encoded, padding: false) do
      {:ok, data} -> {:ok, data}
      :error -> {:error, "invalid url-safe base64 encoding"}
    end
  end
end
```

### 10. Security Module (`AuthPlatform.Security`)

```elixir
defmodule AuthPlatform.Security do
  @moduledoc "Security utilities for the Auth Platform."

  @spec constant_time_compare(binary(), binary()) :: boolean()
  def constant_time_compare(a, b) when is_binary(a) and is_binary(b) do
    :crypto.hash_equals(a, b)
  end

  @spec generate_token(pos_integer()) :: String.t()
  def generate_token(length \\ 32) do
    length
    |> :crypto.strong_rand_bytes()
    |> Base.url_encode64(padding: false)
    |> binary_part(0, length)
  end

  @spec mask_sensitive(String.t(), keyword()) :: String.t()
  def mask_sensitive(value, opts \\ []) when is_binary(value) do
    visible = Keyword.get(opts, :visible, 4)
    mask_char = Keyword.get(opts, :mask_char, "*")

    len = String.length(value)
    if len <= visible * 2 do
      String.duplicate(mask_char, len)
    else
      prefix = String.slice(value, 0, visible)
      suffix = String.slice(value, -visible, visible)
      masked = String.duplicate(mask_char, len - visible * 2)
      "#{prefix}#{masked}#{suffix}"
    end
  end

  @spec sanitize_html(String.t()) :: String.t()
  def sanitize_html(input) when is_binary(input) do
    input
    |> String.replace("&", "&amp;")
    |> String.replace("<", "&lt;")
    |> String.replace(">", "&gt;")
    |> String.replace("\"", "&quot;")
    |> String.replace("'", "&#x27;")
  end

  @sql_injection_patterns [
    ~r/(\b(SELECT|INSERT|UPDATE|DELETE|DROP|UNION|ALTER)\b)/i,
    ~r/(--|#|\/\*)/,
    ~r/(\bOR\b\s+\d+\s*=\s*\d+)/i,
    ~r/('\s*(OR|AND)\s*')/i
  ]

  @spec detect_sql_injection(String.t()) :: boolean()
  def detect_sql_injection(input) when is_binary(input) do
    Enum.any?(@sql_injection_patterns, &Regex.match?(&1, input))
  end
end
```


## Data Models

### Configuration Structs

```elixir
# Circuit Breaker Config
%AuthPlatform.Resilience.CircuitBreaker.Config{
  failure_threshold: 5,
  success_threshold: 2,
  timeout_ms: 30_000,
  half_open_max_requests: 3
}

# Retry Config
%AuthPlatform.Resilience.Retry.Config{
  max_retries: 3,
  initial_delay_ms: 100,
  max_delay_ms: 10_000,
  multiplier: 2.0,
  jitter: true
}

# Rate Limiter Config
%AuthPlatform.Resilience.RateLimiter.Config{
  rate: 100,
  burst_size: 100
}

# Bulkhead Config
%AuthPlatform.Resilience.Bulkhead.Config{
  max_concurrent: 10,
  max_queue: 100,
  queue_timeout_ms: 5000
}
```

### Domain Primitive Structs

```elixir
%AuthPlatform.Domain.Email{value: "user@example.com"}
%AuthPlatform.Domain.UUID{value: "550e8400-e29b-41d4-a716-446655440000"}
%AuthPlatform.Domain.Money{amount: 1000, currency: :USD}
%AuthPlatform.Domain.PhoneNumber{value: "+5511999999999"}
%AuthPlatform.Domain.URL{value: "https://example.com", scheme: :https}
```

### Error Struct

```elixir
%AuthPlatform.Errors.AppError{
  code: :not_found,
  message: "User not found",
  details: %{user_id: "123"},
  correlation_id: "req-abc-123",
  cause: nil,
  retryable: false
}
```

### Telemetry Events

| Event | Measurements | Metadata |
|-------|--------------|----------|
| `[:auth_platform, :circuit_breaker, :state_change]` | `%{}` | `%{name, from_state, to_state}` |
| `[:auth_platform, :circuit_breaker, :reset]` | `%{}` | `%{name, from_state, to_state}` |
| `[:auth_platform, :retry, :attempt]` | `%{delay_ms}` | `%{attempt, reason}` |
| `[:auth_platform, :rate_limiter, :rejected]` | `%{}` | `%{name}` |
| `[:auth_platform, :bulkhead, :rejected]` | `%{}` | `%{name}` |


## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Functional Types Round-Trip

*For any* value `v`, creating a Result with `ok(v)` and then calling `unwrap!/1` SHALL return `v`. Similarly, *for any* error `e`, creating a Result with `error(e)` and calling `unwrap_error!/1` SHALL return `e`. The same applies to Option: `some(v) |> unwrap!/1` SHALL return `v`.

**Validates: Requirements 1.3, 1.4**

### Property 2: Functor Law Compliance

*For any* Result or Option value and identity function `id`, `map(value, id)` SHALL equal `value`. *For any* two functions `f` and `g`, `map(map(value, f), g)` SHALL equal `map(value, fn x -> g.(f.(x)) end)`.

**Validates: Requirements 1.5, 1.6**

### Property 3: Result/Option Constructor Consistency

*For any* value `v`, `ok(v) |> is_ok?()` SHALL return `true` and `ok(v) |> is_error?()` SHALL return `false`. *For any* error `e`, `error(e) |> is_error?()` SHALL return `true`. *For any* value `v`, `some(v) |> is_some?()` SHALL return `true` and `none() |> is_none?()` SHALL return `true`.

**Validates: Requirements 1.1, 1.2**

### Property 4: Error Code Mapping Consistency

*For any* AppError with a known error code, `http_status/1` SHALL return a valid HTTP status code (100-599) and `grpc_code/1` SHALL return a valid gRPC code (0-16). The mapping SHALL be deterministic: the same error code always maps to the same HTTP/gRPC codes.

**Validates: Requirements 2.4, 2.5**

### Property 5: Retryable Error Classification

*For any* AppError created with `rate_limited/0`, `timeout/1`, or `unavailable/1`, `is_retryable?/1` SHALL return `true`. *For any* AppError created with `not_found/1`, `validation/1`, `unauthorized/1`, or `internal/1`, `is_retryable?/1` SHALL return `false`.

**Validates: Requirements 2.6**

### Property 6: Validation Error Accumulation

*For any* list of validation results where N results are errors, `validate_all/1` SHALL return `{:errors, errors}` where `length(errors) >= N`. All individual errors SHALL be preserved in the accumulated result.

**Validates: Requirements 3.6**

### Property 7: Validator Composition

*For any* value `v` and list of validators, `all(validators).(v)` SHALL return `{:ok, v}` if and only if all validators return `{:ok, _}` for `v`. If any validator fails, the result SHALL be `{:errors, _}` containing all failure messages.

**Validates: Requirements 3.5**

### Property 8: String Validator Correctness

*For any* string `s`, `required().(s)` SHALL return `{:ok, s}` if `s` is non-empty, `{:errors, _}` otherwise. *For any* string `s` and integer `n`, `min_length(n).(s)` SHALL return `{:ok, s}` if `String.length(s) >= n`.

**Validates: Requirements 3.2**

### Property 9: Numeric Validator Correctness

*For any* number `n`, `positive().(n)` SHALL return `{:ok, n}` if `n > 0`. *For any* number `n` and range `[min, max]`, `in_range(min, max).(n)` SHALL return `{:ok, n}` if `min <= n <= max`.

**Validates: Requirements 3.3**

### Property 10: Domain Primitive Validation

*For any* valid email string (matching RFC 5322), `Email.new/1` SHALL return `{:ok, %Email{}}`. *For any* invalid email string, `Email.new/1` SHALL return `{:error, _}`. The same pattern applies to UUID, ULID, Money, PhoneNumber, and URL.

**Validates: Requirements 4.1, 4.2, 4.3, 4.4, 4.5, 4.6**

### Property 11: Domain Primitive Serialization Round-Trip

*For any* valid domain primitive (Email, UUID, Money, etc.), encoding to JSON and decoding back SHALL produce an equivalent value. `Jason.encode!(primitive) |> Jason.decode!()` SHALL contain the primitive's value.

**Validates: Requirements 4.8, 4.9**

### Property 12: Circuit Breaker State Machine

*For any* circuit breaker with `failure_threshold = N`, recording N consecutive failures in closed state SHALL transition to open state. *For any* circuit breaker in open state, after `timeout_ms` elapses, `allow_request?/1` SHALL return `true` and state SHALL be half_open. *For any* circuit breaker in half_open state with `success_threshold = M`, recording M consecutive successes SHALL transition to closed state.

**Validates: Requirements 5.2, 5.3, 5.4, 5.5, 5.6**

### Property 13: Retry Policy Exponential Backoff

*For any* retry config with `initial_delay_ms = d` and `multiplier = m`, `delay_for_attempt(config, n)` SHALL return approximately `d * m^n`, capped at `max_delay_ms`. Without jitter, the delay SHALL be exactly `min(d * m^n, max_delay_ms)`.

**Validates: Requirements 6.2, 6.4**

### Property 14: Retry Policy Execution

*For any* operation that fails with a retryable error, `execute/2` SHALL retry up to `max_retries` times. *For any* operation that fails with a non-retryable error, `execute/2` SHALL not retry and return the error immediately.

**Validates: Requirements 6.5, 6.6**

### Property 15: Rate Limiter Token Bucket

*For any* rate limiter with `burst_size = B`, initially `B` consecutive calls to `allow?/1` SHALL return `true`. After exhausting tokens, `allow?/1` SHALL return `false` until tokens are refilled based on `rate`.

**Validates: Requirements 7.1, 7.2, 7.3**

### Property 16: Bulkhead Isolation

*For any* bulkhead with `max_concurrent = C`, at most `C` operations SHALL execute concurrently. When `C` operations are active and queue is full (`max_queue` reached), new requests SHALL be rejected with `{:error, :rejected}`.

**Validates: Requirements 8.1, 8.3, 8.4**

### Property 17: JSON Codec Round-Trip

*For any* JSON-serializable Elixir term `t`, `JSON.decode(JSON.encode!(t))` SHALL return `{:ok, t'}` where `t'` is structurally equivalent to `t` (maps with string keys).

**Validates: Requirements 9.1**

### Property 18: Base64 Codec Round-Trip

*For any* binary `b`, `Base64.decode(Base64.encode(b))` SHALL return `{:ok, b}`. The same applies to URL-safe variant: `Base64.decode_url_safe(Base64.encode_url_safe(b))` SHALL return `{:ok, b}`.

**Validates: Requirements 9.2**

### Property 19: Constant Time Compare Correctness

*For any* two equal binaries `a` and `b`, `constant_time_compare(a, b)` SHALL return `true`. *For any* two different binaries, it SHALL return `false`. The comparison SHALL be timing-safe (constant time regardless of where differences occur).

**Validates: Requirements 11.1**

### Property 20: Token Generation Uniqueness

*For any* length `n`, `generate_token(n)` SHALL return a string of length `n`. *For any* two calls to `generate_token/1`, the results SHALL be different with overwhelming probability (collision resistance).

**Validates: Requirements 11.2**

### Property 21: HTML Sanitization

*For any* string containing `<`, `>`, `&`, `"`, or `'`, `sanitize_html/1` SHALL return a string where these characters are replaced with their HTML entity equivalents (`&lt;`, `&gt;`, `&amp;`, `&quot;`, `&#x27;`).

**Validates: Requirements 11.3**

### Property 22: Sensitive Data Masking

*For any* string `s` with length > 8 and `visible = 4`, `mask_sensitive(s, visible: 4)` SHALL return a string where the first 4 and last 4 characters are visible, and middle characters are replaced with `*`.

**Validates: Requirements 11.4**
