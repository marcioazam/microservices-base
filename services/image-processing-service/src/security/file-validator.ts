/**
 * File Security Validator
 *
 * Provides security validation for uploaded files including:
 * 1. File extension validation (allowlist)
 * 2. MIME type validation
 * 3. Magic bytes (file signature) validation
 * 4. Path traversal protection
 * 5. ClamAV integration (optional, when available)
 *
 * @module security/file-validator
 */

import { AppError } from '@domain/errors';
import { spawn } from 'child_process';
import * as path from 'path';

/**
 * File validation result
 */
export interface FileValidationResult {
  valid: boolean;
  mimeType: string;
  extension: string;
  errors: string[];
  scanResult?: {
    scanned: boolean;
    clean: boolean;
    threat?: string;
  };
}

/**
 * Magic bytes signatures for common image formats
 */
const MAGIC_BYTES: Record<string, { bytes: number[]; offset: number }[]> = {
  'image/jpeg': [
    { bytes: [0xff, 0xd8, 0xff], offset: 0 },
  ],
  'image/png': [
    { bytes: [0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a], offset: 0 },
  ],
  'image/gif': [
    { bytes: [0x47, 0x49, 0x46, 0x38, 0x37, 0x61], offset: 0 }, // GIF87a
    { bytes: [0x47, 0x49, 0x46, 0x38, 0x39, 0x61], offset: 0 }, // GIF89a
  ],
  'image/webp': [
    { bytes: [0x52, 0x49, 0x46, 0x46], offset: 0 }, // RIFF header
  ],
  'image/tiff': [
    { bytes: [0x49, 0x49, 0x2a, 0x00], offset: 0 }, // Little-endian
    { bytes: [0x4d, 0x4d, 0x00, 0x2a], offset: 0 }, // Big-endian
  ],
  'image/bmp': [
    { bytes: [0x42, 0x4d], offset: 0 }, // BM
  ],
};

/**
 * Allowed file extensions for images
 */
const ALLOWED_EXTENSIONS = new Set([
  '.jpg',
  '.jpeg',
  '.png',
  '.gif',
  '.webp',
  '.tiff',
  '.tif',
  '.bmp',
]);

/**
 * Allowed MIME types for images
 */
const ALLOWED_MIME_TYPES = new Set([
  'image/jpeg',
  'image/png',
  'image/gif',
  'image/webp',
  'image/tiff',
  'image/bmp',
]);

/**
 * Dangerous file extensions that should never be allowed
 */
const DANGEROUS_EXTENSIONS = new Set([
  '.exe',
  '.dll',
  '.bat',
  '.cmd',
  '.ps1',
  '.sh',
  '.bash',
  '.com',
  '.msi',
  '.vbs',
  '.js',
  '.jse',
  '.wsf',
  '.wsh',
  '.scr',
  '.pif',
  '.jar',
  '.py',
  '.php',
  '.asp',
  '.aspx',
  '.jsp',
  '.cgi',
  '.pl',
  '.rb',
]);

/**
 * Path patterns that indicate path traversal attempts
 */
const PATH_TRAVERSAL_PATTERNS = [
  /\.\./,                    // Parent directory
  /^\/|^\\|^[a-zA-Z]:/,      // Absolute paths
  /%2e%2e/i,                 // URL-encoded ..
  /%252e%252e/i,             // Double URL-encoded ..
  /\x00/,                    // Null bytes
];

export class FileValidator {
  private static clamavPath: string | null = null;
  private static clamavAvailable: boolean | null = null;

  /**
   * Configure ClamAV scanner path
   * @param executablePath - Path to clamscan or clamdscan executable
   */
  static configureClamAV(executablePath: string): void {
    // Validate the path is safe
    if (!this.isValidExecutablePath(executablePath)) {
      throw new Error(`Invalid ClamAV path: ${executablePath}`);
    }
    this.clamavPath = executablePath;
    this.clamavAvailable = null; // Reset availability check
  }

