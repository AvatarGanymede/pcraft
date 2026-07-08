"use client";

import { useMemo, useState } from "react";
import { IconPlus, IconTrash, IconArrowUp, IconArrowDown } from "@tabler/icons-react";
import { Button } from "@pcraft/ui/button";
import { Input } from "@pcraft/ui/input";
import { Label } from "@pcraft/ui/label";
import { Checkbox } from "@pcraft/ui/checkbox";
import { Card, CardContent, CardHeader, CardTitle } from "@pcraft/ui/card";
import { updateWorkspaceAction } from "@/app/actions/workspaces";
import { useRequest } from "@/lib/http/use-request";
import { useToast } from "@/components/toast-provider";
import { useAppStore } from "@/components/state-provider";
import { UnsavedChangesBadge, UnsavedSaveButton } from "@/components/settings/unsaved-indicator";
import {
  ScriptEditor,
  computeEditorHeight,
} from "@/components/settings/profile-edit/script-editor";
import type { ScriptPlaceholder } from "@/components/settings/profile-edit/script-editor-completions";
import {
  type TaskFormConfig,
  type TaskFormField,
  resolveTaskFormConfig,
  validateDef,
  fieldPlaceholderKey,
} from "@/lib/task-form-config";

type WorkspaceTaskFormCardProps = {
  workspaceId: string;
  initialConfig?: TaskFormConfig | null;
};

function moveItem<T>(arr: T[], from: number, to: number): T[] {
  if (to < 0 || to >= arr.length) return arr;
  const next = arr.slice();
  const [item] = next.splice(from, 1);
  next.splice(to, 0, item);
  return next;
}

/** Per-field validation errors keyed by field index. */
function computeFieldErrors(fields: TaskFormField[]): Record<number, string> {
  const errors: Record<number, string> = {};
  fields.forEach((field, i) => {
    const others = fields.filter((_, j) => j !== i).map((f) => f.def.trim());
    const defErr = validateDef(field.def, others);
    if (defErr) {
      errors[i] = defErr;
      return;
    }
    if (field.label.trim() === "") errors[i] = "Label is required";
  });
  return errors;
}

type FieldRowProps = {
  field: TaskFormField;
  index: number;
  total: number;
  error?: string;
  onChange: (patch: Partial<TaskFormField>) => void;
  onRemove: () => void;
  onMove: (delta: number) => void;
};

function FieldRow({ field, index, total, error, onChange, onRemove, onMove }: FieldRowProps) {
  return (
    <div className="rounded-md border border-border p-3 space-y-3">
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs font-medium text-muted-foreground">
          Field {index + 1} ·{" "}
          <code className="bg-muted px-1 rounded">{`{{${fieldPlaceholderKey(field.def || "key")}}}`}</code>
        </span>
        <div className="flex items-center gap-1">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            disabled={index === 0}
            onClick={() => onMove(-1)}
            aria-label="Move up"
          >
            <IconArrowUp className="h-3.5 w-3.5" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            disabled={index === total - 1}
            onClick={() => onMove(1)}
            aria-label="Move down"
          >
            <IconArrowDown className="h-3.5 w-3.5" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-7 w-7 text-destructive hover:text-destructive"
            onClick={onRemove}
            aria-label="Remove field"
          >
            <IconTrash className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="grid gap-1.5">
          <Label className="text-xs">Label</Label>
          <Input
            value={field.label}
            onChange={(e) => onChange({ label: e.target.value })}
            placeholder="e.g. Requirement"
          />
        </div>
        <div className="grid gap-1.5">
          <Label className="text-xs">Key (placeholder, English, unique)</Label>
          <Input
            value={field.def}
            onChange={(e) => onChange({ def: e.target.value })}
            placeholder="e.g. requirement"
          />
        </div>
        <div className="grid gap-1.5 sm:col-span-2">
          <Label className="text-xs">Placeholder (optional)</Label>
          <Input
            value={field.placeholder ?? ""}
            onChange={(e) => onChange({ placeholder: e.target.value })}
            placeholder="Hint text shown inside the input"
          />
        </div>
      </div>
      <div className="flex items-center gap-6">
        <label className="flex items-center gap-2 text-sm cursor-pointer">
          <Checkbox
            checked={field.required === true}
            onCheckedChange={(c) => onChange({ required: c === true })}
          />
          Required
        </label>
        <label className="flex items-center gap-2 text-sm cursor-pointer">
          <Checkbox
            checked={field.multiline === true}
            onCheckedChange={(c) => onChange({ multiline: c === true })}
          />
          Multiline
        </label>
      </div>
      {error ? <p className="text-xs text-destructive">{error}</p> : null}
    </div>
  );
}

type FieldsEditorProps = {
  fields: TaskFormField[];
  fieldErrors: Record<number, string>;
  onUpdate: (index: number, patch: Partial<TaskFormField>) => void;
  onRemove: (index: number) => void;
  onMove: (index: number, delta: number) => void;
  onAdd: () => void;
};

function FieldsEditor({ fields, fieldErrors, onUpdate, onRemove, onMove, onAdd }: FieldsEditorProps) {
  return (
    <div className="space-y-3">
      {fields.map((field, i) => (
        <FieldRow
          key={i}
          field={field}
          index={i}
          total={fields.length}
          error={fieldErrors[i]}
          onChange={(patch) => onUpdate(i, patch)}
          onRemove={() => onRemove(i)}
          onMove={(delta) => onMove(i, delta)}
        />
      ))}
      {fields.length === 0 && <p className="text-xs text-destructive">At least one field is required.</p>}
      <Button type="button" variant="outline" size="sm" onClick={onAdd} className="gap-1.5">
        <IconPlus className="h-3.5 w-3.5" />
        Add field
      </Button>
    </div>
  );
}

