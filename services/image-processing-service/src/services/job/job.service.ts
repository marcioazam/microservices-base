import { Queue, Job } from 'bullmq';
import { Redis } from 'ioredis';
import { v4 as uuidv4 } from 'uuid';
import { config } from '@config/index';
import { AppError } from '@domain/errors';
import { ImageOperation, JobStatus } from '@domain/types';

export interface JobData {
  id: string;
  userId: string;
  operation: ImageOperation;
  inputKey: string;
  createdAt: string;
}

export interface JobResult {
  id: string;
  status: JobStatus;
  progress: number;
  outputKey?: string;
  error?: string;
  createdAt: string;
  updatedAt: string;
  completedAt?: string;
}

export class JobService {
  private queue: Queue<JobData>;
  private redis: Redis;

  constructor() {
    this.redis = new Redis({
      host: config.redis.host,
      port: config.redis.port,
      password: config.redis.password,
      db: config.redis.db,
      maxRetriesPerRequest: null,
    });

    this.queue = new Queue<JobData>(config.queue.name, {
      connection: this.redis,
      defaultJobOptions: {
        attempts: 3,
        backoff: {
          type: 'exponential',
          delay: 1000,
        },
        removeOnComplete: false,
        removeOnFail: false,
      },
    });
  }

  async enqueue(
    userId: string,
    operation: ImageOperation,
    inputKey: string
  ): Promise<string> {
    const jobId = uuidv4();
    const jobData: JobData = {
      id: jobId,
      userId,
      operation,
      inputKey,
      createdAt: new Date().toISOString(),
    };

    try {
      await this.queue.add(operation.type, jobData, {
        jobId,
      });

      // Store initial job status
      await this.redis.hset(`job:${jobId}`, {
        id: jobId,
        userId,
        status: 'pending',
        progress: 0,
        inputKey,
        operationType: operation.type,
        createdAt: jobData.createdAt,
        updatedAt: jobData.createdAt,
      });

      return jobId;
    } catch (error) {
      throw new AppError(
        'QUEUE_ERROR' as never,
        'Failed to enqueue job',
        { error: error instanceof Error ? error.message : 'Unknown error' }
      );
    }
  }

  async getStatus(jobId: string): Promise<JobResult> {
    const jobData = await this.redis.hgetall(`job:${jobId}`);

    if (!jobData || Object.keys(jobData).length === 0) {
      throw AppError.jobNotFound(`Job not found: ${jobId}`);
    }

    return {
      id: jobData.id,
      status: jobData.status as JobStatus,
      progress: parseInt(jobData.progress || '0', 10),
      outputKey: jobData.outputKey,
      error: jobData.error,
      createdAt: jobData.createdAt,
      updatedAt: jobData.updatedAt,
      completedAt: jobData.completedAt,
    };
  }

  async getResult(jobId: string): Promise<{ outputKey: string } | null> {
    const status = await this.getStatus(jobId);

    if (status.status !== 'completed') {
      return null;
    }

    if (!status.outputKey) {
      throw AppError.processingError('Job completed but no output found');
    }

    return { outputKey: status.outputKey };
  }

  async cancel(jobId: string): Promise<boolean> {
    try {
      const job = await this.queue.getJob(jobId);

      if (!job) {
        throw AppError.jobNotFound(`Job not found: ${jobId}`);
      }

      const state = await job.getState();

      if (state === 'completed' || state === 'failed') {
        return false;
      }

      await job.remove();
      await this.updateJobStatus(jobId, 'cancelled');

      return true;
    } catch (error) {
      if (error instanceof AppError) {
        throw error;
      }
      throw new AppError(
        'QUEUE_ERROR' as never,
        'Failed to cancel job',
        { error: error instanceof Error ? error.message : 'Unknown error' }
      );
    }
  }

  async updateJobStatus(
    jobId: string,
    status: JobStatus,
    updates: Partial<{ progress: number; outputKey: string; error: string }> = {}
  ): Promise<void> {
    const now = new Date().toISOString();
    const updateData: Record<string, string> = {
      status,
      updatedAt: now,
    };

    if (updates.progress !== undefined) {
      updateData.progress = updates.progress.toString();
    }

    if (updates.outputKey) {
      updateData.outputKey = updates.outputKey;
    }

    if (updates.error) {
      updateData.error = updates.error;
    }

    if (status === 'completed' || status === 'failed') {
      updateData.completedAt = now;
    }

    await this.redis.hset(`job:${jobId}`, updateData);
  }

  async getQueue(): Promise<Queue<JobData>> {
    return this.queue;
  }

  async close(): Promise<void> {
    await this.queue.close();
    await this.redis.quit();
  }
}

export const jobService = new JobService();
