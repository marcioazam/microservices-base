import { FastifyRequest, FastifyReply } from 'fastify';
import { jobService } from '@services/job';
import { storageService } from '@services/storage';
import { AppError } from '@domain/errors';
import { sendSuccess, sendError } from '@shared/utils/response';

interface JobParams {
  jobId: string;
}

export class JobController {
  async getStatus(
    request: FastifyRequest<{ Params: JobParams }>,
    reply: FastifyReply
  ): Promise<void> {
    const requestId = request.requestId;

    try {
      const { jobId } = request.params;
      const status = await jobService.getStatus(jobId);

      sendSuccess(reply, requestId, {
        jobId: status.id,
        status: status.status,
        progress: status.progress,
        createdAt: status.createdAt,
        updatedAt: status.updatedAt,
        completedAt: status.completedAt,
        error: status.error,
      });
    } catch (error) {
      if (error instanceof AppError) {
        sendError(reply, requestId, error);
      } else {
        sendError(reply, requestId, AppError.jobNotFound('Job not found'));
      }
    }
  }

  async getResult(
    request: FastifyRequest<{ Params: JobParams }>,
    reply: FastifyReply
  ): Promise<void> {
    const requestId = request.requestId;

    try {
      const { jobId } = request.params;
      const result = await jobService.getResult(jobId);

      if (!result) {
        const status = await jobService.getStatus(jobId);
        sendSuccess(reply, requestId, {
          jobId,
          status: status.status,
          message: 'Job not yet completed',
        });
        return;
      }

      const url = await storageService.getSignedUrl(result.outputKey);

      sendSuccess(reply, requestId, {
        jobId,
        status: 'completed',
        url,
        outputKey: result.outputKey,
      });
    } catch (error) {
      if (error instanceof AppError) {
        sendError(reply, requestId, error);
      } else {
        sendError(reply, requestId, AppError.jobNotFound('Job not found'));
      }
    }
  }

  async cancel(
    request: FastifyRequest<{ Params: JobParams }>,
    reply: FastifyReply
  ): Promise<void> {
    const requestId = request.requestId;

    try {
      const { jobId } = request.params;
      const cancelled = await jobService.cancel(jobId);

      if (cancelled) {
        sendSuccess(reply, requestId, {
          jobId,
          status: 'cancelled',
          message: 'Job cancelled successfully',
        });
      } else {
        sendSuccess(reply, requestId, {
          jobId,
          message: 'Job could not be cancelled (already completed or failed)',
        });
      }
    } catch (error) {
      if (error instanceof AppError) {
        sendError(reply, requestId, error);
      } else {
        sendError(reply, requestId, AppError.jobNotFound('Job not found'));
      }
    }
  }
}

export const jobController = new JobController();
