package store

import (
	"context"

	"github.com/AvatarGanymede/pcraft/internal/user/models"
)

type Repository interface {
	GetUser(ctx context.Context, id string) (*models.User, error)
	GetDefaultUser(ctx context.Context) (*models.User, error)
	GetUserSettings(ctx context.Context, userID string) (*models.UserSettings, error)
	UpsertUserSettings(ctx context.Context, settings *models.UserSettings) error
	Close() error
}
