package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/portal/portalctx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// PortalStore wraps DB to satisfy portalctx.Store without method name conflicts.
type PortalStore struct {
	*DB
}

// NewPortalStore creates a new PortalStore wrapping the given DB.
func NewPortalStore(db *DB) *PortalStore {
	return &PortalStore{DB: db}
}

// CleanupExpiredSessions delegates to CleanupExpiredPortalSessions to avoid
// conflicting with the main app's CleanupExpiredSessions method signature.
func (ps *PortalStore) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	return ps.DB.CleanupExpiredPortalSessions(ctx)
}

// Compile-time check that *PortalStore satisfies portalctx.Store.
var _ portalctx.Store = (*PortalStore)(nil)

// ---------------------------------------------------------------------------
// Customer operations
// ---------------------------------------------------------------------------

// GetCustomerByID returns a customer by their ID.
func (db *DB) GetCustomerByID(ctx context.Context, id uuid.UUID) (*models.Customer, error) {
	var c models.Customer
	err := db.Pool.QueryRow(ctx, `
		SELECT id, email, name, company, password_hash, status,
		       last_login_at, last_login_ip, failed_login_attempts, locked_until,
		       reset_token, reset_token_expires_at, created_at, updated_at
		FROM customers
		WHERE id = $1
	`, id).Scan(
		&c.ID, &c.Email, &c.Name, &c.Company, &c.PasswordHash, &c.Status,
		&c.LastLoginAt, &c.LastLoginIP, &c.FailedLoginAttempts, &c.LockedUntil,
		&c.ResetToken, &c.ResetTokenExpiresAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("customer not found: %w", err)
		}
		return nil, fmt.Errorf("get customer by id: %w", err)
	}
	return &c, nil
}

// GetCustomerByEmail returns a customer by their email address.
func (db *DB) GetCustomerByEmail(ctx context.Context, email string) (*models.Customer, error) {
	var c models.Customer
	err := db.Pool.QueryRow(ctx, `
		SELECT id, email, name, company, password_hash, status,
		       last_login_at, last_login_ip, failed_login_attempts, locked_until,
		       reset_token, reset_token_expires_at, created_at, updated_at
		FROM customers
		WHERE email = $1
	`, email).Scan(
		&c.ID, &c.Email, &c.Name, &c.Company, &c.PasswordHash, &c.Status,
		&c.LastLoginAt, &c.LastLoginIP, &c.FailedLoginAttempts, &c.LockedUntil,
		&c.ResetToken, &c.ResetTokenExpiresAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("customer not found: %w", err)
		}
		return nil, fmt.Errorf("get customer by email: %w", err)
	}
	return &c, nil
}

// CreateCustomer inserts a new customer into the database.
func (db *DB) CreateCustomer(ctx context.Context, customer *models.Customer) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO customers (id, email, name, company, password_hash, status,
		                       last_login_at, last_login_ip, failed_login_attempts, locked_until,
		                       reset_token, reset_token_expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`,
		customer.ID, customer.Email, customer.Name, customer.Company, customer.PasswordHash, customer.Status,
		customer.LastLoginAt, customer.LastLoginIP, customer.FailedLoginAttempts, customer.LockedUntil,
		customer.ResetToken, customer.ResetTokenExpiresAt, customer.CreatedAt, customer.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create customer: %w", err)
	}
	return nil
}

// UpdateCustomer updates an existing customer's profile fields.
func (db *DB) UpdateCustomer(ctx context.Context, customer *models.Customer) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE customers
		SET email = $2, name = $3, company = $4, status = $5, updated_at = $6
		WHERE id = $1
	`, customer.ID, customer.Email, customer.Name, customer.Company, customer.Status, time.Now())
	if err != nil {
		return fmt.Errorf("update customer: %w", err)
	}
	return nil
}

