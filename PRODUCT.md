# Product

## Users

pcraft is for developers driving automated GUI/game development with **Claude Code + dev-gui-plugin** on **Perforce** workspaces. Users arrive with active engineering context: tasks in flight, a Claude Code session running a dev-gui pipeline, P4 changelists pending submit, and file locks coordinating concurrent work against shared P4 clients.

## Product Purpose

pcraft is the GUI control plane for a single, opinionated workflow: turn a requirement into working code through Claude Code's dev-gui-plugin pipeline, with Perforce as the source of truth and pcraft as the lock coordinator. The product should help users launch tasks, watch pipeline progress, resolve file contention, and move work from Backlog to Closed without losing trust in the underlying P4 state.

Success means users can quickly answer:

- What task, P4 workspace, changelist, and dev-gui phase am I in?
- What is the pipeline doing right now, and which files are checked out?
- Why did a task revert to Backlog, and which task is blocking it?
- Is the changelist ready to submit, and what happens when I confirm Closed?

## Core Workflow

1. **Create** — pick a P4 workspace, panel ID, requirements, optional prefab. Saved as Backlog.
2. **Launch** — pcraft injects the dev-gui-plugin prompt and starts Claude Code. Task → In Progress.
3. **Execute** — Claude runs 8 phases; pcraft checks out files via P4 per Write/Edit, reverting to Backlog on conflict.
4. **Done** — pipeline finishes; session stays open for follow-up; changelist pending manual submit.
5. **Close** — user submits in P4 and confirms; pcraft verifies the changelist is submitted, terminates the session, releases locks, and auto-wakes eligible Backlog tasks.

## Brand Personality

Focused, technical, composed.

The interface should feel like a reliable developer workbench: direct, legible, and confident under pressure. It should avoid performative futurism and keep the user oriented in real engineering state — P4 changelists, file locks, and pipeline phases.

## Anti-references

- Generic SaaS dashboards with decorative metrics and inflated spacing.
- Decorative AI gradients, glowing blobs, and "magic" visual language.
- Cluttered IDE chrome that competes with the actual work.
- Novelty terminal cosplay, fake hacker styling, and retro effects used as decoration.

## Design Principles

### Orientation before controls

The user should first understand their current scope: P4 workspace, task, session, changelist, and dev-gui phase. Controls follow orientation instead of crowding it.

### Command-first secondary actions

Primary actions should be obvious and state-aware (launch, confirm submit). Secondary actions should be available through predictable command surfaces — menus, panel headers, contextual toolbars — rather than scattered across global chrome.

### Density without clutter

pcraft is a professional tool with real state density. Use compact controls and information-rich rows, but preserve grouping, alignment, and hierarchy so the surface can be scanned without fatigue.

### State visible but calm

Running pipelines, P4 checkout status, file locks, changelist state, and phase progress should be visible without becoming alarmist. Color and motion indicate meaning, not decoration.

### Familiar affordances preserve developer trust

Use recognizable developer patterns for navigation, changelists, sessions, panels, filters, and settings. Novel interaction should be reserved for places where the workflow genuinely needs it — like the dev-gui phase progress and P4 lock conflict surfacing.

## Accessibility & Inclusion

Target WCAG AA. Interaction must remain reachable by keyboard, with visible focus states and screen-reader labels for icon-only controls. Status must not depend on color alone; pair color with labels, icons, tooltips, or text. Support reduced motion by avoiding nonessential animation and keeping state transitions short.
