export {
  initTracing,
  shutdownTracing,
  getTracer,
  getCurrentTraceContext,
  startSpan,
  recordSpanAttribute,
  recordSpanEvent,
  type TraceContext,
} from './tracing';

export {
  httpRequestsTotal,
  httpRequestDuration,
  imageProcessingTotal,
  imageProcessingDuration,
  imageProcessingSize,
  cacheOperationsTotal,
  jobQueueSize,
  recordHttpRequest,
  recordImageProcessing,
  recordImageSize,
  recordCacheOperation,
  getMetrics,
  getContentType,
  register,
} from './metrics';
