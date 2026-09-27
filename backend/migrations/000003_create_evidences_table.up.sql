-- 000003_create_evidences_table.up.sql

CREATE TABLE IF NOT EXISTS evidences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    raw_content TEXT NOT NULL,
    normalized_url TEXT NOT NULL,
    content_hash VARCHAR(64) NOT NULL,
    source VARCHAR(50) NOT NULL,
    source_id VARCHAR(255),
    author_or_company VARCHAR(255),
    extracted_process VARCHAR(255),
    task_description TEXT,
    frequency VARCHAR(50) DEFAULT 'unknown',
    manuality_score INT DEFAULT 0 CHECK (manuality_score BETWEEN 0 AND 3),
    tools_mentioned TEXT[] DEFAULT '{}',
    sector VARCHAR(100),
    evidence_type VARCHAR(50),
    source_evidence_quote VARCHAR(255),
    evidence_confidence VARCHAR(20) DEFAULT 'mitja',
    is_duplicate_of UUID REFERENCES evidences(id) ON DELETE SET NULL,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_evidences_url ON evidences(normalized_url);
CREATE INDEX IF NOT EXISTS idx_evidences_hash ON evidences(content_hash);
CREATE INDEX IF NOT EXISTS idx_evidences_source ON evidences(source);
CREATE INDEX IF NOT EXISTS idx_evidences_created_at ON evidences(created_at);
CREATE INDEX IF NOT EXISTS idx_evidences_duplicate_of ON evidences(is_duplicate_of);
