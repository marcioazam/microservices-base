#pragma once

/**
 * @file tracing.h
 * @brief Distributed tracing with W3C Trace Context propagation
 * 
 * Implements OpenTelemetry-compatible tracing with:
 * - W3C Trace Context (traceparent, tracestate) propagation
 * - Correlation ID linking traces and logs
 * - Span attributes and events
 * 
 * Requirements: 9.2, 9.3, 9.4
 */

#include <string>
#include <string_view>
#include <chrono>
#include <map>
#include <memory>
#include <random>
#include <optional>
#include <vector>
#include <functional>

namespace crypto {

// ============================================================================
// W3C Trace Context (Requirement 9.3)
// ============================================================================

/**
 * @brief W3C Trace Context for distributed tracing
 * 
 * Format: version-trace_id-span_id-flags
 * Example: 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01
 */
struct TraceContext {
    std::string trace_id;       // 32 hex chars (128-bit)
    std::string span_id;        // 16 hex chars (64-bit)
    std::string parent_span_id; // 16 hex chars (optional)
    bool sampled = true;
    
    // W3C tracestate for vendor-specific data
    std::map<std::string, std::string> tracestate;
    
    /**
     * @brief Parse from W3C traceparent header
     * @param traceparent The traceparent header value
     * @return Parsed TraceContext or nullopt if invalid
     */
    [[nodiscard]] static std::optional<TraceContext> parse(const std::string& traceparent);
    
    /**
     * @brief Parse tracestate header
     * @param tracestate The tracestate header value
     */
    void parseTracestate(const std::string& tracestate);
    
    /**
     * @brief Serialize to W3C traceparent header
     * @return traceparent header value
     */
    [[nodiscard]] std::string toTraceparent() const;
    
    /**
     * @brief Serialize tracestate to header value
     * @return tracestate header value
     */
    [[nodiscard]] std::string toTracestate() const;
    
    /**
     * @brief Get correlation_id for linking traces and logs
     * 
     * The correlation_id is derived from trace_id for consistency.
     * 
     * @return Correlation ID string
     */
    [[nodiscard]] std::string correlationId() const {
        return trace_id.empty() ? "" : trace_id.substr(0, 16);
    }
    
    /**
     * @brief Check if this context is valid
     */
    [[nodiscard]] bool isValid() const {
        return trace_id.size() == 32 && span_id.size() == 16;
    }
};

// ============================================================================
// Span Types
// ============================================================================

/**
 * @brief Span status codes
 */
enum class SpanStatus {
    UNSET,
    OK,
    ERROR
};

/**
 * @brief Span kind (OpenTelemetry compatible)
 */
enum class SpanKind {
    INTERNAL,
    SERVER,
    CLIENT,
    PRODUCER,
    CONSUMER
};

// ============================================================================
// Span
// ============================================================================

/**
 * @brief Represents a single operation in a trace
 */
class Span {
public:
    Span(const std::string& name, SpanKind kind, const TraceContext& context);
    ~Span();
    
    // Disable copy
    Span(const Span&) = delete;
    Span& operator=(const Span&) = delete;
    
    // ========================================================================
    // Attributes (Requirement 9.4)
    // ========================================================================
    
    void setAttribute(const std::string& key, const std::string& value);
    void setAttribute(const std::string& key, int64_t value);
    void setAttribute(const std::string& key, double value);
    void setAttribute(const std::string& key, bool value);
    
    /**
     * @brief Set correlation_id attribute for log linking
     * @param correlation_id The correlation ID
     */
    void setCorrelationId(const std::string& correlation_id);
    
    // ========================================================================
    // Events
    // ========================================================================
    
    void addEvent(const std::string& name,
                  const std::map<std::string, std::string>& attributes = {});
    
    // ========================================================================
    // Status
    // ========================================================================
    
    void setStatus(SpanStatus status, const std::string& description = "");
    
    // ========================================================================
    // Lifecycle
    // ========================================================================
    
    void end();
    
