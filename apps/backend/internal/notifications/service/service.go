package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/AvatarGanymede/pcraft/internal/common/logger"
	gatewayws "github.com/AvatarGanymede/pcraft/internal/gateway/websocket"
	"github.com/AvatarGanymede/pcraft/internal/notifications/models"
	"github.com/AvatarGanymede/pcraft/internal/notifications/providers"
	notificationstore "github.com/AvatarGanymede/pcraft/internal/notifications/store"
	taskmodels "github.com/AvatarGanymede/pcraft/internal/task/models"
	userstore "github.com/AvatarGanymede/pcraft/internal/user/store"
	"go.uber.org/zap"
)

const (
	EventTaskSessionWaitingForInput = "session.waiting_for_input"
	EventOfficeInboxItem            = "office.inbox_item"
)

var ErrProviderNotFound = errors.New("notification provider not found")

// taskGetter is the minimal repository interface needed by the notification service.
type taskGetter interface {
	GetTask(ctx context.Context, id string) (*taskmodels.Task, error)
}

// AssigneeResolver resolves a JNPM ticket's assignee email from a raw ticket
// number. Satisfied by *jnpm.Service; kept as an interface so this package does
// not import the jnpm package directly.
type AssigneeResolver interface {
	Enabled() bool
	ResolveAssigneeEmail(ctx context.Context, rawJnpmID string) (email, name string, err error)
}

type Service struct {
	repo       notificationstore.Repository
	taskRepo   taskGetter
	hub        *gatewayws.Hub
	logger     *logger.Logger
	providers  map[models.ProviderType]providers.Provider
	jnpm       AssigneeResolver
	adminEmail string
}

// NewService builds the notification service. larkSender + jnpmResolver may be
// disabled (nil client); adminEmail is the fallback Lark recipient used when a
// task has no JNPM assignee (or no JNPM id at all).
func NewService(
	repo notificationstore.Repository,
	taskRepo taskGetter,
	hub *gatewayws.Hub,
	log *logger.Logger,
	larkSender providers.LarkSender,
	jnpmResolver AssigneeResolver,
	adminEmail string,
	taskBaseURL string,
) *Service {
	providerMap := map[models.ProviderType]providers.Provider{
		models.ProviderTypeLocal:   providers.NewLocalProvider(hub),
		models.ProviderTypeApprise: providers.NewAppriseProvider(),
		models.ProviderTypeSystem:  providers.NewSystemProvider(),
		models.ProviderTypeLark:    providers.NewLarkProvider(larkSender, taskBaseURL),
	}
	return &Service{
		repo:       repo,
		taskRepo:   taskRepo,
		hub:        hub,
		logger:     log.WithFields(zap.String("component", "notifications-service")),
		providers:  providerMap,
		jnpm:       jnpmResolver,
		adminEmail: strings.TrimSpace(adminEmail),
	}
}

func (s *Service) AppriseAvailable() bool {
	provider := s.providers[models.ProviderTypeApprise]
	if provider == nil {
		return false
	}
	return provider.Available()
}

func (s *Service) AvailableEvents() []string {
	return []string{EventTaskSessionWaitingForInput, EventOfficeInboxItem}
}

