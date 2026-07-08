from fastapi import FastAPI

from .models import HealthResponse, ScoreRequest, ScoreResponse
from .scoring import score_resume

app = FastAPI(title="TrackFolio Match Service", version="1.0.0")


@app.get("/health", response_model=HealthResponse)
def health() -> HealthResponse:
    return HealthResponse(status="ok")


@app.post("/score", response_model=ScoreResponse)
def score(req: ScoreRequest) -> ScoreResponse:
    result = score_resume(req.resume_text, req.job_description_text)
    return ScoreResponse(**result)
