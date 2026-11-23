// Base URL for the Go REST API backend.
// Set NEXT_PUBLIC_API_URL in .env.local for local development or deployment.
// Falls back to the default Docker Compose backend port.
export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
