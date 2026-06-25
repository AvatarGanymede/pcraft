package sqlite

import "fmt"

// initPhase2Schema creates workflow_step_participants and workflow_step_decisions
// tables for multi-agent workflow steps. Empty participant rows preserve
// single-agent behaviour on existing kanban workflows.
func (r *Repository) initPhase2Schema() error {
	participantsSchema := `
	CREATE TABLE IF NOT EXISTS workflow_step_participants (
		id TEXT PRIMARY KEY,
		step_id TEXT NOT NULL REFERENCES workflow_steps(id) ON DELETE CASCADE,
		task_id TEXT NOT NULL DEFAULT '',
		role TEXT NOT NULL CHECK (role IN ('reviewer','approver','watcher','collaborator','runner')),
		agent_profile_id TEXT NOT NULL,
		decision_required INTEGER NOT NULL DEFAULT 0,
		position INTEGER NOT NULL DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_workflow_step_participants_step ON workflow_step_participants(step_id);
	CREATE INDEX IF NOT EXISTS idx_workflow_step_participants_role ON workflow_step_participants(step_id, role);
	CREATE INDEX IF NOT EXISTS idx_workflow_step_participants_task ON workflow_step_participants(task_id) WHERE task_id != '';
	`
	if _, err := r.db.Exec(participantsSchema); err != nil {
		return fmt.Errorf("failed to create workflow_step_participants table: %w", err)
	}

	decisionsSchema := `
	CREATE TABLE IF NOT EXISTS workflow_step_decisions (
		id TEXT PRIMARY KEY,
		task_id TEXT NOT NULL,
		step_id TEXT NOT NULL,
		participant_id TEXT NOT NULL,
		decision TEXT NOT NULL,
		note TEXT DEFAULT '',
		decided_at TIMESTAMP NOT NULL,
		superseded_at TIMESTAMP NULL,
		decider_type TEXT NOT NULL DEFAULT '',
		decider_id TEXT NOT NULL DEFAULT '',
		role TEXT NOT NULL DEFAULT '',
		comment TEXT NOT NULL DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS idx_workflow_step_decisions_task_step ON workflow_step_decisions(task_id, step_id);
	CREATE INDEX IF NOT EXISTS idx_workflow_step_decisions_participant ON workflow_step_decisions(participant_id);
	CREATE INDEX IF NOT EXISTS idx_workflow_step_decisions_active
		ON workflow_step_decisions(task_id, role) WHERE superseded_at IS NULL;
	`
	if _, err := r.db.Exec(decisionsSchema); err != nil {
		return fmt.Errorf("failed to create workflow_step_decisions table: %w", err)
	}

	return nil
}
