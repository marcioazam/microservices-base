# Email Service Dockerfile - PHP 8.3 with gRPC
# State of Art December 2025

FROM php:8.3-fpm-alpine AS base

# Install system dependencies
RUN apk add --no-cache \
    linux-headers \
    zlib-dev \
    libzip-dev \
    icu-dev \
    oniguruma-dev \
    autoconf \
    g++ \
    make \
    curl \
    git

# Install PHP extensions
RUN docker-php-ext-install \
    pdo_mysql \
    zip \
    intl \
    mbstring \
    opcache \
    pcntl \
    sockets

# Install gRPC and protobuf extensions
RUN pecl install grpc protobuf \
    && docker-php-ext-enable grpc protobuf

# Install Composer
COPY --from=composer:2.7 /usr/bin/composer /usr/bin/composer

# Configure OPcache for production
RUN echo "opcache.enable=1" >> /usr/local/etc/php/conf.d/opcache.ini \
    && echo "opcache.memory_consumption=256" >> /usr/local/etc/php/conf.d/opcache.ini \
    && echo "opcache.interned_strings_buffer=16" >> /usr/local/etc/php/conf.d/opcache.ini \
    && echo "opcache.max_accelerated_files=20000" >> /usr/local/etc/php/conf.d/opcache.ini \
    && echo "opcache.validate_timestamps=0" >> /usr/local/etc/php/conf.d/opcache.ini \
    && echo "opcache.save_comments=1" >> /usr/local/etc/php/conf.d/opcache.ini \
    && echo "opcache.jit=1255" >> /usr/local/etc/php/conf.d/opcache.ini \
    && echo "opcache.jit_buffer_size=128M" >> /usr/local/etc/php/conf.d/opcache.ini

# PHP configuration
RUN echo "memory_limit=256M" >> /usr/local/etc/php/conf.d/php.ini \
    && echo "max_execution_time=30" >> /usr/local/etc/php/conf.d/php.ini \
    && echo "upload_max_filesize=10M" >> /usr/local/etc/php/conf.d/php.ini \
    && echo "post_max_size=10M" >> /usr/local/etc/php/conf.d/php.ini

# Create non-root user
RUN addgroup -g 1000 appgroup \
    && adduser -u 1000 -G appgroup -s /bin/sh -D appuser

WORKDIR /app

# Production stage
FROM base AS production

# Copy application files
COPY --chown=appuser:appgroup . .

# Install dependencies (production only)
RUN composer install --no-dev --optimize-autoloader --no-interaction

# Create log directory
RUN mkdir -p /var/log/email-service \
    && chown -R appuser:appgroup /var/log/email-service

# Switch to non-root user
USER appuser

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD php -r "echo 'OK';" || exit 1

EXPOSE 9000

CMD ["php-fpm"]

# Development stage
FROM base AS development

# Install development tools
RUN apk add --no-cache vim

# Copy application files
COPY --chown=appuser:appgroup . .

# Install all dependencies including dev
RUN composer install --no-interaction

# Create log directory
RUN mkdir -p /var/log/email-service \
    && chown -R appuser:appgroup /var/log/email-service

USER appuser

EXPOSE 9000

CMD ["php-fpm"]
