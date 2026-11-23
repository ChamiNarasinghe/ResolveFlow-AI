"use client";

import { useCallback, useEffect, useState } from "react";
import { useAuthContext } from "../../context/AuthContext";
import {
  fetchKpiCounts,
  fetchRecentTickets,
  fetchAnalyticsTickets,
  fetchTeamsData,
} from "../../services/dashboard";
import type { KpiCounts, TeamsData } from "../../services/dashboard";
import type { Ticket } from "../../types";
import { KpiCard } from "./KpiCard";
import { StatusChart } from "./StatusChart";
import { PriorityChart } from "./PriorityChart";
import { TrendChart } from "./TrendChart";
import { RecentTickets } from "./RecentTickets";
import { TeamsOverview } from "./TeamsOverview";
import { Skeleton } from "../ui/Skeleton";
import { ErrorState } from "../ui/ErrorState";

// ─── Section state ────────────────────────────────────────────────────────────

interface SectionState<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
}

function idle<T>(): SectionState<T> {
  return { data: null, loading: true, error: null };
}

// ─── Generic section fetch hook ───────────────────────────────────────────────
// Each call to useSectionFetch is independent: it owns its own state and its
// own refresh key. Incrementing the key re-runs the fetch for that section only.

function useSectionFetch<T>(
  fetchFn: () => Promise<T>,
  errorMessage: string,
): [SectionState<T>, () => void] {
  const [key, setKey] = useState(0);
  const [state, setState] = useState<SectionState<T>>(idle);

  useEffect(() => {
    let cancelled = false;
    fetchFn()
      .then((result) => {
        if (!cancelled) setState({ data: result, loading: false, error: null });
      })
      .catch((err: unknown) => {
        if (!cancelled)
          setState({
            data: null,
            loading: false,
            error: err instanceof Error ? err.message : errorMessage,
          });
      });
    return () => {
      cancelled = true;
    };
  // fetchFn and errorMessage are module-level stable references — safe to include.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key]);

  const retry = useCallback(() => {
    setState(idle());
    setKey((k) => k + 1);
  }, []);

  return [state, retry];
}

// ─── Layout primitives ────────────────────────────────────────────────────────

function SectionCard({
  title,
  children,
  className = "",
}: {
  title: string;
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={`rounded-[var(--radius-lg)] border border-[var(--border)] bg-[var(--bg-surface)] shadow-[var(--shadow-sm)] ${className}`}
    >
      <div className="border-b border-[var(--border)] px-5 py-4">
        <h2 className="text-sm font-semibold text-[var(--text-primary)]">
          {title}
        </h2>
      </div>
      <div className="p-5">{children}</div>
    </div>
  );
}

function KpiSkeleton() {
  return (
    <div className="rounded-[var(--radius-lg)] border border-[var(--border)] bg-[var(--bg-surface)] p-5 shadow-[var(--shadow-sm)]">
      <div className="flex items-center gap-4">
        <Skeleton className="h-11 w-11 rounded-[var(--radius-md)]" />
        <div className="flex flex-col gap-2">
          <Skeleton className="h-6 w-16" />
          <Skeleton className="h-3 w-24" />
        </div>
      </div>
    </div>
  );
}

function SectionSkeleton({ title }: { title: string }) {
  return (
    <div className="rounded-[var(--radius-lg)] border border-[var(--border)] bg-[var(--bg-surface)] shadow-[var(--shadow-sm)]">
      <div className="border-b border-[var(--border)] px-5 py-4">
        <h2 className="text-sm font-semibold text-[var(--text-primary)]">
          {title}
        </h2>
      </div>
      <div className="flex flex-col gap-3 p-5">
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-5/6" />
        <Skeleton className="h-4 w-4/6" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-3/4" />
      </div>
    </div>
  );
}

function SectionError({
  message,
  onRetry,
}: {
  message: string;
  onRetry?: () => void;
}) {
  return (
    <div
      role="alert"
      className="flex flex-col items-center gap-2 py-6 text-center"
    >
      <p className="text-xs font-medium text-[var(--danger)]">{message}</p>
      {onRetry && (
        <button
          onClick={onRetry}
          className="rounded-[var(--radius-sm)] border border-[var(--danger)] px-3 py-1 text-xs font-medium text-[var(--danger)] transition-colors hover:bg-[var(--danger-bg)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--danger)]"
        >
          Retry
        </button>
      )}
    </div>
  );
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function formatGreeting(name: string): string {
  const hour = new Date().getHours();
  const part =
    hour < 12 ? "Good morning" : hour < 17 ? "Good afternoon" : "Good evening";
  const first = name.trim().split(/\s+/)[0];
  return `${part}, ${first}`;
}

function formatDate(): string {
  return new Date().toLocaleDateString("en-US", {
    weekday: "long",
    month: "long",
    day: "numeric",
    year: "numeric",
  });
}

const TicketIcon = () => (
  <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.75} aria-hidden="true">
    <path strokeLinecap="round" strokeLinejoin="round" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
  </svg>
);

const CircleIcon = ({ className }: { className?: string }) => (
  <svg className={`h-5 w-5 ${className ?? ""}`} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.75} aria-hidden="true">
    <circle cx="12" cy="12" r="9" strokeLinecap="round" />
  </svg>
);

// ─── Dashboard ────────────────────────────────────────────────────────────────

export function DashboardClient() {
  const { user } = useAuthContext();

  // Four independent sections — each has its own state and retry function.
  const [kpi, retryKpi] = useSectionFetch<KpiCounts>(
    fetchKpiCounts,
    "Failed to load ticket counts.",
  );
  const [recent, retryRecent] = useSectionFetch<Ticket[]>(
    fetchRecentTickets,
    "Failed to load recent tickets.",
  );
  const [analytics, retryAnalytics] = useSectionFetch<Ticket[]>(
    fetchAnalyticsTickets,
    "Failed to load analytics data.",
  );
  const [teams, retryTeams] = useSectionFetch<TeamsData>(
    fetchTeamsData,
    "Failed to load teams.",
  );

  // Global Refresh — retries all four sections simultaneously.
  const refresh = useCallback(() => {
    retryKpi();
    retryRecent();
    retryAnalytics();
    retryTeams();
  }, [retryKpi, retryRecent, retryAnalytics, retryTeams]);

  const globalLoading =
    kpi.loading || recent.loading || analytics.loading || teams.loading;

  return (
    <div className="flex flex-col gap-6">
      {/* Welcome */}
      <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-xl font-bold text-[var(--text-primary)]">
            {user ? formatGreeting(user.full_name) : "Dashboard"}
          </h1>
          <p className="mt-0.5 text-sm text-[var(--text-muted)]">
            {formatDate()}
          </p>
        </div>
        <button
          onClick={refresh}
          disabled={globalLoading}
          className="mt-3 inline-flex items-center gap-2 self-start rounded-[var(--radius)] border border-[var(--border)] bg-[var(--bg-surface)] px-3 py-2 text-xs font-medium text-[var(--text-secondary)] shadow-[var(--shadow-sm)] transition-colors hover:bg-[var(--bg-subtle)] disabled:opacity-50 sm:mt-0"
          aria-label="Refresh dashboard"
        >
          <svg
            className={`h-3.5 w-3.5 ${globalLoading ? "animate-spin" : ""}`}
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={2}
            aria-hidden="true"
          >
            <path strokeLinecap="round" strokeLinejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          Refresh
        </button>
      </div>

      {/* KPI section — global error only when counts themselves fail */}
      {kpi.error && !kpi.loading ? (
        <ErrorState message={kpi.error} onRetry={retryKpi} />
      ) : (
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
          {kpi.loading ? (
            Array.from({ length: 4 }).map((_, i) => <KpiSkeleton key={i} />)
          ) : kpi.data ? (
            <>
              <KpiCard
                label="Total Tickets"
                value={kpi.data.total}
                accent
                icon={<TicketIcon />}
              />
              <KpiCard
                label="Open"
                value={kpi.data.statusCounts.open}
                icon={<CircleIcon />}
              />
              <KpiCard
                label="In Progress"
                value={kpi.data.statusCounts.in_progress}
                icon={<CircleIcon className="text-[var(--info)]" />}
              />
              <KpiCard
                label="Resolved"
                value={kpi.data.statusCounts.resolved}
                icon={<CircleIcon className="text-[var(--success)]" />}
              />
            </>
          ) : null}
        </div>
      )}

      {/* Analytics row — status chart uses KPI counts; priority + trend use analytics tickets */}
      <div className="grid gap-4 lg:grid-cols-3">
        {/* Status Distribution — depends on KPI counts */}
        {kpi.loading ? (
          <SectionSkeleton title="Status Distribution" />
        ) : kpi.data ? (
          <SectionCard title="Status Distribution">
            <StatusChart counts={kpi.data.statusCounts} total={kpi.data.total} />
          </SectionCard>
        ) : (
          <SectionCard title="Status Distribution">
            <SectionError message={kpi.error ?? "Unavailable"} onRetry={retryKpi} />
          </SectionCard>
        )}

        {/* Priority Distribution — depends on analytics tickets */}
        {analytics.loading ? (
          <SectionSkeleton title="Priority Distribution" />
        ) : analytics.data ? (
          <SectionCard title="Priority Distribution">
            <PriorityChart tickets={analytics.data} />
          </SectionCard>
        ) : (
          <SectionCard title="Priority Distribution">
            <SectionError message={analytics.error ?? "Unavailable"} onRetry={retryAnalytics} />
          </SectionCard>
        )}

        {/* Recent Ticket Activity — depends on analytics tickets */}
        {analytics.loading ? (
          <SectionSkeleton title="Recent Ticket Activity" />
        ) : analytics.data ? (
          <SectionCard title="Recent Ticket Activity">
            <TrendChart tickets={analytics.data} />
          </SectionCard>
        ) : (
          <SectionCard title="Recent Ticket Activity">
            <SectionError message={analytics.error ?? "Unavailable"} onRetry={retryAnalytics} />
          </SectionCard>
        )}
      </div>

      {/* Recent Tickets — independent from analytics */}
      {recent.loading ? (
        <SectionSkeleton title="Recent Tickets" />
      ) : recent.data ? (
        <SectionCard title="Recent Tickets">
          <RecentTickets tickets={recent.data} />
        </SectionCard>
      ) : (
        <SectionCard title="Recent Tickets">
          <SectionError message={recent.error ?? "Unavailable"} onRetry={retryRecent} />
        </SectionCard>
      )}

      {/* Bottom row: Teams + AI insights */}
      <div className="grid gap-4 lg:grid-cols-2">
        {/* Teams — independent section */}
        {teams.loading ? (
          <SectionSkeleton title="Teams" />
        ) : teams.data ? (
          <SectionCard title="Teams">
            <TeamsOverview teams={teams.data.teams} />
          </SectionCard>
        ) : (
          <SectionCard title="Teams">
            <SectionError message={teams.error ?? "Unavailable"} onRetry={retryTeams} />
          </SectionCard>
        )}

        {/* AI Insights — permanently unavailable */}
        <SectionCard title="AI Insights">
          <div className="flex flex-col items-center justify-center gap-3 py-6 text-center">
            <svg
              className="h-8 w-8 text-[var(--text-muted)]"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={1.5}
              aria-hidden="true"
            >
              <path strokeLinecap="round" strokeLinejoin="round" d="M9.75 3.104v5.714a2.25 2.25 0 01-.659 1.591L5 14.5M9.75 3.104c-.251.023-.501.05-.75.082m.75-.082a24.301 24.301 0 014.5 0m0 0v5.714c0 .597.237 1.17.659 1.591L19.8 15.3M14.25 3.104c.251.023.501.05.75.082M19.8 15.3l-1.57.393A9.065 9.065 0 0112 15a9.065 9.065 0 00-6.23-.693L5 14.5m14.8.8l1.402 1.402c1.232 1.232.65 3.318-1.067 3.611A48.309 48.309 0 0112 21c-2.773 0-5.491-.235-8.135-.687-1.718-.293-2.3-2.379-1.067-3.61L5 14.5" />
            </svg>
            <div className="space-y-1">
              <p className="text-sm font-semibold text-[var(--text-primary)]">
                Aggregate AI insights unavailable
              </p>
              <p className="max-w-xs text-xs text-[var(--text-muted)]">
                The backend provides AI predictions per ticket only. Fetching
                predictions for every ticket would create an N+1 request
                pattern. A dedicated aggregate endpoint is needed.
              </p>
            </div>
          </div>
        </SectionCard>
      </div>
    </div>
  );
}
