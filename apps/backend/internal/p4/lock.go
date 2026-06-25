package p4

import (
	"context"
	"errors"
)

var ErrFileLocked = errors.New("p4 file locked by another task")

type CheckoutResult struct {
	Allowed         bool     `json:"allowed"`
	ConflictTaskID  string   `json:"conflict_task_id,omitempty"`
	CheckedOutFiles []string `json:"checked_out_files,omitempty"`
	Changelist      string   `json:"changelist,omitempty"`
}

func (s *Service) Checkout(ctx context.Context, taskID string, files []string) (*CheckoutResult, error) {
	changelist := s.EnsureChangelist(ctx, taskID)

	for _, file := range files {
		owner, ok, err := s.store.GetOwner(ctx, file)
		if err != nil {
			return nil, err
		}
		if ok && owner != taskID {
			return &CheckoutResult{
				Allowed:        false,
				ConflictTaskID: owner,
			}, ErrFileLocked
		}
	}

	if err := s.client.CheckoutFiles(ctx, changelist, files); err != nil {
		return nil, err
	}
	if err := s.store.SetLocks(ctx, taskID, changelist, files); err != nil {
		return nil, err
	}

	return &CheckoutResult{
		Allowed:         true,
		CheckedOutFiles: files,
		Changelist:      changelist,
	}, nil
}

func (s *Service) ReleaseByTask(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.store.ReleaseByTask(context.Background(), taskID)
	delete(s.taskChangelist, taskID)
}

func (s *Service) BindChangelist(taskID, changelist string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.taskChangelist[taskID] = changelist
}
