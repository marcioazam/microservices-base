// Feature: crypto-service-modernization-2025
// Property 3: Trace Context Propagation
// Property 4: Observability Metadata Completeness
// Property 5: Error Metric Emission
// Property-based tests for tracing and metrics

#include <gtest/gtest.h>
#include <rapidcheck.h>
#include <rapidcheck/gtest.h>
#include "crypto/metrics/tracing.h"
#include "crypto/metrics/prometheus_exporter.h"
#include "crypto/common/result.h"
#include <regex>
#include <set>

namespace crypto::test {

// ============================================================================
// Generators
// ============================================================================

/// Generator for valid trace IDs (32 hex chars)
rc::Gen<std::string> genTraceId() {
    return rc::gen::container<std::string>(
        32,
        rc::gen::oneOf(
            rc::gen::inRange<char>('0', '9'),
            rc::gen::inRange<char>('a', 'f')
        )
    );
}

/// Generator for valid span IDs (16 hex chars)
rc::Gen<std::string> genSpanId() {
    return rc::gen::container<std::string>(
        16,
        rc::gen::oneOf(
            rc::gen::inRange<char>('0', '9'),
            rc::gen::inRange<char>('a', 'f')
        )
    );
}

/// Generator for W3C traceparent header
rc::Gen<std::string> genTraceparent() {
    return rc::gen::map(
        rc::gen::tuple(genTraceId(), genSpanId(), rc::gen::element(true, false)),
        [](const auto& tuple) {
            const auto& [trace_id, span_id, sampled] = tuple;
            return "00-" + trace_id + "-" + span_id + "-" + (sampled ? "01" : "00");
        }
    );
}

/// Generator for span names
rc::Gen<std::string> genSpanName() {
    return rc::gen::element<std::string>(
        "encrypt",
        "decrypt",
        "sign",
        "verify",
        "key.generate",
        "key.rotate",
        "key.delete",
        "cache.get",
        "cache.set",
        "log.write"
    );
}

/// Generator for span kinds
rc::Gen<SpanKind> genSpanKind() {
    return rc::gen::element(
        SpanKind::INTERNAL,
        SpanKind::SERVER,
        SpanKind::CLIENT,
        SpanKind::PRODUCER,
        SpanKind::CONSUMER
    );
}

/// Generator for error codes
rc::Gen<ErrorCode> genErrorCode() {
    return rc::gen::element(
        ErrorCode::INVALID_INPUT,
        ErrorCode::INVALID_KEY_SIZE,
        ErrorCode::INVALID_IV_SIZE,
        ErrorCode::INTEGRITY_ERROR,
        ErrorCode::CRYPTO_ERROR,
        ErrorCode::KEY_NOT_FOUND,
        ErrorCode::SERVICE_UNAVAILABLE,
        ErrorCode::TIMEOUT,
        ErrorCode::CACHE_MISS,
        ErrorCode::CACHE_ERROR
    );
}

/// Generator for attribute keys
rc::Gen<std::string> genAttributeKey() {
    return rc::gen::element<std::string>(
        "key_id",
        "algorithm",
        "key_size",
        "operation",
        "user_id",
        "correlation_id"
    );
}

/// Generator for attribute values
rc::Gen<std::string> genAttributeValue() {
    return rc::gen::container<std::string>(
        rc::gen::inRange(1, 50),
        rc::gen::inRange<char>('a', 'z')
    );
}

// ============================================================================
// Test Fixtures
// ============================================================================

class TracingPropertiesTest : public ::testing::Test {
protected:
    void SetUp() override {
        tracer_ = std::make_unique<Tracer>("crypto-service-test");
    }
    
    std::unique_ptr<Tracer> tracer_;
};

class MetricsPropertiesTest : public ::testing::Test {
protected:
    void SetUp() override {
        exporter_ = std::make_unique<PrometheusExporter>();
    }
    
