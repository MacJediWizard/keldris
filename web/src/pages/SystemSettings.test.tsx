import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { SystemSettings } from './SystemSettings';

// --- Mock state containers ---

const mockMe = {
	data: null as {
		id: string;
		email: string;
		name: string;
		current_org_role?: string;
	} | null,
};

const mockSettings = {
	data: null as Record<string, unknown> | null,
	isLoading: false,
};

const mockAuditLogs = {
	data: null as {
		logs: {
			id: string;
			setting_key: string;
			changed_by: string;
			changed_by_email?: string;
			changed_at: string;
			ip_address?: string;
		}[];
	} | null,
};

const mockUpdateSMTP = {
	mutateAsync: vi.fn().mockResolvedValue(undefined),
	isPending: false,
};

const mockTestSMTP = {
	mutateAsync: vi.fn().mockResolvedValue({ success: true, message: 'OK' }),
	isPending: false,
};

const mockUpdateOIDC = {
	mutateAsync: vi.fn().mockResolvedValue(undefined),
	isPending: false,
};

const mockTestOIDC = {
	mutateAsync: vi.fn().mockResolvedValue({ success: true, message: 'OK' }),
	isPending: false,
};

const mockUpdateStorage = {
	mutateAsync: vi.fn().mockResolvedValue(undefined),
	isPending: false,
};

const mockUpdateSecurity = {
	mutateAsync: vi.fn().mockResolvedValue(undefined),
	isPending: false,
};

// --- Mocks ---

vi.mock('../hooks/useAuth', () => ({
	useMe: () => mockMe,
}));

vi.mock('../hooks/useSystemSettings', () => ({
	useSystemSettings: () => mockSettings,
	useSettingsAuditLog: () => mockAuditLogs,
	useUpdateSMTPSettings: () => mockUpdateSMTP,
	useTestSMTP: () => mockTestSMTP,
	useUpdateOIDCSettings: () => mockUpdateOIDC,
	useTestOIDC: () => mockTestOIDC,
	useUpdateStorageDefaultSettings: () => mockUpdateStorage,
	useUpdateSecuritySettings: () => mockUpdateSecurity,
}));

// --- Default settings data ---

const defaultSettingsData = {
	smtp: {
		host: 'smtp.example.com',
		port: 587,
		username: 'user@example.com',
		password: 'secret',
		from_email: 'noreply@example.com',
		from_name: 'Keldris',
		encryption: 'starttls' as const,
		enabled: false,
		skip_tls_verify: false,
		connection_timeout_seconds: 30,
	},
	oidc: {
		enabled: false,
		issuer: 'https://accounts.google.com',
		client_id: 'my-client-id',
		client_secret: 'my-secret',
		redirect_url: 'https://keldris.example.com/auth/callback',
		scopes: ['openid', 'profile', 'email'],
		auto_create_users: false,
		default_role: 'member' as const,
		allowed_domains: [],
		require_email_verification: true,
	},
	storage_defaults: {
		default_retention_days: 30,
		max_retention_days: 365,
		default_storage_backend: 'local' as const,
		max_backup_size_gb: 100,
		enable_compression: true,
		compression_level: 6,
		default_encryption_method: 'aes256' as const,
		prune_schedule: '0 2 * * *',
		auto_prune_enabled: true,
	},
	security: {
		session_timeout_minutes: 480,
		max_concurrent_sessions: 5,
		require_mfa: false,
		mfa_grace_period_days: 7,
		allowed_ip_ranges: [],
		blocked_ip_ranges: [],
		failed_login_lockout_attempts: 5,
		failed_login_lockout_minutes: 30,
		api_key_expiration_days: 0,
		enable_audit_logging: true,
		audit_log_retention_days: 90,
		force_https: true,
		allow_password_login: true,
	},
};

// --- Helpers ---

function createQueryClient() {
	return new QueryClient({
		defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
	});
}

function renderSystemSettings() {
	const queryClient = createQueryClient();
	return render(
		<QueryClientProvider client={queryClient}>
			<MemoryRouter>
				<SystemSettings />
			</MemoryRouter>
		</QueryClientProvider>,
	);
}

