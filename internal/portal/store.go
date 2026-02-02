package portal

import (
	"context"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// Store defines the interface for portal data persistence operations.
type Store interface {
	// Customer operations
	GetCustomerByID(ctx context.Context, id uuid.UUID) (*models.Customer, error)
	GetCustomerByEmail(ctx context.Context, email string) (*models.Customer, error)
	CreateCustomer(ctx context.Context, customer *models.Customer) error
	UpdateCustomer(ctx context.Context, customer *models.Customer) error
	UpdateCustomerPassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	UpdateCustomerResetToken(ctx context.Context, id uuid.UUID, token string, expiresAt time.Time) error
	ClearCustomerResetToken(ctx context.Context, id uuid.UUID) error
	IncrementCustomerFailedLogin(ctx context.Context, id uuid.UUID) error
	ResetCustomerFailedLogin(ctx context.Context, id uuid.UUID) error
	LockCustomerAccount(ctx context.Context, id uuid.UUID, until time.Time) error
	UpdateCustomerLastLogin(ctx context.Context, id uuid.UUID, ip string) error

	// Session operations
	CreateSession(ctx context.Context, session *Session) error
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (*Session, error)
	DeleteSession(ctx context.Context, id uuid.UUID) error
	DeleteSessionsByCustomerID(ctx context.Context, customerID uuid.UUID) error
	CleanupExpiredSessions(ctx context.Context) (int64, error)

	// License operations
	GetLicensesByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*models.License, error)
	GetLicenseByID(ctx context.Context, id uuid.UUID) (*models.License, error)
	GetLicenseByKey(ctx context.Context, key string) (*models.License, error)
	CreateLicense(ctx context.Context, license *models.License) error
	UpdateLicense(ctx context.Context, license *models.License) error

	// Invoice operations
	GetInvoicesByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*models.Invoice, error)
	GetInvoiceByID(ctx context.Context, id uuid.UUID) (*models.Invoice, error)
	GetInvoiceByNumber(ctx context.Context, number string) (*models.Invoice, error)
	GetInvoiceItems(ctx context.Context, invoiceID uuid.UUID) ([]*models.InvoiceItem, error)
	CreateInvoice(ctx context.Context, invoice *models.Invoice) error
	CreateInvoiceItem(ctx context.Context, item *models.InvoiceItem) error
	UpdateInvoice(ctx context.Context, invoice *models.Invoice) error

	// Admin operations
	ListCustomers(ctx context.Context, limit, offset int) ([]*models.Customer, int, error)
	ListLicenses(ctx context.Context, limit, offset int) ([]*models.LicenseWithCustomer, int, error)
	ListInvoices(ctx context.Context, limit, offset int) ([]*models.InvoiceWithCustomer, int, error)
	GenerateInvoiceNumber(ctx context.Context) (string, error)
}
