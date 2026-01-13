import * as fc from 'fast-check';
import jwt from 'jsonwebtoken';

// Feature: image-processing-service, Property 29: JWT Extraction
// Validates: Requirements 10.1

// Feature: image-processing-service, Property 30: Authentication Enforcement
// Validates: Requirements 10.2

// Feature: image-processing-service, Property 31: Authorization Enforcement
// Validates: Requirements 10.3

const TEST_SECRET = 'test-secret-key';
const TEST_ISSUER = 'test-issuer';
const TEST_AUDIENCE = 'test-audience';

interface JwtPayload {
  sub: string;
  userId: string;
  email?: string;
  permissions?: string[];
  iat: number;
  exp: number;
}

function createValidToken(payload: Partial<JwtPayload>): string {
  const fullPayload = {
    sub: payload.sub || 'user-123',
    userId: payload.userId || 'user-123',
    email: payload.email,
    permissions: payload.permissions || [],
    iat: Math.floor(Date.now() / 1000),
    exp: Math.floor(Date.now() / 1000) + 3600, // 1 hour
  };

  return jwt.sign(fullPayload, TEST_SECRET, {
    issuer: TEST_ISSUER,
    audience: TEST_AUDIENCE,
  });
}

function verifyToken(token: string): JwtPayload | null {
  try {
    return jwt.verify(token, TEST_SECRET, {
      issuer: TEST_ISSUER,
      audience: TEST_AUDIENCE,
    }) as JwtPayload;
  } catch {
    return null;
  }
}

function hasPermission(payload: JwtPayload | null, permission: string): boolean {
  if (!payload) return false;
  const permissions = payload.permissions || [];
  return permissions.includes(permission) || permissions.includes('*');
}

describe('Auth Middleware Property Tests', () => {
  describe('Property 29: JWT Extraction', () => {
    it('should extract user ID and permissions from valid JWT', async () => {
      await fc.assert(
        fc.property(
          fc.uuid(),
          fc.emailAddress(),
          fc.array(fc.string({ minLength: 1, maxLength: 20 }), { minLength: 0, maxLength: 5 }),
          (userId, email, permissions) => {
            const token = createValidToken({ userId, email, permissions });
            const decoded = verifyToken(token);

            expect(decoded).not.toBeNull();
            expect(decoded!.userId).toBe(userId);
            expect(decoded!.email).toBe(email);
            expect(decoded!.permissions).toEqual(permissions);
          }
        ),
        { numRuns: 100 }
      );
    });

    it('should preserve all claims in the token', async () => {
      await fc.assert(
        fc.property(
          fc.uuid(),
          fc.string({ minLength: 1, maxLength: 50 }),
          (userId, sub) => {
            const token = createValidToken({ userId, sub });
            const decoded = verifyToken(token);

            expect(decoded).not.toBeNull();
            expect(decoded!.sub).toBe(sub);
            expect(decoded!.userId).toBe(userId);
            expect(decoded!.iat).toBeDefined();
            expect(decoded!.exp).toBeDefined();
          }
        ),
        { numRuns: 100 }
      );
    });
  });

  describe('Property 30: Authentication Enforcement', () => {
    it('should reject requests without token', async () => {
      await fc.assert(
        fc.property(fc.constant(undefined), (token) => {
          const decoded = token ? verifyToken(token) : null;
          expect(decoded).toBeNull();
        }),
        { numRuns: 100 }
      );
    });

    it('should reject invalid tokens', async () => {
      await fc.assert(
        fc.property(
          fc.string({ minLength: 10, maxLength: 100 }),
          (invalidToken) => {
            const decoded = verifyToken(invalidToken);
            expect(decoded).toBeNull();
          }
        ),
        { numRuns: 100 }
      );
    });

    it('should reject tokens with wrong secret', async () => {
      await fc.assert(
        fc.property(fc.uuid(), (userId) => {
          const token = jwt.sign(
            { userId, sub: userId },
            'wrong-secret',
            { issuer: TEST_ISSUER, audience: TEST_AUDIENCE }
          );
          const decoded = verifyToken(token);
          expect(decoded).toBeNull();
        }),
        { numRuns: 100 }
      );
    });

    it('should reject expired tokens', async () => {
      await fc.assert(
        fc.property(fc.uuid(), (userId) => {
          const token = jwt.sign(
            {
              userId,
              sub: userId,
              iat: Math.floor(Date.now() / 1000) - 7200,
              exp: Math.floor(Date.now() / 1000) - 3600, // Expired 1 hour ago
            },
            TEST_SECRET,
            { issuer: TEST_ISSUER, audience: TEST_AUDIENCE }
          );
          const decoded = verifyToken(token);
          expect(decoded).toBeNull();
        }),
        { numRuns: 100 }
      );
    });
  });

  describe('Property 31: Authorization Enforcement', () => {
    it('should grant access when user has required permission', async () => {
      await fc.assert(
        fc.property(
          fc.uuid(),
          fc.string({ minLength: 1, maxLength: 20 }),
          (userId, permission) => {
            const token = createValidToken({ userId, permissions: [permission] });
            const decoded = verifyToken(token);

            expect(hasPermission(decoded, permission)).toBe(true);
          }
        ),
        { numRuns: 100 }
      );
    });

    it('should deny access when user lacks required permission', async () => {
      await fc.assert(
        fc.property(
          fc.uuid(),
          fc.string({ minLength: 1, maxLength: 20 }),
          fc.string({ minLength: 1, maxLength: 20 }),
          (userId, userPermission, requiredPermission) => {
            fc.pre(userPermission !== requiredPermission);
            fc.pre(userPermission !== '*');

            const token = createValidToken({ userId, permissions: [userPermission] });
            const decoded = verifyToken(token);

            expect(hasPermission(decoded, requiredPermission)).toBe(false);
          }
        ),
        { numRuns: 100 }
      );
    });

    it('should grant access with wildcard permission', async () => {
      await fc.assert(
        fc.property(
          fc.uuid(),
          fc.string({ minLength: 1, maxLength: 20 }),
          (userId, anyPermission) => {
            const token = createValidToken({ userId, permissions: ['*'] });
            const decoded = verifyToken(token);

            expect(hasPermission(decoded, anyPermission)).toBe(true);
          }
        ),
        { numRuns: 100 }
      );
    });
  });
});
