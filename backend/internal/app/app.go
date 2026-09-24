package app

import (
	"context"
	"net/http"
	"strings"
	"time"

	"mediguide/internal/buildinfo"
	cachepkg "mediguide/internal/cache"
	"mediguide/internal/config"
	"mediguide/internal/db"
	"mediguide/internal/handlers"
	"mediguide/internal/mailer"
	"mediguide/internal/middleware"
	"mediguide/internal/observability"
	"mediguide/internal/redisx"
	"mediguide/internal/storage"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type App struct {
	stopUploads context.CancelFunc
	Router      *gin.Engine
	DB          *gorm.DB
	Redis       *redis.Client
	Cache       *cachepkg.Store
}

func New(cfg config.Config) (*App, error) {
	ginMode := configureGinMode(cfg.AppEnv)
	log.Info().Str("app_env", cfg.AppEnv).Str("gin_mode", ginMode).Msg("configured Gin runtime mode")

	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	store, err := storage.NewMinioStore(cfg)
	if err != nil {
		return nil, err
	}
	emailSender, err := mailer.New(cfg)
	if err != nil {
		return nil, err
	}
	redisClient, err := redisx.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	redisContext, cancelRedis := context.WithTimeout(context.Background(), 3*time.Second)
	if err := redisx.Ping(redisContext, redisClient); err != nil {
		log.Warn().Err(err).Msg("redis unavailable during startup; fallbacks will be used")
	}
	cancelRedis()
	cacheStore := cachepkg.New(redisClient, cfg.RedisKeyPrefix, cfg.CacheEnabled, cfg.CacheMaxItemBytes)
	rateLimiter := middleware.NewRateLimiter(redisClient, cfg.RedisKeyPrefix, cfg.RateLimitEnabled).WithAuditDB(database)

	r := gin.New()
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, err
	}
	r.Use(gin.Recovery(), middleware.RequestLogger(), observability.OutbreakHTTP())

	// Build CORS allow-list from config (comma-separated).
	allowedOrigins := []string{}
	for _, o := range strings.Split(cfg.AllowedOrigins, ",") {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			allowedOrigins = append(allowedOrigins, trimmed)
		}
	}
	r.Use(cors.New(cors.Config{
		AllowOrigins:  allowedOrigins,
		AllowHeaders:  []string{"Accept", "Authorization", "Content-Type", "If-Match", "If-None-Match"},
		ExposeHeaders: []string{"ETag", "Last-Modified", "Retry-After", "RateLimit-Limit", "RateLimit-Remaining", "RateLimit-Reset"},
		AllowMethods:  []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
	}))

	r.GET("/swagger", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerChooserHTML))
	})
	r.GET("/swagger/all/*any", ginSwagger.WrapHandler(swaggerFiles.NewHandler(), ginSwagger.InstanceName("all"), ginSwagger.URL("/swagger/all/doc.json")))
	r.GET("/swagger/v1/*any", ginSwagger.WrapHandler(swaggerFiles.NewHandler(), ginSwagger.InstanceName("v1"), ginSwagger.URL("/swagger/v1/doc.json")))
	r.GET("/swagger/v2/*any", ginSwagger.WrapHandler(swaggerFiles.NewHandler(), ginSwagger.InstanceName("v2"), ginSwagger.URL("/swagger/v2/doc.json")))
	r.GET("/api/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"ok":       true,
			"service":  cfg.AppName,
			"version":  buildinfo.Version,
			"revision": buildinfo.Revision,
		})
	})
	r.GET("/api/metrics", handlers.OperationsHandler{DB: database, Cache: cacheStore, Store: store}.Metrics)

	// Readiness probe: verify DB connectivity.
	r.GET("/api/readyz", func(c *gin.Context) {
		sqlDB, err := database.DB()
		if err != nil || sqlDB.Ping() != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"ok": false, "reason": "db_unavailable"})
			return
		}
		if cfg.RateLimitEnabled || cfg.CacheEnabled {
			ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
			defer cancel()
			if err := redisx.Ping(ctx, redisClient); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"ok": false, "reason": "redis_unavailable"})
				return
			}
		}
		storageContext, cancelStorage := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancelStorage()
		if checker, ok := any(store).(interface{ Health(context.Context) error }); ok && checker.Health(storageContext) != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"ok": false, "reason": "managed_storage_unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"ok":       true,
			"service":  cfg.AppName,
			"version":  buildinfo.Version,
			"revision": buildinfo.Revision,
		})
	})

	wired, err := wireRoutes(cfg, database, store, cacheStore, emailSender)
	if err != nil {
		return nil, err
	}

	legacyV1 := r.Group("/api/v1")
	legacyV1.GET("/stats", rateLimiter.Limit(middleware.Policy("legacy-public", 60, time.Minute, 10), middleware.IPIdentity), wired.legacyAPIH.Stats)
	legacyV1.GET("/health-facilities/tree", rateLimiter.Limit(middleware.Policy("legacy-public", 60, time.Minute, 10), middleware.IPIdentity), wired.legacyAPIH.HealthFacilitiesTree)
	legacyV1.GET("/ministry-directory/tree", rateLimiter.Limit(middleware.Policy("legacy-public", 60, time.Minute, 10), middleware.IPIdentity), wired.legacyAPIH.MinistryDirectoryTree)
	legacyProtected := legacyV1.Group("")
	legacyProtected.Use(middleware.AuthRequired(cfg, database), middleware.PrivateNoStore())
	legacyProtected.GET("/overview", wired.legacyAPIH.Overview)

	legacyCompat := r.Group("/api")
	legacyCompat.Use(middleware.AuthRequired(cfg, database), middleware.PrivateNoStore())
	legacyCompat.GET("/overview", wired.legacyAPIH.Overview)

	public := r.Group("/api/public")
	public.Use(rateLimiter.Limit(middleware.Policy("public-guidelines", 120, time.Minute, 20), middleware.IPIdentity))
	{
		registerPublicGuidelinesRoutes(public, rateLimiter, wired.publicGuidelineH, wired.searchH)
		registerPublicHubsRoutes(public, wired.contentHubH, wired.diseaseH)
		registerPublicAiRoutes(public, rateLimiter, wired.ragH)
		registerPublicOutbreakRoutes(public, rateLimiter, wired.supportH, wired.outbreakH, wired.contentHubH)
	}

	v2 := r.Group("/api/v2")
	{
		privateNoStore := middleware.PrivateNoStore()
		v2.POST("/auth/register", privateNoStore, rateLimiter.Limit(middleware.Policy("auth-register", 5, time.Hour, 1), middleware.IPIdentity), wired.authH.Register)
		v2.POST("/auth/login",
			privateNoStore,
			rateLimiter.Limit(middleware.Policy("auth-login-ip", 10, 5*time.Minute, 2), middleware.IPIdentity),
			rateLimiter.Limit(middleware.Policy("auth-login-account", 5, 15*time.Minute, 0), middleware.IPAndJSONFieldIdentity("email")), wired.authH.Login)
		v2.POST("/auth/refresh",
			privateNoStore,
			rateLimiter.Limit(middleware.Policy("auth-refresh-ip", 60, time.Minute, 5), middleware.IPIdentity),
			rateLimiter.Limit(middleware.Policy("auth-refresh-session", 30, time.Minute, 5), middleware.IPAndJSONFieldIdentity("refresh_token")), wired.authH.Refresh)
		v2.POST("/auth/password-reset/request",
			privateNoStore,
			rateLimiter.Limit(middleware.Policy("password-reset-ip", 5, 15*time.Minute, 0), middleware.IPIdentity),
			rateLimiter.Limit(middleware.Policy("password-reset-account", 3, time.Hour, 0), middleware.IPAndJSONFieldIdentity("email")), wired.authH.RequestPasswordReset)
		v2.POST("/auth/password-reset/confirm",
			privateNoStore,
			rateLimiter.Limit(middleware.Policy("password-reset-confirm-ip", 10, 15*time.Minute, 0), middleware.IPIdentity),
			rateLimiter.Limit(middleware.Policy("password-reset-confirm-token", 5, 15*time.Minute, 0), middleware.IPAndJSONFieldIdentity("token")), wired.authH.ConfirmPasswordReset)
		v2.POST("/auth/email-verification/request", privateNoStore, rateLimiter.Limit(middleware.Policy("email-verification-request", 5, 15*time.Minute, 0), middleware.IPAndJSONFieldIdentity("email")), wired.authH.RequestEmailVerification)
		v2.POST("/auth/email-verification/confirm", privateNoStore, rateLimiter.Limit(middleware.Policy("email-verification-confirm", 10, 15*time.Minute, 0), middleware.IPAndJSONFieldIdentity("token")), wired.authH.ConfirmEmailVerification)
		protected := v2.Group("")
		protected.Use(
			middleware.AuthRequired(cfg, database),
			privateNoStore,
			middleware.ByMethod(
				rateLimiter.Limit(middleware.Policy("authenticated-read", 300, time.Minute, 30), middleware.UserIdentity),
				rateLimiter.Limit(middleware.Policy("authenticated-write", 120, time.Minute, 20), middleware.UserIdentity),
			),
		)
		protected.POST("/auth/logout", wired.authH.Logout)
		protected.GET("/me", wired.authH.Me)
		protected.POST("/me/password", rateLimiter.Limit(middleware.Policy("password-change", 5, time.Hour, 0), middleware.UserIdentity), wired.authH.ChangePassword)

		registerOutbreakRoutes(protected, wired.outbreakAdminH)

		registerCalculatorsRoutes(protected, rateLimiter, wired.calculatorH)

		registerDrugsRoutes(protected, wired.drugH)

		registerUsersRoutes(protected, wired.userH)

		registerNotificationsRoutes(protected, rateLimiter, database, wired.notificationH, wired.firebaseH)

		registerSupportRoutes(protected, rateLimiter, wired.supportH, wired.helpContentH)

		registerClinicalContentRoutes(protected, wired.guidelineContentH, wired.diseaseH, wired.contentDiseaseH, wired.contentHubH, wired.emergencyProtocolH, wired.contentReferenceH)

		registerDrugReferenceRoutes(protected, wired.drugReferenceH)

		registerGuidelineEditorRoutes(protected, rateLimiter, wired.guidelineH)

		registerDiscoveryRoutes(protected, rateLimiter, wired.searchH, wired.ragH, wired.protocolH)

		registerProfileRoutes(protected, rateLimiter, wired.referenceH, wired.contentReferenceH, wired.progressUsageH, wired.guidelineLibraryH, wired.conversationH)

		registerFacilitiesRoutes(protected, wired.facilityH)

		registerSyncRoutes(protected, rateLimiter, wired.syncH)

	}
	uploadContext, stopUploads := context.WithCancel(context.Background())
	if wired.guidelineH.DirectUploads {
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-uploadContext.Done():
					return
				case <-ticker.C:
				}
				ctx, cancel := context.WithTimeout(uploadContext, 25*time.Second)
				if err := wired.guidelineSvc.MaintainSourceUploads(ctx); err != nil {
					log.Error().Err(err).Msg("guideline upload maintenance failed")
				}
				cancel()
			}
		}()
	}
	return &App{Router: r, DB: database, Redis: redisClient, Cache: cacheStore, stopUploads: stopUploads}, nil
}

func (a *App) Close() error {
	if a != nil && a.stopUploads != nil {
		a.stopUploads()
	}
	if a == nil || a.Redis == nil {
		return nil
	}
	return a.Redis.Close()
}