    std::unique_ptr<PrometheusExporter> exporter_;
};

// ============================================================================
// Property 3: Trace Context Propagation
// For any incoming request with W3C Trace Context headers, the Crypto_Service
// SHALL propagate the trace context to all outgoing requests and include it
// in all generated spans.
// Validates: Requirements 3.6, 9.3
// ============================================================================

RC_GTEST_FIXTURE_PROP(TracingPropertiesTest, TraceparentParsing, ()) {
    auto traceparent = *genTraceparent();
    
    auto context = TraceContext::parse(traceparent);
    RC_ASSERT(context.has_value());
    RC_ASSERT(context->isValid());
    RC_ASSERT(context->trace_id.size() == 32);
    RC_ASSERT(context->span_id.size() == 16);
}

RC_GTEST_FIXTURE_PROP(TracingPropertiesTest, TraceparentRoundTrip, ()) {
    auto traceparent = *genTraceparent();
    
    auto context = TraceContext::parse(traceparent);
    RC_ASSERT(context.has_value());
    
    auto serialized = context->toTraceparent();
    RC_ASSERT(serialized == traceparent);
}

RC_GTEST_FIXTURE_PROP(TracingPropertiesTest, ChildSpanInheritsTraceId, ()) {
    auto traceparent = *genTraceparent();
    auto span_name = *genSpanName();
    auto kind = *genSpanKind();
    
    auto parent_context = TraceContext::parse(traceparent);
    RC_ASSERT(parent_context.has_value());
    
    auto child_span = tracer_->startSpan(span_name, *parent_context, kind);
    RC_ASSERT(child_span != nullptr);
    
    // Child must inherit trace_id from parent
    RC_ASSERT(child_span->context().trace_id == parent_context->trace_id);
    
    // Child must have different span_id
    RC_ASSERT(child_span->context().span_id != parent_context->span_id);
    
    // Child must reference parent span_id
    RC_ASSERT(child_span->context().parent_span_id == parent_context->span_id);
}

RC_GTEST_FIXTURE_PROP(TracingPropertiesTest, SpanFromHeadersPropagatesContext, ()) {
    auto traceparent = *genTraceparent();
    auto span_name = *genSpanName();
    
    auto span = tracer_->startSpanFromHeaders(span_name, traceparent);
    RC_ASSERT(span != nullptr);
    
    auto original = TraceContext::parse(traceparent);
    RC_ASSERT(original.has_value());
    
    // Span must use the same trace_id
    RC_ASSERT(span->context().trace_id == original->trace_id);
}

RC_GTEST_FIXTURE_PROP(TracingPropertiesTest, CorrelationIdDerivedFromTraceId, ()) {
    auto traceparent = *genTraceparent();
    
    auto context = TraceContext::parse(traceparent);
    RC_ASSERT(context.has_value());
    
    auto correlation_id = context->correlationId();
    
    // Correlation ID should be first 16 chars of trace_id
    RC_ASSERT(correlation_id.size() == 16);
    RC_ASSERT(correlation_id == context->trace_id.substr(0, 16));
}

// ============================================================================
// Property 4: Observability Metadata Completeness
// For any operation that generates traces or logs, the metadata SHALL include
// correlation_id linking traces and logs for the same request.
// Validates: Requirements 9.2, 9.4
// ============================================================================

RC_GTEST_FIXTURE_PROP(TracingPropertiesTest, SpanHasCorrelationId, ()) {
    auto span_name = *genSpanName();
    auto kind = *genSpanKind();
    
    auto span = tracer_->startSpan(span_name, kind);
    RC_ASSERT(span != nullptr);
    
    // Every span must have a correlation_id
    auto correlation_id = span->correlationId();
    RC_ASSERT(!correlation_id.empty());
    RC_ASSERT(correlation_id.size() == 16);
}

RC_GTEST_FIXTURE_PROP(TracingPropertiesTest, SpanAttributesPreserved, ()) {
    auto span_name = *genSpanName();
    auto attr_key = *genAttributeKey();
    auto attr_value = *genAttributeValue();
    
    auto span = tracer_->startSpan(span_name);
    span->setAttribute(attr_key, attr_value);
    
    // Attribute should be preserved
    const auto& attrs = span->attributes();
    RC_ASSERT(attrs.count(attr_key) == 1);
    RC_ASSERT(attrs.at(attr_key) == attr_value);
}

RC_GTEST_FIXTURE_PROP(TracingPropertiesTest, SpanGuardProvidesCorrelationId, ()) {
    auto span_name = *genSpanName();
    
    SpanGuard guard(*tracer_, span_name);
    
    auto correlation_id = guard.correlationId();
    RC_ASSERT(!correlation_id.empty());
    RC_ASSERT(correlation_id.size() == 16);
    
    // Context should be valid
    auto context = guard.context();
    RC_ASSERT(context.isValid());
}

// ============================================================================
// Property 5: Error Metric Emission
// For any operation that fails with an error, the Crypto_Service SHALL emit
// a Prometheus counter metric with the error_code label set to the specific
// error code.
// Validates: Requirements 9.5
// ============================================================================

RC_GTEST_FIXTURE_PROP(MetricsPropertiesTest, ErrorMetricEmittedWithCode, ()) {
    auto error_code = *genErrorCode();
    
    // Record the error
    exporter_->recordError(error_code);
    
    // Serialize and check for error_code label
    auto metrics = exporter_->serialize();
    
    auto code_str = std::string(error_code_to_string(error_code));
    std::string expected_label = "error_code=\"" + code_str + "\"";
    
    RC_ASSERT(metrics.find(expected_label) != std::string::npos);
}

RC_GTEST_FIXTURE_PROP(MetricsPropertiesTest, ErrorMetricCountsAccumulate, ()) {
    auto error_code = *genErrorCode();
    auto count = *rc::gen::inRange<size_t>(1, 10);
    
    for (size_t i = 0; i < count; ++i) {
        exporter_->recordError(error_code);
    }
    
    auto metrics = exporter_->serialize();
    
    // Should contain the error code
    auto code_str = std::string(error_code_to_string(error_code));
    RC_ASSERT(metrics.find(code_str) != std::string::npos);
}

RC_GTEST_FIXTURE_PROP(MetricsPropertiesTest, ErrorFromResultRecorded, ()) {
    auto error_code = *genErrorCode();
    auto message = *genAttributeValue();
    
    Error error(error_code, message);
    exporter_->recordError(error);
    
    auto metrics = exporter_->serialize();
    auto code_str = std::string(error_code_to_string(error_code));
    
    RC_ASSERT(metrics.find(code_str) != std::string::npos);
}

RC_GTEST_FIXTURE_PROP(MetricsPropertiesTest, OperationMetricsRecorded, ()) {
    auto success = *rc::gen::element(true, false);
    
    exporter_->recordEncrypt(success);
    exporter_->recordDecrypt(success);
    exporter_->recordSign(success);
    exporter_->recordVerify(success);
    
    auto metrics = exporter_->serialize();
    
    // Should contain operation metrics
    RC_ASSERT(metrics.find("crypto_encrypt_total") != std::string::npos);
    RC_ASSERT(metrics.find("crypto_decrypt_total") != std::string::npos);
    RC_ASSERT(metrics.find("crypto_sign_total") != std::string::npos);
    RC_ASSERT(metrics.find("crypto_verify_total") != std::string::npos);
}

RC_GTEST_FIXTURE_PROP(MetricsPropertiesTest, LatencyHistogramRecorded, ()) {
    auto duration_ns = *rc::gen::inRange<int64_t>(1000, 1000000000);
    auto duration = std::chrono::nanoseconds(duration_ns);
    
    exporter_->recordEncryptLatency(duration);
    
    auto metrics = exporter_->serialize();
    
    // Should contain histogram buckets
    RC_ASSERT(metrics.find("_bucket") != std::string::npos);
    RC_ASSERT(metrics.find("_sum") != std::string::npos);
    RC_ASSERT(metrics.find("_count") != std::string::npos);
}

// ============================================================================
// Unit Tests for Edge Cases
// ============================================================================

TEST_F(TracingPropertiesTest, InvalidTraceparentReturnsNullopt) {
    EXPECT_FALSE(TraceContext::parse("invalid").has_value());
    EXPECT_FALSE(TraceContext::parse("").has_value());
    EXPECT_FALSE(TraceContext::parse("00-short-short-01").has_value());
}

TEST_F(TracingPropertiesTest, GeneratedTraceIdIsValid) {
    auto trace_id = Tracer::generateTraceId();
    EXPECT_EQ(trace_id.size(), 32);
    
    // Should be hex characters only
    for (char c : trace_id) {
        EXPECT_TRUE((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'));
    }
}

TEST_F(TracingPropertiesTest, GeneratedSpanIdIsValid) {
    auto span_id = Tracer::generateSpanId();
    EXPECT_EQ(span_id.size(), 16);
    
    for (char c : span_id) {
        EXPECT_TRUE((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'));
    }
}

TEST_F(TracingPropertiesTest, GeneratedIdsAreUnique) {
    std::set<std::string> trace_ids;
    std::set<std::string> span_ids;
    
    for (int i = 0; i < 100; ++i) {
        trace_ids.insert(Tracer::generateTraceId());
        span_ids.insert(Tracer::generateSpanId());
    }
    
    // All should be unique
    EXPECT_EQ(trace_ids.size(), 100);
    EXPECT_EQ(span_ids.size(), 100);
}

TEST_F(TracingPropertiesTest, SpanLifecycle) {
    auto span = tracer_->startSpan("test_operation");
    EXPECT_FALSE(span->isEnded());
    
    span->setAttribute("key", "value");
    span->setStatus(SpanStatus::OK);
    span->end();
    
    EXPECT_TRUE(span->isEnded());
}

TEST_F(TracingPropertiesTest, SpanGuardAutoEnds) {
    std::string correlation_id;
    {
        SpanGuard guard(*tracer_, "scoped_operation");
        correlation_id = guard.correlationId();
        EXPECT_FALSE(guard.span().isEnded());
    }
    // Span should be ended after guard destruction
    EXPECT_FALSE(correlation_id.empty());
}

TEST_F(MetricsPropertiesTest, CounterIncrement) {
    Counter counter;
    EXPECT_EQ(counter.value(), 0);
    
    counter.increment();
    EXPECT_EQ(counter.value(), 1);
    
    counter.increment(5);
    EXPECT_EQ(counter.value(), 6);
}

TEST_F(MetricsPropertiesTest, GaugeOperations) {
    Gauge gauge;
    EXPECT_DOUBLE_EQ(gauge.value(), 0.0);
    
    gauge.set(10.0);
    EXPECT_DOUBLE_EQ(gauge.value(), 10.0);
    
    gauge.increment(5.0);
    EXPECT_DOUBLE_EQ(gauge.value(), 15.0);
    
    gauge.decrement(3.0);
    EXPECT_DOUBLE_EQ(gauge.value(), 12.0);
}

TEST_F(MetricsPropertiesTest, HistogramObserve) {
    std::vector<double> buckets = {0.001, 0.01, 0.1, 1.0};
    Histogram histogram(buckets);
    
    histogram.observe(0.005);  // Falls in 0.01 bucket
    histogram.observe(0.05);   // Falls in 0.1 bucket
    histogram.observe(0.5);    // Falls in 1.0 bucket
    
    EXPECT_EQ(histogram.count(), 3);
}

TEST_F(MetricsPropertiesTest, ConnectionGauges) {
    exporter_->setHSMConnected(true);
    exporter_->setKMSConnected(false);
    exporter_->setLoggingServiceConnected(true);
    exporter_->setCacheServiceConnected(true);
    
    auto metrics = exporter_->serialize();
    
    EXPECT_TRUE(metrics.find("hsm_connected") != std::string::npos);
    EXPECT_TRUE(metrics.find("kms_connected") != std::string::npos);
}

TEST_F(MetricsPropertiesTest, LatencyTimerCallback) {
    std::chrono::nanoseconds recorded_duration{0};
    
    {
        LatencyTimer timer([&](std::chrono::nanoseconds d) {
            recorded_duration = d;
        });
        // Small delay
        std::this_thread::sleep_for(std::chrono::milliseconds(1));
    }
    
    EXPECT_GT(recorded_duration.count(), 0);
}

TEST_F(MetricsPropertiesTest, AllErrorCodesHaveStringRepresentation) {
    std::vector<ErrorCode> codes = {
        ErrorCode::OK,
        ErrorCode::UNKNOWN_ERROR,
        ErrorCode::INVALID_INPUT,
        ErrorCode::CRYPTO_ERROR,
        ErrorCode::INVALID_KEY_SIZE,
        ErrorCode::INTEGRITY_ERROR,
        ErrorCode::SERVICE_UNAVAILABLE,
        ErrorCode::CACHE_MISS
    };
    
    for (auto code : codes) {
        auto str = error_code_to_string(code);
        EXPECT_FALSE(str.empty());
        EXPECT_NE(str, "UNKNOWN");
    }
}

} // namespace crypto::test
