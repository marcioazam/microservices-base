// Package main provides the entry point for IAM Policy Service.
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/auth-platform/iam-policy-service/internal/cache"
	"github.com/auth-platform/iam-policy-service/internal/caep"
	"github.com/auth-platform/iam-policy-service/internal/config"
	"github.com/auth-platform/iam-policy-service/internal/crypto"
	iamgrpc "github.com/auth-platform/iam-policy-service/internal/grpc"
	"github.com/auth-platform/iam-policy-service/internal/grpc/handlers"
	"github.com/auth-platform/iam-policy-service/internal/health"
	"github.com/auth-platform/iam-policy-service/internal/logging"
	"github.com/auth-platform/iam-policy-service/internal/observability"
	"github.com/auth-platform/iam-policy-service/internal/policy"
	"github.com/auth-platform/iam-policy-service/internal/rbac"
	"github.com/auth-platform/iam-policy-service/internal/server"
	"github.com/auth-platform/iam-policy-service/internal/service"
	pb "github.com/auth-platform/iam-policy-service/proto"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	logger, err := logging.NewLogger(cfg.Logging)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Close()

	logger.Info(ctx, "starting IAM Policy Service",
		logging.String("version", "2.0.0"),
		logging.String("host", cfg.Host),
		logging.Int("port", cfg.Port))

	// Initialize components
	deps, err := initializeDependencies(ctx, cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize dependencies: %w", err)
	}
	defer deps.Close()

	// Create shutdown manager
	shutdownMgr := server.NewShutdownManager(server.ShutdownConfig{
		Timeout: cfg.ShutdownTimeout,
		Logger:  logger,
	})

	// Register shutdown hooks
	registerShutdownHooks(shutdownMgr, deps)

	// Start servers
	if err := startServers(ctx, cfg, deps, shutdownMgr); err != nil {
		return err
	}

	// Wait for shutdown signal
	return shutdownMgr.WaitForSignal(ctx)
}

type dependencies struct {
	logger         *logging.Logger
	cache          *cache.DecisionCache
	encryptedCache *cache.EncryptedDecisionCache
	cryptoClient   *crypto.Client
	cryptoMetrics  *crypto.Metrics
	signer         *crypto.DecisionSigner
	engine         *policy.Engine
	hierarchy      *rbac.RoleHierarchy
	emitter        *caep.Emitter
	authService    *service.AuthorizationService
	healthMgr      *health.Manager
	metrics        *observability.Metrics
	grpcServer     *grpc.Server
	httpServer     *http.Server
}

func (d *dependencies) Close() {
	if d.cryptoClient != nil {
		d.cryptoClient.Close()
	}
	if d.encryptedCache != nil {
		d.encryptedCache.Close()
	} else if d.cache != nil {
		d.cache.Close()
	}
	if d.logger != nil {
		d.logger.Flush()
	}
}

