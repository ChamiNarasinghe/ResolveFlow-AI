import { Badge } from "../ui/Badge";
import type { BadgeVariant } from "../ui/Badge";
import type { TicketStatus } from "../../types";

const VARIANT: Record<TicketStatus, BadgeVariant> = {
  open: "default",
  in_progress: "info",
  waiting_for_customer: "warning",
  resolved: "success",
  closed: "neutral",
};

const LABEL: Record<TicketStatus, string> = {
  open: "Open",
  in_progress: "In Progress",
  waiting_for_customer: "Waiting",
  resolved: "Resolved",
  closed: "Closed",
};

export function TicketStatusBadge({ status }: { status: TicketStatus }) {
  return <Badge variant={VARIANT[status]}>{LABEL[status]}</Badge>;
}
