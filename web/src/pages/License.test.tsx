import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

const mockActivateMutateAsync = vi.fn();
const mockDeactivateMutateAsync = vi.fn();
const mockStartTrialMutateAsync = vi.fn();

vi.mock('../hooks/useLicense', () => ({
	useLicense: vi.fn(),
	usePricingPlans: vi.fn().mockReturnValue({ data: undefined }),
	useActivateLicense: () => ({
		mutateAsync: mockActivateMutateAsync,
		isPending: false,
	}),
	useDeactivateLicense: () => ({
		mutateAsync: mockDeactivateMutateAsync,
		isPending: false,
	}),
	useStartTrial: () => ({
		mutateAsync: mockStartTrialMutateAsync,
		isPending: false,
	}),
}));

vi.mock('../hooks/useLicenses', () => ({
	useCurrentLicense: vi.fn().mockReturnValue({ data: undefined }),
	useLicenseHistory: vi.fn().mockReturnValue({
		data: undefined,
		isLoading: false,
	}),
}));

import { useLicense, usePricingPlans } from '../hooks/useLicense';
import { useCurrentLicense, useLicenseHistory } from '../hooks/useLicenses';
import License from './License';

const baseLicense = {
	tier: 'pro' as const,
	customer_id: 'cust_123',
	customer_name: 'Acme Corp',
	expires_at: '2027-12-31T00:00:00Z',
	issued_at: '2024-01-01T00:00:00Z',
	features: ['oidc', 'api_access', 'custom_reports'],
	limits: {
		max_agents: 50,
		max_servers: 10,
		max_users: 25,
		max_orgs: 3,
		max_storage_bytes: 1099511627776, // 1 TB
	},
	license_key_source: 'database' as const,
	is_trial: false,
};

function setLicenseMock(
	overrides: Record<string, unknown> = {},
	loading = false,
	error: Error | null = null,
) {
	vi.mocked(useLicense).mockReturnValue({
		data: { ...baseLicense, ...overrides },
		isLoading: loading,
		error,
		isError: !!error,
	} as ReturnType<typeof useLicense>);
}

