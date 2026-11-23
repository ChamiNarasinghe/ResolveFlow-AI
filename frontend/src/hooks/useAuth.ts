"use client";

import { useCallback, useEffect, useState } from "react";
import { ApiError, TOKEN_KEY } from "../lib/api-client";
import { authService } from "../services/auth";
import type { LoginRequest, User } from "../types";

interface AuthState {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
}

interface UseAuthReturn extends AuthState {
  login: (credentials: LoginRequest) => Promise<void>;
  logout: () => void;
  isAdmin: boolean;
}

export function useAuth(): UseAuthReturn {
  const [state, setState] = useState<AuthState>({
    user: null,
    isLoading: true,
    isAuthenticated: false,
  });

  // On mount, attempt to restore the session from the stored token.
  // All setState calls are inside async callbacks — never synchronous in the
  // effect body — which satisfies the react-hooks/set-state-in-effect rule.
  useEffect(() => {
    const token = localStorage.getItem(TOKEN_KEY);

    // Resolve the no-token path asynchronously so setState is never called
    // synchronously inside the effect body.
    const hydrate = token
      ? authService.me().then(({ user }) => {
          setState({ user, isLoading: false, isAuthenticated: true });
        })
      : Promise.resolve().then(() => {
          setState({ user: null, isLoading: false, isAuthenticated: false });
        });

    hydrate.catch(() => {
      // Token is expired or invalid — clear it and reset to unauthenticated.
      localStorage.removeItem(TOKEN_KEY);
      setState({ user: null, isLoading: false, isAuthenticated: false });
    });
  }, []);

  const login = useCallback(async (credentials: LoginRequest): Promise<void> => {
    const { user, access_token } = await authService.login(credentials);
    localStorage.setItem(TOKEN_KEY, access_token);
    setState({ user, isLoading: false, isAuthenticated: true });
  }, []);

  const logout = useCallback((): void => {
    localStorage.removeItem(TOKEN_KEY);
    setState({ user: null, isLoading: false, isAuthenticated: false });
  }, []);

  return {
    ...state,
    login,
    logout,
    // Only the admin role has elevated permissions in the current backend.
    // manager has no additional route permissions beyond agent.
    isAdmin: state.user?.role === "admin",
  };
}

// Re-export ApiError so consumers can catch and inspect it without
// importing from the internal api-client module directly.
export { ApiError };
