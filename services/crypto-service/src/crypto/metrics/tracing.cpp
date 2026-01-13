#include "crypto/metrics/tracing.h"
#include <sstream>
#include <iomanip>
#include <iostream>
#include <mutex>
#include <algorithm>

namespace crypto {

// ============================================================================
// TraceContext Implementation
// ============================================================================

std::optional<TraceContext> TraceContext::parse(const std::string& traceparent) {
    // Format: version-trace_id-span_id-flags
    // Example: 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01
    
    if (traceparent.size() < 55) {
        return std::nullopt;
    }
    
    // Validate version (must be 00)
    if (traceparent[0] != '0' || traceparent[1] != '0' || traceparent[2] != '-') {
        return std::nullopt;
    }
    
    TraceContext ctx;
    size_t pos = 3;
    
    // Extract trace_id (32 hex chars)
    ctx.trace_id = traceparent.substr(pos, 32);
    if (ctx.trace_id.size() != 32) return std::nullopt;
    
    // Validate all zeros is invalid
    if (std::all_of(ctx.trace_id.begin(), ctx.trace_id.end(), 
                    [](char c) { return c == '0'; })) {
        return std::nullopt;
    }
    
    pos += 33;  // 32 + '-'
    
    // Extract span_id (16 hex chars)
    ctx.span_id = traceparent.substr(pos, 16);
    if (ctx.span_id.size() != 16) return std::nullopt;
    
    // Validate all zeros is invalid
    if (std::all_of(ctx.span_id.begin(), ctx.span_id.end(),
                    [](char c) { return c == '0'; })) {
        return std::nullopt;
    }
    
    pos += 17;  // 16 + '-'
    
    // Extract flags (2 hex chars)
    if (pos + 2 <= traceparent.size()) {
        std::string flags = traceparent.substr(pos, 2);
        ctx.sampled = (flags[1] == '1');
    }
    
    return ctx;
}

void TraceContext::parseTracestate(const std::string& tracestate_header) {
    // Format: key1=value1,key2=value2
    if (tracestate_header.empty()) return;
    
    std::istringstream ss(tracestate_header);
    std::string pair;
    
    while (std::getline(ss, pair, ',')) {
        auto eq_pos = pair.find('=');
        if (eq_pos != std::string::npos) {
            std::string key = pair.substr(0, eq_pos);
            std::string value = pair.substr(eq_pos + 1);
            // Trim whitespace
            key.erase(0, key.find_first_not_of(" \t"));
            key.erase(key.find_last_not_of(" \t") + 1);
            value.erase(0, value.find_first_not_of(" \t"));
            value.erase(value.find_last_not_of(" \t") + 1);
            tracestate[key] = value;
        }
    }
}

std::string TraceContext::toTraceparent() const {
    std::ostringstream ss;
    ss << "00-" << trace_id << "-" << span_id << "-";
    ss << (sampled ? "01" : "00");
    return ss.str();
}

std::string TraceContext::toTracestate() const {
    if (tracestate.empty()) return "";
    
    std::ostringstream ss;
    bool first = true;
    for (const auto& [key, value] : tracestate) {
        if (!first) ss << ",";
        ss << key << "=" << value;
        first = false;
    }
    return ss.str();
}

// ============================================================================
// Span Implementation
// ============================================================================

Span::Span(const std::string& name, SpanKind kind, const TraceContext& context)
    : name_(name)
    , kind_(kind)
    , context_(context)
    , start_time_(std::chrono::steady_clock::now()) {
    // Auto-set correlation_id attribute
    if (!context_.trace_id.empty()) {
        attributes_["correlation_id"] = context_.correlationId();
    }
}

Span::~Span() {
    if (!ended_) {
        end();
    }
}

void Span::setAttribute(const std::string& key, const std::string& value) {
    attributes_[key] = value;
}

void Span::setAttribute(const std::string& key, int64_t value) {
    attributes_[key] = std::to_string(value);
}

void Span::setAttribute(const std::string& key, double value) {
    attributes_[key] = std::to_string(value);
}

void Span::setAttribute(const std::string& key, bool value) {
    attributes_[key] = value ? "true" : "false";
}

void Span::setCorrelationId(const std::string& correlation_id) {
    attributes_["correlation_id"] = correlation_id;
}

void Span::addEvent(const std::string& name,
                    const std::map<std::string, std::string>& attributes) {
    // In a full implementation, events would be stored and exported
    (void)name;
    (void)attributes;
}

void Span::setStatus(SpanStatus status, const std::string& description) {
    status_ = status;
    status_description_ = description;
}

void Span::end() {
    if (!ended_) {
        end_time_ = std::chrono::steady_clock::now();
        ended_ = true;
    }
}

// ============================================================================
// ConsoleSpanExporter Implementation
// ============================================================================

void ConsoleSpanExporter::exportSpan(const Span& span) {
    std::cout << "[TRACE] trace_id=" << span.context().trace_id 
              << " span_id=" << span.context().span_id
              << " correlation_id=" << span.correlationId()
              << " name=" << span.name()
              << std::endl;
}

// ============================================================================
// Tracer Implementation
// ============================================================================

Tracer::Tracer(const std::string& service_name)
    : service_name_(service_name) {}

std::unique_ptr<Span> Tracer::startSpan(const std::string& name, SpanKind kind) {
    TraceContext ctx;
    ctx.trace_id = generateTraceId();
    ctx.span_id = generateSpanId();
    ctx.sampled = true;
    
    auto span = std::make_unique<Span>(name, kind, ctx);
    span->setAttribute("service.name", service_name_);
    return span;
}

std::unique_ptr<Span> Tracer::startSpan(const std::string& name,
                                         const TraceContext& parent,
                                         SpanKind kind) {
    TraceContext ctx;
    ctx.trace_id = parent.trace_id;
    ctx.span_id = generateSpanId();
    ctx.parent_span_id = parent.span_id;
    ctx.sampled = parent.sampled;
    ctx.tracestate = parent.tracestate;  // Propagate tracestate
    
    auto span = std::make_unique<Span>(name, kind, ctx);
    span->setAttribute("service.name", service_name_);
    return span;
}

std::unique_ptr<Span> Tracer::startSpanFromHeaders(
    const std::string& name,
    const std::string& traceparent,
    const std::string& tracestate,
    SpanKind kind) {
    
    auto parent_ctx = TraceContext::parse(traceparent);
    if (parent_ctx) {
        if (!tracestate.empty()) {
            parent_ctx->parseTracestate(tracestate);
        }
        return startSpan(name, *parent_ctx, kind);
    }
    // No valid parent context, create root span
    return startSpan(name, kind);
}

void Tracer::addExporter(std::shared_ptr<ISpanExporter> exporter) {
    exporters_.push_back(std::move(exporter));
}

std::string Tracer::generateTraceId() {
    static std::random_device rd;
    static std::mt19937_64 gen(rd());
    static std::uniform_int_distribution<uint64_t> dis;
    
    std::ostringstream ss;
    ss << std::hex << std::setfill('0');
    ss << std::setw(16) << dis(gen);
    ss << std::setw(16) << dis(gen);
    return ss.str();
}

std::string Tracer::generateSpanId() {
    static std::random_device rd;
    static std::mt19937_64 gen(rd());
    static std::uniform_int_distribution<uint64_t> dis;
    
    std::ostringstream ss;
    ss << std::hex << std::setfill('0') << std::setw(16) << dis(gen);
    return ss.str();
}

// ============================================================================
// SpanGuard Implementation
// ============================================================================

SpanGuard::SpanGuard(Tracer& tracer, const std::string& name, SpanKind kind)
    : span_(tracer.startSpan(name, kind)) {}

SpanGuard::SpanGuard(Tracer& tracer, const std::string& name,
                     const TraceContext& parent, SpanKind kind)
    : span_(tracer.startSpan(name, parent, kind)) {}

SpanGuard::~SpanGuard() {
    span_->end();
}

// ============================================================================
// Global Tracer
// ============================================================================

static std::map<std::string, std::unique_ptr<Tracer>> tracers;
static std::mutex tracers_mutex;

Tracer& getTracer(const std::string& service_name) {
    std::lock_guard<std::mutex> lock(tracers_mutex);
    auto it = tracers.find(service_name);
    if (it == tracers.end()) {
        tracers[service_name] = std::make_unique<Tracer>(service_name);
    }
    return *tracers[service_name];
}

} // namespace crypto
