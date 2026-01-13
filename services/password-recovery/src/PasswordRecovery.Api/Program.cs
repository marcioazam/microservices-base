using FluentValidation;
using FluentValidation.AspNetCore;
using Microsoft.EntityFrameworkCore;
using OpenTelemetry.Metrics;
using OpenTelemetry.Resources;
using OpenTelemetry.Trace;
using PasswordRecovery.Api.Middleware;
using PasswordRecovery.Api.Services;
using PasswordRecovery.Api.Validators;
using PasswordRecovery.Application.Interfaces;
using PasswordRecovery.Application.Services;
using PasswordRecovery.Infrastructure.Data;
using PasswordRecovery.Infrastructure.Logging;
using PasswordRecovery.Infrastructure.Messaging;
using PasswordRecovery.Infrastructure.RateLimiting;
using PasswordRecovery.Infrastructure.Repositories;
using PasswordRecovery.Infrastructure.Security;
using RabbitMQ.Client;
using Serilog;
using Serilog.Events;
using StackExchange.Redis;

Log.Logger = new LoggerConfiguration()
    .MinimumLevel.Information()
    .MinimumLevel.Override("Microsoft", LogEventLevel.Warning)
    .MinimumLevel.Override("Microsoft.EntityFrameworkCore", LogEventLevel.Warning)
    .Enrich.FromLogContext()
    .Enrich.WithProperty("Service", "password-recovery")
    .WriteTo.Console(outputTemplate: 
        "[{Timestamp:HH:mm:ss} {Level:u3}] {Message:lj} {Properties:j}{NewLine}{Exception}")
    .CreateLogger();

var builder = WebApplication.CreateBuilder(args);
builder.Host.UseSerilog();

// Add controllers
builder.Services.AddControllers();
builder.Services.AddEndpointsApiExplorer();
builder.Services.AddOpenApi();

// Add FluentValidation
builder.Services.AddFluentValidationAutoValidation();
builder.Services.AddValidatorsFromAssemblyContaining<RecoveryRequestValidator>();

// Add DbContext
builder.Services.AddDbContext<PasswordRecoveryDbContext>(options =>
    options.UseNpgsql(builder.Configuration.GetConnectionString("PostgreSQL")));

// Add Redis
builder.Services.AddSingleton<IConnectionMultiplexer>(sp =>
    ConnectionMultiplexer.Connect(builder.Configuration.GetConnectionString("Redis") ?? "localhost:6379"));

// Add RabbitMQ
builder.Services.AddSingleton<IConnection>(sp =>
{
    var factory = new ConnectionFactory
    {
        HostName = builder.Configuration["RabbitMQ:Host"] ?? "localhost",
        UserName = builder.Configuration["RabbitMQ:Username"] ?? "guest",
        Password = builder.Configuration["RabbitMQ:Password"] ?? "guest"
    };
    return factory.CreateConnectionAsync().GetAwaiter().GetResult();
});

// Add OpenTelemetry
builder.Services.AddOpenTelemetry()
    .ConfigureResource(r => r.AddService("password-recovery"))
    .WithTracing(tracing => tracing
        .AddAspNetCoreInstrumentation()
        .AddHttpClientInstrumentation())
    .WithMetrics(metrics => metrics
        .AddAspNetCoreInstrumentation()
        .AddHttpClientInstrumentation()
        .AddPrometheusExporter());

// Add Infrastructure services
builder.Services.AddScoped<ITokenRepository, TokenRepository>();
builder.Services.AddScoped<IUserRepository, UserRepository>();
builder.Services.AddScoped<ITokenGenerator, TokenGenerator>();
builder.Services.AddScoped<IPasswordHasher, Argon2PasswordHasher>();
builder.Services.AddScoped<IRateLimiter, RedisRateLimiter>();
builder.Services.AddScoped<IEmailPublisher, RabbitMqEmailPublisher>();
builder.Services.AddScoped<IAuditLogger, AuditLogger>();

// Add Application services
builder.Services.AddSingleton(new RecoveryServiceOptions
{
    BaseUrl = builder.Configuration["Recovery:BaseUrl"] ?? "https://example.com",
    TokenValidityMinutes = int.Parse(builder.Configuration["Recovery:TokenValidityMinutes"] ?? "15")
});
builder.Services.AddScoped<IRecoveryService, RecoveryService>();

// Add Background Services
builder.Services.AddHostedService<TokenCleanupService>();

// Add Health Checks
builder.Services.AddHealthChecks()
    .AddNpgSql(builder.Configuration.GetConnectionString("PostgreSQL") ?? "")
    .AddRedis(builder.Configuration.GetConnectionString("Redis") ?? "localhost:6379")
    .AddRabbitMQ();

var app = builder.Build();

// Configure the HTTP request pipeline
if (app.Environment.IsDevelopment())
{
    app.MapOpenApi();
}

app.UseSerilogRequestLogging();
app.UseCorrelationId();
app.UseRouting();
app.MapControllers();
app.MapHealthChecks("/health/live");
app.MapHealthChecks("/health/ready");
app.MapPrometheusScrapingEndpoint();

app.Run();

public partial class Program { }
