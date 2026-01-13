import Fastify, { FastifyInstance, FastifyRequest, FastifyReply } from 'fastify';
import fastifyMultipart from '@fastify/multipart';
import fastifyCors from '@fastify/cors';
import fastifyHelmet from '@fastify/helmet';
import { config } from '@config/index';
import { AppError } from '@domain/errors';
import { v4 as uuidv4 } from 'uuid';
import { logger } from './logging';
import { getCurrentTraceContext } from './observability';
import { recordHttpRequest } from './observability/metrics';
import { registerHealthEndpoints } from './health';

declare module 'fastify' {
  interface FastifyRequest {
    requestId: string;
    startTime: number;
  }
}

export async function createServer(): Promise<FastifyInstance> {
  const server = Fastify({
    logger: false, // Using platform logging client instead
    genReqId: () => uuidv4(),
  });

  await server.register(fastifyCors, { origin: true, credentials: true });
  await server.register(fastifyHelmet, { contentSecurityPolicy: false });
  await server.register(fastifyMultipart, {
    limits: { fileSize: config.storage.maxFileSizeBytes, files: 10 },
  });

  server.addHook('onRequest', async (request: FastifyRequest) => {
    request.requestId = request.id as string;
    request.startTime = Date.now();
  });

  server.addHook('onResponse', async (request: FastifyRequest, reply: FastifyReply) => {
    const duration = Date.now() - request.startTime;
    const traceContext = getCurrentTraceContext();
    
    logger.info('Request completed', {
      requestId: request.requestId,
      method: request.method,
      path: request.url,
      statusCode: reply.statusCode,
      duration,
      traceId: traceContext?.traceId,
      spanId: traceContext?.spanId,
    });

    recordHttpRequest(request.method, request.url, reply.statusCode, duration);
  });

  server.setErrorHandler(async (error: Error, request: FastifyRequest, reply: FastifyReply) => {
    const requestId = request.requestId || uuidv4();
    const traceContext = getCurrentTraceContext();

    if (error instanceof AppError) {
      logger.warn('Application error', {
        requestId,
        traceId: traceContext?.traceId,
        error: error.toJSON(),
      });

      return reply
        .status(error.httpStatus)
        .header('X-Request-Id', requestId)
        .send({ success: false, requestId, error: error.toJSON() });
    }

    if ('validation' in error) {
      return reply
        .status(400)
        .header('X-Request-Id', requestId)
        .send({
          success: false,
          requestId,
          error: { code: 'VALIDATION_ERROR', message: error.message },
        });
    }

    logger.error('Unexpected error', error, {
      requestId,
      traceId: traceContext?.traceId,
      url: request.url,
      method: request.method,
    });

    return reply
      .status(500)
      .header('X-Request-Id', requestId)
      .send({
        success: false,
        requestId,
        error: { code: 'INTERNAL_ERROR', message: 'An unexpected error occurred' },
      });
  });

  registerHealthEndpoints(server);

  return server;
}

export async function startServer(server: FastifyInstance): Promise<void> {
  try {
    await server.listen({ port: config.server.port, host: config.server.host });
    logger.info(`Server listening on ${config.server.host}:${config.server.port}`);
  } catch (error) {
    logger.fatal('Failed to start server', error as Error);
    process.exit(1);
  }
}
