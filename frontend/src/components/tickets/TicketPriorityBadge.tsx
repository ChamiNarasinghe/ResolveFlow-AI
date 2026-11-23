import { Badge } from "../ui/Badge";
import type { BadgeVariant } from "../ui/Badge";
import type { TicketPriority } from "../../types";

const VARIANT: Record<TicketPriority, BadgeVariant> = {
  urgent: "danger",
  high: "warning",
  medium: "default",
  low: "neutral",
};

const LABEL: Record<TicketPriority, string> = {
  urgent: "Urgent",
  high: "High",
  medium: "Medium",
  low: "Low",
};

export function TicketPriorityBadge({ priority }: { priority: TicketPriority }) {
  return <Badge variant={VARIANT[priority]}>{LABEL[priority]}</Badge>;
}
