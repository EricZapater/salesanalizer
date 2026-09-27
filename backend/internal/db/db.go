package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func Connect(databaseURL string) (*DB, error) {
	conn, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("error opening db: %w", err)
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(10)
	conn.SetConnMaxLifetime(5 * time.Minute)

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("error pinging db: %w", err)
	}

	log.Println("Database connection established successfully")
	return &DB{conn}, nil
}

func (d *DB) RunAutoMigrations() error {
	schema := `
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

	CREATE TABLE IF NOT EXISTS signal_settings (
		id INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
		deep_fetch_enabled BOOLEAN NOT NULL DEFAULT FALSE,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	INSERT INTO signal_settings (id, deep_fetch_enabled, updated_at)
	VALUES (1, FALSE, NOW())
	ON CONFLICT (id) DO NOTHING;

	-- Afegir columna signal_type si la taula ja existia prèviament
	ALTER TABLE job_offers ADD COLUMN IF NOT EXISTS signal_type VARCHAR(50) DEFAULT 'oferta_feina';

	-- Neteja d'enllaços no funcionals de proves inicials
	UPDATE job_offers SET url = 'https://feinaactiva.gencat.cat' WHERE url LIKE '%feinaactiva.gencat.cat/oferta/%' AND source = 'soc';
	UPDATE job_offers SET url = 'https://www.infofeina.com/ofertes-feina' WHERE url LIKE '%infofeina.com/oferta/%' AND source = 'infofeina';

	-- Taules V2 Radar de Processos de Negoci & Micro-SaaS
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
		status VARCHAR(50) NOT NULL DEFAULT 'emerging',
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
		viability_tier VARCHAR(20) NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_opportunities_global_score ON opportunities(global_score DESC);
	CREATE INDEX IF NOT EXISTS idx_opportunities_cluster ON opportunities(cluster_id);
	CREATE INDEX IF NOT EXISTS idx_opportunities_tier ON opportunities(viability_tier);
	`
	_, err := d.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed executing schema migration: %w", err)
	}
	log.Println("Database schema migrations verified")
	return nil
}