beforeEach(() => {
	vi.clearAllMocks();
	mockMe.data = {
		id: 'user-1',
		email: 'admin@test.com',
		name: 'Admin',
		current_org_role: 'owner',
	};
	mockSettings.data = { ...defaultSettingsData };
	mockSettings.isLoading = false;
	mockAuditLogs.data = { logs: [] };
	mockUpdateSMTP.mutateAsync.mockResolvedValue(undefined);
	mockUpdateSMTP.isPending = false;
	mockTestSMTP.mutateAsync.mockResolvedValue({ success: true, message: 'OK' });
	mockTestSMTP.isPending = false;
	mockUpdateOIDC.mutateAsync.mockResolvedValue(undefined);
	mockUpdateOIDC.isPending = false;
	mockTestOIDC.mutateAsync.mockResolvedValue({ success: true, message: 'OK' });
	mockTestOIDC.isPending = false;
	mockUpdateStorage.mutateAsync.mockResolvedValue(undefined);
	mockUpdateStorage.isPending = false;
	mockUpdateSecurity.mutateAsync.mockResolvedValue(undefined);
	mockUpdateSecurity.isPending = false;
});

// --- Tests ---

describe('SystemSettings page', () => {
	describe('loading state', () => {
		it('renders skeleton placeholders while loading', () => {
			mockSettings.isLoading = true;
			renderSystemSettings();
			const pulseElements = document.querySelectorAll('.animate-pulse');
			expect(pulseElements.length).toBeGreaterThan(0);
		});
	});

	describe('access control', () => {
		it('shows access restricted message for member role', () => {
			mockMe.data = {
				id: 'user-1',
				email: 'member@test.com',
				name: 'Member',
				current_org_role: 'member',
			};
			renderSystemSettings();
			expect(screen.getByText('Access Restricted')).toBeInTheDocument();
			expect(
				screen.getByText('System settings require admin or owner access.'),
			).toBeInTheDocument();
		});

		it('shows access restricted message for readonly role', () => {
			mockMe.data = {
				id: 'user-1',
				email: 'ro@test.com',
				name: 'ReadOnly',
				current_org_role: 'readonly',
			};
			renderSystemSettings();
			expect(screen.getByText('Access Restricted')).toBeInTheDocument();
		});

		it('shows settings for admin role', () => {
			mockMe.data = {
				id: 'user-1',
				email: 'admin@test.com',
				name: 'Admin',
				current_org_role: 'admin',
			};
			renderSystemSettings();
			expect(screen.getByText('System Settings')).toBeInTheDocument();
		});

		it('shows settings for owner role', () => {
			renderSystemSettings();
			expect(screen.getByText('System Settings')).toBeInTheDocument();
		});
	});

	describe('tab rendering', () => {
		it('renders all five setting tabs', () => {
			renderSystemSettings();
			expect(screen.getByText('Email (SMTP)')).toBeInTheDocument();
			expect(screen.getByText('Single Sign-On')).toBeInTheDocument();
			expect(screen.getByText('Storage Defaults')).toBeInTheDocument();
			expect(screen.getByText('Security')).toBeInTheDocument();
			expect(screen.getByText('Audit Log')).toBeInTheDocument();
		});

		it('defaults to the SMTP tab', () => {
			renderSystemSettings();
			expect(screen.getByText('SMTP Configuration')).toBeInTheDocument();
		});

		it('switches to OIDC tab on click', async () => {
			const user = userEvent.setup();
			renderSystemSettings();
			await user.click(screen.getByText('Single Sign-On'));
			expect(screen.getByText('OIDC / Single Sign-On')).toBeInTheDocument();
		});

		it('switches to Storage tab on click', async () => {
			const user = userEvent.setup();
			renderSystemSettings();
			await user.click(screen.getByText('Storage Defaults'));
			expect(
				screen.getByText('Configure default storage and retention settings'),
			).toBeInTheDocument();
		});

		it('switches to Security tab on click', async () => {
			const user = userEvent.setup();
			renderSystemSettings();
			await user.click(screen.getByText('Security'));
			expect(screen.getByText('Security Settings')).toBeInTheDocument();
		});

		it('switches to Audit Log tab on click', async () => {
			const user = userEvent.setup();
			renderSystemSettings();
			await user.click(screen.getByText('Audit Log'));
			expect(screen.getByText('Settings Change History')).toBeInTheDocument();
		});
	});

	describe('SMTP settings form', () => {
		it('loads settings values into the SMTP form', () => {
			renderSystemSettings();
			const hostInput = screen.getByLabelText('SMTP Host') as HTMLInputElement;
			expect(hostInput.value).toBe('smtp.example.com');
			const portInput = screen.getByLabelText('Port') as HTMLInputElement;
			expect(portInput.value).toBe('587');
		});

		it('renders an Edit button and clicking it enables saving', async () => {
			const user = userEvent.setup();
			renderSystemSettings();
			const editBtn = screen.getByRole('button', { name: /^edit$/i });
			expect(editBtn).toBeInTheDocument();
			await user.click(editBtn);
			expect(
				screen.getByRole('button', { name: /save changes/i }),
			).toBeInTheDocument();
			expect(
				screen.getByRole('button', { name: /cancel/i }),
			).toBeInTheDocument();
		});

		it('calls updateSMTP on Save', async () => {
			const user = userEvent.setup();
			renderSystemSettings();
			await user.click(screen.getByRole('button', { name: /^edit$/i }));
			await user.click(screen.getByRole('button', { name: /save changes/i }));
			expect(mockUpdateSMTP.mutateAsync).toHaveBeenCalled();
		});

		it('Cancel button resets the form and hides save controls', async () => {
			const user = userEvent.setup();
			renderSystemSettings();
			await user.click(screen.getByRole('button', { name: /^edit$/i }));
			expect(
				screen.getByRole('button', { name: /save changes/i }),
			).toBeInTheDocument();
			await user.click(screen.getByRole('button', { name: /cancel/i }));
			expect(
				screen.queryByRole('button', { name: /save changes/i }),
			).not.toBeInTheDocument();
		});

		it('does not show SMTP test section when disabled', () => {
			renderSystemSettings();
			expect(
				screen.queryByText('Test SMTP Connection'),
			).not.toBeInTheDocument();
		});

		it('shows SMTP test section when enabled', () => {
			mockSettings.data = {
				...defaultSettingsData,
				smtp: { ...defaultSettingsData.smtp, enabled: true },
			};
			renderSystemSettings();
			expect(screen.getByText('Test SMTP Connection')).toBeInTheDocument();
		});
	});

	describe('OIDC settings form', () => {
		it('loads settings into the OIDC form', async () => {
			const user = userEvent.setup();
			renderSystemSettings();
			await user.click(screen.getByText('Single Sign-On'));
			const issuerInput = screen.getByLabelText(
				'Issuer URL',
			) as HTMLInputElement;
			expect(issuerInput.value).toBe('https://accounts.google.com');
		});

		it('does not show OIDC test section when disabled', async () => {
			const user = userEvent.setup();
			renderSystemSettings();
			await user.click(screen.getByText('Single Sign-On'));
			expect(
				screen.queryByText('Test OIDC Configuration'),
			).not.toBeInTheDocument();
		});

		it('shows OIDC test section when enabled', async () => {
			mockSettings.data = {
				...defaultSettingsData,
				oidc: { ...defaultSettingsData.oidc, enabled: true },
			};
			const user = userEvent.setup();
			renderSystemSettings();
			await user.click(screen.getByText('Single Sign-On'));
			expect(screen.getByText('Test OIDC Configuration')).toBeInTheDocument();
		});

		it('calls updateOIDC on Save', async () => {
			const user = userEvent.setup();
			renderSystemSettings();
			await user.click(screen.getByText('Single Sign-On'));
			await user.click(screen.getByRole('button', { name: /^edit$/i }));
			await user.click(screen.getByRole('button', { name: /save changes/i }));
			expect(mockUpdateOIDC.mutateAsync).toHaveBeenCalled();
		});
	});

	describe('Storage settings form', () => {
		it('loads settings into the storage form', async () => {
			const user = userEvent.setup();
			renderSystemSettings();
			await user.click(screen.getByText('Storage Defaults'));
			const retentionInput = screen.getByLabelText(
				'Default Retention (days)',
			) as HTMLInputElement;
			expect(retentionInput.value).toBe('30');
		});

		it('calls updateStorage on Save', async () => {
			const user = userEvent.setup();
			renderSystemSettings();
			await user.click(screen.getByText('Storage Defaults'));
			await user.click(screen.getByRole('button', { name: /^edit$/i }));
			await user.click(screen.getByRole('button', { name: /save changes/i }));
			expect(mockUpdateStorage.mutateAsync).toHaveBeenCalled();
		});
	});

	describe('Security settings form', () => {
		it('loads settings into the security form', async () => {
			const user = userEvent.setup();
			renderSystemSettings();
			await user.click(screen.getByText('Security'));
			const timeoutInput = screen.getByLabelText(
				'Session Timeout (minutes)',
			) as HTMLInputElement;
			expect(timeoutInput.value).toBe('480');
		});

		it('calls updateSecurity on Save', async () => {
			const user = userEvent.setup();
			renderSystemSettings();
			await user.click(screen.getByText('Security'));
			await user.click(screen.getByRole('button', { name: /^edit$/i }));
			await user.click(screen.getByRole('button', { name: /save changes/i }));
			expect(mockUpdateSecurity.mutateAsync).toHaveBeenCalled();
		});

		it('displays session management section', async () => {
			const user = userEvent.setup();
			renderSystemSettings();
			await user.click(screen.getByText('Security'));
			expect(screen.getByText('Session Management')).toBeInTheDocument();
		});

		it('displays MFA section', async () => {
			const user = userEvent.setup();
			renderSystemSettings();
			await user.click(screen.getByText('Security'));
			expect(
				screen.getByText('Multi-Factor Authentication'),
			).toBeInTheDocument();
		});

		it('displays login protection section', async () => {
			const user = userEvent.setup();
			renderSystemSettings();
			await user.click(screen.getByText('Security'));
			expect(screen.getByText('Login Protection')).toBeInTheDocument();
		});
	});

	describe('Audit Log tab', () => {
		it('shows empty state when no logs', async () => {
			const user = userEvent.setup();
			renderSystemSettings();
			await user.click(screen.getByText('Audit Log'));
			expect(
				screen.getByText('No settings changes recorded yet'),
			).toBeInTheDocument();
		});

		it('shows audit log entries with correct columns', async () => {
			mockAuditLogs.data = {
				logs: [
					{
						id: 'log-1',
						setting_key: 'smtp',
						changed_by: 'user-1',
						changed_by_email: 'admin@test.com',
						changed_at: '2026-03-01T10:00:00Z',
						ip_address: '192.168.1.1',
					},
				],
			};
			const user = userEvent.setup();
			renderSystemSettings();
			await user.click(screen.getByText('Audit Log'));
			expect(screen.getByText('smtp')).toBeInTheDocument();
			expect(screen.getByText('admin@test.com')).toBeInTheDocument();
			expect(screen.getByText('192.168.1.1')).toBeInTheDocument();
		});

		it('renders the table headers', async () => {
			mockAuditLogs.data = {
				logs: [
					{
						id: 'log-1',
						setting_key: 'smtp',
						changed_by: 'user-1',
						changed_at: '2026-03-01T10:00:00Z',
					},
				],
			};
			const user = userEvent.setup();
			renderSystemSettings();
			await user.click(screen.getByText('Audit Log'));
			expect(screen.getByText('Timestamp')).toBeInTheDocument();
			expect(screen.getByText('Setting')).toBeInTheDocument();
			expect(screen.getByText('Changed By')).toBeInTheDocument();
			expect(screen.getByText('IP Address')).toBeInTheDocument();
		});
	});
});
