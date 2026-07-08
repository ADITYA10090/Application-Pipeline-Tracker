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
  current_status: Status;
  match_score: number | null;
  resume_version: string | null;
  applied_date: string | null;
  created_at: string;
  updated_at: string;
  title: string;
  url: string;
  location: string | null;
  mongo_jd_ref: string | null;
  company: string;
}

export interface StatusEvent {
  status: string;
  source: string;
  occurred_at: string;
}

export interface ApplicationDetail extends Application {
  status_events: StatusEvent[];
}

export interface Job {
  id: number;
  title: string;
  url: string;
  location: string | null;
  posted_date: string | null;
  company: string;
  has_application: boolean;
}

export interface MatchResult {
  score: number;
  matched_keywords: string[];
  missing_keywords: string[];
}

export interface DashboardStats {
  total_applications: number;
  counts_by_status: Record<Status, number>;
  submitted: number;
  response_rate: number;
  avg_match_score: number | null;
  applications_over_time: { date: string; count: number }[];
}
