import type {
	Customer,
	CustomerLoginRequest,
	CustomerRegisterRequest,
	Invoice,
	InvoiceItem,
	License,
	LicenseDownload,
	LicensesResponse,
	InvoicesResponse,
} from './types';

const API_BASE = '/api/v1';

class ApiError extends Error {
	status: number;

	constructor(message: string, status: number) {
		super(message);
		this.status = status;
		this.name = 'ApiError';
	}
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
	const response = await fetch(`${API_BASE}${path}`, {
		...options,
		headers: {
			'Content-Type': 'application/json',
			...options?.headers,
		},
		credentials: 'include',
	});

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: 'Unknown error' }));
		throw new ApiError(error.error || 'Request failed', response.status);
	}

	return response.json();
}

// Auth API
export const authApi = {
	login: (data: CustomerLoginRequest): Promise<Customer> =>
		request('/auth/login', { method: 'POST', body: JSON.stringify(data) }),

	register: (data: CustomerRegisterRequest): Promise<Customer> =>
		request('/auth/register', { method: 'POST', body: JSON.stringify(data) }),

	logout: (): Promise<{ message: string }> =>
		request('/auth/logout', { method: 'POST' }),

	me: (): Promise<Customer> => request('/auth/me'),

	changePassword: (currentPassword: string, newPassword: string): Promise<{ message: string }> =>
		request('/auth/change-password', {
			method: 'POST',
			body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
		}),
};

// Licenses API
export const licensesApi = {
	list: (): Promise<LicensesResponse> => request('/licenses'),

	get: (id: string): Promise<License> => request(`/licenses/${id}`),

	download: (id: string): Promise<LicenseDownload> => request(`/licenses/${id}/download`),
};

// Invoices API
export const invoicesApi = {
	list: (): Promise<InvoicesResponse> => request('/invoices'),

	get: (id: string): Promise<{ invoice: Invoice; items: InvoiceItem[] }> =>
		request(`/invoices/${id}`),

	download: (id: string): Promise<{
		invoice: Invoice;
		items: InvoiceItem[];
		customer_name: string;
		customer_email: string;
		customer_company?: string;
	}> => request(`/invoices/${id}/download`),
};

export { ApiError };
