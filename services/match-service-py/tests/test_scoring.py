"""Unit tests for the scoring function on fixed input pairs."""

from app.scoring import keyword_gap, score, similarity_score

BACKEND_RESUME = """
Senior backend engineer with 6 years building distributed systems in Go and
Python. Experience with PostgreSQL, Redis, Docker and Kubernetes. Built
REST APIs and microservices deployed on AWS.
"""

BACKEND_JD = """
We are hiring a backend engineer to build microservices in Go. You will work
with PostgreSQL, Redis and Kubernetes, and deploy to AWS. Experience with
distributed systems and REST APIs required. Terraform is a plus.
"""

FRONTEND_JD = """
Frontend engineer wanted. Build UIs in React and TypeScript with Redux and
Tailwind. Experience with Vite and modern CSS required.
"""


def test_similar_docs_score_higher_than_dissimilar():
    same_field = similarity_score(BACKEND_RESUME, BACKEND_JD)
    cross_field = similarity_score(BACKEND_RESUME, FRONTEND_JD)
    assert same_field > cross_field
    assert 0.0 <= cross_field <= same_field <= 1.0


def test_score_is_bounded():
    s = similarity_score(BACKEND_RESUME, BACKEND_JD)
    assert 0.0 <= s <= 1.0


def test_identical_text_scores_near_one():
    assert similarity_score(BACKEND_JD, BACKEND_JD) == 1.0


def test_empty_input_scores_zero():
    assert similarity_score("", BACKEND_JD) == 0.0
    assert similarity_score(BACKEND_RESUME, "") == 0.0


def test_keyword_gap_matched_and_missing():
    matched, missing = keyword_gap(BACKEND_RESUME, BACKEND_JD)
    # Skills in both resume and JD.
    for kw in ("go", "postgresql", "redis", "kubernetes", "aws"):
        assert kw in matched, f"{kw} should be matched"
    # JD requires terraform; resume lacks it.
    assert "terraform" in missing
    # A skill in neither the resume nor the JD is reported in neither list.
    assert "rust" not in matched and "rust" not in missing


def test_keyword_gap_only_considers_jd_skills():
    # Resume mentions Docker, but the frontend JD does not -> not a gap.
    matched, missing = keyword_gap(BACKEND_RESUME, FRONTEND_JD)
    assert "docker" not in missing
    assert "react" in missing  # required by JD, absent from resume


def test_full_score_shape():
    result = score(BACKEND_RESUME, BACKEND_JD)
    assert set(result) == {"score", "matched_keywords", "missing_keywords"}
    assert isinstance(result["score"], float)
    assert isinstance(result["matched_keywords"], list)


def test_special_char_skills_match():
    matched, _ = keyword_gap("I use C++ and CI/CD daily", "C++ and CI/CD required")
    assert "c++" in matched
    assert "ci/cd" in matched
