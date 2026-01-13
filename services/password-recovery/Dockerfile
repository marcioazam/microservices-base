# Build stage
FROM mcr.microsoft.com/dotnet/sdk:10.0-preview AS build
WORKDIR /src

# Copy csproj files and restore
COPY src/PasswordRecovery.Domain/PasswordRecovery.Domain.csproj src/PasswordRecovery.Domain/
COPY src/PasswordRecovery.Application/PasswordRecovery.Application.csproj src/PasswordRecovery.Application/
COPY src/PasswordRecovery.Infrastructure/PasswordRecovery.Infrastructure.csproj src/PasswordRecovery.Infrastructure/
COPY src/PasswordRecovery.Api/PasswordRecovery.Api.csproj src/PasswordRecovery.Api/
COPY PasswordRecovery.sln .

RUN dotnet restore

# Copy source code and build
COPY src/ src/
RUN dotnet publish src/PasswordRecovery.Api/PasswordRecovery.Api.csproj -c Release -o /app/publish --no-restore

# Runtime stage
FROM mcr.microsoft.com/dotnet/aspnet:10.0-preview AS runtime
WORKDIR /app

# Create non-root user for security
RUN groupadd -r appgroup && useradd -r -g appgroup appuser

# Copy published app
COPY --from=build /app/publish .

# Set ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health/live || exit 1

# Set environment variables
ENV ASPNETCORE_URLS=http://+:8080
ENV ASPNETCORE_ENVIRONMENT=Production

ENTRYPOINT ["dotnet", "PasswordRecovery.Api.dll"]
