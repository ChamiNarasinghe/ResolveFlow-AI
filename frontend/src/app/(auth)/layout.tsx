import { AuthGuard } from "../../components/layout/AuthGuard";

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <AuthGuard redirectIfAuthenticated>
      <div className="flex min-h-screen items-center justify-center bg-[var(--bg-base)] px-4">
        <div className="w-full max-w-sm">{children}</div>
      </div>
    </AuthGuard>
  );
}