// UpdateCustomerPassword updates a customer's password hash.
func (db *DB) UpdateCustomerPassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE customers
		SET password_hash = $2, reset_token = NULL, reset_token_expires_at = NULL, updated_at = NOW()
		WHERE id = $1
	`, id, passwordHash)
	if err != nil {
		return fmt.Errorf("update customer password: %w", err)
	}
	return nil
}

// UpdateCustomerResetToken sets a password-reset token and its expiry.
func (db *DB) UpdateCustomerResetToken(ctx context.Context, id uuid.UUID, token string, expiresAt time.Time) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE customers
		SET reset_token = $2, reset_token_expires_at = $3, updated_at = NOW()
		WHERE id = $1
	`, id, token, expiresAt)
	if err != nil {
		return fmt.Errorf("update customer reset token: %w", err)
	}
	return nil
}

// ClearCustomerResetToken removes the password-reset token from a customer.
func (db *DB) ClearCustomerResetToken(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE customers
		SET reset_token = NULL, reset_token_expires_at = NULL, updated_at = NOW()
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("clear customer reset token: %w", err)
	}
	return nil
}

// IncrementCustomerFailedLogin increments the failed login attempt counter.
func (db *DB) IncrementCustomerFailedLogin(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE customers
		SET failed_login_attempts = failed_login_attempts + 1, updated_at = NOW()
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("increment customer failed login: %w", err)
	}
	return nil
}

// ResetCustomerFailedLogin resets the failed login attempt counter to zero.
func (db *DB) ResetCustomerFailedLogin(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE customers
		SET failed_login_attempts = 0, updated_at = NOW()
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("reset customer failed login: %w", err)
	}
	return nil
}

// LockCustomerAccount locks a customer account until the given time.
func (db *DB) LockCustomerAccount(ctx context.Context, id uuid.UUID, until time.Time) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE customers
		SET locked_until = $2, updated_at = NOW()
		WHERE id = $1
	`, id, until)
	if err != nil {
		return fmt.Errorf("lock customer account: %w", err)
	}
	return nil
}

// UpdateCustomerLastLogin updates the last login timestamp and IP address.
func (db *DB) UpdateCustomerLastLogin(ctx context.Context, id uuid.UUID, ip string) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE customers
		SET last_login_at = NOW(), last_login_ip = $2, updated_at = NOW()
		WHERE id = $1
	`, id, ip)
	if err != nil {
		return fmt.Errorf("update customer last login: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Session operations
// ---------------------------------------------------------------------------

// CreateSession inserts a new portal session.
func (db *DB) CreateSession(ctx context.Context, session *portalctx.Session) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO customer_sessions (id, customer_id, token_hash, ip_address, user_agent, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, session.ID, session.CustomerID, session.TokenHash, session.IPAddress, session.UserAgent, session.ExpiresAt, session.CreatedAt)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// GetSessionByTokenHash returns a session matching the given token hash.
func (db *DB) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*portalctx.Session, error) {
	var s portalctx.Session
	err := db.Pool.QueryRow(ctx, `
		SELECT id, customer_id, token_hash, ip_address, user_agent, expires_at, created_at
		FROM customer_sessions
		WHERE token_hash = $1
	`, tokenHash).Scan(
		&s.ID, &s.CustomerID, &s.TokenHash, &s.IPAddress, &s.UserAgent, &s.ExpiresAt, &s.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("session not found: %w", err)
		}
		return nil, fmt.Errorf("get session by token hash: %w", err)
	}
	return &s, nil
}

