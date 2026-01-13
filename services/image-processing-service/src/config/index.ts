import { z } from 'zod';

const configSchema = z.object({
  server: z.object({
    port: z.number().default(3000),
    host: z.string().default('0.0.0.0'),
  }),
  redis: z.object({
    host: z.string().default('localhost'),
    port: z.number().default(6379),
    password: z.string().optional(),
    db: z.number().default(0),
  }),
  s3: z.object({
    endpoint: z.string().optional(),
    region: z.string().default('us-east-1'),
    bucket: z.string(),
    accessKeyId: z.string(),
    secretAccessKey: z.string(),
    forcePathStyle: z.boolean().default(true),
  }),
  auth: z.object({
    jwtSecret: z.string(),
    jwtIssuer: z.string().default('auth-platform'),
    jwtAudience: z.string().default('image-processing-service'),
  }),
  cache: z.object({
    ttlSeconds: z.number().default(3600),
    prefix: z.string().default('img:'),
    endpoint: z.string().optional(),
  }),
  storage: z.object({
    tempTtlSeconds: z.number().default(86400),
    maxFileSizeBytes: z.number().default(52428800), // 50MB
  }),
  queue: z.object({
    name: z.string().default('image-processing'),
    concurrency: z.number().default(5),
  }),
  rateLimit: z.object({
    max: z.number().default(100),
    timeWindowMs: z.number().default(60000),
  }),
  logging: z.object({
    level: z.enum(['fatal', 'error', 'warn', 'info', 'debug', 'trace']).default('info'),
    endpoint: z.string().optional(),
  }),
  tracing: z.object({
    endpoint: z.string().optional(),
    enabled: z.boolean().default(true),
  }),
});

export type Config = z.infer<typeof configSchema>;

function loadConfig(): Config {
  const rawConfig = {
    server: {
      port: parseInt(process.env.PORT || '3000', 10),
      host: process.env.HOST || '0.0.0.0',
    },
    redis: {
      host: process.env.REDIS_HOST || 'localhost',
      port: parseInt(process.env.REDIS_PORT || '6379', 10),
      password: process.env.REDIS_PASSWORD,
      db: parseInt(process.env.REDIS_DB || '0', 10),
    },
    s3: {
      endpoint: process.env.S3_ENDPOINT,
      region: process.env.S3_REGION || 'us-east-1',
      bucket: process.env.S3_BUCKET || 'image-processing',
      accessKeyId: process.env.S3_ACCESS_KEY_ID || '',
      secretAccessKey: process.env.S3_SECRET_ACCESS_KEY || '',
      forcePathStyle: process.env.S3_FORCE_PATH_STYLE === 'true',
    },
    auth: {
      jwtSecret: process.env.JWT_SECRET || 'dev-secret-change-in-production',
      jwtIssuer: process.env.JWT_ISSUER || 'auth-platform',
      jwtAudience: process.env.JWT_AUDIENCE || 'image-processing-service',
    },
    cache: {
      ttlSeconds: parseInt(process.env.CACHE_TTL_SECONDS || '3600', 10),
      prefix: process.env.CACHE_PREFIX || 'img:',
      endpoint: process.env.CACHE_SERVICE_ENDPOINT,
    },
    storage: {
      tempTtlSeconds: parseInt(process.env.STORAGE_TEMP_TTL_SECONDS || '86400', 10),
      maxFileSizeBytes: parseInt(process.env.MAX_FILE_SIZE_BYTES || '52428800', 10),
    },
    queue: {
      name: process.env.QUEUE_NAME || 'image-processing',
      concurrency: parseInt(process.env.QUEUE_CONCURRENCY || '5', 10),
    },
    rateLimit: {
      max: parseInt(process.env.RATE_LIMIT_MAX || '100', 10),
      timeWindowMs: parseInt(process.env.RATE_LIMIT_WINDOW_MS || '60000', 10),
    },
    logging: {
      level: (process.env.LOG_LEVEL || 'info') as Config['logging']['level'],
      endpoint: process.env.LOGGING_SERVICE_ENDPOINT,
    },
    tracing: {
      endpoint: process.env.TRACING_ENDPOINT || 'http://localhost:4318/v1/traces',
      enabled: process.env.TRACING_ENABLED !== 'false',
    },
  };

  return configSchema.parse(rawConfig);
}

export const config = loadConfig();
