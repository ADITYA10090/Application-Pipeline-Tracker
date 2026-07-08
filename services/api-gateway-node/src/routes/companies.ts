import { Router } from "express";
import { query } from "../db.js";
import { asyncHandler } from "../util.js";

export const companiesRouter = Router();

/** GET /api/companies — companies with a count of their scraped postings. */
companiesRouter.get(
  "/",
  asyncHandler(async (_req, res) => {
    const { rows } = await query(
      `SELECT c.id, c.name, c.website, c.career_page_url, c.industry,
              COUNT(j.id) AS posting_count
         FROM companies c
         LEFT JOIN job_postings j ON j.company_id = c.id
         GROUP BY c.id
         ORDER BY c.name`,
    );
    res.json(rows);
  }),
);
