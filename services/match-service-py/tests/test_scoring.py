from app.scoring import cosine_score, extract_skills, score_resume


def test_identical_text_scores_near_one():
    text = "Senior Python engineer with Django and PostgreSQL experience"
    assert cosine_score(text, text) == 1.0


def test_unrelated_text_scores_low():
    resume = "Pastry chef specializing in French desserts and bread baking"
    jd = "Kubernetes platform engineer with Go and Terraform"
    assert cosine_score(resume, jd) < 0.1


def test_partial_overlap_is_between():
    resume = "Backend engineer with Python, PostgreSQL, and Redis"
    jd = "Backend engineer needing Python, Redis, and Kafka experience"
    s = cosine_score(resume, jd)
    assert 0.1 < s < 1.0


def test_empty_input_scores_zero():
    assert cosine_score("", "anything") == 0.0
    assert cosine_score("anything", "   ") == 0.0


def test_skill_extraction_aliases():
    skills = extract_skills("Experienced in JS, k8s, and Postgres")
    assert "javascript" in skills
    assert "kubernetes" in skills
    assert "postgresql" in skills


def test_matched_and_missing_keywords():
    resume = "I know Python, React, and Docker."
    jd = "Looking for Python, Kubernetes, and Go skills."
    result = score_resume(resume, jd)
    assert "python" in result["matched_keywords"]
    assert "kubernetes" in result["missing_keywords"]
    assert "go" in result["missing_keywords"]
    # React is on the resume but not asked for -> neither list.
    assert "react" not in result["matched_keywords"]
    assert "react" not in result["missing_keywords"]


def test_response_shape():
    result = score_resume("Python developer", "Python role")
    assert set(result.keys()) == {"score", "matched_keywords", "missing_keywords"}
    assert 0.0 <= result["score"] <= 1.0
