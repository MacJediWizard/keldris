-- Migration: Drop offline_licenses table
-- Air-gap license verification now uses local Ed25519 key validation
-- instead of a separate offline license package upload system.

DROP TABLE IF EXISTS offline_licenses;
