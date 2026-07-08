import { z } from "zod";

export const STATUSES = [
  "wishlist",
  "applied",
  "oa",
  "interview",
  "offer",
  "rejected",
] as const;

export const statusSchema = z.enum(STATUSES);

export const createApplicationSchema = z.object({
  job_posting_id: z.number().int().positive(),
  resume_version: z.string().max(200).optional(),
  applied_date: z.string().optional(), // ISO date
  current_status: statusSchema.default("wishlist"),
  match_score: z.number().min(0).max(1).optional(),
});

export const updateStatusSchema = z.object({
  status: statusSchema,
  source: z.enum(["manual", "email_parsed"]).default("manual"),
});

export const listApplicationsQuerySchema = z.object({
  status: statusSchema.optional(),
  company: z.string().optional(),
});

export const matchScoreSchema = z.object({
  resume_text: z.string().min(1),
  job_description_text: z.string().min(1),
});

export type CreateApplication = z.infer<typeof createApplicationSchema>;
export type UpdateStatus = z.infer<typeof updateStatusSchema>;
