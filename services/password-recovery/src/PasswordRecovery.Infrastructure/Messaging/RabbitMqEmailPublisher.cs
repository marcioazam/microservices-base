using System.Text;
using System.Text.Json;
using PasswordRecovery.Application.Interfaces;
using PasswordRecovery.Application.Messages;
using RabbitMQ.Client;

namespace PasswordRecovery.Infrastructure.Messaging;

public class RabbitMqEmailPublisher : IEmailPublisher, IAsyncDisposable
{
    private readonly IConnection _connection;
    private readonly IChannel _channel;
    private const string ExchangeName = "password-recovery";
    private const string RecoveryQueueName = "recovery-emails";
    private const string PasswordChangedQueueName = "password-changed-emails";

    public RabbitMqEmailPublisher(IConnection connection)
    {
        _connection = connection;
        _channel = _connection.CreateChannelAsync().GetAwaiter().GetResult();
        InitializeAsync().GetAwaiter().GetResult();
    }

    private async Task InitializeAsync()
    {
        await _channel.ExchangeDeclareAsync(ExchangeName, ExchangeType.Direct, durable: true);
        await _channel.QueueDeclareAsync(RecoveryQueueName, durable: true, exclusive: false, autoDelete: false);
        await _channel.QueueDeclareAsync(PasswordChangedQueueName, durable: true, exclusive: false, autoDelete: false);
        await _channel.QueueBindAsync(RecoveryQueueName, ExchangeName, "recovery");
        await _channel.QueueBindAsync(PasswordChangedQueueName, ExchangeName, "password-changed");
    }

    public async Task PublishRecoveryEmailAsync(RecoveryEmailMessage message, CancellationToken ct = default)
    {
        var json = JsonSerializer.Serialize(message);
        var body = Encoding.UTF8.GetBytes(json);

        var properties = new BasicProperties
        {
            Persistent = true,
            ContentType = "application/json",
            CorrelationId = message.CorrelationId.ToString()
        };

        await _channel.BasicPublishAsync(
            exchange: ExchangeName,
            routingKey: "recovery",
            mandatory: true,
            basicProperties: properties,
            body: body,
            cancellationToken: ct);
    }

    public async Task PublishPasswordChangedEmailAsync(PasswordChangedEmailMessage message, CancellationToken ct = default)
    {
        var json = JsonSerializer.Serialize(message);
        var body = Encoding.UTF8.GetBytes(json);

        var properties = new BasicProperties
        {
            Persistent = true,
            ContentType = "application/json",
            CorrelationId = message.CorrelationId.ToString()
        };

        await _channel.BasicPublishAsync(
            exchange: ExchangeName,
            routingKey: "password-changed",
            mandatory: true,
            basicProperties: properties,
            body: body,
            cancellationToken: ct);
    }

    public async ValueTask DisposeAsync()
    {
        await _channel.CloseAsync();
        await _connection.CloseAsync();
        GC.SuppressFinalize(this);
    }
}
