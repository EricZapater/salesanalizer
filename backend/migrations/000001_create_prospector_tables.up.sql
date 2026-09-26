-- 000001_create_prospector_tables.up.sql

CREATE TABLE IF NOT EXISTS job_offers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source VARCHAR(50) NOT NULL,
    external_id VARCHAR(255),
    title VARCHAR(500) NOT NULL,
    company VARCHAR(255),
    location VARCHAR(255),
    url TEXT UNIQUE NOT NULL,
    raw_text TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'analyzed',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_job_offers_source ON job_offers(source);
CREATE INDEX IF NOT EXISTS idx_job_offers_status ON job_offers(status);
CREATE INDEX IF NOT EXISTS idx_job_offers_created_at ON job_offers(created_at);

CREATE TABLE IF NOT EXISTS opportunity_analyses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_offer_id UUID UNIQUE NOT NULL REFERENCES job_offers(id) ON DELETE CASCADE,
    ineficiencia_manual TEXT NOT NULL,
    proposta_micro_saas TEXT NOT NULL,
    viabilitat_plg_score INT NOT NULL CHECK (viabilitat_plg_score BETWEEN 1 AND 5),
    decisor_compra VARCHAR(255) NOT NULL,
    ganxo_venda TEXT NOT NULL,
    raw_llm_response JSONB,
    analyzed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_opportunity_plg_score ON opportunity_analyses(viabilitat_plg_score DESC);

CREATE TABLE IF NOT EXISTS scraper_runs (
    id SERIAL PRIMARY KEY,
    source VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL,
    items_found INT NOT NULL DEFAULT 0,
    error_message TEXT,
    run_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
