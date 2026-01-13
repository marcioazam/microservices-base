import { Worker, Job } from 'bullmq';
import { Redis } from 'ioredis';
import { config } from '@config/index';
import { imageService } from '@services/image';
import { storageService } from '@services/storage';
import { jobService, JobData } from '@services/job';
import { logger } from './logging';
import { recordImageProcessing } from './observability/metrics';

export async function createWorker(): Promise<Worker<JobData>> {
  const redis = new Redis({
    host: config.redis.host,
    port: config.redis.port,
    password: config.redis.password,
    db: config.redis.db,
    maxRetriesPerRequest: null,
  });

  const worker = new Worker<JobData>(
    config.queue.name,
    async (job: Job<JobData>) => {
      const { id, operation, inputKey } = job.data;
      const startTime = Date.now();

      logger.info('Processing job', { jobId: id, operation: operation.type });

      try {
        await jobService.updateJobStatus(id, 'processing', { progress: 10 });
        const inputBuffer = await storageService.download(inputKey);
        await jobService.updateJobStatus(id, 'processing', { progress: 30 });

        const result = await processOperation(inputBuffer, operation);
        await jobService.updateJobStatus(id, 'processing', { progress: 70 });

        const outputKey = await storageService.upload(result.buffer, undefined, {
          contentType: `image/${result.metadata.format}`,
          metadata: { width: String(result.metadata.width), height: String(result.metadata.height), jobId: id },
        });

        await jobService.updateJobStatus(id, 'completed', { progress: 100, outputKey });
        
        const duration = Date.now() - startTime;
        recordImageProcessing(operation.type, true, duration);
        logger.info('Job completed', { jobId: id, outputKey, duration });

        return { outputKey };
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : 'Unknown error';
        const duration = Date.now() - startTime;
        
        recordImageProcessing(operation.type, false, duration);
        logger.error('Job failed', error as Error, { jobId: id });
        await jobService.updateJobStatus(id, 'failed', { error: errorMessage });
        throw error;
      }
    },
    { connection: redis, concurrency: config.queue.concurrency }
  );

  worker.on('completed', (job) => logger.info('Job completed', { jobId: job.id }));
  worker.on('failed', (job, error) => logger.error('Job failed', error, { jobId: job?.id }));
  worker.on('error', (error) => logger.error('Worker error', error));

  return worker;
}

async function processOperation(input: Buffer, operation: JobData['operation']) {
  switch (operation.type) {
    case 'resize': return imageService.resize(input, operation.options);
    case 'convert': return imageService.convert(input, operation.options);
    case 'adjust': return imageService.adjust(input, operation.options);
    case 'rotate': return imageService.rotate(input, operation.options);
    case 'flip': return imageService.flip(input, operation.options);
    case 'watermark': return imageService.watermark(input, operation.options);
    case 'compress': return imageService.compress(input, operation.options);
    default: throw new Error(`Unknown operation: ${(operation as { type: string }).type}`);
  }
}

export async function startWorker(): Promise<void> {
  const worker = await createWorker();
  logger.info('Worker started');

  const shutdown = async () => {
    logger.info('Shutting down worker...');
    await worker.close();
    process.exit(0);
  };

  process.on('SIGTERM', shutdown);
  process.on('SIGINT', shutdown);
}
