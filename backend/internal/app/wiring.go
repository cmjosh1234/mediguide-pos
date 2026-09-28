package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mediguide/internal/aiworkergrpc"
	cachepkg "mediguide/internal/cache"
	"mediguide/internal/config"
	"mediguide/internal/handlers"
	"mediguide/internal/mailer"
	"mediguide/internal/services"
	"mediguide/internal/storage"

	"gorm.io/gorm"
)

type routeWiring struct {
	guidelineSvc   services.GuidelineService
	aiWorkerClient *aiworkergrpc.Client
	authH           handlers.AuthHandler
	guidelineH      handlers.GuidelineHandler
	publicGuidelineH handlers.PublicGuidelineHandler
	outbreakH handlers.OutbreakHandler
	outbreakAdminH handlers.OutbreakAdminHandler
	searchH handlers.SearchHandler
	ragH handlers.RAGHandler
	protocolH handlers.ProtocolHandler
	syncH handlers.SyncHandler
	referenceH handlers.ReferenceHandler
	calculatorH handlers.CalculatorHandler
	drugH handlers.DrugHandler
	drugReferenceH handlers.DrugReferenceHandler
	userH handlers.UserHandler
	notificationH handlers.NotificationHandler
	supportH handlers.SupportHandler
	helpContentH handlers.HelpContentHandler
	guidelineContentH handlers.GuidelineContentHandler
	diseaseH handlers.DiseaseHandler
	contentDiseaseH handlers.ContentDiseaseHandler
	contentHubH handlers.ContentHubHandler
	emergencyProtocolH handlers.EmergencyProtocolHandler
	contentReferenceH handlers.ContentReferenceHandler
	progressUsageH handlers.ProgressUsageHandler
	guidelineLibraryH handlers.GuidelineLibraryHandler
	conversationH handlers.ConversationHandler
	legacyAPIH handlers.LegacyAPIHandler
	facilityH handlers.FacilityHandler
	firebaseH handlers.FirebaseHandler
}

