/**
 * CAEP Subscriber - handles security event subscriptions
 */

import { CaepEvent, CaepEventType, Unsubscribe } from './types';
import { CaepError } from './errors';

type EventHandler = (event: CaepEvent) => void;

export class CaepSubscriber {
  private baseUrl: string;
  private getAccessToken: () => Promise<string>;
  private handlers: Map<CaepEventType | '*', Set<EventHandler>> = new Map();
  private eventSource: EventSource | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;

  constructor(baseUrl: string, getAccessToken: () => Promise<string>) {
    this.baseUrl = baseUrl;
    this.getAccessToken = getAccessToken;
  }

  /**
   * Subscribe to security events
   */
  subscribe(handler: EventHandler, eventTypes?: CaepEventType[]): Unsubscribe {
    const types = eventTypes || ['*' as CaepEventType];

    for (const type of types) {
      const key = type as CaepEventType | '*';
      if (!this.handlers.has(key)) {
        this.handlers.set(key, new Set());
      }
      this.handlers.get(key)!.add(handler);
    }

    // Start SSE connection if not already connected
    if (!this.eventSource) {
      this.connect();
    }

    return () => {
      for (const type of types) {
        const key = type as CaepEventType | '*';
        this.handlers.get(key)?.delete(handler);
      }

      // Disconnect if no more handlers
      if (this.getTotalHandlers() === 0) {
        this.disconnect();
      }
    };
  }

  /**
   * Subscribe to a specific event type
   */
  on(eventType: CaepEventType, handler: EventHandler): Unsubscribe {
    return this.subscribe(handler, [eventType]);
  }

  /**
   * Connect to SSE endpoint
   */
  private async connect(): Promise<void> {
    try {
      const token = await this.getAccessToken();
      const url = new URL(`${this.baseUrl}/caep/events`);
      url.searchParams.set('token', token);

      this.eventSource = new EventSource(url.toString());

      this.eventSource.onopen = () => {
        this.reconnectAttempts = 0;
      };

      this.eventSource.onmessage = (event) => {
        try {
          const caepEvent = this.parseEvent(event.data);
          this.dispatch(caepEvent);
        } catch (error) {
          console.error('Failed to parse CAEP event:', error);
        }
      };

      this.eventSource.onerror = () => {
        this.handleError();
      };
    } catch (error) {
      this.handleError();
    }
  }

  /**
   * Disconnect from SSE endpoint
   */
  disconnect(): void {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
  }

  /**
   * Handle connection errors with exponential backoff
   */
  private handleError(): void {
    this.disconnect();

    if (this.reconnectAttempts < this.maxReconnectAttempts && this.getTotalHandlers() > 0) {
      const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts);
      this.reconnectAttempts++;

      setTimeout(() => {
        this.connect();
      }, delay);
    }
  }

  /**
   * Parse SSE event data into CaepEvent
   */
  private parseEvent(data: string): CaepEvent {
    const parsed = JSON.parse(data);
    return {
      type: parsed.type,
      subject: parsed.subject,
      timestamp: new Date(parsed.timestamp),
      reason: parsed.reason,
    };
  }

  /**
   * Dispatch event to handlers
   */
  private dispatch(event: CaepEvent): void {
    // Dispatch to specific type handlers
    const typeHandlers = this.handlers.get(event.type);
    if (typeHandlers) {
      for (const handler of typeHandlers) {
        try {
          handler(event);
        } catch (error) {
          console.error('CAEP handler error:', error);
        }
      }
    }

    // Dispatch to wildcard handlers
    const wildcardHandlers = this.handlers.get('*');
    if (wildcardHandlers) {
      for (const handler of wildcardHandlers) {
        try {
          handler(event);
        } catch (error) {
          console.error('CAEP handler error:', error);
        }
      }
    }
  }

  private getTotalHandlers(): number {
    let total = 0;
    for (const handlers of this.handlers.values()) {
      total += handlers.size;
    }
    return total;
  }
}
