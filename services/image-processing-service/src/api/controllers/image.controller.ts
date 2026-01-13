import { FastifyRequest, FastifyReply } from 'fastify';
import { imageService } from '@services/image';
import { storageService } from '@services/storage';
import { AppError } from '@domain/errors';
import { sendSuccess, sendError, sendImage } from '@shared/utils/response';
import {
  validateResizeInput,
  validateConvertInput,
  validateAdjustInput,
  validateRotateInput,
  validateFlipInput,
  validateWatermarkInput,
  validateCompressInput,
} from '@api/validators';
import {
  ResizeOptions,
  ConvertOptions,
  AdjustOptions,
  RotateOptions,
  FlipOptions,
  WatermarkOptions,
  CompressOptions,
} from '@domain/types';

interface ProcessRequest {
  imageId?: string;
  async?: boolean;
}

export class ImageController {
  async resize(
    request: FastifyRequest<{ Body: ProcessRequest & ResizeOptions }>,
    reply: FastifyReply
  ): Promise<void> {
    const requestId = request.requestId;

    try {
      const { imageId, async: isAsync, ...options } = request.body;
      const validatedOptions = validateResizeInput(options);

      const buffer = await this.getImageBuffer(imageId, request);
      const result = await imageService.resize(buffer, validatedOptions);

      await this.sendProcessedImage(reply, requestId, result.buffer, result.metadata, request);
    } catch (error) {
      this.handleError(reply, requestId, error);
    }
  }

  async convert(
    request: FastifyRequest<{ Body: ProcessRequest & ConvertOptions }>,
    reply: FastifyReply
  ): Promise<void> {
    const requestId = request.requestId;

    try {
      const { imageId, async: isAsync, ...options } = request.body;
      const validatedOptions = validateConvertInput(options);

      const buffer = await this.getImageBuffer(imageId, request);
      const result = await imageService.convert(buffer, validatedOptions);

      await this.sendProcessedImage(reply, requestId, result.buffer, result.metadata, request);
    } catch (error) {
      this.handleError(reply, requestId, error);
    }
  }

  async adjust(
    request: FastifyRequest<{ Body: ProcessRequest & AdjustOptions }>,
    reply: FastifyReply
  ): Promise<void> {
    const requestId = request.requestId;

    try {
      const { imageId, async: isAsync, ...options } = request.body;
      const validatedOptions = validateAdjustInput(options);

      const buffer = await this.getImageBuffer(imageId, request);
      const result = await imageService.adjust(buffer, validatedOptions);

      await this.sendProcessedImage(reply, requestId, result.buffer, result.metadata, request);
    } catch (error) {
      this.handleError(reply, requestId, error);
    }
  }

  async rotate(
    request: FastifyRequest<{ Body: ProcessRequest & RotateOptions }>,
    reply: FastifyReply
  ): Promise<void> {
    const requestId = request.requestId;

    try {
      const { imageId, async: isAsync, ...options } = request.body;
      const validatedOptions = validateRotateInput(options);

      const buffer = await this.getImageBuffer(imageId, request);
      const result = await imageService.rotate(buffer, validatedOptions);

      await this.sendProcessedImage(reply, requestId, result.buffer, result.metadata, request);
    } catch (error) {
      this.handleError(reply, requestId, error);
    }
  }

  async flip(
    request: FastifyRequest<{ Body: ProcessRequest & FlipOptions }>,
    reply: FastifyReply
  ): Promise<void> {
    const requestId = request.requestId;

    try {
      const { imageId, async: isAsync, ...options } = request.body;
      const validatedOptions = validateFlipInput(options);

      const buffer = await this.getImageBuffer(imageId, request);
      const result = await imageService.flip(buffer, validatedOptions);

      await this.sendProcessedImage(reply, requestId, result.buffer, result.metadata, request);
    } catch (error) {
      this.handleError(reply, requestId, error);
    }
  }

  async watermark(
    request: FastifyRequest<{ Body: ProcessRequest & WatermarkOptions }>,
    reply: FastifyReply
  ): Promise<void> {
    const requestId = request.requestId;

    try {
      const { imageId, async: isAsync, ...options } = request.body;
      const validatedOptions = validateWatermarkInput(options);

      const buffer = await this.getImageBuffer(imageId, request);
      const result = await imageService.watermark(buffer, validatedOptions as WatermarkOptions);

      await this.sendProcessedImage(reply, requestId, result.buffer, result.metadata, request);
    } catch (error) {
      this.handleError(reply, requestId, error);
    }
  }

  async compress(
    request: FastifyRequest<{ Body: ProcessRequest & CompressOptions }>,
    reply: FastifyReply
  ): Promise<void> {
    const requestId = request.requestId;

    try {
      const { imageId, async: isAsync, ...options } = request.body;
      const validatedOptions = validateCompressInput(options);

      const buffer = await this.getImageBuffer(imageId, request);
      const result = await imageService.compress(buffer, validatedOptions);

      sendSuccess(reply, requestId, {
        metadata: result.metadata,
        compressionStats: result.compressionStats,
      });
    } catch (error) {
      this.handleError(reply, requestId, error);
    }
  }

  private async getImageBuffer(imageId: string | undefined, request: FastifyRequest): Promise<Buffer> {
    if (imageId) {
      return storageService.download(imageId);
    }

    // Try to get image from multipart
    const file = await request.file();
    if (file) {
      const chunks: Buffer[] = [];
      for await (const chunk of file.file) {
        chunks.push(chunk);
      }
      return Buffer.concat(chunks);
    }

    throw AppError.invalidImage('No image provided. Either provide imageId or upload a file.');
  }

  private async sendProcessedImage(
    reply: FastifyReply,
    requestId: string,
    buffer: Buffer,
    metadata: { width: number; height: number; format: string; size: number; hasAlpha: boolean },
    request: FastifyRequest
  ): Promise<void> {
    const acceptHeader = request.headers.accept;
    sendImage(reply, requestId, buffer, metadata, acceptHeader);
  }

  private handleError(reply: FastifyReply, requestId: string, error: unknown): void {
    if (error instanceof AppError) {
      sendError(reply, requestId, error);
    } else if (error instanceof Error && error.name === 'ZodError') {
      sendError(reply, requestId, new AppError('MISSING_REQUIRED_FIELD' as never, error.message));
    } else {
      sendError(reply, requestId, AppError.processingError('Image processing failed'));
    }
  }
}

export const imageController = new ImageController();
