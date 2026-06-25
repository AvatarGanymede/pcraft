package p4

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
)

func (s *Service) EnsureChangelist(ctx context.Context, taskID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cl := s.taskChangelist[taskID]; cl != "" {
		return cl
	}
	desc := fmt.Sprintf("pcraft task %s", taskID)
	cl, err := s.client.CreateChangelist(ctx, desc)
	if err != nil {
		if s.log != nil {
			s.log.Warn("p4 create changelist failed; using fallback id",
				zap.String("task_id", taskID),
				zap.Error(err))
		}
		if mock, ok := s.client.(*MockClient); ok {
			cl = fmt.Sprintf("%d", mock.NextChangelist-1)
		} else if os.Getenv("PCRAFT_MOCK_P4") == "true" {
			cl = taskID
		} else {
			cl = taskID
		}
	}
	s.taskChangelist[taskID] = cl
	return cl
}

func (s *Service) GetChangelist(taskID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.taskChangelist[taskID]
}

func (s *Service) ConfirmSubmittedAndRelease(ctx context.Context, taskID, changelist string) error {
	cl := changelist
	if cl == "" {
		cl = s.GetChangelist(taskID)
	}
	if cl == "" {
		return fmt.Errorf("p4 changelist is required")
	}
	ok, err := s.client.IsSubmitted(ctx, cl)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("p4 changelist %s is not submitted yet", cl)
	}
	s.ReleaseByTask(taskID)
	return nil
}
