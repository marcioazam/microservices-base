import crypto from 'crypto';
import { config } from '@config/index';
import { ProcessedImage, ImageMetadata } from '@domain/types';
import { ImageOperation } from '@domain/types/requests';

interface CachedImage {
  buffer: string; // base64 encoded
  metadata: ImageMetadata;
}

export interface CacheClient {
  get<T>(key: string): Promise<T | null>;
  set<T>(key: string, value: T, ttlSeconds?: number): Promise<void>;
  delete(key: string): Promise<boolean>;
  exists(key: string): Promise<boolean>;
  invalidatePattern(pattern: string): Promise<number>;
}

export interface ImageCacheClient extends CacheClient {
  getImage(key: string): Promise<ProcessedImage | null>;
  setImage(key: string, image: ProcessedImage, ttlSeconds?: number): Promise<void>;
  generateKey(inputHash: string, operation: ImageOperation): string;
  generateInputHash(buffer: Buffer): string;
}

export class PlatformCacheClient implements ImageCacheClient {
  private readonly namespace = 'img';
  private readonly prefix = 'img:';
  private connected = false;
  private localCache = new Map<string, { value: string; expiresAt: number }>();

  constructor(
    private readonly endpoint: string,
    private readonly defaultTtl: number = 3600
  ) {
    this.initConnection();
  }

  private async initConnection(): Promise<void> {
    try {
      // In production: establish gRPC connection to Cache Service
      this.connected = this.endpoint !== '';
    } catch {
      this.connected = false;
    }
  }

  generateKey(inputHash: string, operation: ImageOperation): string {
    const operationString = JSON.stringify(operation);
    const hash = crypto
      .createHash('sha256')
      .update(`${inputHash}:${operationString}`)
      .digest('hex');
    return `${this.prefix}${hash}`;
  }

  generateInputHash(buffer: Buffer): string {
    return crypto.createHash('sha256').update(buffer).digest('hex');
  }

  async get<T>(key: string): Promise<T | null> {
    const prefixedKey = this.ensurePrefix(key);
    
    if (this.connected) {
      // In production: call gRPC Get with namespace
      return this.localGet<T>(prefixedKey);
    }
    return this.localGet<T>(prefixedKey);
  }

  async set<T>(key: string, value: T, ttlSeconds?: number): Promise<void> {
    const prefixedKey = this.ensurePrefix(key);
    const ttl = ttlSeconds ?? this.defaultTtl;

    if (this.connected) {
      // In production: call gRPC Set with namespace
      this.localSet(prefixedKey, value, ttl);
    } else {
      this.localSet(prefixedKey, value, ttl);
    }
  }

  async delete(key: string): Promise<boolean> {
    const prefixedKey = this.ensurePrefix(key);
    
    if (this.connected) {
      // In production: call gRPC Delete
      return this.localDelete(prefixedKey);
    }
    return this.localDelete(prefixedKey);
  }

  async exists(key: string): Promise<boolean> {
    const prefixedKey = this.ensurePrefix(key);
    const cached = this.localCache.get(prefixedKey);
    
    if (!cached) return false;
    if (Date.now() > cached.expiresAt) {
      this.localCache.delete(prefixedKey);
      return false;
    }
    return true;
  }

  async invalidatePattern(pattern: string): Promise<number> {
    const prefixedPattern = this.ensurePrefix(pattern);
    let count = 0;

    for (const key of this.localCache.keys()) {
      if (key.startsWith(prefixedPattern.replace('*', ''))) {
        this.localCache.delete(key);
        count++;
      }
    }
    return count;
  }

  async getImage(key: string): Promise<ProcessedImage | null> {
    const cached = await this.get<CachedImage>(key);
    if (!cached) return null;

    return {
      buffer: Buffer.from(cached.buffer, 'base64'),
      metadata: cached.metadata,
    };
  }

  async setImage(key: string, image: ProcessedImage, ttlSeconds?: number): Promise<void> {
    const cached: CachedImage = {
      buffer: image.buffer.toString('base64'),
      metadata: image.metadata,
    };
    await this.set(cached, key, ttlSeconds);
  }

  async healthCheck(): Promise<{ healthy: boolean; latencyMs: number }> {
    const start = Date.now();
    try {
      // In production: call gRPC Health
      await this.exists('health-check');
      return { healthy: true, latencyMs: Date.now() - start };
    } catch {
      return { healthy: false, latencyMs: Date.now() - start };
    }
  }

  private ensurePrefix(key: string): string {
    return key.startsWith(this.prefix) ? key : `${this.prefix}${key}`;
  }

  private localGet<T>(key: string): T | null {
    const cached = this.localCache.get(key);
    if (!cached) return null;
    
    if (Date.now() > cached.expiresAt) {
      this.localCache.delete(key);
      return null;
    }
    
    return JSON.parse(cached.value) as T;
  }

  private localSet<T>(key: string, value: T, ttlSeconds: number): void {
    this.localCache.set(key, {
      value: JSON.stringify(value),
      expiresAt: Date.now() + ttlSeconds * 1000,
    });
  }

  private localDelete(key: string): boolean {
    return this.localCache.delete(key);
  }
}

// Singleton instance
export const cacheClient = new PlatformCacheClient(
  config.cache.endpoint || '',
  config.cache.ttlSeconds
);
