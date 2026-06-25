package cron

// EngineTrigger identifies workflow engine triggers used by cron handlers.
// Kept local so scheduler/cron does not depend on the removed workflow engine package.
type EngineTrigger string

const (
	TriggerOnHeartbeat   EngineTrigger = "on_heartbeat"
	TriggerOnBudgetAlert EngineTrigger = "on_budget_alert"
)

// OnHeartbeatPayload accompanies TriggerOnHeartbeat.
type OnHeartbeatPayload struct{}

// OnBudgetAlertPayload accompanies TriggerOnBudgetAlert.
type OnBudgetAlertPayload struct {
	BudgetPct int
	Scope     string
}
