package office

import (
	"github.com/gin-gonic/gin"
	"github.com/AvatarGanymede/pcraft/internal/common/logger"
	"github.com/AvatarGanymede/pcraft/internal/office/agents"
	"github.com/AvatarGanymede/pcraft/internal/office/approvals"
	"github.com/AvatarGanymede/pcraft/internal/office/channels"
	"github.com/AvatarGanymede/pcraft/internal/office/config"
	"github.com/AvatarGanymede/pcraft/internal/office/costs"
	"github.com/AvatarGanymede/pcraft/internal/office/dashboard"
	"github.com/AvatarGanymede/pcraft/internal/office/labels"
	"github.com/AvatarGanymede/pcraft/internal/office/onboarding"
	"github.com/AvatarGanymede/pcraft/internal/office/projects"
	"github.com/AvatarGanymede/pcraft/internal/office/routines"
	officeruntime "github.com/AvatarGanymede/pcraft/internal/office/runtime"
	"github.com/AvatarGanymede/pcraft/internal/office/skills"
	"github.com/AvatarGanymede/pcraft/internal/office/tree_controls"
	"github.com/AvatarGanymede/pcraft/internal/office/workspaces"
)

// RegisterAllRoutes delegates route registration to each feature package.
func RegisterAllRoutes(router *gin.RouterGroup, svcs *Services, log *logger.Logger) {
	agents.RegisterRoutes(router, svcs.Agents, log)
	officeruntime.RegisterRoutes(router, officeruntime.NewHandler(
		svcs.Agents,
		officeruntime.NewActions(officeruntime.ActionDependencies{
			Comments:      svcs.Dashboard,
			Tasks:         svcs.Workspaces,
			TaskStatus:    svcs.Dashboard,
			Agents:        svcs.Agents,
			Approvals:     svcs.Approvals,
			Runs:          svcs.Workspaces,
			AgentModifier: svcs.Agents,
			Skills:        svcs.Skills,
		}),
		svcs.Skills,
		svcs.Workspaces,
	))

	skillsHandler := skills.NewHandler(svcs.Skills)
	skillsHandler.RegisterRoutes(router)

	projectsHandler := projects.NewHandler(svcs.Projects)
	projects.RegisterRoutes(router, projectsHandler)

	costsHandler := costs.NewHandler(svcs.Costs)
	costsHandler.RegisterRoutes(router)

	routinesHandler := routines.NewHandler(svcs.Routines)
	routines.RegisterRoutes(router, routinesHandler)

	approvals.RegisterRoutes(router, svcs.Approvals)

	channelsHandler := channels.NewHandler(svcs.Channels)
	channels.RegisterRoutes(router, channelsHandler)

	configHandler := config.NewHandler(svcs.Config, log)
	config.RegisterRoutes(router, configHandler)

	dashboard.RegisterRoutes(router, svcs.Dashboard, svcs.Repo, svcs.GitManager, log)

	if svcs.Documents != nil {
		docHandler := dashboard.NewDocumentHandler(svcs.Documents, svcs.KandevHome, log)
		dashboard.RegisterDocumentRoutes(router, docHandler)
	}

	onboarding.RegisterRoutes(router, svcs.Onboarding, log)

	labels.RegisterRoutes(router, svcs.Labels)

	tree_controls.RegisterRoutes(router, tree_controls.NewHandler(svcs.TreeControls))
	workspaces.RegisterRoutes(router, workspaces.NewHandler(svcs.Workspaces))
}
