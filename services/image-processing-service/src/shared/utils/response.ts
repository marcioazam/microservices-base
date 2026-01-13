import { FastifyReply, FastifyRequest } from 'fastify';
import { AppError, ErrorDetails } from '@domain/errors';
import { ImageMetadata } from '@domain/types';

export interface SuccessResponse<T = unknown> {
  success: true;
  requestId: string;
  data: T;
}

export interface ErrorResponse {
  success: false;
  requestId: string;
  error: ErrorDetails;
}

export type ApiResponse<T = unknown> = SuccessResponse<T> | ErrorResponse;

// Required headers for all responses
export const REQUIRED_HEADERS = {
  REQUEST_ID: 'X-Request-Id',
} as const;

// Required headers for image responses
export const IMAGE_HEADERS = {
  CONTENT_TYPE: 'Content-Type',
  WIDTH: 'X-Image-Width',
  HEIGHT: 'X-Image-Height',
  SIZE: 'X-Image-Size',
  FORMAT: 'X-Image-Format',
} as const;

export function getRequestId(request: FastifyRequest): string {
  return (request.id as string) || crypto.randomUUID();
}

export interface ImageResponseData {
  image?: string;
  url?: string;
  metadata: ImageMetadata;
}

export interface JobResponseData {
  jobId: string;
  status: string;
  progress?: number;
  createdAt: string;
  updatedAt: string;
  error?: string;
}

export function sendSuccess<T>(
  reply: FastifyReply,
  requestId: string,
  data: T,
  statusCode = 200
): void {
  const response: SuccessResponse<T> = {
    success: true,
    requestId,
    data,
  };
  reply
    .status(statusCode)
    .header(REQUIRED_HEADERS.REQUEST_ID, requestId)
    .send(response);
}

export function sendError(
  reply: FastifyReply,
  requestId: string,
  error: AppError | Error
): void {
  if (error instanceof AppError) {
    const response: ErrorResponse = {
      success: false,
      requestId,
      error: error.toJSON(),
    };
    reply
      .status(error.httpStatus)
      .header(REQUIRED_HEADERS.REQUEST_ID, requestId)
      .send(response);
  } else {
    const response: ErrorResponse = {
      success: false,
      requestId,
      error: {
        code: 'INTERNAL_ERROR' as const,
        message: 'An unexpected error occurred',
      },
    };
    reply
      .status(500)
      .header(REQUIRED_HEADERS.REQUEST_ID, requestId)
      .send(response);
  }
}

export function sendImage(
  reply: FastifyReply,
  requestId: string,
  buffer: Buffer,
  metadata: ImageMetadata,
  acceptHeader?: string
): void {
  const wantsJson = acceptHeader?.includes('application/json');

  if (wantsJson) {
    const response: SuccessResponse<ImageResponseData> = {
      success: true,
      requestId,
      data: {
        image: buffer.toString('base64'),
        metadata,
      },
    };
    reply
      .status(200)
      .header(REQUIRED_HEADERS.REQUEST_ID, requestId)
      .send(response);
  } else {
    reply
      .status(200)
      .header(IMAGE_HEADERS.CONTENT_TYPE, `image/${metadata.format}`)
      .header(REQUIRED_HEADERS.REQUEST_ID, requestId)
      .header(IMAGE_HEADERS.WIDTH, metadata.width.toString())
      .header(IMAGE_HEADERS.HEIGHT, metadata.height.toString())
      .header(IMAGE_HEADERS.SIZE, metadata.size.toString())
      .header(IMAGE_HEADERS.FORMAT, metadata.format)
      .send(buffer);
  }
}
