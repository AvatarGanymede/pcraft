package p4

import "context"

func (s *Service) RevertChangelist(ctx context.Context, changelist string) error {
	if changelist == "" {
		return nil
	}
	return s.client.RevertChangelist(ctx, changelist)
}
