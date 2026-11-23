import { Badge } from "../ui/Badge";
import type { TeamMemberRole } from "../../types";

const VARIANT = {
  lead: "default",
  member: "neutral",
} as const;

const LABEL: Record<TeamMemberRole, string> = {
  lead: "Lead",
  member: "Member",
};

export function TeamRoleBadge({ role }: { role: TeamMemberRole }) {
  return <Badge variant={VARIANT[role]}>{LABEL[role]}</Badge>;
}
