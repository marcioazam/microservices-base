# Build stage
FROM eclipse-temurin:21-jdk-alpine AS builder
WORKDIR /app

# Copy gradle files
COPY build.gradle settings.gradle ./
COPY gradle gradle
COPY gradlew ./

# Download dependencies
RUN ./gradlew dependencies --no-daemon

# Copy source and build
COPY src src
RUN ./gradlew bootJar --no-daemon -x test

# Runtime stage
FROM eclipse-temurin:21-jre-alpine
WORKDIR /app

# Security: run as non-root
RUN addgroup -g 1001 appgroup && adduser -u 1001 -G appgroup -D appuser
USER appuser

# Copy jar from builder
COPY --from=builder /app/build/libs/*.jar app.jar

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=60s --retries=3 \
    CMD wget -q --spider http://localhost:8080/actuator/health || exit 1

# JVM options for containers
ENV JAVA_OPTS="-XX:+UseContainerSupport -XX:MaxRAMPercentage=75.0 -XX:+UseG1GC"

EXPOSE 8080

ENTRYPOINT ["sh", "-c", "java $JAVA_OPTS -jar app.jar"]