    // ========================================================================
    // Accessors
    // ========================================================================
    
    [[nodiscard]] TraceContext context() const { return context_; }
    [[nodiscard]] std::string correlationId() const { return context_.correlationId(); }
    [[nodiscard]] bool isEnded() const { return ended_; }
    [[nodiscard]] const std::string& name() const { return name_; }
    [[nodiscard]] SpanKind kind() const { return kind_; }
    [[nodiscard]] const std::map<std::string, std::string>& attributes() const { return attributes_; }

private:
    std::string name_;
    SpanKind kind_;
    TraceContext context_;
    std::chrono::steady_clock::time_point start_time_;
    std::chrono::steady_clock::time_point end_time_;
    std::map<std::string, std::string> attributes_;
    SpanStatus status_ = SpanStatus::UNSET;
    std::string status_description_;
    bool ended_ = false;
};

// ============================================================================
// Span Exporter Interface
// ============================================================================

class ISpanExporter {
public:
    virtual ~ISpanExporter() = default;
    virtual void exportSpan(const Span& span) = 0;
};

/**
 * @brief Console exporter for debugging
 */
class ConsoleSpanExporter : public ISpanExporter {
public:
    void exportSpan(const Span& span) override;
};

// ============================================================================
// Tracer
// ============================================================================

/**
 * @brief Tracer for creating and managing spans
 */
class Tracer {
public:
    explicit Tracer(const std::string& service_name);
    ~Tracer() = default;
    
    /**
     * @brief Create a new root span
     */
    [[nodiscard]] std::unique_ptr<Span> startSpan(
        const std::string& name,
        SpanKind kind = SpanKind::INTERNAL);
    
    /**
     * @brief Create a child span from parent context (Requirement 9.3)
     * 
     * Propagates W3C Trace Context from parent to child.
     */
    [[nodiscard]] std::unique_ptr<Span> startSpan(
        const std::string& name,
        const TraceContext& parent,
        SpanKind kind = SpanKind::INTERNAL);
    
    /**
     * @brief Create a span from incoming request headers
     * @param name Span name
     * @param traceparent W3C traceparent header value
     * @param tracestate W3C tracestate header value (optional)
     * @param kind Span kind
     */
    [[nodiscard]] std::unique_ptr<Span> startSpanFromHeaders(
        const std::string& name,
        const std::string& traceparent,
        const std::string& tracestate = "",
        SpanKind kind = SpanKind::SERVER);
    
    void addExporter(std::shared_ptr<ISpanExporter> exporter);
    
    [[nodiscard]] static std::string generateTraceId();
    [[nodiscard]] static std::string generateSpanId();
    [[nodiscard]] const std::string& serviceName() const { return service_name_; }

private:
    std::string service_name_;
    std::vector<std::shared_ptr<ISpanExporter>> exporters_;
};

// ============================================================================
// RAII Span Guard
// ============================================================================

/**
 * @brief RAII guard for automatic span lifecycle management
 */
class SpanGuard {
public:
    SpanGuard(Tracer& tracer, const std::string& name,
              SpanKind kind = SpanKind::INTERNAL);
    SpanGuard(Tracer& tracer, const std::string& name,
              const TraceContext& parent,
              SpanKind kind = SpanKind::INTERNAL);
    ~SpanGuard();
    
    SpanGuard(const SpanGuard&) = delete;
    SpanGuard& operator=(const SpanGuard&) = delete;
    
    [[nodiscard]] Span& span() { return *span_; }
    [[nodiscard]] TraceContext context() const { return span_->context(); }
    [[nodiscard]] std::string correlationId() const { return span_->correlationId(); }

private:
    std::unique_ptr<Span> span_;
};

// ============================================================================
// Global Tracer Access
// ============================================================================

/**
 * @brief Get the global tracer instance
 * @param service_name Service name (default: "crypto-service")
 * @return Reference to the tracer
 */
Tracer& getTracer(const std::string& service_name = "crypto-service");

} // namespace crypto
