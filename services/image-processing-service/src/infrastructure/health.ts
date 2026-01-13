import { FastifyInstance, FastifyRequest, FastifyReply } from 'fastify';
import { Redis } from 'ioredis';
import { S3Client, HeadBucketCommand } from '@aws-sdk/client-s3';
import { config } from '@config/index';
import { logger } from './logging';
import { cacheClient } from './cache';

interface HealthStatus {
  status: 'healthy' | 'unhealthy' | 'degraded';
  timestamp: string;
  version: string;
  uptime: number;
  checks: Record<string, ComponentHealth>;
}

interface ComponentHealth {
  status: 'healthy' | 'unhealthy';
  latencyMs: number;
  error?: string;
}

const startTime = Date.now();

async function checkRedis(): Promise<ComponentHealth> {
  const start = Date.now();
  const redis = new Redis({
    host: config.redis.host,
    port: config.redis.port,
    password: config.redis.password,
    db: config.redis.db,
    connectTimeout: 5000,
  });

  try {
    await redis.ping();
    const latencyMs = Date.now() - start;
    await redis.quit();
    return { status: 'healthy', latencyMs };
  } catch (error) {
    const latencyMs = Date.now() - start;
    await redis.quit().catch(() => {});
    return {
      status: 'unhealthy',
      latencyMs,
      error: error instanceof Error ? error.message : 'Unknown error',
    };
  }
}

async function checkS3(): Promise<ComponentHealth> {
  const start = Date.now();
  const client = new S3Client({
    region: config.s3.region,
    endpoint: config.s3.endpoint,
    forcePathStyle: config.s3.forcePathStyle,
    credentials: {
      accessKeyId: config.s3.accessKeyId,
      secretAccessKey: config.s3.secretAccessKey,
    },
  });

  try {
    await client.send(new HeadBucketCommand({ Bucket: config.s3.bucket }));
    const latencyMs = Date.now() - start;
    return { status: 'healthy', latencyMs };
  } catch (error) {
    const latencyMs = Date.now() - start;
    return {
      status: 'unhealthy',
      latencyMs,
      error: error instanceof Error ? error.message : 'Unknown error',
    };
  }
}

async function checkCache(): Promise<ComponentHealth> {
  const result = await cacheClient.healthCheck();
  return {
    status: result.healthy ? 'healthy' : 'unhealthy',
    latencyMs: result.latencyMs,
  };
}

function determineOverallStatus(checks: Record<string, ComponentHealth>): 'healthy' | 'unhealthy' | 'degraded' {
  const statuses = Object.values(checks).map(c => c.status);
  const allHealthy = statuses.every(s => s === 'healthy');
  const anyHealthy = statuses.some(s => s === 'healthy');

  if (allHealthy) return 'healthy';
  if (anyHealthy) return 'degraded';
  return 'unhealthy';
}

export function registerHealthEndpoints(server: FastifyInstance): void {
  server.get('/health/live', async (_request: FastifyRequest, reply: FastifyReply) => {
    reply.send({ status: 'healthy', timestamp: new Date().toISOString() });
  });

  server.get('/health/ready', async (_request: FastifyRequest, reply: FastifyReply) => {
    const [redisHealth, s3Health, cacheHealth] = await Promise.all([
      checkRedis(),
      checkS3(),
      checkCache(),
    ]);

    const checks = { redis: redisHealth, s3: s3Health, cache: cacheHealth };
    const overallStatus = determineOverallStatus(checks);

    const health: HealthStatus = {
      status: overallStatus,
      timestamp: new Date().toISOString(),
      version: process.env.npm_package_version || '2.0.0',
      uptime: Math.floor((Date.now() - startTime) / 1000),
      checks,
    };

    if (overallStatus !== 'healthy') {
      logger.warn('Health check degraded', { checks, status: overallStatus });
    }

    reply.status(overallStatus === 'unhealthy' ? 503 : 200).send(health);
  });

  server.get('/health', async (_request: FastifyRequest, reply: FastifyReply) => {
    const [redisHealth, s3Health, cacheHealth] = await Promise.all([
      checkRedis(),
      checkS3(),
      checkCache(),
    ]);

    const checks = { redis: redisHealth, s3: s3Health, cache: cacheHealth };
    const health: HealthStatus = {
      status: determineOverallStatus(checks),
      timestamp: new Date().toISOString(),
      version: process.env.npm_package_version || '2.0.0',
      uptime: Math.floor((Date.now() - startTime) / 1000),
      checks,
    };

    reply.send(health);
  });
}
