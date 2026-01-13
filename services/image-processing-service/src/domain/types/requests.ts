import { ImageFormat, WatermarkPosition } from './common';

export interface ResizeOptions {
  width?: number;
  height?: number;
  maintainAspectRatio?: boolean;
  fit?: 'cover' | 'contain' | 'fill' | 'inside' | 'outside';
  quality?: number;
}

export interface ConvertOptions {
  format: ImageFormat;
  quality?: number;
  backgroundColor?: string;
}

export interface AdjustOptions {
  brightness?: number;
  contrast?: number;
  saturation?: number;
}

export interface RotateOptions {
  angle: number;
  backgroundColor?: string;
}

export interface FlipOptions {
  horizontal?: boolean;
  vertical?: boolean;
}

export interface FontOptions {
  family?: string;
  size?: number;
  color?: string;
  weight?: 'normal' | 'bold';
}

export interface WatermarkOptions {
  type: 'text' | 'image';
  content: string;
  position: WatermarkPosition;
  opacity?: number;
  font?: FontOptions;
}

export interface CompressOptions {
  mode: 'lossy' | 'lossless';
  quality?: number;
  format?: ImageFormat;
}

export interface BatchResizeOptions {
  images: Buffer[];
  options: ResizeOptions;
}

export type ImageOperation =
  | { type: 'resize'; options: ResizeOptions }
  | { type: 'convert'; options: ConvertOptions }
  | { type: 'adjust'; options: AdjustOptions }
  | { type: 'rotate'; options: RotateOptions }
  | { type: 'flip'; options: FlipOptions }
  | { type: 'watermark'; options: WatermarkOptions }
  | { type: 'compress'; options: CompressOptions };
