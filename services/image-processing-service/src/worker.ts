import { startWorker } from '@infrastructure/worker';
import { initTracing } from '@infrastructure/observability';
import { logger } from '@infrastructure/logging';
import { config } from '@config/index';

async function main(): Promise<void> {
  if (config.tracing.enabled) {
    initTracing();
  }

  logger.info('Starting Image Processing Worker...');
  await startWorker();
  logger.info('Image Processing Worker started successfully');
}

main().catch((error) => {
  logger.fatal('Failed to start worker', error as Error);
  process.exit(1);
});
