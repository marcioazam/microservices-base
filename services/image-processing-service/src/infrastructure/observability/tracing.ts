import { NodeSDK } from '@opentelemetry/sdk-node';
import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { Resource } from '@opentelemetry/resources';
import { ATTR_SERVICE_NAME, ATTR_SERVICE_VERSION } from '@opentelemetry/semantic-conventions';
import { W3CTraceContextPropagator } from '@opentelemetry/core';
import { SpanStatusCode, trace, context, Span } from '@opentelemetry/api';
import { config } from '@config/index';

let sdk: NodeSDK | null = null;
const SERVICE_NAME = 'image-processing-service';

export interface TraceContext {
  traceId: string;
  spanId: string;
}

export function initTracing(): void {
  if (sdk) return;

  const exporter = new OTLPTraceExporter({
    url: config.tracing?.endpoint || 'http://localhost:4318/v1/traces',
  });

  sdk = new NodeSDK({
    resource: new Resource({
      [ATTR_SERVICE_NAME]: SERVICE_NAME,
      [ATTR_SERVICE_VERSION]: process.env.npm_package_version || '1.0.0',
    }),
    traceExporter: exporter,
    textMapPropagator: new W3CTraceContextPropagator(),
    instrumentations: [getNodeAutoInstrumentations({
      '@opentelemetry/instrumentation-fs': { enabled: false },
    })],
  });

  sdk.start();
}

export async function shutdownTracing(): Promise<void> {
  if (sdk) {
    await sdk.shutdown();
    sdk = null;
  }
}

export function getTracer() {
  return trace.getTracer(SERVICE_NAME);
}

export function getCurrentTraceContext(): TraceContext | null {
  const span = trace.getSpan(context.active());
  if (!span) return null;

  const spanContext = span.spanContext();
  return {
    traceId: spanContext.traceId,
    spanId: spanContext.spanId,
  };
}

export function startSpan<T>(name: string, fn: (span: Span) => Promise<T>): Promise<T> {
  const tracer = getTracer();
  return tracer.startActiveSpan(name, async (span) => {
    try {
      const result = await fn(span);
      span.setStatus({ code: SpanStatusCode.OK });
      return result;
    } catch (error) {
      span.setStatus({ code: SpanStatusCode.ERROR, message: (error as Error).message });
      span.recordException(error as Error);
      throw error;
    } finally {
      span.end();
    }
  });
}

export function recordSpanAttribute(key: string, value: string | number | boolean): void {
  const span = trace.getSpan(context.active());
  if (span) {
    span.setAttribute(key, value);
  }
}

export function recordSpanEvent(name: string, attributes?: Record<string, string | number | boolean>): void {
  const span = trace.getSpan(context.active());
  if (span) {
    span.addEvent(name, attributes);
  }
}
