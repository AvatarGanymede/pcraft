"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@pcraft/ui/card";
import { Input } from "@pcraft/ui/input";
import { Label } from "@pcraft/ui/label";

type ProfileDetailsCardProps = {
  name: string;
  onNameChange: (v: string) => void;
};

export function ProfileDetailsCard({ name, onNameChange }: ProfileDetailsCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Profile Details</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="profile-name">Name</Label>
          <Input id="profile-name" value={name} onChange={(e) => onNameChange(e.target.value)} />
        </div>
      </CardContent>
    </Card>
  );
}