func wireRoutes(cfg config.Config, database *gorm.DB, store *storage.MinioStore, cacheStore *cachepkg.Store, emailSender mailer.Sender) (routeWiring, error) {
	authSvc := services.AuthService{DB: database, Cfg: cfg, Mailer: emailSender}
	guidelineSvc, guidelineH, publicGuidelineH, guidelineContentH, guidelineLibraryH := wireGuidelines(cfg, database, store, cacheStore)
	outbreakH, outbreakAdminH := wireOutbreaks(cfg, database, store)
	searchSvc := services.SearchService{DB: database, Cache: cacheStore}
	var aiWorkerClient *aiworkergrpc.Client
	provider := strings.ToLower(strings.TrimSpace(cfg.AIRAGProvider))
	usesWorker := provider == "worker" || provider == "ai-worker"
	if usesWorker && strings.TrimSpace(cfg.AIWorkerGRPCAddr) == "" {
		return routeWiring{}, fmt.Errorf("AI_WORKER_GRPC_ADDR is required when AI_RAG_PROVIDER uses the worker")
	}
	if usesWorker && requiresInternalSecret(cfg.AppEnv) && strings.TrimSpace(cfg.AIWorkerSecret) == "" {
		return routeWiring{}, fmt.Errorf("AI_WORKER_SECRET is required when AI_RAG_PROVIDER uses the worker outside development")
	}
	if strings.TrimSpace(cfg.AIWorkerGRPCAddr) != "" {
		client, err := aiworkergrpc.NewClientWithConfig(context.Background(), aiworkergrpc.Config{
				Address:     cfg.AIWorkerGRPCAddr,
				Secret:      cfg.AIWorkerSecret,
				CallTimeout: time.Duration(cfg.AIWorkerTimeoutSecs) * time.Second,
				Retries:     cfg.AIWorkerGRPCRetries,
			})
			if err != nil {
				return routeWiring{}, err
			}
		aiWorkerClient = client
	}
	ragSvc := services.RAGService{DB: database, Search: searchSvc, Cfg: cfg, Worker: aiWorkerClient}
	protocolSvc := services.ProtocolService{DB: database}
	syncSvc := services.SyncService{DB: database, Store: store, Cfg: cfg}
	referenceSvc := services.ReferenceService{DB: database}
	calculatorSvc := services.CalculatorService{DB: database, LegacyClinicalToolsDir: cfg.LegacyClinicalToolsDir}
	calculatorVersionSvc := services.CalculatorVersionService{DB: database}
	drugSvc := services.DrugService{DB: database}
	drugReferenceSvc := services.DrugReferenceService{DB: database, Cache: cacheStore}
	userSvc := services.UserService{DB: database}
	notificationH, firebaseH, err := wireNotifications(cfg, database)
	if err != nil {
		if aiWorkerClient != nil {
			_ = aiWorkerClient.Close()
		}
		return routeWiring{}, err
	}
	supportSvc := services.SupportService{DB: database}
	helpContentSvc := services.HelpContentService{DB: database, Cache: cacheStore}
	diseaseSvc := services.DiseaseService{DB: database}
	contentDiseaseSvc := services.ContentDiseaseService{DB: database}
	contentHubSvc := services.ContentHubService{DB: database, AllowedExternalHosts: cfg.NotificationActionExternalHosts}
	emergencyProtocolSvc := services.EmergencyProtocolService{DB: database}
	contentReferenceSvc := services.ContentReferenceService{DB: database, Cache: cacheStore}
	legacyAPISvc := services.LegacyAPIService{DB: database, Cache: cacheStore}
	facilityH := wireFacilities(database, cacheStore)

	authH := handlers.AuthHandler{Service: authSvc}
	searchH := handlers.SearchHandler{Service: searchSvc}
	ragH := handlers.RAGHandler{Service: ragSvc}
	protocolH := handlers.ProtocolHandler{Service: protocolSvc}
	syncH := handlers.SyncHandler{Service: syncSvc}
	referenceH := handlers.ReferenceHandler{Service: referenceSvc}
	calculatorH := handlers.CalculatorHandler{Service: calculatorSvc, Versions: calculatorVersionSvc}
	drugH := handlers.DrugHandler{Service: drugSvc}
	drugReferenceH := handlers.DrugReferenceHandler{Service: drugReferenceSvc}
	userH := handlers.UserHandler{Service: userSvc}
	supportH := handlers.SupportHandler{Service: supportSvc}
	helpContentH := handlers.HelpContentHandler{Service: helpContentSvc}
	diseaseH := handlers.DiseaseHandler{Service: diseaseSvc}
	contentDiseaseH := handlers.ContentDiseaseHandler{Service: contentDiseaseSvc}
	contentHubH := handlers.ContentHubHandler{Service: contentHubSvc}
	emergencyProtocolH := handlers.EmergencyProtocolHandler{Service: emergencyProtocolSvc}
	contentReferenceH := handlers.ContentReferenceHandler{Service: contentReferenceSvc}
	progressUsageH := handlers.ProgressUsageHandler{Service: services.ProgressUsageService{DB: database}}
	conversationH := handlers.ConversationHandler{Service: services.ConversationService{DB: database}}
	legacyAPIH := handlers.LegacyAPIHandler{Service: legacyAPISvc, Cfg: cfg}
	return routeWiring{
		guidelineSvc:   guidelineSvc,
		aiWorkerClient: aiWorkerClient,
		authH: authH,
		guidelineH: guidelineH,
		publicGuidelineH: publicGuidelineH,
		outbreakH: outbreakH,
		outbreakAdminH: outbreakAdminH,
		searchH: searchH,
		ragH: ragH,
		protocolH: protocolH,
		syncH: syncH,
		referenceH: referenceH,
		calculatorH: calculatorH,
		drugH: drugH,
		drugReferenceH: drugReferenceH,
		userH: userH,
		notificationH: notificationH,
		supportH: supportH,
		helpContentH: helpContentH,
		guidelineContentH: guidelineContentH,
		diseaseH: diseaseH,
		contentDiseaseH: contentDiseaseH,
		contentHubH: contentHubH,
		emergencyProtocolH: emergencyProtocolH,
		contentReferenceH: contentReferenceH,
		progressUsageH: progressUsageH,
		guidelineLibraryH: guidelineLibraryH,
		conversationH: conversationH,
		legacyAPIH: legacyAPIH,
		facilityH: facilityH,
		firebaseH: firebaseH,
	}, nil
}

func requiresInternalSecret(appEnv string) bool {
	switch strings.ToLower(strings.TrimSpace(appEnv)) {
	case "", "development", "dev", "local", "test", "testing":
		return false
	default:
		return true
	}
}