func initializeDependencies(ctx context.Context, cfg *config.Config, logger *logging.Logger) (*dependencies, error) {
	deps := &dependencies{
		logger:    logger,
		hierarchy: rbac.NewRoleHierarchy(),
		healthMgr: health.NewManager(),
		metrics:   observability.NewMetrics(),
	}

	// Initialize cache
	decisionCache, err := cache.NewDecisionCache(cfg.Cache)
	if err != nil {
		logger.Warn(ctx, "cache initialization failed, using local fallback", logging.Error(err))
	}
	deps.cache = decisionCache

	// Initialize crypto components
	deps.cryptoMetrics = crypto.NewMetrics(nil)
	cryptoClient, encryptedCache, signer := initializeCrypto(ctx, cfg, decisionCache, logger, deps.cryptoMetrics)
	deps.cryptoClient = cryptoClient
	deps.encryptedCache = encryptedCache
	deps.signer = signer

	// Determine which cache to use for policy engine
	var policyCache policy.CacheInterface
	if encryptedCache != nil {
		policyCache = encryptedCache
	} else {
		policyCache = decisionCache
	}

	// Initialize policy engine
	engine, err := policy.NewEngine(policy.EngineConfig{
		PolicyPath: cfg.PolicyPath,
		Cache:      policyCache,
		Logger:     logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize policy engine: %w", err)
	}
	deps.engine = engine

	// Initialize CAEP emitter
	deps.emitter = caep.NewEmitter(caep.EmitterConfig{
		Enabled:      cfg.CAEP.Enabled,
		Transmitter:  cfg.CAEP.TransmitterURL,
		ServiceToken: cfg.CAEP.ServiceToken,
		Issuer:       cfg.CAEP.Issuer,
		Logger:       logger,
	})

	// Initialize authorization service
	deps.authService = service.NewAuthorizationService(service.AuthorizationServiceConfig{
		Engine:    engine,
		Hierarchy: deps.hierarchy,
		Emitter:   deps.emitter,
		Signer:    signer,
		Logger:    logger,
	})

	// Configure health manager with crypto check
	if cryptoClient != nil && cryptoClient.IsEnabled() {
		deps.healthMgr.SetCryptoHealthCheck(func(ctx context.Context) *health.CryptoHealthStatus {
			status, err := cryptoClient.HealthCheck(ctx)
			if err != nil {
				return &health.CryptoHealthStatus{Connected: false}
			}
			return &health.CryptoHealthStatus{
				Connected: status.Connected,
				LatencyMs: status.LatencyMs,
			}
		})
	}

	return deps, nil
}

func initializeCrypto(
	ctx context.Context,
	cfg *config.Config,
	decisionCache *cache.DecisionCache,
	logger *logging.Logger,
	metrics *crypto.Metrics,
) (*crypto.Client, *cache.EncryptedDecisionCache, *crypto.DecisionSigner) {
	if cfg.Crypto == nil || !cfg.Crypto.Enabled {
		logger.Info(ctx, "crypto integration disabled")
		return nil, nil, nil
	}

	// Parse key IDs
	encryptionKeyID, err := crypto.ParseKeyID(cfg.Crypto.EncryptionKeyID)
	if err != nil {
		logger.Warn(ctx, "invalid encryption key ID, crypto disabled",
			logging.String("key_id", cfg.Crypto.EncryptionKeyID),
			logging.Error(err))
		return nil, nil, nil
	}

	signingKeyID, err := crypto.ParseKeyID(cfg.Crypto.SigningKeyID)
	if err != nil {
		logger.Warn(ctx, "invalid signing key ID, crypto disabled",
			logging.String("key_id", cfg.Crypto.SigningKeyID),
			logging.Error(err))
		return nil, nil, nil
	}

	// Create crypto client
	cryptoClient, err := crypto.NewClient(crypto.ClientConfig{
		Address:         cfg.Crypto.Address,
		Timeout:         cfg.Crypto.Timeout,
		EncryptionKeyID: encryptionKeyID,
		SigningKeyID:    signingKeyID,
		KeyCacheTTL:     cfg.Crypto.KeyCacheTTL,
		Enabled:         cfg.Crypto.Enabled,
		CacheEncryption: cfg.Crypto.CacheEncryption,
		DecisionSigning: cfg.Crypto.DecisionSigning,
	}, logger, metrics)
	if err != nil {
		logger.Warn(ctx, "failed to create crypto client, operating without crypto",
			logging.Error(err))
		return nil, nil, nil
	}

	logger.Info(ctx, "crypto client initialized",
		logging.Bool("connected", cryptoClient.IsConnected()),
		logging.Bool("cache_encryption", cfg.Crypto.CacheEncryption),
		logging.Bool("decision_signing", cfg.Crypto.DecisionSigning))

	// Create encrypted cache if enabled
	var encryptedCache *cache.EncryptedDecisionCache
	if cfg.Crypto.CacheEncryption && decisionCache != nil {
		encryptedCache = cache.NewEncryptedDecisionCache(decisionCache, cryptoClient, logger)
		logger.Info(ctx, "encrypted decision cache enabled",
			logging.Bool("encryption_active", encryptedCache.IsEncryptionEnabled()))
	}

	// Create decision signer if enabled
	var signer *crypto.DecisionSigner
	if cfg.Crypto.DecisionSigning {
		signer = crypto.NewDecisionSigner(cryptoClient, logger)
		logger.Info(ctx, "decision signer enabled",
			logging.Bool("signing_active", signer.IsEnabled()))
	}

	return cryptoClient, encryptedCache, signer
}

func registerShutdownHooks(mgr *server.ShutdownManager, deps *dependencies) {
	mgr.RegisterHook(server.NewHealthCheckHook(deps.healthMgr.SetShuttingDown))
	mgr.RegisterHook(server.NewGRPCServerHook(func() {
		if deps.grpcServer != nil {
			deps.grpcServer.GracefulStop()
		}
	}))
	mgr.RegisterHook(server.NewFlushLogsHook(deps.logger.Flush))
	mgr.RegisterHook(server.NewCloseCacheHook(func() error {
		if deps.cache != nil {
			return deps.cache.Close()
		}
		return nil
	}))
}

func startServers(ctx context.Context, cfg *config.Config, deps *dependencies, shutdownMgr *server.ShutdownManager) error {
	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			iamgrpc.RecoveryInterceptor(deps.logger),
			iamgrpc.LoggingInterceptor(deps.logger),
			iamgrpc.MetricsInterceptor(iamgrpc.NewMetrics()),
		),
	)
	deps.grpcServer = grpcServer

	// Register gRPC service
	iamService := handlers.NewIAMPolicyService(handlers.IAMPolicyServiceConfig{
		AuthService: deps.authService,
		Config:      cfg,
		Logger:      deps.logger,
	})
	pb.RegisterIAMPolicyServiceServer(grpcServer, iamService)
	reflection.Register(grpcServer)

	// Start gRPC server
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	go func() {
		deps.logger.Info(ctx, "gRPC server started",
			logging.String("address", listener.Addr().String()))
		if err := grpcServer.Serve(listener); err != nil {
			deps.logger.Error(ctx, "gRPC server error", logging.Error(err))
		}
	}()

	// Start HTTP server for health and metrics
	mux := http.NewServeMux()
	mux.HandleFunc("/health/live", deps.healthMgr.LivenessHandler())
	mux.HandleFunc("/health/ready", deps.healthMgr.ReadinessHandler())
	mux.HandleFunc("/metrics", deps.metrics.Handler())

	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Host, cfg.HealthPort),
		Handler: mux,
	}
	deps.httpServer = httpServer

	go func() {
		deps.logger.Info(ctx, "HTTP server started",
			logging.String("address", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			deps.logger.Error(ctx, "HTTP server error", logging.Error(err))
		}
	}()

	// Start policy watcher
	go deps.engine.WatchPolicies(ctx)

	return nil
}
