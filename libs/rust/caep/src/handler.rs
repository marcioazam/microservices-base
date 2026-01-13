//! Generic event handler trait and implementations.
//!
//! This module provides event handling using native async traits (Rust 2024).

use crate::{CaepError, CaepEvent, CaepEventType, CaepResult, SubjectIdentifier};
use std::future::Future;

/// Generic event handler trait with associated types.
///
/// Uses native async traits (Rust 2024) instead of async-trait macro.
pub trait EventHandler: Send + Sync {
    /// Output type for successful handling
    type Output;

    /// Handle the event and return result.
    fn handle(&self, event: &CaepEvent) -> impl Future<Output = CaepResult<Self::Output>> + Send;

    /// Check if this handler can process the event type.
    fn can_handle(&self, event_type: &CaepEventType) -> bool;
}

/// Session store trait for dependency injection.
pub trait SessionStore: Send + Sync {
    /// Terminate a specific session.
    fn terminate_session(&self, session_id: &str) -> impl Future<Output = CaepResult<()>> + Send;

    /// Terminate all sessions for a user.
    fn terminate_user_sessions(&self, user_id: &str) -> impl Future<Output = CaepResult<u64>> + Send;
}

/// Handler for session-revoked events.
pub struct SessionRevokedHandler<S: SessionStore> {
    session_store: S,
}

impl<S: SessionStore> SessionRevokedHandler<S> {
    /// Create a new session revoked handler.
    #[must_use]
    pub const fn new(session_store: S) -> Self {
        Self { session_store }
    }
}

impl<S: SessionStore> EventHandler for SessionRevokedHandler<S> {
    type Output = u64;

    async fn handle(&self, event: &CaepEvent) -> CaepResult<Self::Output> {
        match &event.subject {
            SubjectIdentifier::SessionId { session_id } => {
                self.session_store.terminate_session(session_id).await?;
                Ok(1)
            }
            SubjectIdentifier::IssSub { sub, .. } => {
                self.session_store.terminate_user_sessions(sub).await
            }
            _ => Err(CaepError::processing(
                "Unsupported subject format for session revocation",
            )),
        }
    }

    fn can_handle(&self, event_type: &CaepEventType) -> bool {
        matches!(event_type, CaepEventType::SessionRevoked)
    }
}

/// Credential cache trait for dependency injection.
pub trait CredentialCache: Send + Sync {
    /// Invalidate cached credentials for a user.
    fn invalidate(&self, user_id: &str) -> impl Future<Output = CaepResult<()>> + Send;
}

/// Handler for credential-change events.
pub struct CredentialChangeHandler<C: CredentialCache> {
    cache: C,
}

impl<C: CredentialCache> CredentialChangeHandler<C> {
    /// Create a new credential change handler.
    #[must_use]
    pub const fn new(cache: C) -> Self {
        Self { cache }
    }
}

impl<C: CredentialCache> EventHandler for CredentialChangeHandler<C> {
    type Output = ();

    async fn handle(&self, event: &CaepEvent) -> CaepResult<Self::Output> {
        match &event.subject {
            SubjectIdentifier::IssSub { sub, .. } => self.cache.invalidate(sub).await,
            _ => Err(CaepError::processing(
                "Unsupported subject format for credential change",
            )),
        }
    }

    fn can_handle(&self, event_type: &CaepEventType) -> bool {
        matches!(event_type, CaepEventType::CredentialChange)
    }
}

/// A boxed event handler for dynamic dispatch.
pub type BoxedHandler = Box<dyn DynEventHandler + Send + Sync>;

/// Dynamic event handler trait for type erasure.
pub trait DynEventHandler: Send + Sync {
    /// Handle the event.
    fn handle_dyn(
        &self,
        event: &CaepEvent,
    ) -> std::pin::Pin<Box<dyn Future<Output = CaepResult<()>> + Send + '_>>;

    /// Check if this handler can process the event type.
    fn can_handle(&self, event_type: &CaepEventType) -> bool;
}

/// Generic event processor that dispatches to registered handlers.
pub struct EventProcessor {
    handlers: Vec<BoxedHandler>,
}

impl EventProcessor {
    /// Create a new event processor.
    #[must_use]
    pub fn new() -> Self {
        Self {
            handlers: Vec::new(),
        }
    }

    /// Register a handler.
    pub fn register(&mut self, handler: BoxedHandler) {
        self.handlers.push(handler);
    }

    /// Process an event through all matching handlers.
    ///
    /// # Errors
    ///
    /// Returns an error if any handler fails.
    pub async fn process(&self, event: &CaepEvent) -> CaepResult<usize> {
        let mut handled = 0;
        for handler in &self.handlers {
            if handler.can_handle(&event.event_type) {
                handler.handle_dyn(event).await?;
                handled += 1;
            }
        }
        Ok(handled)
    }

    /// Get the number of registered handlers.
    #[must_use]
    pub fn handler_count(&self) -> usize {
        self.handlers.len()
    }
}

impl Default for EventProcessor {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::{AtomicU64, Ordering};
    use std::sync::Arc;

    struct MockSessionStore {
        terminated_count: Arc<AtomicU64>,
    }

    impl MockSessionStore {
        fn new() -> Self {
            Self {
                terminated_count: Arc::new(AtomicU64::new(0)),
            }
        }
    }

    impl SessionStore for MockSessionStore {
        async fn terminate_session(&self, _session_id: &str) -> CaepResult<()> {
            self.terminated_count.fetch_add(1, Ordering::SeqCst);
            Ok(())
        }

        async fn terminate_user_sessions(&self, _user_id: &str) -> CaepResult<u64> {
            let count = 5;
            self.terminated_count.fetch_add(count, Ordering::SeqCst);
            Ok(count)
        }
    }

    #[tokio::test]
    async fn test_session_revoked_handler() {
        let store = MockSessionStore::new();
        let handler = SessionRevokedHandler::new(store);

        let subject = SubjectIdentifier::session_id("session-123");
        let event = CaepEvent::session_revoked(subject, None);

        assert!(handler.can_handle(&CaepEventType::SessionRevoked));
        assert!(!handler.can_handle(&CaepEventType::CredentialChange));

        let result = handler.handle(&event).await;
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), 1);
    }

    #[tokio::test]
    async fn test_session_revoked_handler_user_sessions() {
        let store = MockSessionStore::new();
        let handler = SessionRevokedHandler::new(store);

        let subject = SubjectIdentifier::iss_sub("https://issuer.com", "user-123");
        let event = CaepEvent::session_revoked(subject, None);

        let result = handler.handle(&event).await;
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), 5);
    }
}
