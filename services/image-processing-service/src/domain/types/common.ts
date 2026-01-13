export type ImageFormat = 'jpeg' | 'png' | 'gif' | 'webp' | 'tiff';

export type WatermarkPosition =
  | 'top-left'
  | 'top-center'
  | 'top-right'
  | 'center-left'
  | 'center'
  | 'center-right'
  | 'bottom-left'
  | 'bottom-center'
  | 'bottom-right'
  | { x: number; y: number };

export type JobStatus = 'pending' | 'processing' | 'completed' | 'failed' | 'cancelled';

export type OperationType =
  | 'resize'
  | 'convert'
  | 'adjust'
  | 'rotate'
  | 'flip'
  | 'watermark'
  | 'compress'
  | 'batch';

export const SUPPORTED_FORMATS: ImageFormat[] = ['jpeg', 'png', 'gif', 'webp', 'tiff'];

export const LOSSY_FORMATS: ImageFormat[] = ['jpeg', 'webp'];

export const TRANSPARENT_FORMATS: ImageFormat[] = ['png', 'gif', 'webp'];
