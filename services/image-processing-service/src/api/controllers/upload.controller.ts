import { FastifyRequest, FastifyReply } from 'fastify';
import { MultipartFile } from '@fastify/multipart';
import sharp from 'sharp';
import { storageService } from '@services/storage';
import { imageService } from '@services/image';
import { AppError } from '@domain/errors';
import { sendSuccess, sendError } from '@shared/utils/response';
import { config } from '@config/index';
import { UrlValidator } from '@security/url-validator';

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
      // SSRF Protection: Validate URL before making request
      // Blocks private IPs, cloud metadata endpoints, redirects, and non-HTTP protocols
      // Returns ValidatedUrl with pinned IP addresses to prevent DNS rebinding (TOCTOU)
      const validatedUrl = await UrlValidator.validate(url);

      // Build URL with pinned IP to prevent DNS rebinding attacks
      const pinnedRequest = UrlValidator.buildPinnedUrl(validatedUrl);

      // Create safe fetch options with timeout and redirect protection
      const fetchOptions = UrlValidator.createSafeFetchOptions(10000, validatedUrl);

      // Merge pinned IP headers with fetch options
      const mergedOptions = {
        ...fetchOptions,
        headers: {
          ...fetchOptions.headers,
          ...pinnedRequest.headers,
        },
      };

      // Make request to validated IP-pinned URL
      const response = await fetch(pinnedRequest.url, mergedOptions);

      // Validate response to prevent redirect-based SSRF
      UrlValidator.validateResponse(response);

      // Validate content type
      const contentType = response.headers.get('content-type');
      if (!contentType || !ALLOWED_MIME_TYPES.some((type) => contentType.includes(type))) {
        throw AppError.invalidImage(
          `Invalid content type: ${contentType}. Allowed types: ${ALLOWED_MIME_TYPES.join(', ')}`
        );
      }

      // Validate content length to prevent memory exhaustion
      const contentLength = response.headers.get('content-length');
      if (contentLength) {
        const size = parseInt(contentLength, 10);
        if (size > config.storage.maxFileSizeBytes) {
          throw AppError.fileTooLarge(
            `Image size (${size} bytes) exceeds maximum allowed size of ${config.storage.maxFileSizeBytes} bytes`
          );
        }
      }

      // Download with size limit enforcement
      const arrayBuffer = await response.arrayBuffer();

      if (arrayBuffer.byteLength > config.storage.maxFileSizeBytes) {
        throw AppError.fileTooLarge(
          `Image size (${arrayBuffer.byteLength} bytes) exceeds maximum allowed size of ${config.storage.maxFileSizeBytes} bytes`
        );
      }

      return Buffer.from(arrayBuffer);
    } catch (error) {
      if (error instanceof AppError) {
        throw error;
      }

      // Handle abort errors (timeout)
      if (error instanceof Error && error.name === 'AbortError') {
        throw AppError.invalidImage('Request timeout: failed to fetch image within 10 seconds');
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
