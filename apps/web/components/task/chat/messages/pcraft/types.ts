// Shared types used by Pcraft tool renderers.
//
// `RendererProps` is what the dispatcher passes to every per-tool renderer.
// Renderers receive the parsed args and result (already unwrapped from the MCP
// envelope) plus the tool status, and are responsible for returning a fully
// composed `<PcraftRow>` element.

import type { ReactElement } from "react";
import type { PcraftStatus } from "./shared";

export type PcraftRendererProps = {
  args: Record<string, unknown> | undefined;
  result: unknown;
  status: PcraftStatus;
};

export type PcraftRenderer = (props: PcraftRendererProps) => ReactElement;
