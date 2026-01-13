import { ImageFormat, JobStatus } from './common';

export interface ImageMetadata {
  width: number;
  height: number;
  format: ImageFormat | string;
  size: number;
  hasAlpha: boolean;
}

export interface ProcessedImage {
  buffer: Buffer;
  metadata: ImageMetadata;
}

export interface CompressionStats {
  originalSize: number;
  newSize: number;
  ratio: number;
  savedBytes: number;
  savedPercent: number;
}

export interface ProcessedImageWithStats extends ProcessedImage {
  compressionStats?: CompressionStats;
}

export interface JobResult {
  id: string;
  userId: string;
  status: JobStatus;
  progress: number;
  inputKey: string;
  outputKey?: string;
  error?: string;
  createdAt: Date;
  updatedAt: Date;
  completedAt?: Date;
}

export interface CacheStats {
  hits: number;
  misses: number;
  hitRate: number;
  size: number;
}
