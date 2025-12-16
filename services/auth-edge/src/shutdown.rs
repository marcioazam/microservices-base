//! Graceful Shutdown Module
//!
//! Provides structured concurrency with JoinSet and signal handling.

use std::future::Future;
use std::pin::Pin;
use std::sync::Arc;
use std::time::Duration;

use tokio::signal;
use tokio::sync::{broadcast, watch};
use tokio::task::JoinSet;
use tracing::{info, warn, error};

/// Shutdown coordinator for graceful termination
pub struct ShutdownCoordinator {
    /// Broadcast sender for shutdown signal
    shutdown_tx: broadcast::Sender<()>,
    /// Watch channel for shutdown completion
    completion_tx: watch::Sender<bool>,
    /// JoinSet for tracking background tasks
    tasks: JoinSet<()>,
}

impl ShutdownCoordinator {
    /// Creates a new shutdown coordinator
    pub fn new() -> Self {
        let (shutdown_tx, _) = broadcast::channel(1);
        let (completion_tx, _) = watch::channel(false);
        
        Self {
            shutdown_tx,
            completion_tx,
            tasks: JoinSet::new(),
        }
    }

    /// Gets a shutdown receiver
    pub fn subscribe(&self) -> ShutdownSignal {
        ShutdownSignal {
            receiver: self.shutdown_tx.subscribe(),
        }
    }

    /// Spawns a background task that will be tracked
    pub fn spawn<F>(&mut self, name: &'static str, future: F)
    where
        F: Future<Output = ()> + Send + 'static,
    {
        let shutdown = self.subscribe();
        
        self.tasks.spawn(async move {
            tokio::select! {
                _ = future => {
                    info!(task = name, "Background task completed");
                }
                _ = shutdown.recv() => {
                    info!(task = name, "Background task cancelled by shutdown");
                }
            }
        });
    }

    /// Initiates graceful shutdown
    pub async fn shutdown(mut self, timeout: Duration) {
        info!("Initiating graceful shutdown");
        
        // Send shutdown signal
        let _ = self.shutdown_tx.send(());
        
        // Wait for tasks with timeout
        let shutdown_result = tokio::time::timeout(timeout, async {
            while let Some(result) = self.tasks.join_next().await {
                match result {
                    Ok(()) => info!("Task completed successfully"),
                    Err(e) => warn!(error = %e, "Task failed during shutdown"),
                }
            }
        })
        .await;

        match shutdown_result {
            Ok(()) => info!("All tasks completed gracefully"),
            Err(_) => {
                warn!("Shutdown timeout reached, aborting remaining tasks");
                self.tasks.abort_all();
            }
        }

        // Signal completion
        let _ = self.completion_tx.send(true);
        
        info!("Shutdown complete");
    }

    /// Returns the number of active tasks
    pub fn task_count(&self) -> usize {
        self.tasks.len()
    }
}

impl Default for ShutdownCoordinator {
    fn default() -> Self {
        Self::new()
    }
}

/// Shutdown signal receiver
pub struct ShutdownSignal {
    receiver: broadcast::Receiver<()>,
}

impl ShutdownSignal {
    /// Waits for shutdown signal
    pub async fn recv(mut self) {
        let _ = self.receiver.recv().await;
    }

    /// Checks if shutdown has been signaled (non-blocking)
    pub fn is_shutdown(&mut self) -> bool {
        self.receiver.try_recv().is_ok()
    }
}

/// Waits for SIGTERM or SIGINT
pub async fn wait_for_signal() {
    let ctrl_c = async {
        signal::ctrl_c()
            .await
            .expect("Failed to install Ctrl+C handler");
    };

    #[cfg(unix)]
    let terminate = async {
        signal::unix::signal(signal::unix::SignalKind::terminate())
            .expect("Failed to install SIGTERM handler")
            .recv()
            .await;
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {
            info!("Received Ctrl+C, initiating shutdown");
        }
        _ = terminate => {
            info!("Received SIGTERM, initiating shutdown");
        }
    }
}

/// Runs a server with graceful shutdown support
pub async fn run_with_graceful_shutdown<F, S>(
    server_future: F,
    mut shutdown_coordinator: ShutdownCoordinator,
    shutdown_timeout: Duration,
) where
    F: Future<Output = Result<(), S>> + Send,
    S: std::fmt::Display,
{
    tokio::select! {
        result = server_future => {
            match result {
                Ok(()) => info!("Server stopped normally"),
                Err(e) => error!(error = %e, "Server error"),
            }
        }
        _ = wait_for_signal() => {
            info!("Shutdown signal received");
        }
    }

    shutdown_coordinator.shutdown(shutdown_timeout).await;
}
