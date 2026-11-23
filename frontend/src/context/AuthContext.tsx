"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { ApiError, TOKEN_KEY } from "../lib/api-client";
import { authService } from "../services/auth";
import type { LoginRequest, User } from "../types";

interface AuthState {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  isAdmin: boolean;
  login: (credentials: LoginRequest) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    const token = localStorage.getItem(TOKEN_KEY);

    const hydrate = token
      ? authService.me().then(({ user: u }) => {
          if (mountedRef.current) {
            setUser(u);
            setIsLoading(false);
          }
        })
      : Promise.resolve().then(() => {
          if (mountedRef.current) setIsLoading(false);
        });

    hydrate.catch(() => {
      localStorage.removeItem(TOKEN_KEY);
      if (mountedRef.current) {
        setUser(null);
        setIsLoading(false);
      }
    });

    return () => {
      mountedRef.current = false;
    };
  }, []);

  const login = useCallback(async (credentials: LoginRequest) => {
    const { user: u, access_token } = await authService.login(credentials);
    localStorage.setItem(TOKEN_KEY, access_token);
    setUser(u);
    setIsLoading(false);
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem(TOKEN_KEY);
    setUser(null);
  }, []);

  return (
    <AuthContext.Provider
      value={{
        user,
        isLoading,
        isAuthenticated: user !== null,
        isAdmin: user?.role === "admin",
        login,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuthContext(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuthContext must be used within AuthProvider");
  return ctx;
}

export { ApiError };
