package models

import (
	"time"

	"github.com/google/uuid"
)

// OfflineLicense represents an uploaded offline license stored in the database.
type OfflineLicense struct {
	ID          uuid.UUID `json:"id"`
	OrgID       uuid.UUID `json:"org_id"`
	CustomerID  string    `json:"customer_id"`
	Tier        string    `json:"tier"`
	LicenseData []byte    `json:"-"`
	ExpiresAt   time.Time `json:"expires_at"`
	IssuedAt    time.Time `json:"issued_at"`
	UploadedBy  uuid.UUID `json:"uploaded_by"`
	CreatedAt   time.Time `json:"created_at"`
}
