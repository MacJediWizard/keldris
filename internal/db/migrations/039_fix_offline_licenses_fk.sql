-- Migration: Fix offline_licenses uploaded_by foreign key to SET NULL on user deletion

ALTER TABLE offline_licenses ALTER COLUMN uploaded_by DROP NOT NULL;

ALTER TABLE offline_licenses DROP CONSTRAINT IF EXISTS offline_licenses_uploaded_by_fkey;

ALTER TABLE offline_licenses
    ADD CONSTRAINT offline_licenses_uploaded_by_fkey
    FOREIGN KEY (uploaded_by) REFERENCES users(id) ON DELETE SET NULL;
