import { ticketsService } from "./tickets";
import { teamsService } from "./teams";
import type { PaginatedResponse, Team, Ticket, TicketStatus } from "../types";

export interface StatusCounts {
  open: number;
  in_progress: number;
  waiting_for_customer: number;
  resolved: number;
  closed: number;
}

export interface KpiCounts {
  statusCounts: StatusCounts;
  total: number;
}

export interface TeamsData {
  teams: Team[];
}

const STATUSES: TicketStatus[] = [
  "open",
  "in_progress",
  "waiting_for_customer",
  "resolved",
  "closed",
];

// Fetch the true total for a single status cheaply — page_size=1 so the
// backend returns only one row but the pagination.total reflects the full count.
async function fetchStatusTotal(status: TicketStatus): Promise<number> {
  const res: PaginatedResponse<Ticket> = await ticketsService.list({
    status,
    page: 1,
    page_size: 1,
  });
  return res.pagination.total;
}

// Section 1 — KPI counts only (five status-count requests).
// Throws on any failure; without counts the KPI cards have nothing to show.
export async function fetchKpiCounts(): Promise<KpiCounts> {
  const [open, in_progress, waiting_for_customer, resolved, closed] =
    await Promise.all(STATUSES.map(fetchStatusTotal));

  const statusCounts: StatusCounts = {
    open,
    in_progress,
    waiting_for_customer,
    resolved,
    closed,
  };

  return {
    statusCounts,
    total: STATUSES.reduce((sum, s) => sum + statusCounts[s], 0),
  };
}

// Section 2 — Recent tickets (latest 10).
export async function fetchRecentTickets(): Promise<Ticket[]> {
  const res = await ticketsService.list({ page: 1, page_size: 10 });
  return res.data;
}

// Section 3 — Analytics tickets (latest 100 for priority distribution and trend).
export async function fetchAnalyticsTickets(): Promise<Ticket[]> {
  const res = await ticketsService.list({ page: 1, page_size: 100 });
  return res.data;
}

// Section 4 — Teams (independent; a failure here must not affect ticket sections).
export async function fetchTeamsData(): Promise<TeamsData> {
  const res = await teamsService.list();
  return { teams: res.data };
}
