"use client";

import { usePathname } from "next/navigation";
import { useAuthContext } from "../../context/AuthContext";
import { Avatar } from "../ui/Avatar";
import { Dropdown } from "../ui/Dropdown";

const pageTitles: Record<string, string> = {
  "/dashboard": "Overview",
  "/tickets": "Tickets",
  "/teams": "Teams",
  "/users": "Users",
  "/settings": "Settings",
};

function getTitle(pathname: string): string {
  for (const [prefix, title] of Object.entries(pageTitles)) {
    if (pathname === prefix || pathname.startsWith(prefix + "/")) return title;
  }
  return "ResolveFlow AI";
}

interface TopBarProps {
  onMobileMenuOpen: () => void;
}

export function TopBar({ onMobileMenuOpen }: TopBarProps) {
  const pathname = usePathname();
  const { user, logout } = useAuthContext();

  const dropdownItems = [
    {
      label: "Sign out",
      onClick: logout,
      danger: true,
      icon: (
        <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-hidden="true">
          <path strokeLinecap="round" strokeLinejoin="round" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
        </svg>
      ),
    },
  ];

  return (
    <header className="sticky top-0 z-30 flex h-16 shrink-0 items-center gap-4 border-b border-[var(--border)] bg-[var(--bg-surface)] px-4 shadow-[var(--shadow-sm)] sm:px-6">
      {/* Mobile menu button */}
      <button
        type="button"
        onClick={onMobileMenuOpen}
        className="rounded-lg p-2 text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-subtle)] hover:text-[var(--text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-ring)] lg:hidden"
        aria-label="Open navigation menu"
      >
        <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-hidden="true">
          <path strokeLinecap="round" strokeLinejoin="round" d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </button>

      {/* Page title */}
      <h1 className="flex-1 text-base font-semibold text-[var(--text-primary)]">
        {getTitle(pathname)}
      </h1>

      {/* User menu */}
      {user && (
        <Dropdown
          trigger={
            <button
              className="flex items-center gap-2 rounded-lg px-2 py-1.5 transition-colors hover:bg-[var(--bg-subtle)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-ring)]"
              aria-label="User menu"
            >
              <Avatar name={user.full_name} size="sm" />
              <span className="hidden text-sm font-medium text-[var(--text-primary)] sm:block">
                {user.full_name}
              </span>
              <svg className="h-4 w-4 text-[var(--text-muted)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-hidden="true">
                <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
              </svg>
            </button>
          }
          items={dropdownItems}
          align="right"
        />
      )}
    </header>
  );
}
