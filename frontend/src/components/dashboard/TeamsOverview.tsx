import type { Team } from "../../types";

interface TeamsOverviewProps {
  teams: Team[];
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("en-US", {
    month: "short",
    year: "numeric",
  });
}

export function TeamsOverview({ teams }: TeamsOverviewProps) {
  if (teams.length === 0) {
    return (
      <p className="py-6 text-center text-sm text-[var(--text-muted)]">
        No teams configured yet.
      </p>
    );
  }

  return (
    <div className="grid gap-3 sm:grid-cols-2">
      {teams.map((team) => (
        <div
          key={team.id}
          className="rounded-[var(--radius-md)] border border-[var(--border)] bg-[var(--bg-subtle)] px-4 py-3"
        >
          <p className="text-sm font-semibold text-[var(--text-primary)]">
            {team.name}
          </p>
          {team.description && (
            <p className="mt-0.5 text-xs text-[var(--text-muted)] line-clamp-2">
              {team.description}
            </p>
          )}
          <p className="mt-2 text-xs text-[var(--text-muted)]">
            Since {formatDate(team.created_at)}
          </p>
        </div>
      ))}
    </div>
  );
}
