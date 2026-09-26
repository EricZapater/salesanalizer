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
	`
	_, err := d.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed executing schema migration: %w", err)
	}
	log.Println("Database schema migrations verified")
	return nil
}

