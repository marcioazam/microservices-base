/**
 * CAEP Subscriber - handles security event subscriptions via SSE.
 * 
 * Implements the Continuous Access Evaluation Protocol (CAEP) for
 * receiving real-time security events such as session revocations
 * and credential changes.
 * 
 * Features:
 * - Automatic reconnection with exponential backoff
 * - Event ID tracking for resumable connections
 * - Type-specific and wildcard event handlers
 * - Auto-disconnect when all handlers are removed
 * 
 * @see {@link https://openid.net/specs/openid-caep-specification-1_0.html | CAEP Specification}
 * @packageDocumentation
 */

import type { AccessToken } from './types/branded.js';
import type {
  CaepEvent,
  CaepEventHandler,
  CaepSubscriberOptions,
  Unsubscribe,
} from './types/caep.js';
import { CaepEventType } from './types/caep.js';
import { CaepConnectionError, CaepParseError } from './errors/index.js';

type EventTypeKey = CaepEventType | '*';

/** Default maximum reconnection attempts */
const DEFAULT_MAX_RECONNECT_ATTEMPTS = 5;
/** Default initial reconnection delay in milliseconds */
const DEFAULT_INITIAL_RECONNECT_DELAY = 1000;

/**
 * Configuration for CaepSubscriber.
 */
export interface CaepSubscriberConfig {
  /** Base URL of the auth server */
  readonly baseUrl: string;
  /** Function to get current access token */
  readonly getAccessToken: () => Promise<AccessToken>;
  /** Optional subscriber options */
  readonly options?: CaepSubscriberOptions;
}

/**
 * Subscriber for CAEP (Continuous Access Evaluation Protocol) events.
 * 
 * Establishes an SSE connection to receive real-time security events
 * and automatically handles reconnection with exponential backoff.
 * 
 * @example Basic usage
 * ```typescript
 * const subscriber = new CaepSubscriber({
 *   baseUrl: 'https://auth.example.com',
 *   getAccessToken: () => tokenManager.getAccessToken(),
 * });
 * 
 * // Subscribe to all events
 * const unsubscribe = subscriber.subscribe((event) => {
 *   console.log('Security event:', event.type, event.reason);
 * });
 * 
 * // Later, unsubscribe
 * unsubscribe();
 * ```
 * 
 * @example Type-specific handlers
 * ```typescript
 * // Subscribe to specific event types
 * subscriber.on('session-revoked', (event) => {
 *   console.log('Session revoked:', event.subject);
 *   logout();
 * });
 * 
 * subscriber.on('credential-change', (event) => {
 *   console.log('Credentials changed');
 *   refreshTokens();
 * });
 * ```
 */
export class CaepSubscriber {
  private readonly baseUrl: string;
  private readonly getAccessToken: () => Promise<AccessToken>;
  private readonly maxReconnectAttempts: number;
  private readonly initialReconnectDelay: number;
  private readonly handlers = new Map<EventTypeKey, Set<CaepEventHandler>>();
  private eventSource: EventSource | null = null;
  private reconnectAttempts = 0;
  private lastEventId: string | null = null;

  /**
   * Create a new CaepSubscriber instance.
   * 
   * @param config - Subscriber configuration
   */
  constructor(config: CaepSubscriberConfig) {
    this.baseUrl = config.baseUrl;
    this.getAccessToken = config.getAccessToken;
    this.maxReconnectAttempts =
      config.options?.maxReconnectAttempts ?? DEFAULT_MAX_RECONNECT_ATTEMPTS;
    this.initialReconnectDelay =
      config.options?.initialReconnectDelay ?? DEFAULT_INITIAL_RECONNECT_DELAY;
  }

  /**
   * Subscribe to security events.
   * 
   * Establishes an SSE connection if not already connected.
   * Multiple handlers can be registered for the same event types.
   * 
   * @param handler - Event handler function
   * @param eventTypes - Specific event types to subscribe to (default: all events)
   * @returns Unsubscribe function to remove the handler
   * 
   * @example
   * ```typescript
   * // Subscribe to all events
   * const unsubscribe = subscriber.subscribe((event) => {
   *   console.log('Event:', event.type);
   * });
   * 
   * // Subscribe to specific events
   * const unsubscribe2 = subscriber.subscribe(
   *   (event) => console.log('Session event:', event),
   *   ['session-revoked']
   * );
   * ```
   */
  subscribe(handler: CaepEventHandler, eventTypes?: CaepEventType[]): Unsubscribe {
    const types: EventTypeKey[] = eventTypes ?? ['*'];

    for (const type of types) {
      if (!this.handlers.has(type)) {
        this.handlers.set(type, new Set());
      }
      this.handlers.get(type)?.add(handler);
    }

    if (this.eventSource === null) {
      void this.connect();
    }

    return () => {
      this.unsubscribe(handler, types);
    };
  }

