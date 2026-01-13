package com.authplatform.usersvc.api.filter;

import com.authplatform.usersvc.shared.security.SecurityUtils;
import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;
import org.springframework.stereotype.Component;
import org.springframework.web.filter.OncePerRequestFilter;

import java.io.IOException;

/**
 * Filter that manages correlation ID for request tracing.
 */
@Component
@Order(Ordered.HIGHEST_PRECEDENCE)
public class CorrelationIdFilter extends OncePerRequestFilter {

    public static final String CORRELATION_ID_HEADER = "X-Correlation-ID";
    
    private final SecurityUtils securityUtils;

    public CorrelationIdFilter(SecurityUtils securityUtils) {
        this.securityUtils = securityUtils;
    }

    @Override
    protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response, 
                                     FilterChain filterChain) throws ServletException, IOException {
        try {
            // Get or create correlation ID
            String providedId = request.getHeader(CORRELATION_ID_HEADER);
            String correlationId = securityUtils.getOrCreateCorrelationId(providedId);
            
            // Set MDC context
            securityUtils.setMdcContext(correlationId, null);
            
            // Add to response header
            response.setHeader(CORRELATION_ID_HEADER, correlationId);
            
            filterChain.doFilter(request, response);
        } finally {
            // Clear MDC context
            securityUtils.clearMdcContext();
        }
    }
}
