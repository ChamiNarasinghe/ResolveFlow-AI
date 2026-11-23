"use client";

import { useRouter } from "next/navigation";
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
  const { isLoading, isAuthenticated } = useAuthContext();
  const router = useRouter();

  useEffect(() => {
    if (isLoading) return;
    if (redirectIfAuthenticated && isAuthenticated) {
      router.replace("/dashboard");
    } else if (!redirectIfAuthenticated && !isAuthenticated) {
      router.replace("/login");
    }
  }, [isLoading, isAuthenticated, redirectIfAuthenticated, router]);

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center bg-[var(--bg-base)]">
        <Spinner size="lg" className="text-[var(--accent)]" />
      </div>
    );
  }

  // While the redirect is in flight, render nothing to avoid a flash.
  if (redirectIfAuthenticated && isAuthenticated) return null;
  if (!redirectIfAuthenticated && !isAuthenticated) return null;

  return <>{children}</>;
}
