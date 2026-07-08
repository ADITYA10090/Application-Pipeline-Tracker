export const STATUSES = [
  "wishlist",
  "applied",
  "oa",
  "interview",
  "offer",
  "rejected",
] as const;

export type Status = (typeof STATUSES)[number];

export const STATUS_LABELS: Record<Status, string> = {
  wishlist: "Wishlist",
  applied: "Applied",
  oa: "OA",
  interview: "Interview",
  offer: "Offer",
  rejected: "Rejected",
};

export interface Application {
  id: number;
  job_posting_id: number;
  resume_version: string | null;
  applied_date: string | null;
  current_status: Status;
  match_score: number | null;
  created_at: string;
  updated_at: string;
  title: string;
  url: string;
  location: string | null;
  company: string;
  mongo_jd_ref: string | null;
}

export interface StatusEvent {
  status: Status;
  source: "manual" | "email_parsed";
  occurred_at: string;
}

export interface ApplicationDetail extends Application {
  status_events: StatusEvent[];
}

export interface DashboardStats {
  total_applications: number;
  by_status: Record<Status, number>;
  response_rate: number;
  avg_match_score: number;
  applications_over_time: { day: string; count: number }[];
}
