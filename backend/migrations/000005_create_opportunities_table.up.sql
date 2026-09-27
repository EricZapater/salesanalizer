-- 000005_create_opportunities_table.up.sql

CREATE TABLE IF NOT EXISTS opportunities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID UNIQUE NOT NULL REFERENCES pain_clusters(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    target_user VARCHAR(255) NOT NULL,
    buyer_persona VARCHAR(255) NOT NULL,
    core_workflow TEXT NOT NULL,
    value_prop TEXT NOT NULL,
    pricing_model VARCHAR(100) NOT NULL DEFAULT '30-50€/mes',
    outreach_hook TEXT NOT NULL,
    scores JSONB NOT NULL,
    global_score NUMERIC(4,2) NOT NULL,
    viability_tier VARCHAR(20) NOT NULL, -- 'excel·lent', 'prometedora', 'feble', 'descartada'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_opportunities_global_score ON opportunities(global_score DESC);
CREATE INDEX IF NOT EXISTS idx_opportunities_cluster ON opportunities(cluster_id);
CREATE INDEX IF NOT EXISTS idx_opportunities_tier ON opportunities(viability_tier);
