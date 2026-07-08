import type {
  Application,
  ApplicationDetail,
  DashboardStats,
  Job,
  MatchResult,
  Status,
} from "../types";

// Same-origin by default (dev proxy / ingress route /api -> gateway). Override
// with VITE_API_BASE for split deployments.
const BASE = import.meta.env.VITE_API_BASE ?? "";

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const resp = await fetch(`${BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...init,
  });
  if (!resp.ok) {
    let detail = "";
    try {
      detail = JSON.stringify(await resp.json());
    } catch {
      /* ignore */
    }
    throw new Error(`${resp.status} ${resp.statusText} ${detail}`);
  }
  return (await resp.json()) as T;
}

export const api = {
  listApplications: (filters?: { status?: Status; company?: string }) => {
    const p = new URLSearchParams();
    if (filters?.status) p.set("status", filters.status);
    if (filters?.company) p.set("company", filters.company);
    const qs = p.toString();
    return req<Application[]>(`/api/applications${qs ? `?${qs}` : ""}`);
  },

  getApplication: (id: number) =>
    req<ApplicationDetail>(`/api/applications/${id}`),

  createApplication: (body: {
    job_posting_id: number;
    current_status?: Status;
    resume_version?: string;
    match_score?: number;
  }) =>
    req<Application>(`/api/applications`, {
      method: "POST",
      body: JSON.stringify(body),
    }),

  updateStatus: (id: number, status: Status) =>
    req<Application>(`/api/applications/${id}/status`, {
      method: "PATCH",
      body: JSON.stringify({ status }),
    }),

  listJobs: (onlyUnapplied = false) =>
    req<Job[]>(`/api/jobs${onlyUnapplied ? "?applied=false" : ""}`),

  triggerScrape: () =>
    req<{ status: string }>(`/api/jobs/scrape-trigger`, { method: "POST" }),

  matchScore: (resume_text: string, job_description_text: string) =>
    req<MatchResult>(`/api/match-score`, {
      method: "POST",
      body: JSON.stringify({ resume_text, job_description_text }),
    }),

  dashboardStats: () => req<DashboardStats>(`/api/dashboard/stats`),
};
