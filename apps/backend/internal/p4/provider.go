package p4

import (
	"database/sql"
	"os"
	"sync"

	"github.com/AvatarGanymede/pcraft/internal/common/logger"
)

type Service struct {
	client         Client
	log            *logger.Logger
	mu             sync.Mutex
	store          LockStore
	taskChangelist map[string]string
}

func NewService(client Client, log *logger.Logger, store LockStore) *Service {
	if store == nil {
		store = NewMemoryLockStore()
	}
	return &Service{
		client:         client,
		log:            log,
		store:          store,
		taskChangelist: map[string]string{},
	}
}

func Provide(db *sql.DB, log *logger.Logger) (*Service, func() error, error) {
	var client Client = NewCLIClient()
	if os.Getenv("PCRAFT_MOCK_P4") == "true" {
		client = NewMockClient()
	}
	var store LockStore = NewMemoryLockStore()
	if db != nil {
		store = NewSQLLockStore(db)
	}
	svc := NewService(client, log, store)
	return svc, func() error { return nil }, nil
}
