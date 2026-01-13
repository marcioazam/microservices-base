import { FastifyRequest, FastifyReply } from 'fastify';
import { imageService } from '@services/image';
import { storageService } from '@services/storage';
import { AppError } from '@domain/errors';
import { sendSuccess, sendError } from '@shared/utils/response';
import { validateResizeInput } from '@api/validators';
import { ResizeOptions, ProcessedImage, ImageMetadata } from '@domain/types';

interface BatchResizeBody {
  imageIds: string[];
  options: ResizeOptions;
}

interface BatchResult {
  id: string;
  success: boolean;
  metadata?: ImageMetadata;
  url?: string;
  error?: string;
}

const MAX_BATCH_SIZE = 10;
const CONCURRENCY_LIMIT = 5;

export class BatchController {
  async batchResize(
    request: FastifyRequest<{ Body: BatchResizeBody }>,
    reply: FastifyReply
  ): Promise<void> {
    const requestId = request.requestId;

    try {
      const { imageIds, options } = request.body;

      if (!imageIds || !Array.isArray(imageIds) || imageIds.length === 0) {
        throw AppError.invalidImage('imageIds array is required and must not be empty');
      }

      if (imageIds.length > MAX_BATCH_SIZE) {
        throw AppError.invalidImage(`Maximum batch size is ${MAX_BATCH_SIZE} images`);
      }

      const validatedOptions = validateResizeInput(options);

      // Process images with concurrency limit
      const results = await this.processWithConcurrency(
        imageIds,
        validatedOptions,
        CONCURRENCY_LIMIT
      );

      sendSuccess(reply, requestId, {
        total: imageIds.length,
        successful: results.filter((r) => r.success).length,
        failed: results.filter((r) => !r.success).length,
        results,
      });
    } catch (error) {
      if (error instanceof AppError) {
        sendError(reply, requestId, error);
      } else {
        sendError(reply, requestId, AppError.processingError('Batch processing failed'));
      }
    }
  }

  private async processWithConcurrency(
    imageIds: string[],
    options: ResizeOptions,
    concurrency: number
  ): Promise<BatchResult[]> {
    const results: BatchResult[] = [];
    const queue = [...imageIds];

    const processOne = async (imageId: string): Promise<BatchResult> => {
      try {
        const buffer = await storageService.download(imageId);
        const processed = await imageService.resize(buffer, options);

        const outputKey = await storageService.upload(processed.buffer, undefined, {
          contentType: `image/${processed.metadata.format}`,
        });

        const url = await storageService.getSignedUrl(outputKey);

        return {
          id: imageId,
          success: true,
          metadata: processed.metadata,
          url,
        };
      } catch (error) {
        return {
          id: imageId,
          success: false,
          error: error instanceof Error ? error.message : 'Unknown error',
        };
      }
    };

    // Process in batches with concurrency limit
    while (queue.length > 0) {
      const batch = queue.splice(0, concurrency);
      const batchResults = await Promise.all(batch.map(processOne));
      results.push(...batchResults);
    }

    return results;
  }
}

export const batchController = new BatchController();
