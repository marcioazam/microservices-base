# Session/Identity Core - Elixir Multi-Stage Build
# Phoenix/OTP for fault-tolerant session management
# Build context: repository root (../../)

# ============================================
# Stage 1: Build Environment
# ============================================
FROM hexpm/elixir:1.16.0-erlang-26.2.1-alpine-3.19.0 AS builder

RUN apk add --no-cache \
    build-base \
    git \
    nodejs \
    npm \
    protobuf-dev

WORKDIR /build

# Install hex and rebar
RUN mix local.hex --force && \
    mix local.rebar --force

# Set build environment
ENV MIX_ENV=prod

# Copy dependency files
COPY services/session-identity/mix.exs services/session-identity/mix.lock ./
COPY api/proto/ ../proto/

# Install dependencies
RUN mix deps.get --only prod && \
    mix deps.compile

# Copy application code
COPY services/session-identity/config ./config
COPY services/session-identity/lib ./lib
COPY services/session-identity/priv ./priv

# Compile and build release
RUN mix compile && \
    mix release

# ============================================
# Stage 2: Runtime Environment
# ============================================
FROM alpine:3.19 AS runtime

RUN apk add --no-cache \
    libstdc++ \
    openssl \
    ncurses-libs \
    libgcc \
    ca-certificates \
    && addgroup -g 1000 elixir \
    && adduser -u 1000 -G elixir -s /bin/sh -D elixir

LABEL org.opencontainers.image.title="Session Identity Core"
LABEL org.opencontainers.image.description="Elixir/OTP session management with WebSocket support"

WORKDIR /app

COPY --from=builder --chown=elixir:elixir \
    /build/_build/prod/rel/session_identity_core ./

# Expose HTTP, gRPC, and EPMD ports
EXPOSE 4000 8082 4369

ENV HOME=/app
ENV RELEASE_COOKIE=session_identity_core_cookie
ENV RELEASE_NODE=session_identity_core@127.0.0.1

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:4000/health || exit 1

USER elixir:elixir
ENTRYPOINT ["/app/bin/session_identity_core"]
CMD ["start"]
