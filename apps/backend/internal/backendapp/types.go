package backendapp

import (
	settingsstore "github.com/AvatarGanymede/pcraft/internal/agent/settings/store"
	analyticsrepository "github.com/AvatarGanymede/pcraft/internal/analytics/repository"
	"github.com/AvatarGanymede/pcraft/internal/automation"
	editorservice "github.com/AvatarGanymede/pcraft/internal/editors/service"
	editorstore "github.com/AvatarGanymede/pcraft/internal/editors/store"
	"github.com/AvatarGanymede/pcraft/internal/github"
	"github.com/AvatarGanymede/pcraft/internal/gitlab"
	"github.com/AvatarGanymede/pcraft/internal/jira"
	"github.com/AvatarGanymede/pcraft/internal/linear"
	notificationservice "github.com/AvatarGanymede/pcraft/internal/notifications/service"
	notificationstore "github.com/AvatarGanymede/pcraft/internal/notifications/store"
	office "github.com/AvatarGanymede/pcraft/internal/office"
	officesqlite "github.com/AvatarGanymede/pcraft/internal/office/repository/sqlite"
	officeservice "github.com/AvatarGanymede/pcraft/internal/office/service"
	promptservice "github.com/AvatarGanymede/pcraft/internal/prompts/service"
	promptstore "github.com/AvatarGanymede/pcraft/internal/prompts/store"
	"github.com/AvatarGanymede/pcraft/internal/runtimeflags"
	"github.com/AvatarGanymede/pcraft/internal/secrets"
	"github.com/AvatarGanymede/pcraft/internal/sentry"
	"github.com/AvatarGanymede/pcraft/internal/slack"
	sqliterepo "github.com/AvatarGanymede/pcraft/internal/task/repository/sqlite"
	taskservice "github.com/AvatarGanymede/pcraft/internal/task/service"
	"github.com/AvatarGanymede/pcraft/internal/task/share"
	terminalrepo "github.com/AvatarGanymede/pcraft/internal/terminal/repository"
	terminalservice "github.com/AvatarGanymede/pcraft/internal/terminal/service"
	userservice "github.com/AvatarGanymede/pcraft/internal/user/service"
	userstore "github.com/AvatarGanymede/pcraft/internal/user/store"
	utilityservice "github.com/AvatarGanymede/pcraft/internal/utility/service"
	utilitystore "github.com/AvatarGanymede/pcraft/internal/utility/store"
	workflowrepository "github.com/AvatarGanymede/pcraft/internal/workflow/repository"
	workflowservice "github.com/AvatarGanymede/pcraft/internal/workflow/service"
	"github.com/AvatarGanymede/pcraft/internal/worktree"
)

type Repositories struct {
	Task          *sqliterepo.Repository
	Analytics     analyticsrepository.Repository
	AgentSettings settingsstore.Repository
	User          userstore.Repository
	Notification  notificationstore.Repository
	Editor        editorstore.Repository
	Prompts       promptstore.Repository
	Utility       utilitystore.Repository
	Workflow      *workflowrepository.Repository
	Secrets       secrets.SecretStore
	Office        *officesqlite.Repository
	Terminal      *terminalrepo.Repository
	RuntimeFlags  *runtimeflags.SQLiteStore
}

type Services struct {
	Task         *taskservice.Service
	User         *userservice.Service
	Editor       *editorservice.Service
	Notification *notificationservice.Service
	Prompts      *promptservice.Service
	Utility      *utilityservice.Service
	Workflow     *workflowservice.Service
	GitHub       *github.Service
	GitLab       *gitlab.Service
	Jira         *jira.Service
	Linear       *linear.Service
	Sentry       *sentry.Service
	Slack        *slack.Service
	Share        *share.HTTPHandlers
	Office       *officeservice.Service
	OfficeSvcs   *office.Services
	// OrchScheduler is the office SchedulerIntegration constructed by
	// startOfficeSchedulersAndGC. Exposed here so registerRoutes can
	// wire SetTaskContextProvider after the HandoffService is built.
	OrchScheduler *officeservice.SchedulerIntegration
	// WorktreeMgr is the worktree manager. Exposed so the office GC can
	// consult it as the authoritative inventory of live worktrees.
	WorktreeMgr *worktree.Manager
	// Terminal is the first-class user-terminal service (rename, park, etc.).
	// Wired into the gateway once lifecycle.Manager is up so the PTY backend
	// is available.
	Terminal     *terminalservice.Service
	RuntimeFlags *runtimeflags.Service
	// Automation is the trigger-based automation subsystem (cron, GitHub PR
	// events, webhooks). Independent of Office — has its own scheduler and
	// creates tasks via the task service.
	Automation *automation.Components
}