  /**
   * Validate that an executable path is safe
   * Prevents command injection and path traversal
   */
  private static isValidExecutablePath(execPath: string): boolean {
    // Normalize path
    const normalizedPath = path.normalize(execPath);

    // Check for path traversal attempts
    for (const pattern of PATH_TRAVERSAL_PATTERNS) {
      if (pattern.test(execPath)) {
        return false;
      }
    }

    // Only allow specific executable names
    const allowedExecutables = [
      'clamscan',
      'clamscan.exe',
      'clamdscan',
      'clamdscan.exe',
    ];

    const baseName = path.basename(normalizedPath).toLowerCase();
    if (!allowedExecutables.includes(baseName)) {
      return false;
    }

    // Path must be in expected locations (optional security check)
    const allowedPrefixes = [
      '/usr/bin/',
      '/usr/local/bin/',
      '/opt/clamav/',
      'C:\\Program Files\\ClamAV\\',
      'C:\\Program Files (x86)\\ClamAV\\',
    ];

    // Allow if path starts with known safe prefix or is just the executable name
    const isKnownPath = allowedPrefixes.some(prefix =>
      normalizedPath.toLowerCase().startsWith(prefix.toLowerCase())
    );
    const isBareName = !normalizedPath.includes('/') && !normalizedPath.includes('\\');

    return isKnownPath || isBareName;
  }

  /**
   * Validate a filename for security issues
   */
  static validateFilename(filename: string): void {
    // Check for path traversal
    for (const pattern of PATH_TRAVERSAL_PATTERNS) {
      if (pattern.test(filename)) {
        throw AppError.invalidImage('Invalid filename: path traversal detected');
      }
    }

    // Get extension
    const ext = path.extname(filename).toLowerCase();

    // Check for dangerous extensions
    if (DANGEROUS_EXTENSIONS.has(ext)) {
      throw AppError.invalidImage(`Dangerous file type not allowed: ${ext}`);
    }

    // Check for allowed extensions
    if (!ALLOWED_EXTENSIONS.has(ext)) {
      throw AppError.invalidImage(
        `File extension not allowed: ${ext}. Allowed: ${Array.from(ALLOWED_EXTENSIONS).join(', ')}`
      );
    }
  }

  /**
   * Validate MIME type
   */
  static validateMimeType(mimeType: string): void {
    if (!ALLOWED_MIME_TYPES.has(mimeType.toLowerCase())) {
      throw AppError.invalidImage(
        `MIME type not allowed: ${mimeType}. Allowed: ${Array.from(ALLOWED_MIME_TYPES).join(', ')}`
      );
    }
  }

  /**
   * Validate file content using magic bytes
   * @param buffer - File content buffer
   * @returns Detected MIME type or null if unknown
   */
  static validateMagicBytes(buffer: Buffer): string | null {
    for (const [mimeType, signatures] of Object.entries(MAGIC_BYTES)) {
      for (const sig of signatures) {
        if (this.matchesSignature(buffer, sig.bytes, sig.offset)) {
          return mimeType;
        }
      }
    }
    return null;
  }

  /**
   * Check if buffer matches a byte signature
   */
  private static matchesSignature(buffer: Buffer, bytes: number[], offset: number): boolean {
    if (buffer.length < offset + bytes.length) {
      return false;
    }

    for (let i = 0; i < bytes.length; i++) {
      if (buffer[offset + i] !== bytes[i]) {
        return false;
      }
    }

    return true;
  }

  /**
   * Full file validation including optional virus scan
   */
  static async validate(
    buffer: Buffer,
    filename: string,
    declaredMimeType: string,
    options: { skipVirusScan?: boolean } = {}
  ): Promise<FileValidationResult> {
    const errors: string[] = [];
    const ext = path.extname(filename).toLowerCase();

    // Validate filename
    try {
      this.validateFilename(filename);
    } catch (e) {
      errors.push((e as Error).message);
    }

    // Validate declared MIME type
    try {
      this.validateMimeType(declaredMimeType);
    } catch (e) {
      errors.push((e as Error).message);
    }

    // Validate magic bytes and detect actual MIME type
    const detectedMimeType = this.validateMagicBytes(buffer);

    if (!detectedMimeType) {
      errors.push('Unable to detect file type from content');
    } else if (detectedMimeType !== declaredMimeType.toLowerCase()) {
      // Allow some flexibility (e.g., image/jpeg vs image/jpg)
      const normalizedDeclared = declaredMimeType.toLowerCase().replace('jpg', 'jpeg');
      const normalizedDetected = detectedMimeType.toLowerCase().replace('jpg', 'jpeg');

      if (normalizedDeclared !== normalizedDetected) {
        errors.push(
          `MIME type mismatch: declared ${declaredMimeType}, detected ${detectedMimeType}`
        );
      }
    }

    // Virus scan (if available and not skipped)
    let scanResult: FileValidationResult['scanResult'];
    if (!options.skipVirusScan) {
      scanResult = await this.scanForViruses(buffer);
      if (scanResult.scanned && !scanResult.clean) {
        errors.push(`Malware detected: ${scanResult.threat}`);
      }
    }

    return {
      valid: errors.length === 0,
      mimeType: detectedMimeType || declaredMimeType,
      extension: ext,
      errors,
      scanResult,
    };
  }

