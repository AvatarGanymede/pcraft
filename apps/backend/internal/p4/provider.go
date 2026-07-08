package p4

import (
	"database/sql"
	"os"

	"github.com/AvatarGanymede/pcraft/internal/common/logger"
)

// Service exposes the p4 client-listing and root-resolution surface used for
// the workspace↔P4 binding. It performs no p4 mutations of its own — agents
// run p4 commands themselves via workflow-step prompts.
type Service struct {
	client Client
	log    *logger.Logger
}

func NewService(client Client, log *logger.Logger) *Service {
	return &Service{
		client: client,
		log:    log,
	}
}

// Provide constructs the p4 Service. The db argument is unused (kept for
// wiring symmetry); no lock store is created.
func Provide(_ *sql.DB, log *logger.Logger) (*Service, func() error, error) {
	var client Client = NewCLIClient()
	if os.Getenv("PCRAFT_MOCK_P4") == "true" {
		client = NewMockClient()
	}
	svc := NewService(client, log)
	return svc, func() error { return nil }, nil
}
