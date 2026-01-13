// Property-based tests for Audit Logging System
// Validates: Requirements 10.1, 10.2, 10.3, 6.6

#include <rapidcheck.h>
#include <gtest/gtest.h>
#include "crypto/audit/audit_logger.h"
#include "crypto/keys/key_types.h"
#include <set>
#include <regex>

namespace crypto::test {

// Generator for random AuditOperation
rc::Gen<AuditOperation> genAuditOperation() {
    return rc::gen::element(
        AuditOperation::ENCRYPT,
        AuditOperation::DECRYPT,
        AuditOperation::RSA_ENCRYPT,
        AuditOperation::RSA_DECRYPT,
        AuditOperation::SIGN,
        AuditOperation::VERIFY,
        AuditOperation::KEY_GENERATE,
        AuditOperation::KEY_ROTATE,
        AuditOperation::KEY_DELETE,
        AuditOperation::KEY_ACCESS,
        AuditOperation::FILE_ENCRYPT,
        AuditOperation::FILE_DECRYPT
    );
}

// Generator for random KeyId
rc::Gen<KeyId> genKeyId() {
    return rc::gen::map(
        rc::gen::tuple(
            rc::gen::nonEmpty(rc::gen::string<std::string>()),
            rc::gen::inRange(1, 100)
        ),
        [](const std::tuple<std::string, int>& t) {
            return KeyId(std::get<0>(t), generateUUID(), std::get<1>(t));
        }
    );
}

// Generator for random IP address
rc::Gen<std::string> genIpAddress() {
    return rc::gen::map(
        rc::gen::tuple(
            rc::gen::inRange(0, 255),
            rc::gen::inRange(0, 255),
            rc::gen::inRange(0, 255),
            rc::gen::inRange(0, 255)
        ),
        [](const std::tuple<int, int, int, int>& t) {
            return std::to_string(std::get<0>(t)) + "." +
                   std::to_string(std::get<1>(t)) + "." +
                   std::to_string(std::get<2>(t)) + "." +
                   std::to_string(std::get<3>(t));
        }
    );
}

// Generator for random AuditEntry
rc::Gen<AuditEntry> genAuditEntry() {
    return rc::gen::build<AuditEntry>(
        rc::gen::set(&AuditEntry::correlation_id, rc::gen::nonEmpty(rc::gen::string<std::string>())),
        rc::gen::set(&AuditEntry::timestamp, rc::gen::just(std::chrono::system_clock::now())),
        rc::gen::set(&AuditEntry::operation, genAuditOperation()),
        rc::gen::set(&AuditEntry::key_id, genKeyId()),
        rc::gen::set(&AuditEntry::caller_identity, rc::gen::nonEmpty(rc::gen::string<std::string>())),
        rc::gen::set(&AuditEntry::caller_service, rc::gen::nonEmpty(rc::gen::string<std::string>())),
        rc::gen::set(&AuditEntry::success, rc::gen::arbitrary<bool>()),
        rc::gen::set(&AuditEntry::source_ip, genIpAddress())
    );
}

// Property 20: Audit Entry Completeness
// All audit entries must contain required fields
TEST(AuditPropertiesTest, AuditEntryCompleteness) {
    rc::check("All audit entries contain required fields", []() {
        InMemoryAuditLogger logger;
        
        auto correlation_id = *rc::gen::nonEmpty(rc::gen::string<std::string>());
        auto operation = *genAuditOperation();
        auto key_id = *genKeyId();
        auto caller_identity = *rc::gen::nonEmpty(rc::gen::string<std::string>());
        auto caller_service = *rc::gen::nonEmpty(rc::gen::string<std::string>());
        auto success = *rc::gen::arbitrary<bool>();
        auto source_ip = *genIpAddress();
        
        AuditEntry entry = AuditEntryBuilder()
            .setCorrelationId(correlation_id)
            .setOperation(operation)
            .setKeyId(key_id)
            .setCallerIdentity(caller_identity)
            .setCallerService(caller_service)
            .setSuccess(success)
            .setSourceIp(source_ip)
            .build();
        
        logger.logOperation(entry);
        
        auto entries = logger.getAllEntries();
        RC_ASSERT(entries.size() == 1);
        
        const auto& logged = entries[0];
        
        // Verify all required fields are present
        RC_ASSERT(!logged.correlation_id.empty());
        RC_ASSERT(logged.timestamp.time_since_epoch().count() > 0);
        RC_ASSERT(!logged.key_id.toString().empty());
        RC_ASSERT(!logged.caller_identity.empty());
        RC_ASSERT(!logged.caller_service.empty());
        RC_ASSERT(!logged.source_ip.empty());
        
        // Verify values match
        RC_ASSERT(logged.correlation_id == correlation_id);
        RC_ASSERT(logged.operation == operation);
        RC_ASSERT(logged.caller_identity == caller_identity);
        RC_ASSERT(logged.caller_service == caller_service);
        RC_ASSERT(logged.success == success);
        RC_ASSERT(logged.source_ip == source_ip);
    });
}

// Property: Audit entries never contain sensitive data
TEST(AuditPropertiesTest, NoSensitiveDataInLogs) {
    rc::check("Audit entries never contain plaintext, ciphertext, or key material", []() {
        InMemoryAuditLogger logger;
        
        // Generate some "sensitive" data patterns
        auto sensitive_patterns = std::vector<std::string>{
            "-----BEGIN PRIVATE KEY-----",
            "-----BEGIN RSA PRIVATE KEY-----",
            "-----BEGIN EC PRIVATE KEY-----",
            "password",
            "secret"
        };
        
        auto entry = *genAuditEntry();
        logger.logOperation(entry);
        
        auto entries = logger.getAllEntries();
        RC_ASSERT(entries.size() == 1);
        
        // Convert entry to JSON and check for sensitive patterns
        std::string json = entries[0].toJson();
        
        for (const auto& pattern : sensitive_patterns) {
            RC_ASSERT(json.find(pattern) == std::string::npos);
        }
    });
}

// Property: Audit query filters work correctly
TEST(AuditPropertiesTest, QueryFiltersWork) {
    rc::check("Audit query filters return correct results", []() {
        InMemoryAuditLogger logger;
        
        // Log multiple entries with different operations
        auto num_entries = *rc::gen::inRange(5, 20);
        auto target_operation = *genAuditOperation();
        int expected_count = 0;
        
        for (int i = 0; i < num_entries; ++i) {
            auto entry = *genAuditEntry();
            if (i % 3 == 0) {
                entry.operation = target_operation;
                expected_count++;
            }
            logger.logOperation(entry);
        }
        
        // Query by operation
        AuditQuery query;
        query.operation = target_operation;
        query.limit = 1000;
        
        auto result = logger.query(query);
        RC_ASSERT(result.has_value());
        
        // All returned entries should have the target operation
        for (const auto& entry : *result) {
            RC_ASSERT(entry.operation == target_operation);
        }
    });
}

// Property: Audit pagination works correctly
TEST(AuditPropertiesTest, PaginationWorks) {
    rc::check("Audit pagination returns correct subsets", []() {
        InMemoryAuditLogger logger;
        
        auto num_entries = *rc::gen::inRange(10, 50);
        
        for (int i = 0; i < num_entries; ++i) {
            auto entry = *genAuditEntry();
            logger.logOperation(entry);
        }
        
        auto page_size = *rc::gen::inRange(1, 10);
        auto offset = *rc::gen::inRange(0, num_entries - 1);
        
        AuditQuery query;
        query.limit = page_size;
        query.offset = offset;
        
        auto result = logger.query(query);
        RC_ASSERT(result.has_value());
        
        // Check result size
        size_t expected_size = std::min(
            static_cast<size_t>(page_size),
            static_cast<size_t>(num_entries - offset)
        );
        RC_ASSERT(result->size() == expected_size);
    });
}

// Property 16: Key Rotation Audit
// All key rotation operations must be logged
TEST(AuditPropertiesTest, KeyRotationAudit) {
    rc::check("Key rotation operations are properly logged", []() {
        InMemoryAuditLogger logger;
        
        auto key_id = *genKeyId();
        auto caller = *rc::gen::nonEmpty(rc::gen::string<std::string>());
        auto correlation_id = generateUUID();
        
        // Log key rotation
        AuditEntry entry = AuditEntryBuilder()
            .setCorrelationId(correlation_id)
            .setOperation(AuditOperation::KEY_ROTATE)
            .setKeyId(key_id)
            .setCallerIdentity(caller)
            .setCallerService("key-service")
            .setSuccess(true)
            .setSourceIp("127.0.0.1")
            .addMetadata("old_version", std::to_string(key_id.version))
            .addMetadata("new_version", std::to_string(key_id.version + 1))
            .build();
        
        logger.logOperation(entry);
        
        // Query for rotation events
        AuditQuery query;
        query.operation = AuditOperation::KEY_ROTATE;
        
        auto result = logger.query(query);
        RC_ASSERT(result.has_value());
        RC_ASSERT(result->size() == 1);
        
        const auto& logged = (*result)[0];
        RC_ASSERT(logged.operation == AuditOperation::KEY_ROTATE);
        RC_ASSERT(logged.correlation_id == correlation_id);
        RC_ASSERT(logged.metadata.count("old_version") > 0);
        RC_ASSERT(logged.metadata.count("new_version") > 0);
    });
}

// Property: JSON serialization is valid
TEST(AuditPropertiesTest, JsonSerializationValid) {
    rc::check("Audit entry JSON serialization produces valid JSON", []() {
        auto entry = *genAuditEntry();
        
        std::string json = entry.toJson();
        
        // Basic JSON structure validation
        RC_ASSERT(!json.empty());
        RC_ASSERT(json.front() == '{');
        RC_ASSERT(json.back() == '}');
        
        // Check required fields are present
        RC_ASSERT(json.find("\"correlation_id\"") != std::string::npos);
        RC_ASSERT(json.find("\"timestamp\"") != std::string::npos);
        RC_ASSERT(json.find("\"operation\"") != std::string::npos);
        RC_ASSERT(json.find("\"key_id\"") != std::string::npos);
        RC_ASSERT(json.find("\"caller_identity\"") != std::string::npos);
        RC_ASSERT(json.find("\"success\"") != std::string::npos);
    });
}

// Property: Timestamp is always set
TEST(AuditPropertiesTest, TimestampAlwaysSet) {
    rc::check("Audit entry timestamp is always set to current time", []() {
        auto before = std::chrono::system_clock::now();
        
        AuditEntry entry = AuditEntryBuilder()
            .setCorrelationId("test")
            .setOperation(AuditOperation::ENCRYPT)
            .setKeyId(KeyId("ns", "id", 1))
            .setCallerIdentity("caller")
            .setCallerService("service")
            .setSuccess(true)
            .setSourceIp("127.0.0.1")
            .build();
        
        auto after = std::chrono::system_clock::now();
        
        RC_ASSERT(entry.timestamp >= before);
        RC_ASSERT(entry.timestamp <= after);
    });
}

// Property: ScopedAuditLog logs on destruction
TEST(AuditPropertiesTest, ScopedAuditLogLogsOnDestruction) {
    rc::check("ScopedAuditLog logs entry when destroyed", []() {
        InMemoryAuditLogger logger;
        auto success = *rc::gen::arbitrary<bool>();
        
        {
            ScopedAuditLog scoped(logger, 
                AuditEntryBuilder()
                    .setCorrelationId("test")
                    .setOperation(AuditOperation::ENCRYPT)
                    .setKeyId(KeyId("ns", "id", 1))
                    .setCallerIdentity("caller")
                    .setCallerService("service")
                    .setSourceIp("127.0.0.1")
            );
            scoped.setSuccess(success);
        }
        
        auto entries = logger.getAllEntries();
        RC_ASSERT(entries.size() == 1);
        RC_ASSERT(entries[0].success == success);
    });
}

} // namespace crypto::test
