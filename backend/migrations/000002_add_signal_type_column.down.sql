-- 000002_add_signal_type_column.down.sql
ALTER TABLE job_offers DROP COLUMN IF EXISTS signal_type;
