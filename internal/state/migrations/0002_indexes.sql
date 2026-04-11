CREATE INDEX IF NOT EXISTS idx_services_status ON services(status);
CREATE INDEX IF NOT EXISTS idx_jobs_last_status ON jobs(last_status);
CREATE INDEX IF NOT EXISTS idx_triggers_last_status ON triggers(last_status);