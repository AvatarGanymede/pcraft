export const STORAGE_KEYS = {
  BACKEND_URL: "pcraft.settings.backendUrl",
  ONBOARDING_COMPLETED: "pcraft.onboarding.completed",
  LAST_AGENT_PROFILE_ID: "pcraft.dialog.lastAgentProfileId",
  LAST_REPOSITORY_ID: "pcraft.dialog.lastRepositoryId",
  LAST_BRANCH: "pcraft.dialog.lastBranch",
  LAST_EXECUTOR_PROFILE_ID: "pcraft.dialog.lastExecutorProfileId",
} as const;

export const DEFAULT_BACKEND_URL = "http://localhost:38429";

// Kanban Preview Panel Settings
export const PREVIEW_PANEL = {
  // Minimum width of the preview panel in pixels (prevents panel from being too narrow)
  MIN_WIDTH_PX: 300,

  // Default width of the preview panel in pixels when first opened
  DEFAULT_WIDTH_PX: 500,

  // Maximum width of the preview panel in viewport width percentage (prevents covering entire screen)
  MAX_WIDTH_VW: 95,

  // Minimum width of the kanban board as a percentage of viewport before the panel switches to floating mode
  MIN_KANBAN_WIDTH_PERCENT: 50,
} as const;
