//! Generic event handler trait and implementations.

use crate::{CaepError, CaepEvent, CaepEventType};
use async_trait::async_trait;
use std::marker::PhantomData;

/// Generic event handler trait with associated types
#[async_trait]
pub trait EventHandler: Send + Sync {
    /// Output type for successful handling
    type Output;

    /// Handle the event and return result
    async fn handle(&self, event: &CaepEvent) -> Result<Self::Output, CaepError>;

    /// Check if this handler can process the event type
    fn can_handle(&self, event_type: &CaepEventType) -> bool;
}

/// Handler for session-revoked events
pub struct SessionRevokedHandler<S: SessionStore> {
    session_store: S,
}

/// Session store trait for dependency injection
#[async_trait]
pub trait SessionStore: Send + Sync {
    async fn terminate_session(&self, session_id: &str) -> Result<(), CaepError>;
    async fn terminate_user_sessions(&self, user_id: &str) -> Result<u64, CaepError>;
}

impl<S: SessionStore> SessionRevokedHandler<S> {
    pub fn new(session_store: S) -> Self {
        Self { session_store }
    }
}

#[async_trait]
impl<S: SessionStore> EventHandler for SessionRevokedHandler<S> {
    type Output = u64;

    async fn handle(&self, event: &CaepEvent) -> Result<Self::Output, CaepError> {
        match &event.subject {
            crate::SubjectIdentifier::SessionId { session_id } => {
                self.session_store.terminate_session(session_id).await?;
                Ok(1)
            }
            crate::SubjectIdentifier::IssSub { sub, .. } => {
                self.session_store.terminate_user_sessions(sub).await
            }
            _ => Err(CaepError::ProcessingError(
                "Unsupported subject format for session revocation".to_string(),
            )),
        }
    }

    fn can_handle(&self, event_type: &CaepEventType) -> bool {
        matches!(event_type, CaepEventType::SessionRevoked)
    }
}

/// Handler for credential-change events
pub struct CredentialChangeHandler<C: CredentialCache> {
    cache: C,
}

/// Credential cache trait for dependency injection
#[async_trait]
pub trait CredentialCache: Send + Sync {
    async fn invalidate(&self, user_id: &str) -> Result<(), CaepError>;
}

impl<C: CredentialCache> CredentialChangeHandler<C> {
    pub fn new(cache: C) -> Self {
        Self { cache }
    }
}

#[async_trait]
impl<C: CredentialCache> EventHandler for CredentialChangeHandler<C> {
    type Output = ();

    async fn handle(&self, event: &CaepEvent) -> Result<Self::Output, CaepError> {
        match &event.subject {
            crate::SubjectIdentifier::IssSub { sub, .. } => {
                self.cache.invalidate(sub).await
            }
            _ => Err(CaepError::ProcessingError(
                "Unsupported subject format for credential change".to_string(),
            )),
        }
    }

    fn can_handle(&self, event_type: &CaepEventType) -> bool {
        matches!(event_type, CaepEventType::CredentialChange)
    }
}

/// Generic event processor that dispatches to registered handlers
pub struct EventProcessor {
    handlers: Vec<Box<dyn EventHandler<Output = ()> + Send + Sync>>,
}

impl EventProcessor {
    pub fn new() -> Self {
        Self {
            handlers: Vec::new(),
        }
    }

    pub fn register<H>(mut self, handler: H) -> Self
    where
        H: EventHandler<Output = ()> + Send + Sync + 'static,
    {
        self.handlers.push(Box::new(handler));
        self
    }

    pub async fn process(&self, event: &CaepEvent) -> Result<usize, CaepError> {
        let mut handled = 0;
        for handler in &self.handlers {
            if handler.can_handle(&event.event_type) {
                handler.handle(event).await?;
                handled += 1;
            }
        }
        Ok(handled)
    }
}

impl Default for EventProcessor {
    fn default() -> Self {
        Self::new()
    }
}
