/**
 * Auth Platform SDK Error Classes
 */

export class AuthPlatformError extends Error {
  public readonly code: string;
  public readonly statusCode?: number;

  constructor(message: string, code: string, statusCode?: number) {
    super(message);
    this.name = 'AuthPlatformError';
    this.code = code;
    this.statusCode = statusCode;
  }
}

export class TokenExpiredError extends AuthPlatformError {
  constructor(message = 'Access token has expired') {
    super(message, 'TOKEN_EXPIRED', 401);
    this.name = 'TokenExpiredError';
  }
}

export class TokenRefreshError extends AuthPlatformError {
  constructor(message = 'Failed to refresh token') {
    super(message, 'TOKEN_REFRESH_FAILED', 401);
    this.name = 'TokenRefreshError';
  }
}

export class NetworkError extends AuthPlatformError {
  constructor(message = 'Network request failed') {
    super(message, 'NETWORK_ERROR');
    this.name = 'NetworkError';
  }
}

export class InvalidConfigError extends AuthPlatformError {
  constructor(message: string) {
    super(message, 'INVALID_CONFIG');
    this.name = 'InvalidConfigError';
  }
}

export class RateLimitError extends AuthPlatformError {
  public readonly retryAfter?: number;

  constructor(message = 'Rate limit exceeded', retryAfter?: number) {
    super(message, 'RATE_LIMITED', 429);
    this.name = 'RateLimitError';
    this.retryAfter = retryAfter;
  }
}

export class PasskeyError extends AuthPlatformError {
  constructor(message: string, code: string) {
    super(message, code);
    this.name = 'PasskeyError';
  }
}

export class PasskeyNotSupportedError extends PasskeyError {
  constructor() {
    super('Passkeys are not supported on this device/browser', 'PASSKEY_NOT_SUPPORTED');
    this.name = 'PasskeyNotSupportedError';
  }
}

export class PasskeyCancelledError extends PasskeyError {
  constructor() {
    super('Passkey operation was cancelled by user', 'PASSKEY_CANCELLED');
    this.name = 'PasskeyCancelledError';
  }
}

export class CaepError extends AuthPlatformError {
  constructor(message: string, code: string) {
    super(message, code);
    this.name = 'CaepError';
  }
}
