"use client";

import { Badge } from "@pcraft/ui/badge";
import { cn } from "@/lib/utils";

const GUI_PHASES = [
  "gui-plan",
  "gui-draft",
  "gui-prefab",
  "gui-config",
  "gui-review",
  "gui-verify",
  "gui-improve",
  "gui-learn",
] as const;

const PHASE_LABELS: Record<(typeof GUI_PHASES)[number], string> = {
  "gui-plan": "Plan",
  "gui-draft": "Draft",
  "gui-prefab": "Prefab",
  "gui-config": "Config",
  "gui-review": "Review",
  "gui-verify": "Verify",
  "gui-improve": "Improve",
  "gui-learn": "Learn",
};

type GuiPhaseProgressProps = {
  phase?: string | null;
  phaseStatus?: string | null;
  className?: string;
};

function phaseState(
  phaseId: (typeof GUI_PHASES)[number],
  current?: string | null,
  status?: string | null,
): "completed" | "current" | "future" {
  const currentIdx = GUI_PHASES.findIndex((p) => p === current);
  const idx = GUI_PHASES.findIndex((p) => p === phaseId);
  if (currentIdx < 0) {
    return idx === 0 ? "current" : "future";
  }
  if (idx < currentIdx) return "completed";
  if (idx === currentIdx) {
    if (status === "done" || status === "terminal") return "completed";
    return "current";
  }
  return "future";
}

export function GuiPhaseProgress({ phase, phaseStatus, className }: GuiPhaseProgressProps) {
  return (
    <div className={cn("flex flex-wrap gap-1.5", className)} data-testid="gui-phase-progress">
      {GUI_PHASES.map((phaseId) => {
        const state = phaseState(phaseId, phase, phaseStatus);
        return (
          <Badge
            key={phaseId}
            variant={state === "current" ? "default" : state === "completed" ? "secondary" : "outline"}
            className={cn("text-[11px] cursor-default", state === "future" && "text-muted-foreground")}
          >
            {PHASE_LABELS[phaseId]}
          </Badge>
        );
      })}
    </div>
  );
}
