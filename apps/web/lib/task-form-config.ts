// Per-workspace customizable "new task" form.
//
// A workspace can define the fields shown in the new-task dialog and a template
// that concatenates the entered values into the task's base prompt (the
// {{task_prompt}} that workflow STEP PROMPTs reference). Each field is
// referenced inside the template — and inside STEP PROMPTs — as
// `{{prompt_<def>}}`. The template itself must NOT use {{task_prompt}} because
// that placeholder IS the template's output and would recurse.

export type TaskFormField = {
  /** Placeholder key referenced as {{prompt_<def>}}. Unique per workspace,
   * English letters/digits/underscore, must start with a letter. */
  def: string;
  /** Human-facing field label shown in the dialog. */
  label: string;
  /** Optional input placeholder text. */
  placeholder?: string;
  /** Whether the field must be filled before the task can be created. */
  required?: boolean;
  /** Render a textarea instead of a single-line input. */
  multiline?: boolean;
};

export type TaskFormConfig = {
  fields: TaskFormField[];
  /** Concatenation template referencing {{prompt_<def>}} tokens. */
  template: string;
};

/** The implicit form used when a workspace has no custom configuration: a
 * single required "Prompt" field passed through unchanged. */
export function defaultTaskFormConfig(): TaskFormConfig {
  return {
    fields: [{ def: "default", label: "Prompt", required: true, multiline: true }],
    template: "{{prompt_default}}",
  };
}

/** Returns the workspace's configured form, or the default form when none is
 * set (no fields). */
export function resolveTaskFormConfig(config: TaskFormConfig | null | undefined): TaskFormConfig {
  if (!config || !Array.isArray(config.fields) || config.fields.length === 0) {
    return defaultTaskFormConfig();
  }
  return config;
}

/** The placeholder token for a field, e.g. def "title" -> "prompt_title". */
export function fieldPlaceholderKey(def: string): string {
  return `prompt_${def}`;
}

export const DEF_PATTERN = /^[a-zA-Z][a-zA-Z0-9_]*$/;

/** Validates a single def value. Returns an error message, or null when valid. */
export function validateDef(def: string, existingDefs: string[]): string | null {
  const trimmed = def.trim();
  if (trimmed === "") return "Key is required";
  if (!DEF_PATTERN.test(trimmed)) {
    return "Key must be English letters/digits/underscore and start with a letter";
  }
  if (existingDefs.includes(trimmed)) return "Key must be unique within the workspace";
  return null;
}

/** True when every required field in the config has a non-empty value. */
export function isTaskFormComplete(
  config: TaskFormConfig,
  values: Record<string, string>,
): boolean {
  return config.fields.every((f) => !f.required || (values[f.def] ?? "").trim() !== "");
}

/**
 * Builds the base task prompt by replacing every {{prompt_<def>}} token in the
 * template with the corresponding entered value. Missing/blank values resolve
 * to an empty string (optional fields left blank).
 */
export function buildPromptFromTemplate(
  template: string,
  values: Record<string, string>,
): string {
  return template.replace(/\{\{(prompt_[a-zA-Z0-9_]+)\}\}/g, (_match, key: string) => {
    return values[key] ?? "";
  });
}