// DeleteSession removes a session by ID.
func (db *DB) DeleteSession(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM customer_sessions WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// DeleteSessionsByCustomerID removes all sessions for a customer.
func (db *DB) DeleteSessionsByCustomerID(ctx context.Context, customerID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM customer_sessions WHERE customer_id = $1
	`, customerID)
	if err != nil {
		return fmt.Errorf("delete sessions by customer id: %w", err)
	}
	return nil
}

// CleanupExpiredPortalSessions removes all expired portal customer sessions and returns the count deleted.
func (db *DB) CleanupExpiredPortalSessions(ctx context.Context) (int64, error) {
	tag, err := db.Pool.Exec(ctx, `
		DELETE FROM customer_sessions WHERE expires_at < NOW()
	`)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired portal sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}

// ---------------------------------------------------------------------------
// License operations
// ---------------------------------------------------------------------------

// scanPortalLicense scans a single portal license row into a PortalLicense struct.
func scanPortalLicense(row pgx.Row) (*models.PortalLicense, error) {
	var l models.PortalLicense
	var featuresJSON []byte
	err := row.Scan(
		&l.ID, &l.CustomerID, &l.LicenseKey, &l.LicenseType, &l.ProductName, &l.Status,
		&l.MaxAgents, &l.MaxRepos, &l.MaxStorage, &featuresJSON,
		&l.IssuedAt, &l.ExpiresAt, &l.ActivatedAt, &l.LastVerified,
		&l.Notes, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(featuresJSON) > 0 {
		if jsonErr := json.Unmarshal(featuresJSON, &l.Features); jsonErr != nil {
			log.Warn().Err(jsonErr).Str("license_id", l.ID.String()).Msg("failed to unmarshal portal license features")
			l.Features = nil
		}
	}
	return &l, nil
}

// portalLicenseColumns is the column list used by all license queries.
const portalLicenseColumns = `id, customer_id, license_key, license_type, product_name, status,
	max_agents, max_repos, max_storage_gb, features,
	issued_at, expires_at, activated_at, last_verified,
	notes, created_at, updated_at`

// GetLicensesByCustomerID returns all licenses belonging to a customer.
func (db *DB) GetLicensesByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*models.PortalLicense, error) {
	rows, err := db.Pool.Query(ctx, fmt.Sprintf(`
		SELECT %s
		FROM licenses
		WHERE customer_id = $1
		ORDER BY created_at DESC
	`, portalLicenseColumns), customerID)
	if err != nil {
		return nil, fmt.Errorf("get licenses by customer id: %w", err)
	}
	defer rows.Close()

	var licenses []*models.PortalLicense
	for rows.Next() {
		var l models.PortalLicense
		var featuresJSON []byte
		if err := rows.Scan(
			&l.ID, &l.CustomerID, &l.LicenseKey, &l.LicenseType, &l.ProductName, &l.Status,
			&l.MaxAgents, &l.MaxRepos, &l.MaxStorage, &featuresJSON,
			&l.IssuedAt, &l.ExpiresAt, &l.ActivatedAt, &l.LastVerified,
			&l.Notes, &l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan license: %w", err)
		}
		if len(featuresJSON) > 0 {
			if jsonErr := json.Unmarshal(featuresJSON, &l.Features); jsonErr != nil {
				log.Warn().Err(jsonErr).Str("license_id", l.ID.String()).Msg("failed to unmarshal portal license features")
			}
		}
		licenses = append(licenses, &l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate licenses: %w", err)
	}
	return licenses, nil
}

// GetLicenseByID returns a license by its ID.
func (db *DB) GetLicenseByID(ctx context.Context, id uuid.UUID) (*models.PortalLicense, error) {
	row := db.Pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT %s FROM licenses WHERE id = $1
	`, portalLicenseColumns), id)

	l, err := scanPortalLicense(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("license not found: %w", err)
		}
		return nil, fmt.Errorf("get license by id: %w", err)
	}
	return l, nil
}

// GetLicenseByKey returns a license by its license key.
func (db *DB) GetLicenseByKey(ctx context.Context, key string) (*models.PortalLicense, error) {
	row := db.Pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT %s FROM licenses WHERE license_key = $1
	`, portalLicenseColumns), key)

	l, err := scanPortalLicense(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("license not found: %w", err)
		}
		return nil, fmt.Errorf("get license by key: %w", err)
	}
	return l, nil
}

// CreateLicense inserts a new portal license.
func (db *DB) CreateLicense(ctx context.Context, license *models.PortalLicense) error {
	featuresJSON, err := json.Marshal(license.Features)
	if err != nil {
		return fmt.Errorf("marshal license features: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO licenses (id, customer_id, license_key, license_type, product_name, status,
		                      max_agents, max_repos, max_storage_gb, features,
		                      issued_at, expires_at, activated_at, last_verified,
		                      notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`,
		license.ID, license.CustomerID, license.LicenseKey, license.LicenseType, license.ProductName, license.Status,
		license.MaxAgents, license.MaxRepos, license.MaxStorage, featuresJSON,
		license.IssuedAt, license.ExpiresAt, license.ActivatedAt, license.LastVerified,
		license.Notes, license.CreatedAt, license.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create license: %w", err)
	}
	return nil
}