describe('License page', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		setLicenseMock();
	});

	it('shows loading spinner when data is loading', () => {
		vi.mocked(useLicense).mockReturnValue({
			data: undefined,
			isLoading: true,
			error: null,
			isError: false,
		} as ReturnType<typeof useLicense>);
		renderWithProviders(<License />);
		// LoadingSpinner renders a spinner element
		const spinner = document.querySelector('.animate-spin');
		expect(spinner).toBeInTheDocument();
	});

	it('shows error state when license fetch fails', () => {
		vi.mocked(useLicense).mockReturnValue({
			data: undefined,
			isLoading: false,
			error: new Error('Network error'),
			isError: true,
		} as ReturnType<typeof useLicense>);
		renderWithProviders(<License />);
		expect(
			screen.getByText('Failed to load license information.'),
		).toBeInTheDocument();
	});

	it('renders page title and tier badge', () => {
		renderWithProviders(<License />);
		expect(screen.getByText('License')).toBeInTheDocument();
		expect(screen.getByText('Pro')).toBeInTheDocument();
	});

	it('displays license details section with correct data', () => {
		renderWithProviders(<License />);
		expect(screen.getByText('License Details')).toBeInTheDocument();
		expect(screen.getByText('Acme Corp')).toBeInTheDocument();
		expect(screen.getByText('pro')).toBeInTheDocument();
		expect(screen.getByText('database')).toBeInTheDocument();
	});

	it('displays formatted issue and expiry dates', () => {
		renderWithProviders(<License />);
		// formatDate uses toLocaleDateString which is locale-dependent;
		// verify the dates are rendered by matching the expected output for the default locale
		const issuedDate = new Date('2024-01-01T00:00:00Z').toLocaleDateString(
			undefined,
			{ year: 'numeric', month: 'long', day: 'numeric' },
		);
		const expiresDate = new Date('2027-12-31T00:00:00Z').toLocaleDateString(
			undefined,
			{ year: 'numeric', month: 'long', day: 'numeric' },
		);
		expect(screen.getByText(issuedDate)).toBeInTheDocument();
		expect(screen.getByText(expiresDate)).toBeInTheDocument();
	});

	it('displays resource limits section', () => {
		renderWithProviders(<License />);
		expect(screen.getByText('Resource Limits')).toBeInTheDocument();
		expect(screen.getByText('Agents')).toBeInTheDocument();
		expect(screen.getByText('50')).toBeInTheDocument();
		expect(screen.getByText('Servers')).toBeInTheDocument();
		expect(screen.getByText('10')).toBeInTheDocument();
		expect(screen.getByText('Users')).toBeInTheDocument();
		expect(screen.getByText('25')).toBeInTheDocument();
		expect(screen.getByText('Organizations')).toBeInTheDocument();
		expect(screen.getByText('3')).toBeInTheDocument();
		expect(screen.getByText('1 TB')).toBeInTheDocument();
	});

	it("shows 'Unlimited' for zero-value limits", () => {
		setLicenseMock({
			limits: {
				max_agents: 0,
				max_servers: 0,
				max_users: 0,
				max_orgs: 0,
				max_storage_bytes: 0,
			},
		});
		renderWithProviders(<License />);
		const unlimitedElements = screen.getAllByText('Unlimited');
		expect(unlimitedElements.length).toBe(5);
	});

	it('displays included features', () => {
		renderWithProviders(<License />);
		expect(screen.getByText('Included Features')).toBeInTheDocument();
		expect(screen.getByText('oidc')).toBeInTheDocument();
		expect(screen.getByText('api access')).toBeInTheDocument();
		expect(screen.getByText('custom reports')).toBeInTheDocument();
	});

	it('shows no features message when features array is empty', () => {
		setLicenseMock({ features: [] });
		renderWithProviders(<License />);
		expect(
			screen.getByText('No features included in the current plan.'),
		).toBeInTheDocument();
	});

	it('shows deactivate button for database-sourced non-free license', () => {
		renderWithProviders(<License />);
		expect(screen.getByText('Deactivate')).toBeInTheDocument();
	});

	it('hides deactivate button for env-sourced license', () => {
		setLicenseMock({ license_key_source: 'env' });
		renderWithProviders(<License />);
		expect(screen.queryByText('Deactivate')).not.toBeInTheDocument();
	});

	it('hides deactivate button for free tier', () => {
		setLicenseMock({ tier: 'free', license_key_source: 'database' });
		renderWithProviders(<License />);
		expect(screen.queryByText('Deactivate')).not.toBeInTheDocument();
	});

	it('shows env var notice for env-configured licenses', () => {
		setLicenseMock({ license_key_source: 'env' });
		renderWithProviders(<License />);
		expect(screen.getByText(/LICENSE_KEY/)).toBeInTheDocument();
		expect(
			screen.getByText(/remove the environment variable and restart/),
		).toBeInTheDocument();
	});

	it('shows expired indicator when license is expired', () => {
		setLicenseMock({ expires_at: '2020-01-01T00:00:00Z' });
		renderWithProviders(<License />);
		expect(screen.getByText(/\(Expired\)/)).toBeInTheDocument();
	});

	// Trial states
	it('shows trial badge when is_trial is true', () => {
		setLicenseMock({ is_trial: true, trial_days_left: 10 });
		renderWithProviders(<License />);
		expect(screen.getByText('Trial')).toBeInTheDocument();
	});

	it('shows active trial banner with days remaining', () => {
		setLicenseMock({ is_trial: true, trial_days_left: 7 });
		renderWithProviders(<License />);
		expect(screen.getByText(/7 days remaining/)).toBeInTheDocument();
		expect(screen.getByText('Upgrade Now')).toBeInTheDocument();
	});

	it('handles singular day remaining text', () => {
		setLicenseMock({ is_trial: true, trial_days_left: 1 });
		renderWithProviders(<License />);
		expect(screen.getByText(/1 day remaining/)).toBeInTheDocument();
	});

	it('shows expired trial banner when trial is expired', () => {
		setLicenseMock({
			is_trial: true,
			trial_days_left: 0,
			expires_at: '2020-01-01T00:00:00Z',
		});
		renderWithProviders(<License />);
		expect(screen.getByText('Trial expired')).toBeInTheDocument();
		expect(screen.getByText(/Your trial has ended/)).toBeInTheDocument();
	});

	// Free tier: trial start + activate form
	it('shows trial start section for free tier without license', () => {
		setLicenseMock({
			tier: 'free',
			license_key_source: 'none',
			is_trial: false,
		});
		renderWithProviders(<License />);
		expect(screen.getByText('Start Free 14-Day Trial')).toBeInTheDocument();
		expect(
			screen.getByPlaceholderText('Enter your email...'),
		).toBeInTheDocument();
		expect(screen.getByText('Start 14-Day Trial')).toBeInTheDocument();
	});

	it('shows activate license form for free tier', () => {
		setLicenseMock({
			tier: 'free',
			license_key_source: 'none',
			is_trial: false,
		});
		renderWithProviders(<License />);
		expect(screen.getByText('Activate License')).toBeInTheDocument();
		expect(
			screen.getByPlaceholderText('Enter your license key...'),
		).toBeInTheDocument();
	});

	it('trial start button is disabled when email is empty', () => {
		setLicenseMock({
			tier: 'free',
			license_key_source: 'none',
			is_trial: false,
		});
		renderWithProviders(<License />);
		const startButton = screen.getByText('Start 14-Day Trial');
		expect(startButton).toBeDisabled();
	});

	it('activate button is disabled when license key is empty', () => {
		setLicenseMock({
			tier: 'free',
			license_key_source: 'none',
			is_trial: false,
		});
		renderWithProviders(<License />);
		const activateButton = screen.getByText('Activate');
		expect(activateButton).toBeDisabled();
	});

	it('calls startTrial mutation when trial form is submitted', async () => {
		const user = userEvent.setup();
		setLicenseMock({
			tier: 'free',
			license_key_source: 'none',
			is_trial: false,
		});
		renderWithProviders(<License />);

		const emailInput = screen.getByPlaceholderText('Enter your email...');
		await user.type(emailInput, 'test@example.com');

		const startButton = screen.getByText('Start 14-Day Trial');
		await user.click(startButton);

		expect(mockStartTrialMutateAsync).toHaveBeenCalledWith({
			email: 'test@example.com',
			tier: 'pro',
		});
	});

	it('shows trial error when startTrial mutation fails', async () => {
		const user = userEvent.setup();
		mockStartTrialMutateAsync.mockRejectedValueOnce(
			new Error('Trial limit reached'),
		);
		setLicenseMock({
			tier: 'free',
			license_key_source: 'none',
			is_trial: false,
		});
		renderWithProviders(<License />);

		const emailInput = screen.getByPlaceholderText('Enter your email...');
		await user.type(emailInput, 'test@example.com');
		await user.click(screen.getByText('Start 14-Day Trial'));

		expect(await screen.findByText('Trial limit reached')).toBeInTheDocument();
	});

	it('calls activate mutation when license key form is submitted', async () => {
		const user = userEvent.setup();
		setLicenseMock({
			tier: 'free',
			license_key_source: 'none',
			is_trial: false,
		});
		renderWithProviders(<License />);

		const keyInput = screen.getByPlaceholderText('Enter your license key...');
		await user.type(keyInput, 'LK-ABCDEF-123456');
		await user.click(screen.getByText('Activate'));

		expect(mockActivateMutateAsync).toHaveBeenCalledWith('LK-ABCDEF-123456');
	});

	it('shows activate error when activation fails', async () => {
		const user = userEvent.setup();
		mockActivateMutateAsync.mockRejectedValueOnce(
			new Error('Invalid license key'),
		);
		setLicenseMock({
			tier: 'free',
			license_key_source: 'none',
			is_trial: false,
		});
		renderWithProviders(<License />);

		const keyInput = screen.getByPlaceholderText('Enter your license key...');
		await user.type(keyInput, 'BAD-KEY');
		await user.click(screen.getByText('Activate'));

		expect(await screen.findByText('Invalid license key')).toBeInTheDocument();
	});

	// Pricing plans
	it('shows available plans section for free tier when plans exist', () => {
		setLicenseMock({
			tier: 'free',
			license_key_source: 'none',
			is_trial: false,
		});
		vi.mocked(usePricingPlans).mockReturnValue({
			data: [
				{
					id: 'plan_1',
					product_id: 'prod_1',
					tier: 'pro',
					name: 'Pro',
					base_price_cents: 4900,
					agent_price_cents: 500,
					included_agents: 10,
					included_servers: 5,
					features: null,
					is_active: true,
					created_at: '2024-01-01T00:00:00Z',
					updated_at: '2024-01-01T00:00:00Z',
				},
			],
		} as ReturnType<typeof usePricingPlans>);
		renderWithProviders(<License />);
		expect(screen.getByText('Available Plans')).toBeInTheDocument();
		expect(screen.getByText('$49.00')).toBeInTheDocument();
		expect(screen.getByText('10 agents included')).toBeInTheDocument();
		expect(screen.getByText('5 servers included')).toBeInTheDocument();
		expect(screen.getByText('$5.00/extra agent')).toBeInTheDocument();
	});

	it('hides plans section for non-free tier', () => {
		vi.mocked(usePricingPlans).mockReturnValue({
			data: [
				{
					id: 'plan_1',
					product_id: 'prod_1',
					tier: 'pro',
					name: 'Pro',
					base_price_cents: 4900,
					agent_price_cents: 0,
					included_agents: 10,
					included_servers: 5,
					features: null,
					is_active: true,
					created_at: '2024-01-01T00:00:00Z',
					updated_at: '2024-01-01T00:00:00Z',
				},
			],
		} as ReturnType<typeof usePricingPlans>);
		setLicenseMock({ tier: 'pro' });
		renderWithProviders(<License />);
		expect(screen.queryByText('Available Plans')).not.toBeInTheDocument();
	});

	// License history
	it('shows license history with entries', () => {
		vi.mocked(useLicenseHistory).mockReturnValue({
			data: {
				history: [
					{
						id: 'h1',
						license_id: 'lic_1',
						org_id: 'org_1',
						action: 'activated' as const,
						previous_tier: undefined,
						new_tier: 'pro' as const,
						notes: 'Activated via dashboard',
						created_at: '2024-06-15T00:00:00Z',
					},
					{
						id: 'h2',
						license_id: 'lic_1',
						org_id: 'org_1',
						action: 'upgraded' as const,
						previous_tier: 'pro' as const,
						new_tier: 'enterprise' as const,
						created_at: '2024-09-01T00:00:00Z',
					},
				],
				total_count: 2,
			},
			isLoading: false,
		} as ReturnType<typeof useLicenseHistory>);
		renderWithProviders(<License />);
		expect(screen.getByText('License History')).toBeInTheDocument();
		expect(screen.getByText('License Activated')).toBeInTheDocument();
		expect(screen.getByText('Activated via dashboard')).toBeInTheDocument();
		expect(screen.getByText('Plan Upgraded')).toBeInTheDocument();
		expect(screen.getByText('pro \u2192 enterprise')).toBeInTheDocument();
	});

	it('shows empty license history state', () => {
		vi.mocked(useLicenseHistory).mockReturnValue({
			data: { history: [], total_count: 0 },
			isLoading: false,
		} as ReturnType<typeof useLicenseHistory>);
		renderWithProviders(<License />);
		expect(screen.getByText('No license history')).toBeInTheDocument();
		expect(
			screen.getByText('License changes will appear here.'),
		).toBeInTheDocument();
	});

	it('shows loading skeleton for license history', () => {
		vi.mocked(useLicenseHistory).mockReturnValue({
			data: undefined,
			isLoading: true,
		} as ReturnType<typeof useLicenseHistory>);
		renderWithProviders(<License />);
		// Three loading skeleton divs with animate-pulse
		screen
			.getByText('License History')
			.closest('div.bg-white, div.dark\\:bg-gray-800');
		const pulses = document.querySelectorAll('.animate-pulse');
		expect(pulses.length).toBeGreaterThanOrEqual(3);
	});

	// Usage / Limits section with currentLicense
	it('shows limits section when currentLicense has limits', () => {
		vi.mocked(useCurrentLicense).mockReturnValue({
			data: {
				tier: 'pro',
				customer_id: 'c1',
				expires_at: '2027-12-31T00:00:00Z',
				issued_at: '2024-01-01T00:00:00Z',
				features: ['oidc', 'api_access'],
				limits: {
					max_agents: 50,
					max_servers: 10,
					max_users: 25,
					max_orgs: 3,
					max_storage_bytes: 1099511627776,
				},
				license_key_source: 'database',
				is_trial: false,
			},
		} as ReturnType<typeof useCurrentLicense>);
		renderWithProviders(<License />);
		expect(screen.getByText('Limits')).toBeInTheDocument();
	});

	// Extended features section
	it('shows extended features section when currentLicense has features', () => {
		vi.mocked(useCurrentLicense).mockReturnValue({
			data: {
				tier: 'pro',
				customer_id: 'c1',
				expires_at: '2027-12-31T00:00:00Z',
				issued_at: '2024-01-01T00:00:00Z',
				features: ['oidc', 'api_access', 'custom_reports', 'white_label'],
				limits: {
					max_agents: 50,
					max_servers: 10,
					max_users: 25,
					max_orgs: 3,
					max_storage_bytes: 0,
				},
				license_key_source: 'database',
				is_trial: false,
			},
		} as ReturnType<typeof useCurrentLicense>);
		renderWithProviders(<License />);
		expect(screen.getByText('Features')).toBeInTheDocument();
		expect(screen.getByText('Single Sign-On (SSO)')).toBeInTheDocument();
		expect(screen.getByText('API Access')).toBeInTheDocument();
		expect(screen.getByText('Advanced Reporting')).toBeInTheDocument();
		expect(screen.getByText('Custom Branding')).toBeInTheDocument();
	});

	// Key source display
	it("shows 'Not configured' for key source none", () => {
		setLicenseMock({
			tier: 'free',
			license_key_source: 'none',
			is_trial: false,
		});
		renderWithProviders(<License />);
		expect(screen.getByText('Not configured')).toBeInTheDocument();
	});
});
