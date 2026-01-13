import {
  S3Client,
  PutObjectCommand,
  GetObjectCommand,
  DeleteObjectCommand,
} from '@aws-sdk/client-s3';
import { getSignedUrl } from '@aws-sdk/s3-request-presigner';
import { config } from '@config/index';
import { AppError } from '@domain/errors';
import { v4 as uuidv4 } from 'uuid';

export interface StorageOptions {
  contentType?: string;
  metadata?: Record<string, string>;
  expiresIn?: number;
}

export class StorageService {
  private client: S3Client;
  private bucket: string;

  constructor() {
    this.client = new S3Client({
      region: config.s3.region,
      endpoint: config.s3.endpoint,
      forcePathStyle: config.s3.forcePathStyle,
      credentials: {
        accessKeyId: config.s3.accessKeyId,
        secretAccessKey: config.s3.secretAccessKey,
      },
    });
    this.bucket = config.s3.bucket;
  }

  async upload(
    buffer: Buffer,
    key?: string,
    options: StorageOptions = {}
  ): Promise<string> {
    const storageKey = key || this.generateKey();

    try {
      await this.client.send(
        new PutObjectCommand({
          Bucket: this.bucket,
          Key: storageKey,
          Body: buffer,
          ContentType: options.contentType || 'application/octet-stream',
          Metadata: options.metadata,
        })
      );

      return storageKey;
    } catch (error) {
      throw AppError.storageError('Failed to upload file to storage', {
        key: storageKey,
        error: error instanceof Error ? error.message : 'Unknown error',
      });
    }
  }

  async download(key: string): Promise<Buffer> {
    try {
      const response = await this.client.send(
        new GetObjectCommand({
          Bucket: this.bucket,
          Key: key,
        })
      );

      if (!response.Body) {
        throw AppError.imageNotFound('Image not found in storage', { key });
      }

      const chunks: Uint8Array[] = [];
      const stream = response.Body as AsyncIterable<Uint8Array>;

      for await (const chunk of stream) {
        chunks.push(chunk);
      }

      return Buffer.concat(chunks);
    } catch (error) {
      if (error instanceof AppError) {
        throw error;
      }
      throw AppError.storageError('Failed to download file from storage', {
        key,
        error: error instanceof Error ? error.message : 'Unknown error',
      });
    }
  }

  async getSignedUrl(key: string, expiresIn?: number): Promise<string> {
    const expiration = expiresIn || config.storage.tempTtlSeconds;

    try {
      const command = new GetObjectCommand({
        Bucket: this.bucket,
        Key: key,
      });

      return await getSignedUrl(this.client, command, { expiresIn: expiration });
    } catch (error) {
      throw AppError.storageError('Failed to generate signed URL', {
        key,
        error: error instanceof Error ? error.message : 'Unknown error',
      });
    }
  }

  async delete(key: string): Promise<void> {
    try {
      await this.client.send(
        new DeleteObjectCommand({
          Bucket: this.bucket,
          Key: key,
        })
      );
    } catch (error) {
      throw AppError.storageError('Failed to delete file from storage', {
        key,
        error: error instanceof Error ? error.message : 'Unknown error',
      });
    }
  }

  async exists(key: string): Promise<boolean> {
    try {
      await this.client.send(
        new GetObjectCommand({
          Bucket: this.bucket,
          Key: key,
        })
      );
      return true;
    } catch {
      return false;
    }
  }

  private generateKey(): string {
    const timestamp = Date.now();
    const uuid = uuidv4();
    return `images/${timestamp}/${uuid}`;
  }
}

export const storageService = new StorageService();
