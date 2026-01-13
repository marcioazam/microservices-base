# Logging Service - Multi-stage build
# Build only - infrastructure managed in /deploy/docker/docker-compose.yml

# Build stage
FROM mcr.microsoft.com/dotnet/sdk:10.0 AS build
ARG BUILD_CONFIGURATION=Release
WORKDIR /src

COPY ["platform/logging-service/src/LoggingService.Api/LoggingService.Api.csproj", "src/LoggingService.Api/"]
COPY ["platform/logging-service/src/LoggingService.Core/LoggingService.Core.csproj", "src/LoggingService.Core/"]
COPY ["platform/logging-service/src/LoggingService.Infrastructure/LoggingService.Infrastructure.csproj", "src/LoggingService.Infrastructure/"]
COPY ["platform/logging-service/src/LoggingService.Worker/LoggingService.Worker.csproj", "src/LoggingService.Worker/"]
COPY ["platform/logging-service/Directory.Build.props", "."]
COPY ["platform/logging-service/Directory.Packages.props", "."]

RUN dotnet restore "src/LoggingService.Api/LoggingService.Api.csproj"
RUN dotnet restore "src/LoggingService.Worker/LoggingService.Worker.csproj"

COPY platform/logging-service/ .

WORKDIR "/src/src/LoggingService.Api"
RUN dotnet publish "LoggingService.Api.csproj" -c $BUILD_CONFIGURATION -o /app/publish/api /p:UseAppHost=false

WORKDIR "/src/src/LoggingService.Worker"
RUN dotnet publish "LoggingService.Worker.csproj" -c $BUILD_CONFIGURATION -o /app/publish/worker /p:UseAppHost=false

# Runtime - API
FROM mcr.microsoft.com/dotnet/aspnet:10.0 AS api
WORKDIR /app
EXPOSE 8080 8081

RUN groupadd -r appgroup && useradd -r -g appgroup appuser
USER appuser

COPY --from=build /app/publish/api .

ENV ASPNETCORE_URLS=http://+:8080
ENV DOTNET_RUNNING_IN_CONTAINER=true

ENTRYPOINT ["dotnet", "LoggingService.Api.dll"]

# Runtime - Worker
FROM mcr.microsoft.com/dotnet/aspnet:10.0 AS worker
WORKDIR /app

RUN groupadd -r appgroup && useradd -r -g appgroup appuser
USER appuser

COPY --from=build /app/publish/worker .

ENV DOTNET_RUNNING_IN_CONTAINER=true

ENTRYPOINT ["dotnet", "LoggingService.Worker.dll"]