func (s *Service) ListProviders(ctx context.Context, userID string) ([]*models.Provider, map[string][]string, error) {
	if err := s.ensureDefaultProviders(ctx, userID); err != nil {
		return nil, nil, err
	}
	providers, err := s.repo.ListProvidersByUser(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	subscriptions := make(map[string][]string, len(providers))
	for _, provider := range providers {
		subs, err := s.repo.ListSubscriptionsByProvider(ctx, provider.ID)
		if err != nil {
			return nil, nil, err
		}
		for _, sub := range subs {
			if sub.Enabled {
				subscriptions[provider.ID] = append(subscriptions[provider.ID], sub.EventType)
			}
		}
	}
	return providers, subscriptions, nil
}

func (s *Service) CreateProvider(ctx context.Context, userID, name string, providerType models.ProviderType, config map[string]interface{}, enabled bool, events []string) (*models.Provider, error) {
	if err := s.validateProvider(providerType, config); err != nil {
		return nil, err
	}
	if err := s.validateEvents(events); err != nil {
		return nil, err
	}
	provider := &models.Provider{
		UserID:  userID,
		Name:    name,
		Type:    providerType,
		Config:  config,
		Enabled: enabled,
	}
	if err := s.repo.CreateProvider(ctx, provider); err != nil {
		return nil, err
	}
	if err := s.repo.ReplaceSubscriptions(ctx, provider.ID, userID, events); err != nil {
		return nil, err
	}
	return provider, nil
}

func (s *Service) UpdateProvider(ctx context.Context, providerID string, updates ProviderUpdate) (*models.Provider, error) {
	provider, err := s.repo.GetProvider(ctx, providerID)
	if err != nil {
		return nil, ErrProviderNotFound
	}
	if updates.Name != nil {
		provider.Name = strings.TrimSpace(*updates.Name)
	}
	if updates.Enabled != nil {
		provider.Enabled = *updates.Enabled
	}
	if updates.Config != nil {
		provider.Config = updates.Config
	}
	if updates.Type != nil {
		provider.Type = *updates.Type
	}
	if err := s.validateProvider(provider.Type, provider.Config); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateProvider(ctx, provider); err != nil {
		return nil, err
	}
	if updates.Events != nil {
		if err := s.validateEvents(*updates.Events); err != nil {
			return nil, err
		}
		if err := s.repo.ReplaceSubscriptions(ctx, provider.ID, provider.UserID, *updates.Events); err != nil {
			return nil, err
		}
	}
	return provider, nil
}

func (s *Service) DeleteProvider(ctx context.Context, providerID string) error {
	return s.repo.DeleteProvider(ctx, providerID)
}

type ProviderUpdate struct {
	Name    *string
	Enabled *bool
	Type    *models.ProviderType
	Config  map[string]interface{}
	Events  *[]string
}

func (s *Service) HandleTaskSessionStateChanged(ctx context.Context, taskID, sessionID, newState string) {
	if newState != "WAITING_FOR_INPUT" {
		return
	}
	userID := userstore.DefaultUserID
	providers, subscriptions, err := s.ListProviders(ctx, userID)
	if err != nil {
		s.logger.Error("failed to load notification providers", zap.Error(err))
		return
	}
	title, body := s.buildWaitingForInputMessage(ctx, taskID)
	targetEmail := s.resolveRecipientEmail(ctx, taskID)
	larkAvailable := false
	if p := s.providers[models.ProviderTypeLark]; p != nil {
		larkAvailable = p.Available()
	}
	s.logger.Info("waiting-for-input notification",
		zap.String("task_id", taskID),
		zap.String("session_id", sessionID),
		zap.String("recipient_email", targetEmail),
		zap.Bool("lark_available", larkAvailable),
		zap.Int("provider_count", len(providers)))
	for _, provider := range providers {
		if !provider.Enabled {
			continue
		}
		events := subscriptions[provider.ID]
		if !containsEvent(events, EventTaskSessionWaitingForInput) {
			continue
		}
		delivery := &models.Delivery{
			UserID:        userID,
			ProviderID:    provider.ID,
			EventType:     EventTaskSessionWaitingForInput,
			TaskSessionID: sessionID,
		}
		inserted, err := s.repo.InsertDelivery(ctx, delivery)
		if err != nil {
			s.logger.Error("failed to record notification delivery", zap.Error(err))
			continue
		}
		if !inserted {
			continue
		}
		if err := s.dispatchProvider(ctx, provider, waitingForInputPayload{
			TaskID:        taskID,
			TaskSessionID: sessionID,
			Title:         title,
			Body:          body,
			TargetEmail:   targetEmail,
		}); err != nil {
			s.logger.Warn("notification delivery failed",
				zap.String("provider_id", provider.ID),
				zap.String("provider_type", string(provider.Type)),
				zap.Error(err))
			_ = s.repo.DeleteDelivery(ctx, provider.ID, EventTaskSessionWaitingForInput, sessionID)
		} else {
			s.logger.Info("notification delivered",
				zap.String("provider_type", string(provider.Type)),
				zap.String("recipient_email", targetEmail))
		}
	}
}

// HandleInboxItem sends notifications for a new office inbox item.
func (s *Service) HandleInboxItem(ctx context.Context, itemType, title string) {
	userID := userstore.DefaultUserID
	providers, subscriptions, err := s.ListProviders(ctx, userID)
	if err != nil {
		s.logger.Error("failed to load notification providers for inbox item", zap.Error(err))
		return
	}
	notifTitle := "New inbox item"
	body := title
	if itemType != "" {
		notifTitle = fmt.Sprintf("Inbox: %s", itemType)
	}
	// Inbox items are not task-scoped, so there is no JNPM assignee to resolve;
	// Lark delivery goes to the configured admin address.
	targetEmail := s.adminEmail
	for _, provider := range providers {
		if !provider.Enabled {
			continue
		}
		events := subscriptions[provider.ID]
		if !containsEvent(events, EventOfficeInboxItem) {
			continue
		}
		if err := s.dispatchGenericNotification(ctx, provider, EventOfficeInboxItem, notifTitle, body, targetEmail); err != nil {
			s.logger.Warn("inbox item notification delivery failed",
				zap.String("provider_id", provider.ID), zap.Error(err))
		}
	}
}

func (s *Service) dispatchGenericNotification(ctx context.Context, provider *models.Provider, eventType, title, body, targetEmail string) error {
	adapter := s.providers[provider.Type]
	if adapter == nil {
		return fmt.Errorf("unknown provider type: %s", provider.Type)
	}
	return adapter.Send(ctx, providers.Message{
		EventType:   eventType,
		Title:       title,
		Body:        body,
		UserID:      userstore.DefaultUserID,
		Config:      provider.Config,
		TargetEmail: targetEmail,
	})
}

type waitingForInputPayload struct {
	TaskID        string
	TaskSessionID string
	Title         string
	Body          string
	TargetEmail   string
}

func (s *Service) dispatchProvider(ctx context.Context, provider *models.Provider, payload waitingForInputPayload) error {
	adapter := s.providers[provider.Type]
	if adapter == nil {
		return fmt.Errorf("unknown provider type: %s", provider.Type)
	}
	return adapter.Send(ctx, providers.Message{
		EventType:     EventTaskSessionWaitingForInput,
		Title:         payload.Title,
		Body:          payload.Body,
		TaskID:        payload.TaskID,
		TaskSessionID: payload.TaskSessionID,
		UserID:        userstore.DefaultUserID,
		Config:        provider.Config,
		TargetEmail:   payload.TargetEmail,
	})
}

// resolveRecipientEmail picks the Lark recipient for a task-scoped
// notification: the JNPM ticket assignee when the task carries a jnpm_id and
// JNPM resolution succeeds, otherwise the configured admin address.
func (s *Service) resolveRecipientEmail(ctx context.Context, taskID string) string {
	if taskID == "" || s.taskRepo == nil {
		return s.adminEmail
	}
	task, err := s.taskRepo.GetTask(ctx, taskID)
	if err != nil || task == nil {
		return s.adminEmail
	}
	jnpmID := metadataString(task.Metadata, taskmodels.MetaKeyJnpmID)
	if jnpmID == "" || s.jnpm == nil || !s.jnpm.Enabled() {
		return s.adminEmail
	}
	email, _, err := s.jnpm.ResolveAssigneeEmail(ctx, jnpmID)
	if err != nil {
		s.logger.Warn("jnpm assignee resolution failed; falling back to admin",
			zap.String("task_id", taskID), zap.String("jnpm_id", jnpmID), zap.Error(err))
		return s.adminEmail
	}
	if strings.TrimSpace(email) == "" {
		return s.adminEmail
	}
	return email
}

// metadataString reads a string value from a task metadata map, tolerating a
// nil map and non-string values.
func metadataString(meta map[string]interface{}, key string) string {
	if meta == nil {
		return ""
	}
	if v, ok := meta[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func (s *Service) buildWaitingForInputMessage(ctx context.Context, taskID string) (string, string) {
	title := "Task needs your input"
	body := "An agent is waiting for your input."
	if taskID == "" || s.taskRepo == nil {
		return title, body
	}
	task, err := s.taskRepo.GetTask(ctx, taskID)
	if err != nil || task == nil {
		return title, body
	}
	if task.Title != "" {
		body = fmt.Sprintf("An agent is waiting for your input on \"%s\".", task.Title)
	}
	return title, body
}

func (s *Service) ensureDefaultProviders(ctx context.Context, userID string) error {
	providers, err := s.repo.ListProvidersByUser(ctx, userID)
	if err != nil {
		return err
	}
	hasLocal := false
	hasSystem := false
	hasLark := false
	for _, provider := range providers {
		switch provider.Type {
		case models.ProviderTypeLocal:
			hasLocal = true
		case models.ProviderTypeSystem:
			hasSystem = true
		case models.ProviderTypeLark:
			hasLark = true
		}
	}
	if !hasLocal {
		provider := &models.Provider{
			ID:      uuid.New().String(),
			UserID:  userID,
			Name:    "Desktop Notifications",
			Type:    models.ProviderTypeLocal,
			Config:  map[string]interface{}{},
			Enabled: true,
		}
		if err := s.repo.CreateProvider(ctx, provider); err != nil {
			return err
		}
		if err := s.repo.ReplaceSubscriptions(ctx, provider.ID, userID, []string{
			EventTaskSessionWaitingForInput,
			EventOfficeInboxItem,
		}); err != nil {
			return err
		}
	}
	if !hasSystem {
		if err := s.ensureSystemProvider(ctx, userID); err != nil {
			return err
		}
	}
	if !hasLark {
		if err := s.ensureLarkProvider(ctx, userID); err != nil {
			return err
		}
	}
	return nil
}

// ensureLarkProvider seeds the Lark (Feishu) bot provider when the adapter is
// available (app credentials configured). This is the primary delivery channel
// now that OS/desktop notifications are permanently disabled.
func (s *Service) ensureLarkProvider(ctx context.Context, userID string) error {
	adapter := s.providers[models.ProviderTypeLark]
	if adapter == nil || !adapter.Available() {
		return nil
	}
	provider := &models.Provider{
		ID:      uuid.New().String(),
		UserID:  userID,
		Name:    "Lark Bot",
		Type:    models.ProviderTypeLark,
		Config:  map[string]interface{}{},
		Enabled: true,
	}
	if err := s.repo.CreateProvider(ctx, provider); err != nil {
		return err
	}
	return s.repo.ReplaceSubscriptions(ctx, provider.ID, userID, []string{
		EventTaskSessionWaitingForInput,
		EventOfficeInboxItem,
	})
}

// ensureSystemProvider creates the system notification provider if the adapter is available.
func (s *Service) ensureSystemProvider(ctx context.Context, userID string) error {
	adapter := s.providers[models.ProviderTypeSystem]
	if adapter == nil || !adapter.Available() {
		return nil
	}
	provider := &models.Provider{
		ID:     uuid.New().String(),
		UserID: userID,
		Name:   "System Notifications",
		Type:   models.ProviderTypeSystem,
		Config: map[string]interface{}{
			"sound_enabled": false,
		},
		Enabled: true,
	}
	if err := s.repo.CreateProvider(ctx, provider); err != nil {
		return err
	}
	return s.repo.ReplaceSubscriptions(ctx, provider.ID, userID, []string{
		EventTaskSessionWaitingForInput,
		EventOfficeInboxItem,
	})
}

func (s *Service) TestProvider(ctx context.Context, providerID string) error {
	provider, err := s.repo.GetProvider(ctx, providerID)
	if err != nil {
		return ErrProviderNotFound
	}
	adapter := s.providers[provider.Type]
	if adapter == nil {
		return fmt.Errorf("unknown provider type: %s", provider.Type)
	}
	return adapter.Send(ctx, providers.Message{
		EventType: EventTaskSessionWaitingForInput,
		Title:     "Test notification",
		Body:      "If you can read this, notifications are working.",
		Config:    provider.Config,
	})
}

func (s *Service) validateProvider(providerType models.ProviderType, config map[string]interface{}) error {
	adapter := s.providers[providerType]
	if adapter == nil {
		return fmt.Errorf("unsupported provider type: %s", providerType)
	}
	return adapter.Validate(config)
}

func (s *Service) validateEvents(events []string) error {
	allowed := map[string]struct{}{
		EventTaskSessionWaitingForInput: {},
		EventOfficeInboxItem:            {},
	}
	for _, event := range events {
		if _, ok := allowed[event]; !ok {
			return fmt.Errorf("unsupported event type: %s", event)
		}
	}
	return nil
}

func containsEvent(events []string, target string) bool {
	for _, event := range events {
		if event == target {
			return true
		}
	}
	return false
}
