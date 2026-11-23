import { Skeleton } from "../ui/Skeleton";

export function TicketDetailSkeleton() {
  return (
    <div className="flex flex-col gap-6">
      {/* Header skeleton */}
      <div className="flex flex-col gap-3">
        <Skeleton className="h-4 w-24" />
        <div className="flex flex-wrap items-center gap-3">
          <Skeleton className="h-7 w-12" />
          <Skeleton className="h-7 w-64" />
          <Skeleton className="h-6 w-20" />
          <Skeleton className="h-6 w-16" />
        </div>
      </div>

      {/* Two-column body skeleton */}
      <div className="grid gap-5 lg:grid-cols-3">
        {/* Main column */}
        <div className="flex flex-col gap-5 lg:col-span-2">
          {/* Description card */}
          <div className="rounded-[var(--radius-lg)] border border-[var(--border)] bg-[var(--bg-surface)] p-5 shadow-[var(--shadow-sm)]">
            <Skeleton className="mb-4 h-4 w-28" />
            <Skeleton className="mb-2 h-3 w-full" />
            <Skeleton className="mb-2 h-3 w-full" />
            <Skeleton className="h-3 w-3/4" />
          </div>
          {/* Customer + metadata card */}
          <div className="rounded-[var(--radius-lg)] border border-[var(--border)] bg-[var(--bg-surface)] p-5 shadow-[var(--shadow-sm)]">
            <Skeleton className="mb-4 h-4 w-36" />
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="mb-3 flex justify-between">
                <Skeleton className="h-3 w-24" />
                <Skeleton className="h-3 w-32" />
              </div>
            ))}
          </div>
          {/* History card */}
          <div className="rounded-[var(--radius-lg)] border border-[var(--border)] bg-[var(--bg-surface)] p-5 shadow-[var(--shadow-sm)]">
            <Skeleton className="mb-4 h-4 w-28" />
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="mb-4 flex gap-3">
                <Skeleton className="mt-0.5 h-7 w-7 shrink-0 rounded-full" />
                <div className="flex-1">
                  <Skeleton className="mb-1.5 h-3 w-48" />
                  <Skeleton className="h-3 w-32" />
                </div>
              </div>
            ))}
          </div>
          {/* Comments card */}
          <div className="rounded-[var(--radius-lg)] border border-[var(--border)] bg-[var(--bg-surface)] p-5 shadow-[var(--shadow-sm)]">
            <Skeleton className="mb-4 h-4 w-24" />
            {Array.from({ length: 2 }).map((_, i) => (
              <div key={i} className="mb-3 rounded-[var(--radius)] border border-[var(--border)] p-4">
                <div className="mb-3 flex items-center gap-2">
                  <Skeleton className="h-6 w-6 rounded-full" />
                  <Skeleton className="h-3 w-28" />
                  <Skeleton className="h-3 w-20" />
                </div>
                <Skeleton className="mb-1.5 h-3 w-full" />
                <Skeleton className="h-3 w-2/3" />
              </div>
            ))}
          </div>
          {/* Composer card */}
          <div className="rounded-[var(--radius-lg)] border border-[var(--border)] bg-[var(--bg-surface)] p-5 shadow-[var(--shadow-sm)]">
            <Skeleton className="mb-4 h-4 w-28" />
            <Skeleton className="mb-3 h-24 w-full" />
            <div className="flex justify-end">
              <Skeleton className="h-9 w-32" />
            </div>
          </div>
        </div>

        {/* Side column */}
        <div className="flex flex-col gap-5">
          {/* AI card */}
          <div className="rounded-[var(--radius-lg)] border border-[var(--border)] bg-[var(--bg-surface)] p-5 shadow-[var(--shadow-sm)]">
            <Skeleton className="mb-4 h-4 w-28" />
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="mb-3 flex justify-between">
                <Skeleton className="h-3 w-20" />
                <Skeleton className="h-3 w-24" />
              </div>
            ))}
          </div>
          {/* Assignment card */}
          <div className="rounded-[var(--radius-lg)] border border-[var(--border)] bg-[var(--bg-surface)] p-5 shadow-[var(--shadow-sm)]">
            <Skeleton className="mb-4 h-4 w-24" />
            {Array.from({ length: 2 }).map((_, i) => (
              <div key={i} className="mb-3 flex justify-between">
                <Skeleton className="h-3 w-20" />
                <Skeleton className="h-3 w-28" />
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
