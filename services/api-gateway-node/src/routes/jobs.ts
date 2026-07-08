import { Router } from "express";
import { query } from "../db.js";
import { triggerScrape } from "../clients.js";
import { asyncHandler } from "../util.js";

export const jobsRouter = Router();

/**
 * GET /api/jobs?applied=false
 * Lists scraped postings with company name and whether an application exists,
 * so the frontend can offer un-applied postings to add to the pipeline.
 */
jobsRouter.get(
  "/",
  asyncHandler(async (req, res) => {
    const onlyUnapplied = req.query.applied === "false";
    const { rows } = await query(
      `SELECT j.id, j.title, j.url, j.location, j.posted_date, j.mongo_jd_ref,
              c.name AS company,
              (a.id IS NOT NULL) AS has_application
         FROM job_postings j
         JOIN companies c ON c.id = j.company_id
         LEFT JOIN applications a ON a.job_posting_id = j.id
         ${onlyUnapplied ? "WHERE a.id IS NULL" : ""}
         ORDER BY j.scraped_at DESC
         LIMIT 200`,
    );
    res.json(rows);
  }),
);

/** POST /api/jobs/scrape-trigger — proxy to the Go scraper's /trigger. */
jobsRouter.post(
  "/scrape-trigger",
  asyncHandler(async (_req, res) => {
    const result = await triggerScrape();
    res.status(202).json(result);
  }),
);
