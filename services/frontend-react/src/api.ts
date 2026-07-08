import type {
  Application,
  ApplicationDetail,
  DashboardStats,
  Status,
} from "./types";

// All requests are relative to /api, served by the gateway (dev: Vite proxy,
// prod: k8s ingress).
const BASE = import.meta.env.VITE_API_URL ?? "/api";

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...init,
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`${res.status} ${res.statusText}: ${body}`);
  }
  return res.json() as Promise<T>;
}

export const api = {
  listApplications: (filters?: { status?: Status; company?: string }) => {
    const q = new URLSearchParams();
    if (filters?.status) q.set("status", filters.status);
    if (filters?.company) q.set("company", filters.company);
    const qs = q.toString();
    return req<Application[]>(`/applications${qs ? `?${qs}` : ""}`);
  },

  getApplication: (id: number) =>
    req<ApplicationDetail>(`/applications/${id}`),

  updateStatus: (id: number, status: Status) =>
    req<Application>(`/applications/${id}/status`, {
      method: "PATCH",
      body: JSON.stringify({ status }),
    }),

  getStats: () => req<DashboardStats>("/dashboard/stats"),

  triggerScrape: () =>
    req<{ status: string }>("/jobs/scrape-trigger", { method: "POST" }),
};
