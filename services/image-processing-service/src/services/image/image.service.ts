import sharp from 'sharp';
import {
  ResizeOptions, ConvertOptions, AdjustOptions,
  RotateOptions, FlipOptions, WatermarkOptions, CompressOptions,
} from '@domain/types/requests';
import {
  ProcessedImage, ImageMetadata, ProcessedImageWithStats, CompressionStats,
} from '@domain/types/responses';
import { ImageFormat, WatermarkPosition } from '@domain/types/common';
import { AppError } from '@domain/errors';

/**
 * Pure image processing service - no validation, caching, or logging.
 * Validation is handled by centralized validators at controller level.
 * Caching is handled by platform cache client at controller level.
 */
export class ImageService {
  async resize(input: Buffer, options: ResizeOptions): Promise<ProcessedImage> {
    let sharpInstance = sharp(input);
    const metadata = await sharpInstance.metadata();

    sharpInstance = sharpInstance.resize({
      width: options.width,
      height: options.height,
      fit: options.maintainAspectRatio !== false ? (options.fit || 'inside') : 'fill',
      withoutEnlargement: false,
    });

    if (options.quality && metadata.format) {
      sharpInstance = this.applyQuality(sharpInstance, metadata.format as ImageFormat, options.quality);
    }

    const buffer = await sharpInstance.toBuffer();
    return this.buildResult(buffer);
  }

  async convert(input: Buffer, options: ConvertOptions): Promise<ProcessedImage> {
    let sharpInstance = sharp(input);
    const inputMetadata = await sharpInstance.metadata();

    if (options.format === 'jpeg' && this.hasTransparency(inputMetadata)) {
      sharpInstance = sharpInstance.flatten({ background: options.backgroundColor || '#ffffff' });
    }

    sharpInstance = sharpInstance.toFormat(options.format, { quality: options.quality || 80 });
    const buffer = await sharpInstance.toBuffer();
    return this.buildResult(buffer);
  }

  async adjust(input: Buffer, options: AdjustOptions): Promise<ProcessedImage> {
    let sharpInstance = sharp(input);

    if (options.brightness !== undefined && options.brightness !== 0) {
      sharpInstance = sharpInstance.modulate({ brightness: 1 + (options.brightness / 100) });
    }

    if (options.saturation !== undefined && options.saturation !== 0) {
      sharpInstance = sharpInstance.modulate({ saturation: 1 + (options.saturation / 100) });
    }

    if (options.contrast !== undefined && options.contrast !== 0) {
      const factor = 1 + (options.contrast / 100);
      sharpInstance = sharpInstance.linear(factor, -(128 * factor) + 128);
    }

    const buffer = await sharpInstance.toBuffer();
    return this.buildResult(buffer);
  }

  async rotate(input: Buffer, options: RotateOptions): Promise<ProcessedImage> {
    const rotateOpts: sharp.RotateOptions = {};
    if (options.backgroundColor) rotateOpts.background = options.backgroundColor;

    const buffer = await sharp(input).rotate(options.angle, rotateOpts).toBuffer();
    return this.buildResult(buffer);
  }

  async flip(input: Buffer, options: FlipOptions): Promise<ProcessedImage> {
    let sharpInstance = sharp(input);
    if (options.vertical) sharpInstance = sharpInstance.flip();
    if (options.horizontal) sharpInstance = sharpInstance.flop();

    const buffer = await sharpInstance.toBuffer();
    return this.buildResult(buffer);
  }

  async watermark(input: Buffer, options: WatermarkOptions): Promise<ProcessedImage> {
    const sharpInstance = sharp(input);
    const inputMetadata = await sharpInstance.metadata();

    if (!inputMetadata.width || !inputMetadata.height) {
      throw AppError.processingError('Unable to read image dimensions');
    }

    const opacity = (options.opacity ?? 100) / 100;
    if (opacity === 0) {
      const buffer = await sharpInstance.toBuffer();
      return this.buildResult(buffer);
    }

    const overlayBuffer = options.type === 'text'
      ? await this.createTextOverlay(options.content, options.font, opacity)
      : await this.createImageOverlay(options.content, opacity);

    const position = this.calculatePosition(options.position, inputMetadata.width, inputMetadata.height);

    const buffer = await sharpInstance
      .composite([{ input: overlayBuffer, top: position.top, left: position.left }])
      .toBuffer();

    return this.buildResult(buffer);
  }