  /**
   * Check if ClamAV is available
   */
  private static async checkClamAVAvailability(): Promise<boolean> {
    if (this.clamavAvailable !== null) {
      return this.clamavAvailable;
    }

    if (!this.clamavPath) {
      // Try to find clamscan in PATH
      const defaultPaths = ['clamscan', 'clamdscan'];

      for (const cmd of defaultPaths) {
        try {
          await this.executeCommand(cmd, ['--version']);
          this.clamavPath = cmd;
          this.clamavAvailable = true;
          return true;
        } catch {
          // Continue to next option
        }
      }

      this.clamavAvailable = false;
      return false;
    }

    try {
      await this.executeCommand(this.clamavPath, ['--version']);
      this.clamavAvailable = true;
      return true;
    } catch {
      this.clamavAvailable = false;
      return false;
    }
  }

  /**
   * Scan buffer for viruses using ClamAV
   */
  private static async scanForViruses(buffer: Buffer): Promise<{
    scanned: boolean;
    clean: boolean;
    threat?: string;
  }> {
    const available = await this.checkClamAVAvailability();

    if (!available || !this.clamavPath) {
      return { scanned: false, clean: true };
    }

    try {
      // Use stdin to pass file content (avoids temp file)
      const result = await this.executeClamScan(buffer);

      return {
        scanned: true,
        clean: result.clean,
        threat: result.threat,
      };
    } catch (error) {
      // Log error but don't fail - ClamAV is optional
      console.warn('ClamAV scan failed:', error);
      return { scanned: false, clean: true };
    }
  }

  /**
   * Execute ClamAV scan on buffer
   */
  private static async executeClamScan(buffer: Buffer): Promise<{
    clean: boolean;
    threat?: string;
  }> {
    return new Promise((resolve, reject) => {
      if (!this.clamavPath) {
        reject(new Error('ClamAV not configured'));
        return;
      }

      const proc = spawn(this.clamavPath, [
        '--stdin',
        '--no-summary',
        '--infected',
      ]);

      let stdout = '';
      let stderr = '';

      proc.stdout.on('data', (data) => {
        stdout += data.toString();
      });

      proc.stderr.on('data', (data) => {
        stderr += data.toString();
      });

      proc.on('close', (code) => {
        if (code === 0) {
          // No virus found
          resolve({ clean: true });
        } else if (code === 1) {
          // Virus found - parse output for threat name
          const match = stdout.match(/:\s*(.+)\s+FOUND/);
          resolve({
            clean: false,
            threat: match ? match[1] : 'Unknown threat',
          });
        } else {
          // Error
          reject(new Error(`ClamAV error (code ${code}): ${stderr}`));
        }
      });

      proc.on('error', (error) => {
        reject(error);
      });

      // Write buffer to stdin
      proc.stdin.write(buffer);
      proc.stdin.end();
    });
  }

  /**
   * Execute a command and return output
   */
  private static async executeCommand(cmd: string, args: string[]): Promise<string> {
    return new Promise((resolve, reject) => {
      const proc = spawn(cmd, args);
      let stdout = '';
      let stderr = '';

      proc.stdout.on('data', (data) => {
        stdout += data.toString();
      });

      proc.stderr.on('data', (data) => {
        stderr += data.toString();
      });

      proc.on('close', (code) => {
        if (code === 0) {
          resolve(stdout);
        } else {
          reject(new Error(`Command failed (code ${code}): ${stderr}`));
        }
      });

      proc.on('error', (error) => {
        reject(error);
      });
    });
  }

  /**
   * Get list of allowed extensions
   */
  static getAllowedExtensions(): string[] {
    return Array.from(ALLOWED_EXTENSIONS);
  }

  /**
   * Get list of allowed MIME types
   */
  static getAllowedMimeTypes(): string[] {
    return Array.from(ALLOWED_MIME_TYPES);
  }
}
