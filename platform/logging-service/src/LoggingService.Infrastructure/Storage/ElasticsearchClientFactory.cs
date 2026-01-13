using Elastic.Clients.Elasticsearch;
using Elastic.Transport;
using LoggingService.Core.Configuration;
using Microsoft.Extensions.Options;

namespace LoggingService.Infrastructure.Storage;

/// <summary>
/// Factory for creating configured Elasticsearch clients.
/// </summary>
public static class ElasticsearchClientFactory
{
    /// <summary>
    /// Creates an Elasticsearch client with the specified options.
    /// </summary>
    public static ElasticsearchClient Create(ElasticSearchOptions options)
    {
        var nodes = options.Nodes.Select(n => new Uri(n)).ToArray();

        ElasticsearchClientSettings settings;

        if (nodes.Length == 1)
        {
            settings = new ElasticsearchClientSettings(nodes[0]);
        }
        else
        {
            var pool = new StaticNodePool(nodes);
            settings = new ElasticsearchClientSettings(pool);
        }

        settings = settings
            .DefaultIndex($"{options.IndexPrefix}-default")
            .EnableDebugMode()
            .PrettyJson(false)
            .RequestTimeout(TimeSpan.FromSeconds(30))
            .MaxRetryTimeout(TimeSpan.FromSeconds(60));

        // Configure authentication if provided
        if (!string.IsNullOrEmpty(options.Username) && !string.IsNullOrEmpty(options.Password))
        {
            settings = settings.Authentication(
                new BasicAuthentication(options.Username, options.Password));
        }

        return new ElasticsearchClient(settings);
    }

    /// <summary>
    /// Creates an Elasticsearch client from IOptions.
    /// </summary>
    public static ElasticsearchClient Create(IOptions<ElasticSearchOptions> options)
    {
        return Create(options.Value);
    }
}
