"use client";

import { PageTopbar } from "@/components/page-topbar";

type RemovedIntegrationPageProps = {
  name: string;
};

export function RemovedIntegrationPage({ name }: RemovedIntegrationPageProps) {
  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <PageTopbar title={`${name} integration`} />
      <div className="p-6 text-sm text-muted-foreground">
        The {name} integration is not available in this build.
      </div>
    </div>
  );
}