type TemplateEditorProps = {
  template: string;
  onChange: (value: string) => void;
  placeholders: ScriptPlaceholder[];
  usesTaskPrompt: boolean;
};

function TemplateEditor({ template, onChange, placeholders, usesTaskPrompt }: TemplateEditorProps) {
  return (
    <div className="space-y-2">
      <Label className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
        Prompt template
      </Label>
      <div className="rounded-md border overflow-hidden">
        <ScriptEditor
          value={template}
          onChange={onChange}
          language="markdown"
          height={computeEditorHeight(template)}
          lineNumbers="off"
          placeholders={placeholders}
        />
      </div>
      <p className="text-[11px] text-muted-foreground/60">
        Type {"{{"} to see available placeholders.{" "}
        <code className="bg-muted px-1 py-0.5 rounded text-[10px]">{"{{task_prompt}}"}</code> is not
        allowed here (it is this template&apos;s output and would recurse infinitely).
      </p>
      {usesTaskPrompt && (
        <p className="text-xs text-destructive">{"{{task_prompt}}"} cannot be used in this template.</p>
      )}
    </div>
  );
}

function useTaskFormDraft(workspaceId: string, initialConfig?: TaskFormConfig | null) {
  const { toast } = useToast();
  const workspaces = useAppStore((state) => state.workspaces.items);
  const setWorkspaces = useAppStore((state) => state.setWorkspaces);
  const saveRequest = useRequest(updateWorkspaceAction);

  const resolved = useMemo(() => resolveTaskFormConfig(initialConfig), [initialConfig]);
  const [fields, setFields] = useState<TaskFormField[]>(resolved.fields);
  const [template, setTemplate] = useState(resolved.template);
  const [savedSnapshot, setSavedSnapshot] = useState(() =>
    JSON.stringify({ fields: resolved.fields, template: resolved.template }),
  );

  const fieldErrors = useMemo(() => computeFieldErrors(fields), [fields]);
  const usesTaskPrompt = template.includes("{{task_prompt}}");
  const hasErrors = Object.keys(fieldErrors).length > 0 || fields.length === 0 || usesTaskPrompt;
  const isDirty = JSON.stringify({ fields, template }) !== savedSnapshot;

  // Only fields with a valid, unique key can be referenced from the template.
  const placeholders: ScriptPlaceholder[] = useMemo(
    () =>
      fields
        .filter((f) => f.def.trim() !== "" && !fieldErrors[fields.indexOf(f)])
        .map((f) => ({
          key: fieldPlaceholderKey(f.def.trim()),
          description: f.label || f.def,
          example: f.placeholder ?? "",
          executor_types: [],
        })),
    [fields, fieldErrors],
  );

  const updateField = (index: number, patch: Partial<TaskFormField>) =>
    setFields((prev) => prev.map((f, i) => (i === index ? { ...f, ...patch } : f)));
  const addField = () =>
    setFields((prev) => [...prev, { def: "", label: "", required: false, multiline: false }]);
  const removeField = (index: number) => setFields((prev) => prev.filter((_, i) => i !== index));
  const moveField = (index: number, delta: number) =>
    setFields((prev) => moveItem(prev, index, index + delta));

  const handleSave = async () => {
    if (!isDirty || hasErrors) return;
    const config: TaskFormConfig = {
      fields: fields.map((f) => ({
        def: f.def.trim(),
        label: f.label.trim(),
        placeholder: f.placeholder?.trim() || undefined,
        required: f.required || undefined,
        multiline: f.multiline || undefined,
      })),
      template,
    };
    try {
      await saveRequest.run(workspaceId, { task_form_config: config });
      setSavedSnapshot(JSON.stringify({ fields, template }));
      setWorkspaces(
        workspaces.map((ws) => (ws.id === workspaceId ? { ...ws, task_form_config: config } : ws)),
      );
      toast({ title: "Task form saved", variant: "success" });
    } catch (error) {
      toast({
        title: "Failed to save task form",
        description: error instanceof Error ? error.message : "Request failed",
        variant: "error",
      });
    }
  };

  return {
    fields,
    template,
    setTemplate,
    fieldErrors,
    usesTaskPrompt,
    hasErrors,
    isDirty,
    placeholders,
    updateField,
    addField,
    removeField,
    moveField,
    handleSave,
    saveRequest,
  };
}

export function WorkspaceTaskFormCard({ workspaceId, initialConfig }: WorkspaceTaskFormCardProps) {
  const {
    fields,
    template,
    setTemplate,
    fieldErrors,
    usesTaskPrompt,
    hasErrors,
    isDirty,
    placeholders,
    updateField,
    addField,
    removeField,
    moveField,
    handleSave,
    saveRequest,
  } = useTaskFormDraft(workspaceId, initialConfig);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <span>Task Form</span>
          {isDirty && <UnsavedChangesBadge />}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <p className="text-sm text-muted-foreground">
            Customize the fields shown when creating a task in this workspace, and use the template
            below to assemble them into the task prompt. Reference a field&apos;s value with{" "}
            <code className="bg-muted px-1 rounded">{"{{prompt_<key>}}"}</code>.
          </p>
          <FieldsEditor
            fields={fields}
            fieldErrors={fieldErrors}
            onUpdate={updateField}
            onRemove={removeField}
            onMove={moveField}
            onAdd={addField}
          />
          <TemplateEditor
            template={template}
            onChange={setTemplate}
            placeholders={placeholders}
            usesTaskPrompt={usesTaskPrompt}
          />
          <div className="flex justify-end pt-2">
            <UnsavedSaveButton
              isDirty={isDirty && !hasErrors}
              isLoading={saveRequest.isLoading}
              status={saveRequest.status}
              onClick={handleSave}
            />
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
