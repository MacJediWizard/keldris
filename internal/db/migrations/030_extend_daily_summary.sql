-- 030_extend_daily_summary.sql
-- Migration: Add total_duration_secs and agents_active to metrics_daily_summary

ALTER TABLE metrics_daily_summary ADD COLUMN total_duration_secs BIGINT NOT NULL DEFAULT 0;
ALTER TABLE metrics_daily_summary ADD COLUMN agents_active INTEGER NOT NULL DEFAULT 0;
