pub mod cache;
pub mod encrypted_cache;

// Legacy Redis module - deprecated, use CacheStorage
#[deprecated(since = "2.0.0", note = "Use CacheStorage with rust-common::CacheClient")]
pub mod redis;

pub use cache::CacheStorage;
pub use encrypted_cache::EncryptedCacheStorage;

// Re-export for backward compatibility during migration
#[allow(deprecated)]
pub use redis::RedisStorage;
