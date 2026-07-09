package service

import (
	"context"
	"errors"
	"testing"

	"github.com/AvatarGanymede/pcraft/internal/common/logger"
	taskmodels "github.com/AvatarGanymede/pcraft/internal/task/models"
)

type fakeTaskRepo struct {
	task *taskmodels.Task
	err  error
}

func (f *fakeTaskRepo) GetTask(_ context.Context, _ string) (*taskmodels.Task, error) {
	return f.task, f.err
}

type fakeResolver struct {
	enabled bool
	email   string
	name    string
	err     error
}

func (f *fakeResolver) Enabled() bool { return f.enabled }
func (f *fakeResolver) ResolveAssigneeEmail(_ context.Context, _ string) (string, string, error) {
	return f.email, f.name, f.err
}

func newTestService(repo *fakeTaskRepo, resolver AssigneeResolver) *Service {
	return &Service{
		taskRepo:   repo,
		jnpm:       resolver,
		adminEmail: "admin@example.com",
		logger:     logger.Default(),
	}
}

func taskWithJnpm(id string) *taskmodels.Task {
	return &taskmodels.Task{Metadata: map[string]interface{}{taskmodels.MetaKeyJnpmID: id}}
}

func TestResolveRecipientEmail(t *testing.T) {
	const admin = "admin@example.com"
	cases := []struct {
		name     string
		taskID   string
		repo     *fakeTaskRepo
		resolver AssigneeResolver
		want     string
	}{
		{
			name:   "empty task id falls back to admin",
			taskID: "",
			repo:   &fakeTaskRepo{},
			want:   admin,
		},
		{
			name:   "task lookup error falls back to admin",
			taskID: "t1",
			repo:   &fakeTaskRepo{err: errors.New("boom")},
			want:   admin,
		},
		{
			name:   "no jnpm id falls back to admin",
			taskID: "t1",
			repo:   &fakeTaskRepo{task: &taskmodels.Task{}},
			want:   admin,
		},
		{
			name:     "resolver disabled falls back to admin",
			taskID:   "t1",
			repo:     &fakeTaskRepo{task: taskWithJnpm("#755621")},
			resolver: &fakeResolver{enabled: false},
			want:     admin,
		},
		{
			name:     "resolver error falls back to admin",
			taskID:   "t1",
			repo:     &fakeTaskRepo{task: taskWithJnpm("#755621")},
			resolver: &fakeResolver{enabled: true, err: errors.New("nope")},
			want:     admin,
		},
		{
			name:     "resolver empty email falls back to admin",
			taskID:   "t1",
			repo:     &fakeTaskRepo{task: taskWithJnpm("#755621")},
			resolver: &fakeResolver{enabled: true, email: ""},
			want:     admin,
		},
		{
			name:     "resolver returns assignee email",
			taskID:   "t1",
			repo:     &fakeTaskRepo{task: taskWithJnpm("#755621")},
			resolver: &fakeResolver{enabled: true, email: "alice@example.com"},
			want:     "alice@example.com",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestService(tc.repo, tc.resolver)
			if got := s.resolveRecipientEmail(context.Background(), tc.taskID); got != tc.want {
				t.Fatalf("resolveRecipientEmail = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMetadataString(t *testing.T) {
	if got := metadataString(nil, "k"); got != "" {
		t.Errorf("nil map = %q, want empty", got)
	}
	if got := metadataString(map[string]interface{}{"k": 42}, "k"); got != "" {
		t.Errorf("non-string = %q, want empty", got)
	}
	if got := metadataString(map[string]interface{}{"k": "  v  "}, "k"); got != "v" {
		t.Errorf("string = %q, want v", got)
	}
}
