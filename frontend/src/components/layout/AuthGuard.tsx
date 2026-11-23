"use client";

import { usePathname, useRouter } from "next/navigation";
import { useEffect } from "react";
import { useAuthContext } from "../../context/AuthContext";
import { Spinner } from "../ui/Spinner";

interface AuthGuardProps {
  children: React.ReactNode;
  /** When true, redirects authenticated users away (used on the login page). */
  redirectIfAuthenticated?: boolean;
}

export function AuthGuard({
  children,
  redirectIfAuthenticated = false,
}: AuthGuardProps) {
  const { isLoading, isAuthenticated, mustChangePassword } = useAuthContext();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (isLoading) return;

    if (redirectIfAuthenticated) {
      // Auth layout: redirect authenticated users to dashboard, but only if
      // they don't need to change their password. A forced-change user must
      // stay on /change-password — don't redirect them away.
      if (isAuthenticated && !mustChangePassword && pathname !== "/change-password") {
        router.replace("/dashboard");
      }
    } else {
      // App layout: unauthenticated users go to login.
      if (!isAuthenticated) {
        router.replace("/login");
      } else if (mustChangePassword) {
        // Authenticated but forced to change password — block app routes.
        router.replace("/change-password");
      }
    }
  }, [isLoading, isAuthenticated, mustChangePassword, redirectIfAuthenticated, pathname, router]);

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center bg-[var(--bg-base)]">
        <Spinner size="lg" className="text-[var(--accent)]" />
      </div>
    );
  }

  if (redirectIfAuthenticated) {
    // Allow access to /change-password even when authenticated (forced-change flow).
    if (isAuthenticated && !mustChangePassword && pathname !== "/change-password") return null;
  } else {
    if (!isAuthenticated) return null;
    if (mustChangePassword) return null;
  }

  return <>{children}</>;
}
