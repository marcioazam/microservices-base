/**
 * @file result_test.cpp
 * @brief Unit tests for Result type and error handling utilities
 * 
 * Requirements: 7.2
 */

#include <gtest/gtest.h>
#include <crypto/common/result.h>
#include <vector>
#include <string>

namespace crypto::test {

// ============================================================================
// ErrorCode Tests
// ============================================================================

TEST(ErrorCodeTest, ErrorCodeToStringReturnsCorrectValues) {
    EXPECT_EQ(error_code_to_string(ErrorCode::OK), "OK");
    EXPECT_EQ(error_code_to_string(ErrorCode::INVALID_INPUT), "INVALID_INPUT");
    EXPECT_EQ(error_code_to_string(ErrorCode::CRYPTO_ERROR), "CRYPTO_ERROR");
    EXPECT_EQ(error_code_to_string(ErrorCode::KEY_NOT_FOUND), "KEY_NOT_FOUND");
    EXPECT_EQ(error_code_to_string(ErrorCode::SERVICE_UNAVAILABLE), "SERVICE_UNAVAILABLE");
    EXPECT_EQ(error_code_to_string(ErrorCode::CACHE_MISS), "CACHE_MISS");
    EXPECT_EQ(error_code_to_string(ErrorCode::CONFIG_ERROR), "CONFIG_ERROR");
}

TEST(ErrorCodeTest, IsRetryableIdentifiesRetryableErrors) {
    // Retryable errors
    EXPECT_TRUE(is_retryable(ErrorCode::SERVICE_UNAVAILABLE));
    EXPECT_TRUE(is_retryable(ErrorCode::TIMEOUT));
    EXPECT_TRUE(is_retryable(ErrorCode::KMS_UNAVAILABLE));
    EXPECT_TRUE(is_retryable(ErrorCode::CACHE_UNAVAILABLE));
    EXPECT_TRUE(is_retryable(ErrorCode::LOGGING_UNAVAILABLE));
    
    // Non-retryable errors
    EXPECT_FALSE(is_retryable(ErrorCode::OK));
    EXPECT_FALSE(is_retryable(ErrorCode::INVALID_INPUT));
    EXPECT_FALSE(is_retryable(ErrorCode::CRYPTO_ERROR));
    EXPECT_FALSE(is_retryable(ErrorCode::KEY_NOT_FOUND));
    EXPECT_FALSE(is_retryable(ErrorCode::AUTHENTICATION_FAILED));
}

TEST(ErrorCodeTest, IsClientErrorIdentifiesClientErrors) {
    // Client errors
    EXPECT_TRUE(is_client_error(ErrorCode::INVALID_INPUT));
    EXPECT_TRUE(is_client_error(ErrorCode::INVALID_KEY_SIZE));
    EXPECT_TRUE(is_client_error(ErrorCode::AUTHENTICATION_FAILED));
    EXPECT_TRUE(is_client_error(ErrorCode::PERMISSION_DENIED));
    EXPECT_TRUE(is_client_error(ErrorCode::KEY_NOT_FOUND));
    
    // Server errors
    EXPECT_FALSE(is_client_error(ErrorCode::OK));
    EXPECT_FALSE(is_client_error(ErrorCode::INTERNAL_ERROR));
    EXPECT_FALSE(is_client_error(ErrorCode::SERVICE_UNAVAILABLE));
    EXPECT_FALSE(is_client_error(ErrorCode::CRYPTO_ERROR));
}

// ============================================================================
// Error Structure Tests
// ============================================================================

TEST(ErrorTest, ConstructorSetsFields) {
    Error err(ErrorCode::INVALID_INPUT, "test message", "corr-123");
    
    EXPECT_EQ(err.code, ErrorCode::INVALID_INPUT);
    EXPECT_EQ(err.message, "test message");
    EXPECT_EQ(err.correlation_id, "corr-123");
}

TEST(ErrorTest, DefaultConstructorValues) {
    Error err(ErrorCode::CRYPTO_ERROR);
    
    EXPECT_EQ(err.code, ErrorCode::CRYPTO_ERROR);
    EXPECT_TRUE(err.message.empty());
    EXPECT_TRUE(err.correlation_id.empty());
}

TEST(ErrorTest, IsRetryableDelegates) {
    Error retryable(ErrorCode::SERVICE_UNAVAILABLE);
    Error nonRetryable(ErrorCode::INVALID_INPUT);
    
    EXPECT_TRUE(retryable.is_retryable());
    EXPECT_FALSE(nonRetryable.is_retryable());
}

TEST(ErrorTest, IsClientErrorDelegates) {
    Error clientErr(ErrorCode::INVALID_INPUT);
    Error serverErr(ErrorCode::INTERNAL_ERROR);
    
    EXPECT_TRUE(clientErr.is_client_error());
    EXPECT_FALSE(serverErr.is_client_error());
}

TEST(ErrorTest, CodeStringReturnsCorrectValue) {
    Error err(ErrorCode::KEY_NOT_FOUND);
    EXPECT_EQ(err.code_string(), "KEY_NOT_FOUND");
}

TEST(ErrorTest, ToLogStringFormatsCorrectly) {
    Error errWithCorr(ErrorCode::CRYPTO_ERROR, "encryption failed", "req-456");
    EXPECT_EQ(errWithCorr.to_log_string(), 
              "[CRYPTO_ERROR] encryption failed (correlation_id=req-456)");
    
    Error errNoCorr(ErrorCode::INVALID_INPUT, "bad data");
    EXPECT_EQ(errNoCorr.to_log_string(), "[INVALID_INPUT] bad data");
}

TEST(ErrorTest, EqualityComparesCode) {
    Error err1(ErrorCode::CRYPTO_ERROR, "message 1");
    Error err2(ErrorCode::CRYPTO_ERROR, "message 2");
    Error err3(ErrorCode::INVALID_INPUT, "message 1");
    
    EXPECT_EQ(err1, err2);  // Same code, different message
    EXPECT_NE(err1, err3);  // Different code
}

// ============================================================================
// Result<T> Tests
// ============================================================================

TEST(ResultTest, OkCreatesSuccessResult) {
    auto result = Ok(42);
    
    EXPECT_TRUE(result.has_value());
    EXPECT_EQ(result.value(), 42);
    EXPECT_EQ(*result, 42);
}

TEST(ResultTest, OkWithVectorValue) {
    std::vector<uint8_t> data = {1, 2, 3, 4, 5};
    auto result = Ok(data);
    
    EXPECT_TRUE(result.has_value());
    EXPECT_EQ(result.value(), data);
}

TEST(ResultTest, OkWithStringValue) {
    auto result = Ok(std::string("hello"));
    
    EXPECT_TRUE(result.has_value());
    EXPECT_EQ(*result, "hello");
}

TEST(ResultTest, ErrCreatesErrorResult) {
    auto result = Err<int>(ErrorCode::INVALID_INPUT, "bad input");
    
    EXPECT_FALSE(result.has_value());
    EXPECT_EQ(result.error().code, ErrorCode::INVALID_INPUT);
    EXPECT_EQ(result.error().message, "bad input");
}

TEST(ResultTest, ErrWithCorrelationId) {
    auto result = Err<int>(ErrorCode::CRYPTO_ERROR, "failed", "corr-789");
    
    EXPECT_FALSE(result.has_value());
    EXPECT_EQ(result.error().correlation_id, "corr-789");
}

TEST(ResultTest, ErrFromErrorObject) {
    Error err(ErrorCode::KEY_NOT_FOUND, "key missing", "req-123");
    auto result = Err<std::string>(err);
    
    EXPECT_FALSE(result.has_value());
    EXPECT_EQ(result.error().code, ErrorCode::KEY_NOT_FOUND);
    EXPECT_EQ(result.error().message, "key missing");
    EXPECT_EQ(result.error().correlation_id, "req-123");
}

TEST(ResultTest, BoolConversion) {
    auto success = Ok(100);
    auto failure = Err<int>(ErrorCode::INTERNAL_ERROR);
    
    EXPECT_TRUE(static_cast<bool>(success));
    EXPECT_FALSE(static_cast<bool>(failure));
    
    if (success) {
        EXPECT_EQ(*success, 100);
    } else {
        FAIL() << "Expected success result";
    }
}

TEST(ResultTest, ValueOrReturnsValueOnSuccess) {
    auto result = Ok(42);
    EXPECT_EQ(result.value_or(0), 42);
}

TEST(ResultTest, ValueOrReturnsDefaultOnError) {
    auto result = Err<int>(ErrorCode::INVALID_INPUT);
    EXPECT_EQ(result.value_or(99), 99);
}

// ============================================================================
// Result<void> Tests
// ============================================================================

TEST(ResultVoidTest, OkCreatesSuccessResult) {
    auto result = Ok();
    
    EXPECT_TRUE(result.has_value());
}

TEST(ResultVoidTest, ErrCreatesErrorResult) {
    auto result = Err(ErrorCode::CRYPTO_ERROR, "operation failed");
    
    EXPECT_FALSE(result.has_value());
    EXPECT_EQ(result.error().code, ErrorCode::CRYPTO_ERROR);
}

TEST(ResultVoidTest, BoolConversion) {
    auto success = Ok();
    auto failure = Err(ErrorCode::INTERNAL_ERROR);
    
    EXPECT_TRUE(static_cast<bool>(success));
    EXPECT_FALSE(static_cast<bool>(failure));
}

// ============================================================================
// Result Combinator Tests
// ============================================================================

TEST(ResultCombinatorsTest, TransformAppliesFunctionOnSuccess) {
    auto result = Ok(10);
    auto transformed = transform(result, [](int x) { return x * 2; });
    
    EXPECT_TRUE(transformed.has_value());
    EXPECT_EQ(*transformed, 20);
}

TEST(ResultCombinatorsTest, TransformPreservesErrorOnFailure) {
    auto result = Err<int>(ErrorCode::INVALID_INPUT, "bad");
    auto transformed = transform(result, [](int x) { return x * 2; });
    
    EXPECT_FALSE(transformed.has_value());
    EXPECT_EQ(transformed.error().code, ErrorCode::INVALID_INPUT);
}

TEST(ResultCombinatorsTest, AndThenChainsOnSuccess) {
    auto result = Ok(5);
    auto chained = and_then(result, [](int x) -> Result<std::string> {
        return Ok(std::to_string(x * 2));
    });
    
    EXPECT_TRUE(chained.has_value());
    EXPECT_EQ(*chained, "10");
}

TEST(ResultCombinatorsTest, AndThenShortCircuitsOnError) {
    auto result = Err<int>(ErrorCode::CRYPTO_ERROR);
    bool called = false;
    auto chained = and_then(result, [&called](int x) -> Result<std::string> {
        called = true;
        return Ok(std::to_string(x));
    });
    
    EXPECT_FALSE(chained.has_value());
    EXPECT_FALSE(called);
    EXPECT_EQ(chained.error().code, ErrorCode::CRYPTO_ERROR);
}

TEST(ResultCombinatorsTest, OrElseProvidesFallbackOnError) {
    auto result = Err<int>(ErrorCode::CACHE_MISS);
    auto recovered = or_else(result, [](const Error& err) -> Result<int> {
        if (err.code == ErrorCode::CACHE_MISS) {
            return Ok(42);  // Default value on cache miss
        }
        return Err<int>(err);
    });
    
    EXPECT_TRUE(recovered.has_value());
    EXPECT_EQ(*recovered, 42);
}

TEST(ResultCombinatorsTest, OrElsePassesThroughOnSuccess) {
    auto result = Ok(100);
    bool called = false;
    auto recovered = or_else(result, [&called](const Error&) -> Result<int> {
        called = true;
        return Ok(0);
    });
    
    EXPECT_TRUE(recovered.has_value());
    EXPECT_FALSE(called);
    EXPECT_EQ(*recovered, 100);
}

// ============================================================================
// Edge Cases
// ============================================================================

TEST(ResultEdgeCasesTest, MoveSemantics) {
    std::vector<uint8_t> large_data(1000, 0x42);
    auto result = Ok(std::move(large_data));
    
    EXPECT_TRUE(result.has_value());
    EXPECT_EQ(result->size(), 1000);
    EXPECT_EQ((*result)[0], 0x42);
}

TEST(ResultEdgeCasesTest, EmptyStringMessage) {
    auto result = Err<int>(ErrorCode::UNKNOWN_ERROR, "");
    
    EXPECT_FALSE(result.has_value());
    EXPECT_TRUE(result.error().message.empty());
}

TEST(ResultEdgeCasesTest, ChainedTransformations) {
    auto result = Ok(2);
    
    auto final_result = transform(
        transform(
            transform(result, [](int x) { return x + 1; }),  // 3
            [](int x) { return x * 2; }                       // 6
        ),
        [](int x) { return std::to_string(x); }              // "6"
    );
    
    EXPECT_TRUE(final_result.has_value());
    EXPECT_EQ(*final_result, "6");
}

} // namespace crypto::test
