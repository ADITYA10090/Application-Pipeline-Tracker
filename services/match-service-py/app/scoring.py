"""Resume/JD scoring.

Approach: TF-IDF vectorization + cosine similarity for the overall match score,
and a curated skill-keyword dictionary for the matched/missing keyword lists.

Why TF-IDF and not embeddings? It is fast, has no model download, is fully
deterministic, and is trivial to explain in an interview: each document becomes
a sparse vector weighted by term frequency times inverse document frequency, and
cosine similarity measures the angle between the two vectors (1.0 = identical
direction, 0.0 = no shared terms). The trade-off is that it matches on surface
tokens, not meaning ("JS" vs "JavaScript" look unrelated) — sentence-transformer
embeddings would capture that at the cost of a heavy dependency and slower cold
start. For this project the keyword layer below compensates for the common
synonym cases we care about.
"""

from __future__ import annotations

import re
from typing import Iterable

from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn.metrics.pairwise import cosine_similarity

# Curated skill vocabulary. Each canonical skill maps to the surface forms that
# should count as a mention of it. This is deliberately hand-maintained rather
# than auto-extracted so the matched/missing lists stay meaningful.
SKILL_ALIASES: dict[str, list[str]] = {
    "python": ["python"],
    "go": ["golang", "go"],
    "javascript": ["javascript", "js", "node", "nodejs", "node.js"],
    "typescript": ["typescript", "ts"],
    "react": ["react", "reactjs", "react.js"],
    "fastapi": ["fastapi"],
    "express": ["express", "expressjs"],
    "postgresql": ["postgresql", "postgres", "psql"],
    "mongodb": ["mongodb", "mongo"],
    "redis": ["redis"],
    "docker": ["docker"],
    "kubernetes": ["kubernetes", "k8s"],
    "aws": ["aws", "amazon web services"],
    "gcp": ["gcp", "google cloud"],
    "terraform": ["terraform"],
    "graphql": ["graphql"],
    "rest": ["rest api", "restful", "rest"],
    "grpc": ["grpc"],
    "kafka": ["kafka"],
    "sql": ["sql"],
    "ci/cd": ["ci/cd", "cicd", "continuous integration", "continuous delivery"],
    "microservices": ["microservice", "microservices"],
    "machine learning": ["machine learning", "scikit", "sklearn"],
    "linux": ["linux"],
    "git": ["git", "github", "gitlab"],
}

# Precompile one word-boundary regex per alias. Word boundaries avoid matching
# "go" inside "google" or "sql" inside "postgresql" while still catching "Go,"
# and "Go." at token edges — far more robust than substring checks.
_ALIAS_PATTERNS: dict[str, list[re.Pattern]] = {
    canonical: [
        re.compile(r"(?<![a-z0-9])" + re.escape(a) + r"(?![a-z0-9])")
        for a in aliases
    ]
    for canonical, aliases in SKILL_ALIASES.items()
}


def extract_skills(text: str) -> set[str]:
    """Return the set of canonical skills mentioned anywhere in `text`."""
    norm = text.lower()
    found: set[str] = set()
    for canonical, patterns in _ALIAS_PATTERNS.items():
        if any(p.search(norm) for p in patterns):
            found.add(canonical)
    return found


def cosine_score(resume_text: str, job_description_text: str) -> float:
    """TF-IDF cosine similarity in [0, 1], rounded to 4 dp."""
    docs = [resume_text or "", job_description_text or ""]
    if not resume_text.strip() or not job_description_text.strip():
        return 0.0
    vectorizer = TfidfVectorizer(stop_words="english", ngram_range=(1, 2))
    matrix = vectorizer.fit_transform(docs)
    sim = cosine_similarity(matrix[0:1], matrix[1:2])[0][0]
    return round(float(sim), 4)


def score_resume(resume_text: str, job_description_text: str) -> dict:
    """Full analysis: overall score plus matched/missing skill keywords.

    matched  = skills the JD asks for that the resume has
    missing  = skills the JD asks for that the resume lacks (the gap to close)
    """
    score = cosine_score(resume_text, job_description_text)
    resume_skills = extract_skills(resume_text)
    jd_skills = extract_skills(job_description_text)

    matched = sorted(jd_skills & resume_skills)
    missing = sorted(jd_skills - resume_skills)

    return {
        "score": score,
        "matched_keywords": matched,
        "missing_keywords": missing,
    }
