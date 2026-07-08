"""Resume/JD similarity scoring and keyword-gap analysis.

Scoring approach: TF-IDF vectorization + cosine similarity.

Why TF-IDF + cosine (over embeddings)?
  - Cosine similarity measures the angle between the two documents' term-weight
    vectors, i.e. how much their *important* vocabulary overlaps, normalized for
    length. TF-IDF down-weights terms common to all documents and up-weights
    distinctive ones, so shared rare skills (e.g. "kubernetes") move the score
    more than shared filler words.
  - It is fast, dependency-light, deterministic, and fully explainable in an
    interview - every point of the score traces to shared weighted terms.
  - The tradeoff vs. sentence-transformers embeddings: TF-IDF is purely lexical,
    so "k8s" and "kubernetes" look unrelated. Embeddings capture that semantic
    similarity but add a heavy model dependency and non-determinism. For a
    resume-vs-JD keyword-heavy match, lexical overlap is a defensible choice.
"""

import re

from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn.metrics.pairwise import cosine_similarity

from .skills import SKILLS


def similarity_score(resume_text: str, jd_text: str) -> float:
    """Return cosine similarity of the TF-IDF vectors, in [0, 1]."""
    resume_text = (resume_text or "").strip()
    jd_text = (jd_text or "").strip()
    if not resume_text or not jd_text:
        return 0.0

    # english stop-word removal + sublinear tf keeps common words from
    # dominating; ngram_range picks up two-word skills like "machine learning".
    vectorizer = TfidfVectorizer(
        stop_words="english",
        ngram_range=(1, 2),
        sublinear_tf=True,
    )
    try:
        matrix = vectorizer.fit_transform([resume_text, jd_text])
    except ValueError:
        # Happens when both docs are entirely stop words -> no vocabulary.
        return 0.0
    sim = cosine_similarity(matrix[0:1], matrix[1:2])[0][0]
    return round(float(sim), 4)


def _contains_skill(text: str, skill: str) -> bool:
    """Word-boundary / phrase match for a skill in lowercased text."""
    # re.escape handles special chars like c++, next.js, ci/cd.
    pattern = r"(?<![A-Za-z0-9])" + re.escape(skill) + r"(?![A-Za-z0-9])"
    return re.search(pattern, text) is not None


def keyword_gap(resume_text: str, jd_text: str) -> tuple[list[str], list[str]]:
    """Return (matched, missing) skills.

    matched  = skills present in BOTH resume and JD.
    missing  = skills required by the JD but absent from the resume.
    Only skills that actually appear in the JD are considered, so we never
    report a "gap" the posting doesn't ask for.
    """
    resume_l = (resume_text or "").lower()
    jd_l = (jd_text or "").lower()

    matched: list[str] = []
    missing: list[str] = []
    seen: set[str] = set()
    for skill in SKILLS:
        if skill in seen:
            continue
        seen.add(skill)
        in_jd = _contains_skill(jd_l, skill)
        if not in_jd:
            continue
        if _contains_skill(resume_l, skill):
            matched.append(skill)
        else:
            missing.append(skill)
    return matched, missing


def score(resume_text: str, jd_text: str) -> dict:
    """Full scoring result combining similarity and keyword gap."""
    matched, missing = keyword_gap(resume_text, jd_text)
    return {
        "score": similarity_score(resume_text, jd_text),
        "matched_keywords": matched,
        "missing_keywords": missing,
    }
