# MFA Service - Elixir Multi-Stage Build
# TOTP, WebAuthn, and Push authentication

# ============================================
# Stage 1: Build Environment
# ============================================
FROM hexpm/elixir:1.16.0-erlang-26.2.1-alpine-3.19.0 AS builder

RUN apk add --no-cache \
    build-base \
    git \
    protobuf-dev

WORKDIR /build

RUN mix local.hex --force && \
    mix local.rebar --force

ENV MIX_ENV=prod

COPY auth/mfa-service/mix.exs auth/mfa-service/mix.lock ./
COPY proto/ ../proto/

RUN mix deps.get --only prod && \
    mix deps.compile

COPY auth/mfa-service/config ./config
COPY auth/mfa-service/lib ./lib
COPY auth/mfa-service/priv ./priv

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

LABEL org.opencontainers.image.title="MFA Service"
LABEL org.opencontainers.image.description="Multi-factor authentication: TOTP, WebAuthn, Push"

WORKDIR /app

COPY --from=builder --chown=elixir:elixir \
    /build/_build/prod/rel/mfa_service ./

EXPOSE 8084 9094

ENV HOME=/app
ENV RELEASE_COOKIE=mfa_service_cookie

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8084/health || exit 1

USER elixir:elixir
ENTRYPOINT ["/app/bin/mfa_service"]
CMD ["start"]
