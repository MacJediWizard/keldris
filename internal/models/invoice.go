package models

import (
	"time"

	"github.com/google/uuid"
)

// InvoiceStatus defines the status of an invoice.
type InvoiceStatus string

const (
	// InvoiceStatusDraft is an invoice that has not been sent.
	InvoiceStatusDraft InvoiceStatus = "draft"
	// InvoiceStatusSent is an invoice that has been sent to the customer.
	InvoiceStatusSent InvoiceStatus = "sent"
	// InvoiceStatusPaid is a paid invoice.
	InvoiceStatusPaid InvoiceStatus = "paid"
	// InvoiceStatusOverdue is an overdue invoice.
	InvoiceStatusOverdue InvoiceStatus = "overdue"
	// InvoiceStatusCancelled is a cancelled invoice.
	InvoiceStatusCancelled InvoiceStatus = "cancelled"
	// InvoiceStatusRefunded is a refunded invoice.
	InvoiceStatusRefunded InvoiceStatus = "refunded"
)

// PaymentMethod defines the payment method used.
type PaymentMethod string

const (
	// PaymentMethodCard is a credit/debit card payment.
	PaymentMethodCard PaymentMethod = "card"
	// PaymentMethodBankTransfer is a bank transfer payment.
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
	// PaymentMethodPayPal is a PayPal payment.
	PaymentMethodPayPal PaymentMethod = "paypal"
	// PaymentMethodInvoice is payment via invoice (net terms).
	PaymentMethodInvoice PaymentMethod = "invoice"
)

// Invoice represents an invoice for license purchases.
type Invoice struct {
	ID              uuid.UUID     `json:"id"`
	CustomerID      uuid.UUID     `json:"customer_id"`
	InvoiceNumber   string        `json:"invoice_number"`
	Status          InvoiceStatus `json:"status"`
	Currency        string        `json:"currency"`
	Subtotal        int64         `json:"subtotal"`         // Amount in cents
	Tax             int64         `json:"tax"`              // Tax amount in cents
	Total           int64         `json:"total"`            // Total amount in cents
	AmountPaid      int64         `json:"amount_paid"`      // Amount paid in cents
	PaymentMethod   PaymentMethod `json:"payment_method,omitempty"`
	PaymentRef      string        `json:"payment_ref,omitempty"` // External payment reference
	BillingAddress  string        `json:"billing_address,omitempty"`
	Notes           string        `json:"notes,omitempty"`
	DueDate         *time.Time    `json:"due_date,omitempty"`
	PaidAt          *time.Time    `json:"paid_at,omitempty"`
	SentAt          *time.Time    `json:"sent_at,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

// InvoiceItem represents a line item on an invoice.
type InvoiceItem struct {
	ID          uuid.UUID  `json:"id"`
	InvoiceID   uuid.UUID  `json:"invoice_id"`
	LicenseID   *uuid.UUID `json:"license_id,omitempty"`
	Description string     `json:"description"`
	Quantity    int        `json:"quantity"`
	UnitPrice   int64      `json:"unit_price"` // Price per unit in cents
	Total       int64      `json:"total"`      // Total for this line in cents
	CreatedAt   time.Time  `json:"created_at"`
}

// NewInvoice creates a new Invoice with the given details.
func NewInvoice(customerID uuid.UUID, invoiceNumber, currency string) *Invoice {
	now := time.Now()
	return &Invoice{
		ID:            uuid.New(),
		CustomerID:   customerID,
		InvoiceNumber: invoiceNumber,
		Status:        InvoiceStatusDraft,
		Currency:      currency,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// NewInvoiceItem creates a new InvoiceItem.
func NewInvoiceItem(invoiceID uuid.UUID, description string, quantity int, unitPrice int64) *InvoiceItem {
	return &InvoiceItem{
		ID:          uuid.New(),
		InvoiceID:   invoiceID,
		Description: description,
		Quantity:    quantity,
		UnitPrice:   unitPrice,
		Total:       int64(quantity) * unitPrice,
		CreatedAt:   time.Now(),
	}
}

// IsPaid returns true if the invoice has been fully paid.
func (i *Invoice) IsPaid() bool {
	return i.Status == InvoiceStatusPaid
}

// IsOverdue returns true if the invoice is overdue.
func (i *Invoice) IsOverdue() bool {
	if i.DueDate == nil || i.IsPaid() {
		return false
	}
	return time.Now().After(*i.DueDate)
}

// Balance returns the remaining balance on the invoice.
func (i *Invoice) Balance() int64 {
	return i.Total - i.AmountPaid
}

// InvoiceWithItems includes invoice items for display.
type InvoiceWithItems struct {
	Invoice
	Items []InvoiceItem `json:"items"`
}

// InvoiceWithCustomer includes customer details for display.
type InvoiceWithCustomer struct {
	Invoice
	CustomerEmail string `json:"customer_email"`
	CustomerName  string `json:"customer_name"`
}

// CreateInvoiceRequest is the request body for creating an invoice (admin).
type CreateInvoiceRequest struct {
	CustomerID     uuid.UUID             `json:"customer_id" binding:"required"`
	Currency       string                `json:"currency" binding:"required,len=3"`
	Items          []CreateInvoiceItemRequest `json:"items" binding:"required,min=1"`
	Tax            int64                 `json:"tax"`
	BillingAddress string                `json:"billing_address,omitempty"`
	Notes          string                `json:"notes,omitempty"`
	DueDate        *time.Time            `json:"due_date,omitempty"`
}

// CreateInvoiceItemRequest is the request for a line item.
type CreateInvoiceItemRequest struct {
	LicenseID   *uuid.UUID `json:"license_id,omitempty"`
	Description string     `json:"description" binding:"required,min=1"`
	Quantity    int        `json:"quantity" binding:"required,min=1"`
	UnitPrice   int64      `json:"unit_price" binding:"required,min=0"`
}

// UpdateInvoiceRequest is the request body for updating an invoice (admin).
type UpdateInvoiceRequest struct {
	Status         *InvoiceStatus `json:"status,omitempty"`
	BillingAddress *string        `json:"billing_address,omitempty"`
	Notes          *string        `json:"notes,omitempty"`
	DueDate        *time.Time     `json:"due_date,omitempty"`
}

// RecordPaymentRequest is the request for recording a payment.
type RecordPaymentRequest struct {
	Amount        int64         `json:"amount" binding:"required,min=1"`
	PaymentMethod PaymentMethod `json:"payment_method" binding:"required,oneof=card bank_transfer paypal invoice"`
	PaymentRef    string        `json:"payment_ref,omitempty"`
}

// InvoiceDownloadResponse contains invoice details for PDF generation.
type InvoiceDownloadResponse struct {
	Invoice        Invoice       `json:"invoice"`
	Items          []InvoiceItem `json:"items"`
	CustomerName   string        `json:"customer_name"`
	CustomerEmail  string        `json:"customer_email"`
	CustomerCompany string       `json:"customer_company,omitempty"`
}
