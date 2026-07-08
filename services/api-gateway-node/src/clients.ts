import { config } from "./config.js";
import { HttpError } from "./util.js";

/** Proxy a resume/JD pair to the Python match service. */
export async function callMatchService(body: {
  resume_text: string;
  job_description_text: string;
}): Promise<{
  score: number;
  matched_keywords: string[];
  missing_keywords: string[];
}> {
  let resp: globalThis.Response;
  try {
    resp = await fetch(`${config.matchServiceUrl}/score`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  } catch {
    throw new HttpError(502, "match service unreachable");
  }
  if (!resp.ok) {
    throw new HttpError(502, `match service error: ${resp.status}`);
  }
  return (await resp.json()) as never;
}

/** Kick off a scrape run on the Go scraper. */
export async function triggerScrape(): Promise<{ status: string }> {
  let resp: globalThis.Response;
  try {
    resp = await fetch(`${config.scraperUrl}/trigger`, { method: "POST" });
  } catch {
    throw new HttpError(502, "scraper unreachable");
  }
  if (resp.status === 409) return { status: "already_running" };
  if (!resp.ok) throw new HttpError(502, `scraper error: ${resp.status}`);
  return (await resp.json()) as { status: string };
}
