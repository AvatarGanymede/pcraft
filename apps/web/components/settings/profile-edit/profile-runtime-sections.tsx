"use client";

import type { ExecutorProfile } from "@/lib/types/http";
import type { SecretListItem } from "@/lib/types/http-secrets";
import {
  RemoteCredentialsCard,
  type GitIdentityMode,
  type GitIdentityState,
} from "@/components/settings/profile-edit/remote-credentials-card";

type RemoteSectionsProps = {
  isRemote: boolean;
  remoteCredentials: string[];
  onRemoteCredentialsChange: (ids: string[]) => void;
  agentEnvVars: Record<string, string | null>;
  onAgentEnvVarChange: (agentId: string, secretId: string | null) => void;
  gitIdentityMode: GitIdentityMode;
  onGitIdentityModeChange: (mode: GitIdentityMode) => void;
  gitUserName: string;
  gitUserEmail: string;
  onGitUserNameChange: (value: string) => void;
  onGitUserEmailChange: (value: string) => void;
  localGitIdentity: GitIdentityState;
  secrets: SecretListItem[];
};

export function RemoteSections({
  isRemote,
  remoteCredentials,
  onRemoteCredentialsChange,
  agentEnvVars,
  onAgentEnvVarChange,
  gitIdentityMode,
  onGitIdentityModeChange,
  gitUserName,
  gitUserEmail,
  onGitUserNameChange,
  onGitUserEmailChange,
  localGitIdentity,
  secrets,
}: RemoteSectionsProps) {
  return (
    <RemoteCredentialsCard
      isRemote={isRemote}
      selectedIds={remoteCredentials}
      onChange={onRemoteCredentialsChange}
      agentEnvVars={agentEnvVars}
      onAgentEnvVarChange={onAgentEnvVarChange}
      secrets={secrets}
      gitIdentityMode={gitIdentityMode}
      onGitIdentityModeChange={onGitIdentityModeChange}
      gitUserName={gitUserName}
      gitUserEmail={gitUserEmail}
      onGitUserNameChange={onGitUserNameChange}
      onGitUserEmailChange={onGitUserEmailChange}
      localGitIdentity={localGitIdentity}
    />
  );
}
