"use client";

import { useCallback, useMemo } from "react";
import type { Branch } from "@/lib/types/http";

export type ParsedGitHubUrl = {
  owner: string;
  repo: string;
  prNumber?: number;
  branch?: string;
};

export type PRInfo = {
  suggestedTitle?: string;
  headBranch?: string;
  baseBranch?: string;
  owner?: string;
  repo?: string;
  prNumber?: number;
};

export type UseBranchesByURLResult = {
  branches: (url: string) => Branch[];
  loading: (url: string) => boolean;
  ensure: (url: string) => void;
};

export type UsePRInfoByURLResult = {
  info: (url: string) => PRInfo | null;
  loading: (url: string) => boolean;
  ensure: (url: string) => void;
};

const EMPTY_BRANCHES: Branch[] = [];

export function parseGitHubAnyUrl(_url: string): ParsedGitHubUrl | null {
  return null;
}

export function useBranchesByURL(): UseBranchesByURLResult {
  const branches = useCallback((_url: string) => EMPTY_BRANCHES, []);
  const loading = useCallback((_url: string) => false, []);
  const ensure = useCallback((_url: string) => {}, []);
  return useMemo(() => ({ branches, loading, ensure }), [branches, loading, ensure]);
}

export function usePRInfoByURL(): UsePRInfoByURLResult {
  const info = useCallback((_url: string) => null, []);
  const loading = useCallback((_url: string) => false, []);
  const ensure = useCallback((_url: string) => {}, []);
  return useMemo(() => ({ info, loading, ensure }), [info, loading, ensure]);
}
