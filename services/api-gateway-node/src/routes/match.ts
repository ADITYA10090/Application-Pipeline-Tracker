import { Router } from "express";
import { matchScoreSchema } from "../schemas.js";
import { callMatchService } from "../clients.js";
import { asyncHandler } from "../util.js";

export const matchRouter = Router();

/** POST /api/match-score — proxy to the Python match service's /score. */
matchRouter.post(
  "/",
  asyncHandler(async (req, res) => {
    const body = matchScoreSchema.parse(req.body);
    const result = await callMatchService(body);
    res.json(result);
  }),
);
