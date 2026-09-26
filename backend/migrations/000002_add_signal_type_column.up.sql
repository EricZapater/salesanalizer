-- 000002_add_signal_type_column.up.sql
ALTER TABLE job_offers ADD COLUMN IF NOT EXISTS signal_type VARCHAR(50) DEFAULT 'oferta_feina';
