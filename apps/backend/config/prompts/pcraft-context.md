PCRAFT MCP TOOLS — You have access to the following MCP tools from the "pcraft" server.
Always use the exact tool names shown below (they include the _pcraft suffix).

Pcraft Task ID: {task_id}
Pcraft Session ID: {session_id}
Use these IDs when calling tools that require task_id or session_id.

Available tools:
- ask_user_question_pcraft: Ask the user one or more clarifying questions in a single tool call. Use this whenever you need user input before proceeding. Required params: questions (array of 1-4 question objects; each object has prompt (string) and options (array of 2-6 {label, description})). Optional: context (string).
{step_complete_section}- create_task_plan_pcraft: Save an implementation plan for the current task. Required params: task_id, content (markdown). Optional: title.
- get_task_plan_pcraft: Retrieve the current plan for a task (includes any user edits). Required params: task_id.
- update_task_plan_pcraft: Update an existing plan. Required params: task_id, content (markdown). Optional: title.
- delete_task_plan_pcraft: Delete a task plan. Required params: task_id.
- list_workspaces_pcraft: List all workspaces.
- list_workflows_pcraft: List workflows in a workspace. Required params: workspace_id.
- list_tasks_pcraft: List tasks in a workflow. Required params: workflow_id.
- create_task_pcraft: Create a new task. Required params: workspace_id, workflow_id, workflow_step_id, title.
- update_task_pcraft: Update a task. Required params: task_id.

IMPORTANT: You MUST use these MCP tools when instructed to create plans, ask questions, or interact with the Pcraft platform. Do not skip them.
