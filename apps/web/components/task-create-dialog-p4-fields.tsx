"use client";

import type { ReactNode } from "react";
import { Input } from "@pcraft/ui/input";
import { Textarea } from "@pcraft/ui/textarea";
import { Label } from "@pcraft/ui/label";
import type { TaskFormConfig } from "@/lib/task-form-config";

/**
 * Create-dialog form values. `values` holds the workspace's custom new-task
 * form entries keyed by each field's `def`. The task's P4 workspace is no
 * longer chosen here — it is derived from the selected pcraft workspace's
 * bound P4 client (1:1), so there is no per-task P4 selector.
 */
export type P4TaskFormValues = {
  values: Record<string, string>;
};

/**
 * Shared field label row: bold label on the left, optional inline hint on the
 * right, and a destructive-colored asterisk when the field is required. Keeps
 * every field in the create dialog visually consistent.
 */
function FieldLabel({
  htmlFor,
  required,
  hint,
  children,
}: {
  htmlFor: string;
  required?: boolean;
  hint?: string;
  children: ReactNode;
}) {
  return (
    <div className="flex items-baseline justify-between gap-2">
      <Label htmlFor={htmlFor}>
        {children}
        {required ? <span className="text-destructive ml-0.5">*</span> : null}
      </Label>
      {hint ? (
        <span className="text-[11px] leading-none text-muted-foreground">{hint}</span>
      ) : null}
    </div>
  );
}

type DynamicTaskFormProps = {
  config: TaskFormConfig;
  values: Record<string, string>;
  onChange: (patch: Partial<P4TaskFormValues>) => void;
  disabled?: boolean;
};

/**
 * Renders the workspace's custom new-task form. Each configured field becomes a
 * single-line input or a textarea (when `multiline`). Values are tracked by the
 * field's `def`; the dialog later concatenates them via the workspace template.
 */
export function DynamicTaskForm({ config, values, onChange, disabled }: DynamicTaskFormProps) {
  const setValue = (def: string, value: string) => {
    onChange({ values: { ...values, [def]: value } });
  };
  return (
    <>
      {config.fields.map((field) => {
        const id = `task-form-${field.def}`;
        return (
          <div key={field.def} className="grid gap-1.5">
            <FieldLabel htmlFor={id} required={field.required}>
              {field.label}
            </FieldLabel>
            {field.multiline ? (
              <Textarea
                id={id}
                value={values[field.def] ?? ""}
                onChange={(e) => setValue(field.def, e.target.value)}
                disabled={disabled}
                placeholder={field.placeholder}
                rows={4}
              />
            ) : (
              <Input
                id={id}
                value={values[field.def] ?? ""}
                onChange={(e) => setValue(field.def, e.target.value)}
                disabled={disabled}
                placeholder={field.placeholder}
              />
            )}
          </div>
        );
      })}
    </>
  );
}
