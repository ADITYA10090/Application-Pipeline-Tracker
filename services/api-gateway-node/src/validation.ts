import { z } from "zod";
import { STATUSES } from "./config";

export const createApplicationSchema = z.object({
  job_posting_id: z.number().int().positive(),
  resume_version: z.string().optional(),
  applied_date: z.string().datetime().optional(),
  current_status: z.enum(STATUSES).default("wishlist"),
  match_score: z.number().min(0).max(1).optional(),
});

export const updateStatusSchema = z.object({
  status: z.enum(STATUSES),
  source: z.enum(["manual", "email_parsed"]).default("manual"),
});

export const matchScoreSchema = z.object({
  resume_text: z.string().min(1),
  job_description_text: z.string().min(1),
});

export const listApplicationsQuery = z.object({
  status: z.enum(STATUSES).optional(),
  company: z.string().optional(),
});
