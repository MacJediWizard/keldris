-- Migration: Add soft delete support for historical records

ALTER TABLE backups ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE dr_tests ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE restores ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE verifications ADD COLUMN deleted_at TIMESTAMPTZ;

CREATE INDEX idx_backups_deleted ON backups(deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_dr_tests_deleted ON dr_tests(deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_restores_deleted ON restores(deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_verifications_deleted ON verifications(deleted_at) WHERE deleted_at IS NULL;
