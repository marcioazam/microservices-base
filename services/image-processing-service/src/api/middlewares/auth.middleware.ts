import { FastifyRequest, FastifyReply, HookHandlerDoneFunction } from 'fastify';
import jwt from 'jsonwebtoken';
import { config } from '@config/index';
import { AppError, ErrorCode } from '@domain/errors';

export interface JwtPayload {
  sub: string;
  userId: string;
  email?: string;
  permissions?: string[];
  iat: number;
  exp: number;
}

declare module 'fastify' {
  interface FastifyRequest {
    user?: JwtPayload;
  }
}

export function authMiddleware(
  request: FastifyRequest,
  reply: FastifyReply,
  done: HookHandlerDoneFunction
): void {
  const authHeader = request.headers.authorization;

  if (!authHeader) {
    const error = new AppError(ErrorCode.MISSING_TOKEN, 'Authorization header is required');
    reply.status(error.httpStatus).send({
      success: false,
      requestId: request.requestId,
      error: error.toJSON(),
    });
    return;
  }

  const [scheme, token] = authHeader.split(' ');

  if (scheme !== 'Bearer' || !token) {
    const error = new AppError(ErrorCode.INVALID_TOKEN, 'Invalid authorization format. Use: Bearer <token>');
    reply.status(error.httpStatus).send({
      success: false,
      requestId: request.requestId,
      error: error.toJSON(),
    });
    return;
  }

  try {
    const decoded = jwt.verify(token, config.auth.jwtSecret, {
      issuer: config.auth.jwtIssuer,
      audience: config.auth.jwtAudience,
    }) as JwtPayload;

    request.user = decoded;
    done();
  } catch (error) {
    if (error instanceof jwt.TokenExpiredError) {
      const appError = new AppError(ErrorCode.EXPIRED_TOKEN, 'Token has expired');
      reply.status(appError.httpStatus).send({
        success: false,
        requestId: request.requestId,
        error: appError.toJSON(),
      });
      return;
    }

    const appError = new AppError(ErrorCode.INVALID_TOKEN, 'Invalid token');
    reply.status(appError.httpStatus).send({
      success: false,
      requestId: request.requestId,
      error: appError.toJSON(),
    });
  }
}

export function requirePermission(permission: string) {
  return (
    request: FastifyRequest,
    reply: FastifyReply,
    done: HookHandlerDoneFunction
  ): void => {
    if (!request.user) {
      const error = new AppError(ErrorCode.MISSING_TOKEN, 'Authentication required');
      reply.status(error.httpStatus).send({
        success: false,
        requestId: request.requestId,
        error: error.toJSON(),
      });
      return;
    }

    const userPermissions = request.user.permissions || [];

    if (!userPermissions.includes(permission) && !userPermissions.includes('*')) {
      const error = new AppError(
        ErrorCode.INSUFFICIENT_PERMISSIONS,
        `Permission '${permission}' is required`
      );
      reply.status(error.httpStatus).send({
        success: false,
        requestId: request.requestId,
        error: error.toJSON(),
      });
      return;
    }

    done();
  };
}

export function optionalAuth(
  request: FastifyRequest,
  _reply: FastifyReply,
  done: HookHandlerDoneFunction
): void {
  const authHeader = request.headers.authorization;

  if (!authHeader) {
    done();
    return;
  }

  const [scheme, token] = authHeader.split(' ');

  if (scheme !== 'Bearer' || !token) {
    done();
    return;
  }

  try {
    const decoded = jwt.verify(token, config.auth.jwtSecret, {
      issuer: config.auth.jwtIssuer,
      audience: config.auth.jwtAudience,
    }) as JwtPayload;

    request.user = decoded;
  } catch {
    // Ignore invalid tokens for optional auth
  }

  done();
}
