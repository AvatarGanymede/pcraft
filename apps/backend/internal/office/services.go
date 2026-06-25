package office

import (
	"github.com/AvatarGanymede/pcraft/internal/office/agents"
	"github.com/AvatarGanymede/pcraft/internal/office/approvals"
	"github.com/AvatarGanymede/pcraft/internal/office/channels"
	"github.com/AvatarGanymede/pcraft/internal/office/config"
	"github.com/AvatarGanymede/pcraft/internal/office/configloader"
	"github.com/AvatarGanymede/pcraft/internal/office/costs"
	"github.com/AvatarGanymede/pcraft/internal/office/dashboard"
	"github.com/AvatarGanymede/pcraft/internal/office/infra"
	"github.com/AvatarGanymede/pcraft/internal/office/labels"
	"github.com/AvatarGanymede/pcraft/internal/office/onboarding"
	"github.com/AvatarGanymede/pcraft/internal/office/projects"
	"github.com/AvatarGanymede/pcraft/internal/office/repository/sqlite"
	"github.com/AvatarGanymede/pcraft/internal/office/routines"
	"github.com/AvatarGanymede/pcraft/internal/office/scheduler"
	officeservice "github.com/AvatarGanymede/pcraft/internal/office/service"
	"github.com/AvatarGanymede/pcraft/internal/office/skills"
	taskservice "github.com/AvatarGanymede/pcraft/internal/task/service"
)

// Services holds references to all feature services in the office domain.
// It is the central wiring point for HTTP handlers and background jobs.
type Services struct {
	Agents       *agents.AgentService
	Skills       *skills.SkillService
	Projects     *projects.ProjectService
	Costs        *costs.CostService
	Routines     *routines.RoutineService
	Approvals    *approvals.ApprovalService
	Channels     *channels.ChannelService
	Config       *config.ConfigService
	Dashboard    *dashboard.DashboardService
	Labels       *labels.LabelService
	Onboarding   *onboarding.OnboardingService
	Scheduler    *scheduler.SchedulerService
	TreeControls *officeservice.Service
	Workspaces   *officeservice.Service
	Documents    *taskservice.DocumentService
	GC           *infra.GarbageCollector
	Reconciler   *infra.Reconciler
	Repo         *sqlite.Repository
	GitManager   *configloader.GitManager
	// KandevHome is the base storage directory for attachment files.
	KandevHome string
}
