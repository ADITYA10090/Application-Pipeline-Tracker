import { Router } from "express";
import { config } from "../config";
import { matchScoreSchema } from "../validation";

export const matchRouter = Router();

// POST /api/match-score — validate, then proxy to the Python match service.
matchRouter.post("/", async (req, res) => {
  const parsed = matchScoreSchema.safeParse(req.body);
  if (!parsed.success) {
    return res.status(400).json({ error: parsed.error.flatten() });
  }
  try {
    const r = await fetch(`${config.matchServiceUrl}/score`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(parsed.data),
    });
    const body = await r.json();
    res.status(r.status).json(body);
  } catch (err) {
    res
      .status(502)
      .json({ error: "match service unreachable", detail: String(err) });
  }
});