  async compress(input: Buffer, options: CompressOptions): Promise<ProcessedImageWithStats> {
    const inputMetadata = await sharp(input).metadata();
    const originalSize = input.length;

    let sharpInstance = sharp(input);
    const format = options.format || (inputMetadata.format as ImageFormat) || 'jpeg';

    sharpInstance = options.mode === 'lossless'
      ? this.applyLosslessCompression(sharpInstance, format)
      : sharpInstance.toFormat(format, { quality: options.quality || 80 });

    const buffer = await sharpInstance.toBuffer();
    const newSize = buffer.length;

    if (newSize >= originalSize && options.mode === 'lossy') {
      return {
        buffer: input,
        metadata: await this.getMetadata(input),
        compressionStats: { originalSize, newSize: originalSize, ratio: 1, savedBytes: 0, savedPercent: 0 },
      };
    }

    const compressionStats: CompressionStats = {
      originalSize,
      newSize,
      ratio: originalSize / newSize,
      savedBytes: originalSize - newSize,
      savedPercent: ((originalSize - newSize) / originalSize) * 100,
    };

    return { ...(await this.buildResult(buffer)), compressionStats };
  }

  async getMetadata(input: Buffer): Promise<ImageMetadata> {
    const metadata = await sharp(input).metadata();
    return this.buildMetadata(metadata, input.length);
  }

  private async buildResult(buffer: Buffer): Promise<ProcessedImage> {
    const metadata = await sharp(buffer).metadata();
    return { buffer, metadata: this.buildMetadata(metadata, buffer.length) };
  }

  private buildMetadata(metadata: sharp.Metadata, size: number): ImageMetadata {
    return {
      width: metadata.width || 0,
      height: metadata.height || 0,
      format: metadata.format || 'unknown',
      size,
      hasAlpha: metadata.hasAlpha || false,
    };
  }

  private applyQuality(instance: sharp.Sharp, format: ImageFormat, quality: number): sharp.Sharp {
    switch (format) {
      case 'jpeg': return instance.jpeg({ quality });
      case 'png': return instance.png({ quality });
      case 'webp': return instance.webp({ quality });
      default: return instance;
    }
  }

  private applyLosslessCompression(instance: sharp.Sharp, format: ImageFormat): sharp.Sharp {
    switch (format) {
      case 'png': return instance.png({ compressionLevel: 9, palette: false });
      case 'webp': return instance.webp({ lossless: true });
      case 'tiff': return instance.tiff({ compression: 'lzw' });
      default: return instance.jpeg({ quality: 100 });
    }
  }

  private hasTransparency(metadata: sharp.Metadata): boolean {
    return metadata.hasAlpha === true || metadata.format === 'png' || metadata.format === 'gif';
  }

  private async createTextOverlay(text: string, font: WatermarkOptions['font'], opacity: number): Promise<Buffer> {
    const fontSize = font?.size || 24;
    const svg = `
      <svg xmlns="http://www.w3.org/2000/svg">
        <text x="0" y="${fontSize}" font-family="${font?.family || 'Arial'}" font-size="${fontSize}"
          font-weight="${font?.weight || 'normal'}" fill="${font?.color || '#ffffff'}" opacity="${opacity}"
        >${this.escapeXml(text)}</text>
      </svg>`;
    return sharp(Buffer.from(svg)).png().toBuffer();
  }

  private async createImageOverlay(imagePath: string, opacity: number): Promise<Buffer> {
    const overlay = sharp(imagePath);
    if (opacity < 1) {
      return overlay.ensureAlpha().composite([{
        input: Buffer.from([255, 255, 255, Math.round(opacity * 255)]),
        raw: { width: 1, height: 1, channels: 4 }, tile: true, blend: 'dest-in',
      }]).toBuffer();
    }
    return overlay.toBuffer();
  }

  private calculatePosition(position: WatermarkPosition, w: number, h: number): { top: number; left: number } {
    const p = 10;
    if (typeof position === 'object' && 'x' in position) return { top: position.y, left: position.x };
    const positions: Record<string, { top: number; left: number }> = {
      'top-left': { top: p, left: p }, 'top-center': { top: p, left: Math.floor(w / 2) },
      'top-right': { top: p, left: w - p - 100 }, 'center-left': { top: Math.floor(h / 2), left: p },
      'center': { top: Math.floor(h / 2), left: Math.floor(w / 2) },
      'center-right': { top: Math.floor(h / 2), left: w - p - 100 },
      'bottom-left': { top: h - p - 30, left: p }, 'bottom-center': { top: h - p - 30, left: Math.floor(w / 2) },
      'bottom-right': { top: h - p - 30, left: w - p - 100 },
    };
    return positions[position as string] || positions['bottom-right'];
  }

  private escapeXml(text: string): string {
    return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;').replace(/'/g, '&apos;');
  }
}

export const imageService = new ImageService();
