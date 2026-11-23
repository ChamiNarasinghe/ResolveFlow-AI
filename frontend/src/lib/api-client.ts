import { API_BASE_URL } from "./api-config";

// ─── Token storage key ────────────────────────────────────────────────────────
export const TOKEN_KEY = "rf_access_token";

// ─── API error ────────────────────────────────────────────────────────────────

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

// ─── Internal fetch wrapper ───────────────────────────────────────────────────

type RequestOptions = Omit<RequestInit, "body"> & {
  body?: unknown;
};

async function request<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const { body, headers: extraHeaders, ...rest } = options;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(extraHeaders as Record<string, string>),
  };

  // Attach the stored JWT if present.
  if (typeof window !== "undefined") {
    const token = localStorage.getItem(TOKEN_KEY);
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...rest,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (!response.ok) {
    // The Go backend returns plain-text error messages from most handlers.
    // Some handlers return a JSON string such as `{"error":"ticket not found"}`.
    // Read as text first, then attempt to extract the inner message if it is JSON.
    const text = await response.text().catch(() => response.statusText);
    const trimmed = text.trim();

    let message = trimmed || response.statusText;
    if (trimmed.startsWith("{")) {
      try {
        const parsed = JSON.parse(trimmed) as Record<string, unknown>;
        if (typeof parsed["error"] === "string" && parsed["error"]) {
          message = parsed["error"];
        }
      } catch {
        // Not valid JSON — use the raw text as-is.
      }
    }

    throw new ApiError(response.status, message);
  }

  // 204 No Content — return undefined cast to T.
  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
}

// ─── Public HTTP helpers ──────────────────────────────────────────────────────

export const apiClient = {
  get<T>(path: string): Promise<T> {
    return request<T>(path, { method: "GET" });
  },

  post<T>(path: string, body?: unknown): Promise<T> {
    return request<T>(path, { method: "POST", body });
  },

  patch<T>(path: string, body?: unknown): Promise<T> {
    return request<T>(path, { method: "PATCH", body });
  },

  delete<T = void>(path: string): Promise<T> {
    return request<T>(path, { method: "DELETE" });
  },
};
