// Customer types
export interface Customer {
	id: string;
	email: string;
	name: string;
	company?: string;
	status: 'active' | 'disabled' | 'pending';
	created_at: string;
}

export interface CustomerLoginRequest {
	email: string;
	password: string;
}

export interface CustomerRegisterRequest {
	email: string;
	name: string;
	company?: string;
	password: string;
}

// License types
export type LicenseStatus = 'active' | 'expired' | 'revoked' | 'suspended';
export type LicenseType = 'trial' | 'standard' | 'professional' | 'enterprise';

export interface License {
	id: string;
	customer_id: string;
	license_key: string;
	license_type: LicenseType;
	product_name: string;
	status: LicenseStatus;
	max_agents?: number;
	max_repos?: number;
	max_storage_gb?: number;
	features?: string[];
	issued_at: string;
	expires_at?: string;
	activated_at?: string;
	created_at: string;
}

export interface LicenseDownload {
	license_key: string;
	license_type: LicenseType;
	product_name: string;
	customer_id: string;
	issued_at: string;
	expires_at?: string;
	max_agents?: number;
	max_repos?: number;
	max_storage_gb?: number;
	features?: string[];
}

// Invoice types
export type InvoiceStatus = 'draft' | 'sent' | 'paid' | 'overdue' | 'cancelled' | 'refunded';
export type PaymentMethod = 'card' | 'bank_transfer' | 'paypal' | 'invoice';

export interface Invoice {
	id: string;
	customer_id: string;
	invoice_number: string;
	status: InvoiceStatus;
	currency: string;
	subtotal: number;
	tax: number;
	total: number;
	amount_paid: number;
	payment_method?: PaymentMethod;
	billing_address?: string;
	notes?: string;
	due_date?: string;
	paid_at?: string;
	created_at: string;
}

export interface InvoiceItem {
	id: string;
	invoice_id: string;
	license_id?: string;
	description: string;
	quantity: number;
	unit_price: number;
	total: number;
}

export interface InvoiceWithItems extends Invoice {
	items: InvoiceItem[];
}

// API response types
export interface LicensesResponse {
	licenses: License[];
}

export interface InvoicesResponse {
	invoices: Invoice[];
}

export interface ErrorResponse {
	error: string;
}