  /**
   * Subscribe to a specific event type.
   * 
   * Convenience method for subscribing to a single event type.
   * 
   * @param eventType - Event type to subscribe to
   * @param handler - Event handler function
   * @returns Unsubscribe function
   * 
   * @example
   * ```typescript
   * subscriber.on('session-revoked', (event) => {
   *   console.log('Session revoked:', event.reason);
   * });
   * ```
   */
  on(eventType: CaepEventType, handler: CaepEventHandler): Unsubscribe {
    return this.subscribe(handler, [eventType]);
  }

  /**
   * Disconnect from the SSE endpoint.
   * 
   * Closes the EventSource connection. The connection will be
   * re-established automatically if new handlers are added.
   */
  disconnect(): void {
    if (this.eventSource !== null) {
      this.eventSource.close();
      this.eventSource = null;
    }
  }

  /**
   * Calculate exponential backoff delay for reconnection.
   * 
   * Formula: initialDelay * 2^attempt
   * 
   * @param attempt - Current reconnection attempt number (0-based)
   * @returns Delay in milliseconds
   * 
   * @example
   * ```typescript
   * // With initialDelay = 1000ms:
   * // attempt 0: 1000ms
   * // attempt 1: 2000ms
   * // attempt 2: 4000ms
   * // attempt 3: 8000ms
   * ```
   */
  calculateBackoffDelay(attempt: number): number {
    return this.initialReconnectDelay * Math.pow(2, attempt);
  }

  private unsubscribe(handler: CaepEventHandler, types: EventTypeKey[]): void {
    for (const type of types) {
      this.handlers.get(type)?.delete(handler);
    }

    if (this.getTotalHandlers() === 0) {
      this.disconnect();
    }
  }

  private async connect(): Promise<void> {
    try {
      const token = await this.getAccessToken();
      const url = new URL(`${this.baseUrl}/caep/events`);
      url.searchParams.set('token', token);

      if (this.lastEventId !== null) {
        url.searchParams.set('lastEventId', this.lastEventId);
      }

      this.eventSource = new EventSource(url.toString());

      this.eventSource.onopen = (): void => {
        this.reconnectAttempts = 0;
      };

      this.eventSource.onmessage = (event: MessageEvent<string>): void => {
        if (event.lastEventId !== '') {
          this.lastEventId = event.lastEventId;
        }
        this.handleMessage(event.data);
      };

      this.eventSource.onerror = (): void => {
        this.handleError();
      };
    } catch {
      this.handleError();
    }
  }

  private handleError(): void {
    this.disconnect();

    if (
      this.reconnectAttempts < this.maxReconnectAttempts &&
      this.getTotalHandlers() > 0
    ) {
      const delay = this.calculateBackoffDelay(this.reconnectAttempts);
      this.reconnectAttempts++;

      setTimeout(() => {
        void this.connect();
      }, delay);
    }
  }

  private handleMessage(data: string): void {
    try {
      const event = this.parseEvent(data);
      this.dispatch(event);
    } catch {
      // Log parse errors but don't crash
    }
  }

  private parseEvent(data: string): CaepEvent {
    const parsed: unknown = JSON.parse(data);

    if (!this.isValidEventData(parsed)) {
      throw new CaepParseError('Invalid event data');
    }

    return {
      type: parsed.type as CaepEventType,
      subject: parsed.subject,
      timestamp: new Date(parsed.timestamp as string),
      reason: parsed.reason,
    };
  }

  private isValidEventData(
    data: unknown
  ): data is { type: string; subject: unknown; timestamp: unknown; reason?: string } {
    if (typeof data !== 'object' || data === null) {
      return false;
    }
    const obj = data as Record<string, unknown>;
    return (
      typeof obj.type === 'string' &&
      typeof obj.subject === 'object' &&
      obj.subject !== null
    );
  }

  private dispatch(event: CaepEvent): void {
    const typeHandlers = this.handlers.get(event.type);
    if (typeHandlers !== undefined) {
      for (const handler of typeHandlers) {
        this.safeCall(handler, event);
      }
    }

    const wildcardHandlers = this.handlers.get('*');
    if (wildcardHandlers !== undefined) {
      for (const handler of wildcardHandlers) {
        this.safeCall(handler, event);
      }
    }
  }

  private safeCall(handler: CaepEventHandler, event: CaepEvent): void {
    try {
      handler(event);
    } catch {
      // Swallow handler errors to prevent one handler from breaking others
    }
  }

  getTotalHandlers(): number {
    let total = 0;
    for (const handlers of this.handlers.values()) {
      total += handlers.size;
    }
    return total;
  }
}
