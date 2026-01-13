import { FastifyRequest, FastifyReply, HookHandlerDoneFunction } from 'fastify';
import { Redis } from 'ioredis';
import { config } from '@config/index';
import { AppError, ErrorCode } from '@domain/errors';

let redis: Redis | null = null;

function getRedis(): Redis {
  if (!redis) {
    redis = new Redis({
      host: config.redis.host,
      port: config.redis.port,
      password: config.redis.password,
      db: config.redis.db,
    });
  }
  return redis;
}

interface RateLimitOptions {
  max?: number;
  timeWindowMs?: number;
  keyPrefix?: string;
}

export function rateLimitMiddleware(options: RateLimitOptions = {}) {
  const max = options.max || config.rateLimit.max;
  const timeWindowMs = options.timeWindowMs || config.rateLimit.timeWindowMs;
  const keyPrefix = options.keyPrefix || 'ratelimit:';

  return async (
    request: FastifyRequest,
    reply: FastifyReply,
    done: HookHandlerDoneFunction
  ): Promise<void> => {
    const client = getRedis();

    // Use user ID if authenticated, otherwise use IP
    const identifier = request.user?.userId || request.ip || 'anonymous';
    const key = `${keyPrefix}${identifier}`;

    try {
      const current = await client.incr(key);

      if (current === 1) {
        // First request, set expiration
        await client.pexpire(key, timeWindowMs);
      }

      // Get TTL for retry-after header
      const ttl = await client.pttl(key);
      const retryAfterSeconds = Math.ceil(ttl / 1000);

      // Set rate limit headers
      reply.header('X-RateLimit-Limit', max.toString());
      reply.header('X-RateLimit-Remaining', Math.max(0, max - current).toString());
      reply.header('X-RateLimit-Reset', retryAfterSeconds.toString());

      if (current > max) {
        const error = new AppError(
          ErrorCode.RATE_LIMIT_EXCEEDED,
          'Rate limit exceeded. Please try again later.',
          { retryAfter: retryAfterSeconds }
        );

        reply
          .status(error.httpStatus)
          .header('Retry-After', retryAfterSeconds.toString())
          .send({
            success: false,
            requestId: request.requestId,
            error: error.toJSON(),
          });
        return;
      }

      done();
    } catch (error) {
      // If Redis fails, allow the request but log the error
      request.log.error({ error }, 'Rate limit check failed');
      done();
    }
  };
}

export async function closeRateLimitRedis(): Promise<void> {
  if (redis) {
    await redis.quit();
    redis = null;
  }
}
