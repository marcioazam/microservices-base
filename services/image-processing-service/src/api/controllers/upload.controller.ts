import { FastifyRequest, FastifyReply } from 'fastify';
import { MultipartFile } from '@fastify/multipart';
import sharp from 'sharp';
import { storageService } from '@services/storage';
import { imageService } from '@services/image';
import { AppError } from '@domain/errors';
import { sendSuccess, sendError } from '@shared/utils/response';
import { config } from '@config/index';

interface UploadFromUrlBody {
  url: string;
}

const ALLOWED_MIME_TYPES = [
  'image/jpeg',
  'image/png',
  'image/gif',
  'image/webp',
  'image/tiff',
];

export class UploadController {
  async uploadFile(request: FastifyRequest, reply: FastifyReply): Promise<void> {
    const requestId = request.requestId;

    try {
      const file = await request.file();

      if (!file) {
        throw AppError.invalidImage('No file provided');
      }

      const buffer = await this.processMultipartFile(file);
      await this.validateImage(buffer);

      const metadata = await imageService.getMetadata(buffer);
      const key = await storageService.upload(buffer, undefined, {
        contentType: `image/${metadata.format}`,
        metadata: {
          width: metadata.width.toString(),
          height: metadata.height.toString(),
        },
      });

      const url = await storageService.getSignedUrl(key);

      sendSuccess(reply, requestId, {
        id: key,
        url,
        metadata,
      }, 201);
    } catch (error) {
      if (error instanceof AppError) {
        sendError(reply, requestId, error);
      } else {
        sendError(reply, requestId, AppError.processingError('Failed to upload file'));
      }
    }
  }

  async uploadFromUrl(
    request: FastifyRequest<{ Body: UploadFromUrlBody }>,
    reply: FastifyReply
  ): Promise<void> {
    const requestId = request.requestId;

    try {
      const { url } = request.body;

      if (!url) {
        throw AppError.invalidImage('URL is required');
      }

      const buffer = await this.fetchImageFromUrl(url);
      await this.validateImage(buffer);

      const metadata = await imageService.getMetadata(buffer);
      const key = await storageService.upload(buffer, undefined, {
        contentType: `image/${metadata.format}`,
        metadata: {
          width: metadata.width.toString(),
          height: metadata.height.toString(),
          sourceUrl: url,
        },
      });

      const signedUrl = await storageService.getSignedUrl(key);

      sendSuccess(reply, requestId, {
        id: key,
        url: signedUrl,
        metadata,
      }, 201);
    } catch (error) {
      if (error instanceof AppError) {
        sendError(reply, requestId, error);
      } else {
        sendError(reply, requestId, AppError.processingError('Failed to fetch image from URL'));
      }
    }
  }

  async getImage(
    request: FastifyRequest<{ Params: { id: string } }>,
    reply: FastifyReply
  ): Promise<void> {
    const requestId = request.requestId;

    try {
      const { id } = request.params;
      const buffer = await storageService.download(id);
      const metadata = await imageService.getMetadata(buffer);

      const acceptHeader = request.headers.accept;

      if (acceptHeader?.includes('application/json')) {
        sendSuccess(reply, requestId, {
          image: buffer.toString('base64'),
          metadata,
        });
      } else {
        reply
          .status(200)
          .header('Content-Type', `image/${metadata.format}`)
          .header('X-Request-Id', requestId)
          .header('X-Image-Width', metadata.width.toString())
          .header('X-Image-Height', metadata.height.toString())
          .send(buffer);
      }
    } catch (error) {
      if (error instanceof AppError) {
        sendError(reply, requestId, error);
      } else {
        sendError(reply, requestId, AppError.imageNotFound('Image not found'));
      }
    }
  }

  private async processMultipartFile(file: MultipartFile): Promise<Buffer> {
    if (!ALLOWED_MIME_TYPES.includes(file.mimetype)) {
      throw AppError.invalidImage(`Invalid file type: ${file.mimetype}. Allowed: ${ALLOWED_MIME_TYPES.join(', ')}`);
    }

    const chunks: Buffer[] = [];
    let totalSize = 0;

    for await (const chunk of file.file) {
      totalSize += chunk.length;
      if (totalSize > config.storage.maxFileSizeBytes) {
        throw AppError.fileTooLarge(
          `File size exceeds maximum allowed size of ${config.storage.maxFileSizeBytes} bytes`
        );
      }
      chunks.push(chunk);
    }

    return Buffer.concat(chunks);
  }

  private async fetchImageFromUrl(url: string): Promise<Buffer> {
    try {
      const response = await fetch(url);

      if (!response.ok) {
        throw AppError.invalidImage(`Failed to fetch image: ${response.statusText}`);
      }

      const contentType = response.headers.get('content-type');
      if (contentType && !ALLOWED_MIME_TYPES.some((type) => contentType.includes(type))) {
        throw AppError.invalidImage(`Invalid content type: ${contentType}`);
      }

      const arrayBuffer = await response.arrayBuffer();
      return Buffer.from(arrayBuffer);
    } catch (error) {
      if (error instanceof AppError) {
        throw error;
      }
      throw AppError.invalidImage('Failed to fetch image from URL');
    }
  }

  private async validateImage(buffer: Buffer): Promise<void> {
    try {
      const metadata = await sharp(buffer).metadata();

      if (!metadata.format) {
        throw AppError.invalidImage('Unable to determine image format');
      }

      if (!metadata.width || !metadata.height) {
        throw AppError.invalidImage('Unable to determine image dimensions');
      }
    } catch (error) {
      if (error instanceof AppError) {
        throw error;
      }
      throw AppError.invalidImage('Invalid or corrupted image file');
    }
  }
}

export const uploadController = new UploadController();
