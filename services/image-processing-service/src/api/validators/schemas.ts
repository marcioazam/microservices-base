import { z } from 'zod';
import { SUPPORTED_FORMATS } from '@domain/types/common';
import { AppError } from '@domain/errors';

// Base schemas
const dimensionSchema = z.number().int().positive().max(10000);
const qualitySchema = z.number().int().min(1).max(100);
const hexColorSchema = z.string().regex(/^#[0-9A-Fa-f]{6}$/, 'Invalid hex color format');
const formatSchema = z.enum(['jpeg', 'png', 'gif', 'webp', 'tiff']);
const adjustmentSchema = z.number().min(-100).max(100);

// Operation schemas
export const resizeSchema = z.object({
  width: dimensionSchema.optional(),
  height: dimensionSchema.optional(),
  maintainAspectRatio: z.boolean().default(true),
  fit: z.enum(['cover', 'contain', 'fill', 'inside', 'outside']).default('inside'),
  quality: qualitySchema.optional(),
}).refine(
  (data) => data.width !== undefined || data.height !== undefined,
  { message: 'At least one of width or height must be provided', path: ['width'] }
);

export const convertSchema = z.object({
  format: formatSchema,
  quality: qualitySchema.optional(),
  backgroundColor: hexColorSchema.optional(),
});

export const adjustSchema = z.object({
  brightness: adjustmentSchema.optional(),
  contrast: adjustmentSchema.optional(),
  saturation: adjustmentSchema.optional(),
}).refine(
  (data) => data.brightness !== undefined || data.contrast !== undefined || data.saturation !== undefined,
  { message: 'At least one adjustment must be provided', path: ['brightness'] }
);

export const rotateSchema = z.object({
  angle: z.number().min(0).max(360),
  backgroundColor: hexColorSchema.optional(),
});

export const flipSchema = z.object({
  horizontal: z.boolean().optional(),
  vertical: z.boolean().optional(),
}).refine(
  (data) => data.horizontal === true || data.vertical === true,
  { message: 'At least one of horizontal or vertical must be true', path: ['horizontal'] }
);

export const fontSchema = z.object({
  family: z.string().optional(),
  size: z.number().positive().optional(),
  color: hexColorSchema.optional(),
  weight: z.enum(['normal', 'bold']).optional(),
});

export const watermarkSchema = z.object({
  type: z.enum(['text', 'image']),
  content: z.string().min(1, 'Watermark content is required'),
  position: z.union([
    z.enum(['top-left', 'top-center', 'top-right', 'center-left', 'center',
            'center-right', 'bottom-left', 'bottom-center', 'bottom-right']),
    z.object({ x: z.number(), y: z.number() }),
  ]),
  opacity: z.number().min(0).max(100).optional(),
  font: fontSchema.optional(),
});

export const compressSchema = z.object({
  mode: z.enum(['lossy', 'lossless']),
  quality: qualitySchema.optional(),
  format: formatSchema.optional(),
});

// Field error type
export interface FieldError {
  field: string;
  message: string;
  code: string;
}

// Validation result type
export type ValidationResult<T> = { success: true; data: T } | { success: false; errors: FieldError[] };

// Generic validation function with structured errors
export function validate<T>(schema: z.ZodSchema<T>, input: unknown): T {
  const result = schema.safeParse(input);
  
  if (!result.success) {
    const errors: FieldError[] = result.error.errors.map(e => ({
      field: e.path.join('.'),
      message: e.message,
      code: e.code,
    }));
    
    throw AppError.validationError('Validation failed', errors);
  }
  
  return result.data;
}

// Safe validation (returns result instead of throwing)
export function validateSafe<T>(schema: z.ZodSchema<T>, input: unknown): ValidationResult<T> {
  const result = schema.safeParse(input);
  
  if (!result.success) {
    return {
      success: false,
      errors: result.error.errors.map(e => ({
        field: e.path.join('.'),
        message: e.message,
        code: e.code,
      })),
    };
  }
  
  return { success: true, data: result.data };
}

// Type exports from schemas
export type ResizeInput = z.infer<typeof resizeSchema>;
export type ConvertInput = z.infer<typeof convertSchema>;
export type AdjustInput = z.infer<typeof adjustSchema>;
export type RotateInput = z.infer<typeof rotateSchema>;
export type FlipInput = z.infer<typeof flipSchema>;
export type WatermarkInput = z.infer<typeof watermarkSchema>;
export type CompressInput = z.infer<typeof compressSchema>;

// Convenience validation functions
export const validateResize = (input: unknown) => validate(resizeSchema, input);
export const validateConvert = (input: unknown) => validate(convertSchema, input);
export const validateAdjust = (input: unknown) => validate(adjustSchema, input);
export const validateRotate = (input: unknown) => validate(rotateSchema, input);
export const validateFlip = (input: unknown) => validate(flipSchema, input);
export const validateWatermark = (input: unknown) => validate(watermarkSchema, input);
export const validateCompress = (input: unknown) => validate(compressSchema, input);
