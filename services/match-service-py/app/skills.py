"""Curated skill vocabulary for keyword-gap analysis.

We use a curated list rather than spaCy noun-phrase extraction because the goal
is *skill* matching, not general phrase extraction. A curated list keeps the
dependency light, the output interpretable, and avoids surfacing noise like
"great opportunity" as a "keyword gap". Multi-word skills are matched as exact
phrases; single tokens are matched on word boundaries (see scoring.py).
"""

SKILLS: list[str] = [
    # languages
    "python", "go", "golang", "javascript", "typescript", "java", "c++", "c#",
    "ruby", "rust", "kotlin", "scala", "php", "swift", "sql", "bash",
    # web / frontend
    "react", "vue", "angular", "next.js", "node.js", "express", "fastapi",
    "django", "flask", "spring", "graphql", "rest", "html", "css", "tailwind",
    "redux", "vite", "webpack",
    # data / ml
    "machine learning", "deep learning", "nlp", "pytorch", "tensorflow",
    "scikit-learn", "pandas", "numpy", "spark", "airflow", "etl",
    "data engineering", "data science", "statistics",
    # databases
    "postgresql", "postgres", "mysql", "mongodb", "redis", "elasticsearch",
    "cassandra", "dynamodb", "kafka", "rabbitmq",
    # infra / devops
    "docker", "kubernetes", "terraform", "ansible", "aws", "gcp", "azure",
    "ci/cd", "jenkins", "github actions", "prometheus", "grafana",
    "microservices", "linux", "nginx", "helm", "istio",
    # practices
    "agile", "scrum", "tdd", "rest api", "distributed systems",
    "system design", "observability", "load testing",
]
