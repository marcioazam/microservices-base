import { ErrorCode, ERROR_HTTP_STATUS } from './error-codes';

export interface FieldError {
  field: string;
  message: string;
  code: string;
}

export interface ErrorDetails {
  code: ErrorCode;
  message: string;
  details?: Record<string, unknown>;
  fields?: FieldError[];
}

export class AppError extends Error {
  public readonly code: ErrorCode;
  public readonly httpStatus: number;
  public readonly details?: Record<string, unknown>;
  public readonly fields?: FieldError[];

  constructor(code: ErrorCode, message: string, details?: Record<string, unknown>, fields?: FieldError[]) {
    super(message);
    this.name = 'AppError';
    this.code = code;
    this.httpStatus = ERROR_HTTP_STATUS[code];
    this.details = details;
    this.fields = fields;
    Error.captureStackTrace(this, this.constructor);
  }

  toJSON(): ErrorDetails {
    return {
      code: this.code,
      message: this.message,
      details: this.details,
      fields: this.fields,
    };
  }

  static validationError(message: string, fields: FieldError[]): AppError {
    return new AppError(ErrorCode.VALIDATION_ERROR, message, { fields }, fields);
  }

  static invalidDimensions(message: string, details?: Record<string, unknown>): AppError {
    return new AppError(ErrorCode.INVALID_DIMENSIONS, message, details);
  }

  static invalidFormat(message: string, details?: Record<string, unknown>): AppError {
    return new AppError(ErrorCode.INVALID_FORMAT, message, details);
  }

  static invalidQuality(message: string, details?: Record<string, unknown>): AppError {
    return new AppError(ErrorCode.INVALID_QUALITY, message, details);
  }

  static invalidAdjustmentValue(message: string, details?: Record<string, unknown>): AppError {
    return new AppError(ErrorCode.INVALID_ADJUSTMENT_VALUE, message, details);
  }

  static invalidImage(message: string, details?: Record<string, unknown>): AppError {
    return new AppError(ErrorCode.INVALID_IMAGE, message, details);
  }

  static invalidWatermark(message: string, details?: Record<string, unknown>): AppError {
    return new AppError(ErrorCode.INVALID_WATERMARK, message, details);
  }

  static fileTooLarge(message: string, details?: Record<string, unknown>): AppError {
    return new AppError(ErrorCode.FILE_TOO_LARGE, message, details);
  }

  static imageNotFound(message: string, details?: Record<string, unknown>): AppError {
    return new AppError(ErrorCode.IMAGE_NOT_FOUND, message, details);
  }

  static jobNotFound(message: string, details?: Record<string, unknown>): AppError {
    return new AppError(ErrorCode.JOB_NOT_FOUND, message, details);
  }

  static processingError(message: string, details?: Record<string, unknown>): AppError {
    return new AppError(ErrorCode.PROCESSING_ERROR, message, details);
  }

  static storageError(message: string, details?: Record<string, unknown>): AppError {
    return new AppError(ErrorCode.STORAGE_ERROR, message, details);
  }

  static unauthorized(message: string, details?: Record<string, unknown>): AppError {
    return new AppError(ErrorCode.MISSING_TOKEN, message, details);
  }

  static forbidden(message: string, details?: Record<string, unknown>): AppError {
    return new AppError(ErrorCode.INSUFFICIENT_PERMISSIONS, message, details);
  }

  static rateLimitExceeded(message: string, details?: Record<string, unknown>): AppError {
    return new AppError(ErrorCode.RATE_LIMIT_EXCEEDED, message, details);
  }
}
