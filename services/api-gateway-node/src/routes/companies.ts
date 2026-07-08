import { Router } from "express";
import { query } from "../db";

export const companiesRouter = Router();

// GET /api/companies — companies with their open-posting counts.
companiesRouter.get("/", async (_req, res) => {
  const rows = await query(
    `SELECT c.id, c.name, c.website, c.career_page_url, c.industry, c.created_at,
            COUNT(jp.id)::int AS posting_count
       FROM companies c
       LEFT JOIN job_postings jp ON jp.company_id = c.id
       GROUP BY c.id
       ORDER BY c.name ASC`
  );
  res.json(rows);
});
