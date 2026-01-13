import { createServer, startServer } from '@infrastructure/server';
import { registerRoutes } from '@api/routes';
import { getMetrics, getContentType } from '@infrastructure/observability';
import { initTracing, shutdownTracing } from '@infrastructure/observability';
import { logger } from '@infrastructure/logging';
import { config } from '@config/index';

async function main(): Promise<void> {
  if (config.tracing.enabled) {
    initTracing();
  }

  const server = await createServer();

  // Metrics endpoint
  server.get('/metrics', async (_req, reply) => {
    const metrics = await getMetrics();
    reply.header('Content-Type', getContentType()).send(metrics);
  });

  await registerRoutes(server);
  await startServer(server);

  logger.info('Image Processing Service started successfully');

  const shutdown = async (signal: string): Promise<void> => {
    logger.info(`Received ${signal}, shutting down...`);
    await server.close();
    await shutdownTracing();
    logger.info('Server closed');
    process.exit(0);
  };

  process.on('SIGTERM', () => shutdown('SIGTERM'));
  process.on('SIGINT', () => shutdown('SIGINT'));
}

main().catch((error) => {
  logger.fatal('Failed to start server', error as Error);
  process.exit(1);
});