// UpdateLicense updates an existing portal license.
func (db *DB) UpdateLicense(ctx context.Context, license *models.PortalLicense) error {
	featuresJSON, err := json.Marshal(license.Features)
	if err != nil {
		return fmt.Errorf("marshal license features: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE licenses
		SET license_type = $2, product_name = $3, status = $4,
		    max_agents = $5, max_repos = $6, max_storage_gb = $7, features = $8,
		    expires_at = $9, activated_at = $10, last_verified = $11,
		    notes = $12, updated_at = NOW()
		WHERE id = $1
	`,
		license.ID, license.LicenseType, license.ProductName, license.Status,
		license.MaxAgents, license.MaxRepos, license.MaxStorage, featuresJSON,
		license.ExpiresAt, license.ActivatedAt, license.LastVerified,
		license.Notes,
	)
	if err != nil {
		return fmt.Errorf("update license: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Invoice operations
// ---------------------------------------------------------------------------

// invoiceColumns is the column list used by all invoice queries.
const invoiceColumns = `id, customer_id, invoice_number, status, currency,
	subtotal, tax, total, amount_paid, payment_method, payment_ref,
	billing_address, notes, due_date, paid_at, sent_at, created_at, updated_at`

// scanInvoice scans a single invoice row.
func scanInvoice(row pgx.Row) (*models.Invoice, error) {
	var inv models.Invoice
	err := row.Scan(
		&inv.ID, &inv.CustomerID, &inv.InvoiceNumber, &inv.Status, &inv.Currency,
		&inv.Subtotal, &inv.Tax, &inv.Total, &inv.AmountPaid, &inv.PaymentMethod, &inv.PaymentRef,
		&inv.BillingAddress, &inv.Notes, &inv.DueDate, &inv.PaidAt, &inv.SentAt, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// GetInvoicesByCustomerID returns all invoices for a customer.
func (db *DB) GetInvoicesByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*models.Invoice, error) {
	rows, err := db.Pool.Query(ctx, fmt.Sprintf(`
		SELECT %s
		FROM invoices
		WHERE customer_id = $1
		ORDER BY created_at DESC
	`, invoiceColumns), customerID)
	if err != nil {
		return nil, fmt.Errorf("get invoices by customer id: %w", err)
	}
	defer rows.Close()

	var invoices []*models.Invoice
	for rows.Next() {
		var inv models.Invoice
		if err := rows.Scan(
			&inv.ID, &inv.CustomerID, &inv.InvoiceNumber, &inv.Status, &inv.Currency,
			&inv.Subtotal, &inv.Tax, &inv.Total, &inv.AmountPaid, &inv.PaymentMethod, &inv.PaymentRef,
			&inv.BillingAddress, &inv.Notes, &inv.DueDate, &inv.PaidAt, &inv.SentAt, &inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan invoice: %w", err)
		}
		invoices = append(invoices, &inv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate invoices: %w", err)
	}
	return invoices, nil
}

// GetInvoiceByID returns an invoice by its ID.
func (db *DB) GetInvoiceByID(ctx context.Context, id uuid.UUID) (*models.Invoice, error) {
	row := db.Pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT %s FROM invoices WHERE id = $1
	`, invoiceColumns), id)

	inv, err := scanInvoice(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("invoice not found: %w", err)
		}
		return nil, fmt.Errorf("get invoice by id: %w", err)
	}
	return inv, nil
}

