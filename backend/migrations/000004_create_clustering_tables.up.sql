-- 000004_create_clustering_tables.up.sql

CREATE TABLE IF NOT EXISTS processes_normalized (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    canonical_name VARCHAR(255) UNIQUE NOT NULL,
    category VARCHAR(100) NOT NULL,
    typical_tools TEXT[] DEFAULT '{}',
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_processes_category ON processes_normalized(category);

CREATE TABLE IF NOT EXISTS pain_clusters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    process_id UUID NOT NULL REFERENCES processes_normalized(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    summary TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'emerging', -- 'emerging', 'consolidated', 'validated', 'discarded'
    evidence_count INT NOT NULL DEFAULT 0,
    company_count INT NOT NULL DEFAULT 0,
    source_count INT NOT NULL DEFAULT 0,
    sector_breadth INT NOT NULL DEFAULT 0,
    last_evidence_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pain_clusters_status ON pain_clusters(status);
CREATE INDEX IF NOT EXISTS idx_pain_clusters_process ON pain_clusters(process_id);
CREATE INDEX IF NOT EXISTS idx_pain_clusters_evidence_count ON pain_clusters(evidence_count DESC);
CREATE INDEX IF NOT EXISTS idx_pain_clusters_updated_at ON pain_clusters(updated_at DESC);

CREATE TABLE IF NOT EXISTS pain_cluster_evidences (
    cluster_id UUID NOT NULL REFERENCES pain_clusters(id) ON DELETE CASCADE,
    evidence_id UUID NOT NULL REFERENCES evidences(id) ON DELETE CASCADE,
    relevance_score FLOAT NOT NULL DEFAULT 1.0,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (cluster_id, evidence_id)
);

CREATE INDEX IF NOT EXISTS idx_pce_evidence ON pain_cluster_evidences(evidence_id);
