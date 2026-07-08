from pydantic import BaseModel, Field


class ScoreRequest(BaseModel):
    resume_text: str = Field(..., min_length=1, description="Full resume text")
    job_description_text: str = Field(
        ..., min_length=1, description="Full job description text"
    )


class ScoreResponse(BaseModel):
    score: float = Field(..., ge=0.0, le=1.0, description="Cosine similarity 0..1")
    matched_keywords: list[str]
    missing_keywords: list[str]


class HealthResponse(BaseModel):
    status: str
