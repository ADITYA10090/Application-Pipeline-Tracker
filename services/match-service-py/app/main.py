from fastapi import FastAPI

from .models import HealthResponse, ScoreRequest, ScoreResponse
from .scoring import score

app = FastAPI(
    title="TrackFolio Match Service",
    description="Scores a resume against a job description (TF-IDF + cosine).",
    version="1.0.0",
)


@app.get("/health", response_model=HealthResponse)
def health() -> HealthResponse:
    return HealthResponse(status="ok")


@app.post("/score", response_model=ScoreResponse)
def score_endpoint(req: ScoreRequest) -> ScoreResponse:
    result = score(req.resume_text, req.job_description_text)
    return ScoreResponse(**result)
