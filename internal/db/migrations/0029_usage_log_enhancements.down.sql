DROP INDEX IF EXISTS idx_usage_events_deployment_time;
DROP INDEX IF EXISTS idx_usage_events_user_time;
DROP INDEX IF EXISTS idx_usage_events_time_id;

ALTER TABLE usage_hourly DROP COLUMN revenue_sum;
ALTER TABLE usage_hourly DROP COLUMN cached_tokens;

ALTER TABLE usage_events DROP COLUMN meta;
ALTER TABLE usage_events DROP COLUMN cache_write_tokens;
ALTER TABLE usage_events DROP COLUMN is_stream;
ALTER TABLE usage_events DROP COLUMN log_type;