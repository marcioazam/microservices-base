import { config } from '@config/index';

export enum LogLevel {
  DEBUG = 'debug',
  INFO = 'info',
  WARN = 'warn',
  ERROR = 'error',
  FATAL = 'fatal',
}

/** Known context fields extracted to structured log entry fields */
const KNOWN_CONTEXT_FIELDS = [
  'requestId', 'traceId', 'spanId', 'userId',
  'operation', 'duration', 'method', 'path', 'statusCode',
] as const;

export interface LogContext {
  requestId?: string;
  traceId?: string;
  spanId?: string;
  userId?: string;
  operation?: string;
  duration?: number;
  method?: string;
  path?: string;
  statusCode?: number;
  [key: string]: unknown;
}

export interface ExceptionInfo {
  type: string;
  message: string;
  stackTrace?: string;
}

export interface LogEntry {
  timestamp: string;
  correlationId?: string;
  serviceId: string;
  level: LogLevel;
  message: string;
  traceId?: string;
  spanId?: string;
  userId?: string;
  requestId?: string;
  method?: string;
  path?: string;
  statusCode?: number;
  durationMs?: number;
  metadata?: Record<string, string>;
  exception?: ExceptionInfo;
}

export interface LoggingClient {
  log(level: LogLevel, message: string, context?: LogContext): void;
  debug(message: string, context?: LogContext): void;
  info(message: string, context?: LogContext): void;
  warn(message: string, context?: LogContext): void;
  error(message: string, error?: Error, context?: LogContext): void;
  fatal(message: string, error?: Error, context?: LogContext): void;
  child(context: LogContext): LoggingClient;
  isConnected(): boolean;
}

export class PlatformLoggingClient implements LoggingClient {
  private static readonly SERVICE_ID = 'image-processing-service';
  private readonly version: string;
  private readonly baseContext: LogContext;
  private connected = false;

  constructor(
    private readonly endpoint: string,
    baseContext: LogContext = {}
  ) {
    this.version = process.env.npm_package_version ?? '1.0.0';
    this.baseContext = baseContext;
    void this.initConnection();
  }

  private async initConnection(): Promise<void> {
    try {
      // In production, this would establish gRPC connection to Logging Service
      // For now, we mark as connected and use structured local logging
      this.connected = this.endpoint.length > 0;
    } catch {
      this.connected = false;
    }
  }

  isConnected(): boolean {
    return this.connected;
  }

  log(level: LogLevel, message: string, context?: LogContext): void {
    const entry = this.buildEntry(level, message, context);
    this.send(entry);
  }

  debug(message: string, context?: LogContext): void {
    this.log(LogLevel.DEBUG, message, context);
  }

  info(message: string, context?: LogContext): void {
    this.log(LogLevel.INFO, message, context);
  }

  warn(message: string, context?: LogContext): void {
    this.log(LogLevel.WARN, message, context);
  }

  error(message: string, error?: Error, context?: LogContext): void {
    const entry = this.buildEntry(LogLevel.ERROR, message, context);
    if (error) {
      entry.exception = this.buildExceptionInfo(error);
    }
    this.send(entry);
  }

  fatal(message: string, error?: Error, context?: LogContext): void {
    const entry = this.buildEntry(LogLevel.FATAL, message, context);
    if (error) {
      entry.exception = this.buildExceptionInfo(error);
    }
    this.send(entry);
  }

  child(context: LogContext): LoggingClient {
    return new PlatformLoggingClient(this.endpoint, {
      ...this.baseContext,
      ...context,
    });
  }

  private buildExceptionInfo(error: Error): ExceptionInfo {
    return {
      type: error.name,
      message: error.message,
      stackTrace: error.stack,
    };
  }

  private buildEntry(level: LogLevel, message: string, context?: LogContext): LogEntry {
    const merged = { ...this.baseContext, ...context };
    const metadata = this.extractMetadata(merged);

    return {
      timestamp: new Date().toISOString(),
      correlationId: merged.requestId,
      serviceId: PlatformLoggingClient.SERVICE_ID,
      level,
      message,
      traceId: merged.traceId,
      spanId: merged.spanId,
      userId: merged.userId,
      requestId: merged.requestId,
      method: merged.method,
      path: merged.path,
      statusCode: merged.statusCode,
      durationMs: merged.duration,
      metadata: Object.keys(metadata).length > 0 ? metadata : undefined,
    };
  }

  private extractMetadata(context: LogContext): Record<string, string> {
    const metadata: Record<string, string> = {};
    const knownFields = new Set<string>(KNOWN_CONTEXT_FIELDS);

    for (const [key, value] of Object.entries(context)) {
      if (!knownFields.has(key) && value != null) {
        metadata[key] = String(value);
      }
    }
    return metadata;
  }

  private send(entry: LogEntry): void {
    // In production: send via gRPC to Logging Service
    // Fallback to local structured logging
    this.localLog(entry);
  }

  private localLog(entry: LogEntry): void {
    const output = {
      ...entry,
      service: PlatformLoggingClient.SERVICE_ID,
      version: this.version,
      environment: process.env.NODE_ENV ?? 'development',
    };

    const logFn = this.getConsoleFn(entry.level);
    logFn(JSON.stringify(output));
  }

  private getConsoleFn(level: LogLevel): (msg: string) => void {
    const consoleFnMap: Record<LogLevel, (msg: string) => void> = {
      [LogLevel.DEBUG]: console.debug,
      [LogLevel.INFO]: console.info,
      [LogLevel.WARN]: console.warn,
      [LogLevel.ERROR]: console.error,
      [LogLevel.FATAL]: console.error,
    };
    return consoleFnMap[level] ?? console.log;
  }
}

// Singleton instance
export const logger = new PlatformLoggingClient(config.logging.endpoint ?? '');