// GetInvoiceByNumber returns an invoice by its invoice number.
func (db *DB) GetInvoiceByNumber(ctx context.Context, number string) (*models.Invoice, error) {
	row := db.Pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT %s FROM invoices WHERE invoice_number = $1
	`, invoiceColumns), number)

	inv, err := scanInvoice(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("invoice not found: %w", err)
		}
		return nil, fmt.Errorf("get invoice by number: %w", err)
	}
	return inv, nil
}

// GetInvoiceItems returns all line items for an invoice.
func (db *DB) GetInvoiceItems(ctx context.Context, invoiceID uuid.UUID) ([]*models.InvoiceItem, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, invoice_id, license_id, description, quantity, unit_price, total, created_at
		FROM invoice_items
		WHERE invoice_id = $1
		ORDER BY created_at ASC
	`, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("get invoice items: %w", err)
	}
	defer rows.Close()

	var items []*models.InvoiceItem
	for rows.Next() {
		var item models.InvoiceItem
		if err := rows.Scan(
			&item.ID, &item.InvoiceID, &item.LicenseID, &item.Description,
			&item.Quantity, &item.UnitPrice, &item.Total, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan invoice item: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate invoice items: %w", err)
	}
	return items, nil
}

// CreateInvoice inserts a new invoice.
func (db *DB) CreateInvoice(ctx context.Context, invoice *models.Invoice) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO invoices (id, customer_id, invoice_number, status, currency,
		                      subtotal, tax, total, amount_paid, payment_method, payment_ref,
		                      billing_address, notes, due_date, paid_at, sent_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`,
		invoice.ID, invoice.CustomerID, invoice.InvoiceNumber, invoice.Status, invoice.Currency,
		invoice.Subtotal, invoice.Tax, invoice.Total, invoice.AmountPaid, invoice.PaymentMethod, invoice.PaymentRef,
		invoice.BillingAddress, invoice.Notes, invoice.DueDate, invoice.PaidAt, invoice.SentAt, invoice.CreatedAt, invoice.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create invoice: %w", err)
	}
	return nil
}

// CreateInvoiceItem inserts a new invoice line item.
func (db *DB) CreateInvoiceItem(ctx context.Context, item *models.InvoiceItem) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO invoice_items (id, invoice_id, license_id, description, quantity, unit_price, total, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, item.ID, item.InvoiceID, item.LicenseID, item.Description, item.Quantity, item.UnitPrice, item.Total, item.CreatedAt)
	if err != nil {
		return fmt.Errorf("create invoice item: %w", err)
	}
	return nil
}

// UpdateInvoice updates an existing invoice.
func (db *DB) UpdateInvoice(ctx context.Context, invoice *models.Invoice) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE invoices
		SET status = $2, currency = $3, subtotal = $4, tax = $5, total = $6,
		    amount_paid = $7, payment_method = $8, payment_ref = $9,
		    billing_address = $10, notes = $11, due_date = $12,
		    paid_at = $13, sent_at = $14, updated_at = NOW()
		WHERE id = $1
	`,
		invoice.ID, invoice.Status, invoice.Currency, invoice.Subtotal, invoice.Tax, invoice.Total,
		invoice.AmountPaid, invoice.PaymentMethod, invoice.PaymentRef,
		invoice.BillingAddress, invoice.Notes, invoice.DueDate,
		invoice.PaidAt, invoice.SentAt,
	)
	if err != nil {
		return fmt.Errorf("update invoice: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Admin operations
// ---------------------------------------------------------------------------

// ListCustomers returns a paginated list of customers and the total count.
func (db *DB) ListCustomers(ctx context.Context, limit, offset int) ([]*models.Customer, int, error) {
	var total int
	err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM customers`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count customers: %w", err)
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, email, name, company, password_hash, status,
		       last_login_at, last_login_ip, failed_login_attempts, locked_until,
		       reset_token, reset_token_expires_at, created_at, updated_at
		FROM customers
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list customers: %w", err)
	}
	defer rows.Close()

	var customers []*models.Customer
	for rows.Next() {
		var c models.Customer
		if err := rows.Scan(
			&c.ID, &c.Email, &c.Name, &c.Company, &c.PasswordHash, &c.Status,
			&c.LastLoginAt, &c.LastLoginIP, &c.FailedLoginAttempts, &c.LockedUntil,
			&c.ResetToken, &c.ResetTokenExpiresAt, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan customer: %w", err)
		}
		customers = append(customers, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate customers: %w", err)
	}
	return customers, total, nil
}

// ListLicenses returns a paginated list of licenses with customer details and the total count.
func (db *DB) ListLicenses(ctx context.Context, limit, offset int) ([]*models.PortalLicenseWithCustomer, int, error) {
	var total int
	err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM licenses`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count licenses: %w", err)
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT l.id, l.customer_id, l.license_key, l.license_type, l.product_name, l.status,
		       l.max_agents, l.max_repos, l.max_storage_gb, l.features,
		       l.issued_at, l.expires_at, l.activated_at, l.last_verified,
		       l.notes, l.created_at, l.updated_at,
		       c.email, c.name
		FROM licenses l
		JOIN customers c ON c.id = l.customer_id
		ORDER BY l.created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list licenses: %w", err)
	}
	defer rows.Close()

	var licenses []*models.PortalLicenseWithCustomer
	for rows.Next() {
		var lc models.PortalLicenseWithCustomer
		var featuresJSON []byte
		if err := rows.Scan(
			&lc.ID, &lc.CustomerID, &lc.LicenseKey, &lc.LicenseType, &lc.ProductName, &lc.Status,
			&lc.MaxAgents, &lc.MaxRepos, &lc.MaxStorage, &featuresJSON,
			&lc.IssuedAt, &lc.ExpiresAt, &lc.ActivatedAt, &lc.LastVerified,
			&lc.Notes, &lc.CreatedAt, &lc.UpdatedAt,
			&lc.CustomerEmail, &lc.CustomerName,
		); err != nil {
			return nil, 0, fmt.Errorf("scan license with customer: %w", err)
		}
		if len(featuresJSON) > 0 {
			if jsonErr := json.Unmarshal(featuresJSON, &lc.Features); jsonErr != nil {
				log.Warn().Err(jsonErr).Str("license_id", lc.ID.String()).Msg("failed to unmarshal portal license features")
			}
		}
		licenses = append(licenses, &lc)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate licenses: %w", err)
	}
	return licenses, total, nil
}

// ListInvoices returns a paginated list of invoices with customer details and the total count.
func (db *DB) ListInvoices(ctx context.Context, limit, offset int) ([]*models.InvoiceWithCustomer, int, error) {
	var total int
	err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM invoices`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count invoices: %w", err)
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT i.id, i.customer_id, i.invoice_number, i.status, i.currency,
		       i.subtotal, i.tax, i.total, i.amount_paid, i.payment_method, i.payment_ref,
		       i.billing_address, i.notes, i.due_date, i.paid_at, i.sent_at, i.created_at, i.updated_at,
		       c.email, c.name
		FROM invoices i
		JOIN customers c ON c.id = i.customer_id
		ORDER BY i.created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list invoices: %w", err)
	}
	defer rows.Close()

	var invoices []*models.InvoiceWithCustomer
	for rows.Next() {
		var ic models.InvoiceWithCustomer
		if err := rows.Scan(
			&ic.ID, &ic.CustomerID, &ic.InvoiceNumber, &ic.Status, &ic.Currency,
			&ic.Subtotal, &ic.Tax, &ic.Total, &ic.AmountPaid, &ic.PaymentMethod, &ic.PaymentRef,
			&ic.BillingAddress, &ic.Notes, &ic.DueDate, &ic.PaidAt, &ic.SentAt, &ic.CreatedAt, &ic.UpdatedAt,
			&ic.CustomerEmail, &ic.CustomerName,
		); err != nil {
			return nil, 0, fmt.Errorf("scan invoice with customer: %w", err)
		}
		invoices = append(invoices, &ic)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate invoices: %w", err)
	}
	return invoices, total, nil
}

// GenerateInvoiceNumber generates the next sequential invoice number in format INV-YYYYMM-NNNN.
func (db *DB) GenerateInvoiceNumber(ctx context.Context) (string, error) {
	now := time.Now()
	prefix := fmt.Sprintf("INV-%d%02d-", now.Year(), now.Month())

	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM invoices WHERE invoice_number LIKE $1
	`, prefix+"%").Scan(&count)
	if err != nil {
		return "", fmt.Errorf("count invoices for number generation: %w", err)
	}

	number := fmt.Sprintf("%s%04d", prefix, count+1)
	return number, nil
}
