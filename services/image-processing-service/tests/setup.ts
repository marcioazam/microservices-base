// Jest setup file
import { jest } from '@jest/globals';

// Increase timeout for property-based tests
jest.setTimeout(30000);

// Mock environment variables for tests
process.env.NODE_ENV = 'test';
process.env.LOG_LEVEL = 'silent';
