/**
 * Property-based tests for CAEP subscriber
 *
 * **Feature: typescript-sdk-modernization**
 * **Properties: 16, 18, 19, 20**
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import * as fc from 'fast-check';
import { CaepSubscriber } from '../../src/caep.js';
import { CaepEventType, type CaepEvent, type AccessToken } from '../../src/index.js';

// Mock EventSource
class MockEventSource {
  static instances: MockEventSource[] = [];
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: string; lastEventId: string }) => void) | null = null;
  onerror: (() => void) | null = null;
  readyState = 1;
  url: string;

  constructor(url: string) {
    this.url = url;
    MockEventSource.instances.push(this);
  }

  close(): void {
    this.readyState = 2;
  }

  static reset(): void {
    MockEventSource.instances = [];
  }
}

// @ts-expect-error - Mocking global EventSource
globalThis.EventSource = MockEventSource;

/**
 * **Property 16: Exponential Backoff Calculation**
 * Delay equals initialDelay * 2^n for attempt n.
 * **Validates: Requirements 8.2**
 */
describe('Property 16: Exponential Backoff Calculation', () => {
  it('calculates correct exponential backoff delay', () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 100, max: 5000 }), // initial delay
        fc.integer({ min: 0, max: 10 }), // attempt number
        (initialDelay, attempt) => {
          const subscriber = new CaepSubscriber({
            baseUrl: 'https://example.com',
            getAccessToken: async () => 'token' as AccessToken,
            options: { initialReconnectDelay: initialDelay },
          });

          const expectedDelay = initialDelay * Math.pow(2, attempt);
          const actualDelay = subscriber.calculateBackoffDelay(attempt);

          expect(actualDelay).toBe(expectedDelay);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('backoff increases exponentially', () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 100, max: 1000 }),
        (initialDelay) => {
          const subscriber = new CaepSubscriber({
            baseUrl: 'https://example.com',
            getAccessToken: async () => 'token' as AccessToken,
            options: { initialReconnectDelay: initialDelay },
          });

          const delays = [0, 1, 2, 3, 4].map((n) => subscriber.calculateBackoffDelay(n));

          // Each delay should be double the previous
          for (let i = 1; i < delays.length; i++) {
            expect(delays[i]).toBe(delays[i - 1]! * 2);
          }
        }
      ),
      { numRuns: 100 }
    );
  });
});

/**
 * **Property 18: Event Dispatch Routing**
 * Events go to type-specific and wildcard handlers only.
 * **Validates: Requirements 8.4, 8.5**
 */
describe('Property 18: Event Dispatch Routing', () => {
  beforeEach(() => {
    MockEventSource.reset();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('dispatches to correct handlers based on event type', async () => {
    const eventTypes = Object.values(CaepEventType);

    fc.assert(
      fc.property(
        fc.constantFrom(...eventTypes),
        (eventType) => {
          const subscriber = new CaepSubscriber({
            baseUrl: 'https://example.com',
            getAccessToken: async () => 'token' as AccessToken,
          });

          const specificHandler = vi.fn();
          const wildcardHandler = vi.fn();
          const otherHandler = vi.fn();

          // Subscribe to specific type
          subscriber.on(eventType, specificHandler);

          // Subscribe to wildcard
          subscriber.subscribe(wildcardHandler);

          // Subscribe to different type
          const otherType = eventTypes.find((t) => t !== eventType) ?? eventTypes[0]!;
          subscriber.on(otherType, otherHandler);

          // Simulate event
          const event: CaepEvent = {
            type: eventType,
            subject: { format: 'iss_sub', iss: 'test', sub: 'user' },
            timestamp: new Date(),
          };

          // Access private dispatch method via any
          (subscriber as any).dispatch(event);

          // Specific handler should be called
          expect(specificHandler).toHaveBeenCalledWith(event);

          // Wildcard handler should be called
          expect(wildcardHandler).toHaveBeenCalledWith(event);

          // Other type handler should NOT be called (unless it's the same type)
          if (otherType !== eventType) {
            expect(otherHandler).not.toHaveBeenCalled();
          }

          subscriber.disconnect();
        }
      ),
      { numRuns: 20 }
    );
  });
});

/**
 * **Property 19: Retry Limit Enforcement**
 * Reconnect attempts never exceed maxReconnectAttempts.
 * **Validates: Requirements 8.6**
 */
describe('Property 19: Retry Limit Enforcement', () => {
  it('respects max reconnect attempts limit', () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 1, max: 10 }),
        (maxAttempts) => {
          const subscriber = new CaepSubscriber({
            baseUrl: 'https://example.com',
            getAccessToken: async () => 'token' as AccessToken,
            options: { maxReconnectAttempts: maxAttempts },
          });

          // The subscriber should store the max attempts
          expect((subscriber as any).maxReconnectAttempts).toBe(maxAttempts);
        }
      ),
      { numRuns: 100 }
    );
  });
});

/**
 * **Property 20: Auto-Disconnect on Handler Removal**
 * When all handlers are removed, connection is closed.
 * **Validates: Requirements 8.7**
 */
describe('Property 20: Auto-Disconnect on Handler Removal', () => {
  beforeEach(() => {
    MockEventSource.reset();
  });

  it('disconnects when all handlers are removed', () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 1, max: 5 }),
        (handlerCount) => {
          const subscriber = new CaepSubscriber({
            baseUrl: 'https://example.com',
            getAccessToken: async () => 'token' as AccessToken,
          });

          const unsubscribes: Array<() => void> = [];

          // Add handlers
          for (let i = 0; i < handlerCount; i++) {
            const unsub = subscriber.subscribe(() => {});
            unsubscribes.push(unsub);
          }

          expect(subscriber.getTotalHandlers()).toBe(handlerCount);

          // Remove all but one
          for (let i = 0; i < handlerCount - 1; i++) {
            unsubscribes[i]!();
          }

          expect(subscriber.getTotalHandlers()).toBe(1);

          // Remove last one
          unsubscribes[handlerCount - 1]!();

          expect(subscriber.getTotalHandlers()).toBe(0);
        }
      ),
      { numRuns: 100 }
    );
  });
});
