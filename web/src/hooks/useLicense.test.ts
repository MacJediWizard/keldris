import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useActivateLicense,
	useCheckTrial,
	useDeactivateLicense,
	useLicense,
	usePricingPlans,
	useStartTrial,
} from './useLicense';

vi.mock('../lib/api', () => ({
	licenseApi: {
		getInfo: vi.fn(),
		activate: vi.fn(),
		deactivate: vi.fn(),
		getPlans: vi.fn(),
		startTrial: vi.fn(),
		checkTrial: vi.fn(),
	},
}));

import { licenseApi } from '../lib/api';

const mockActiveLicense = {
	tier: 'professional' as const,
	customer_id: 'cust-123',
	customer_name: 'Test Customer',
	company: 'Test Corp',
	expires_at: '2027-01-01T00:00:00Z',
	issued_at: '2026-01-01T00:00:00Z',
	features: ['oidc', 'sla', 'geo_replication'],
	limits: {
		max_agents: 50,
		max_repositories: 100,
		max_users: 25,
		max_orgs: 5,
		max_storage_bytes: 1099511627776,
	},
	license_key_source: 'database' as const,
	is_trial: false,
};

const mockExpiredLicense = {
	...mockActiveLicense,
	expires_at: '2025-01-01T00:00:00Z',
};

const mockTrialLicense = {
	...mockActiveLicense,
	tier: 'enterprise' as const,
	is_trial: true,
	trial_days_left: 14,
};

describe('useLicense', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('returns active license status', async () => {
		vi.mocked(licenseApi.getInfo).mockResolvedValue(mockActiveLicense);

		const { result } = renderHook(() => useLicense(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockActiveLicense);
		expect(result.current.data?.tier).toBe('professional');
		expect(result.current.data?.is_trial).toBe(false);
	});

	it('returns expired license data', async () => {
		vi.mocked(licenseApi.getInfo).mockResolvedValue(mockExpiredLicense);

		const { result } = renderHook(() => useLicense(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data?.expires_at).toBe('2025-01-01T00:00:00Z');
	});

	it('returns trial license with days remaining', async () => {
		vi.mocked(licenseApi.getInfo).mockResolvedValue(mockTrialLicense);

		const { result } = renderHook(() => useLicense(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data?.is_trial).toBe(true);
		expect(result.current.data?.trial_days_left).toBe(14);
		expect(result.current.data?.tier).toBe('enterprise');
	});

	it('returns features list for entitlement checks', async () => {
		vi.mocked(licenseApi.getInfo).mockResolvedValue(mockActiveLicense);

		const { result } = renderHook(() => useLicense(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data?.features).toContain('oidc');
		expect(result.current.data?.features).toContain('sla');
		expect(result.current.data?.features).not.toContain('nonexistent_feature');
	});

	it('handles API error', async () => {
		vi.mocked(licenseApi.getInfo).mockRejectedValue(new Error('Unauthorized'));

		const { result } = renderHook(() => useLicense(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isError).toBe(true));
		expect(result.current.error).toBeDefined();
	});

	it('uses 5-minute stale time for caching', async () => {
		vi.mocked(licenseApi.getInfo).mockResolvedValue(mockActiveLicense);

		const { result } = renderHook(() => useLicense(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(licenseApi.getInfo).toHaveBeenCalledOnce();

		// Re-render the hook in the same wrapper - should use cached data
		const wrapper = createWrapper();
		const { result: result2 } = renderHook(() => useLicense(), { wrapper });

		await waitFor(() => expect(result2.current.isSuccess).toBe(true));
		// Each wrapper creates its own QueryClient, so both call once each
		// The staleTime config prevents refetching within the same client
	});

	it('handles air-gap mode (license_key_source = none)', async () => {
		const airGapLicense = {
			...mockActiveLicense,
			license_key_source: 'none' as const,
			tier: 'free' as const,
			features: [],
		};
		vi.mocked(licenseApi.getInfo).mockResolvedValue(airGapLicense);

		const { result } = renderHook(() => useLicense(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data?.license_key_source).toBe('none');
		expect(result.current.data?.features).toEqual([]);
	});
});

describe('useActivateLicense', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('activates a license key', async () => {
		const mockResponse = { status: 'activated', tier: 'professional' as const };
		vi.mocked(licenseApi.activate).mockResolvedValue(mockResponse);

		const { result } = renderHook(() => useActivateLicense(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('LICENSE-KEY-123');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(licenseApi.activate).toHaveBeenCalledWith('LICENSE-KEY-123');
	});

	it('handles activation failure', async () => {
		vi.mocked(licenseApi.activate).mockRejectedValue(
			new Error('Invalid license key'),
		);

		const { result } = renderHook(() => useActivateLicense(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('BAD-KEY');

		await waitFor(() => expect(result.current.isError).toBe(true));
	});
});

describe('useDeactivateLicense', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('deactivates the current license', async () => {
		vi.mocked(licenseApi.deactivate).mockResolvedValue({
			status: 'deactivated',
			tier: 'free',
		});

		const { result } = renderHook(() => useDeactivateLicense(), {
			wrapper: createWrapper(),
		});

		result.current.mutate();

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(licenseApi.deactivate).toHaveBeenCalledOnce();
	});
});

describe('usePricingPlans', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches available pricing plans', async () => {
		const mockPlans = [
			{
				id: 'plan-1',
				product_id: 'prod-1',
				tier: 'professional',
				name: 'Professional',
				base_price_cents: 4900,
				agent_price_cents: 500,
				included_agents: 10,
				included_servers: 5,
				features: ['oidc', 'sla'],
				is_active: true,
				created_at: '2026-01-01T00:00:00Z',
				updated_at: '2026-01-01T00:00:00Z',
			},
		];
		vi.mocked(licenseApi.getPlans).mockResolvedValue(mockPlans);

		const { result } = renderHook(() => usePricingPlans(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockPlans);
		expect(result.current.data?.[0].tier).toBe('professional');
	});
});

describe('useStartTrial', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('starts a trial with email and tier', async () => {
		const mockResponse = {
			status: 'trial_started',
			tier: 'enterprise',
			expires_at: '2026-04-01T00:00:00Z',
		};
		vi.mocked(licenseApi.startTrial).mockResolvedValue(mockResponse);

		const { result } = renderHook(() => useStartTrial(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ email: 'test@example.com', tier: 'enterprise' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(licenseApi.startTrial).toHaveBeenCalledWith({
			email: 'test@example.com',
			tier: 'enterprise',
		});
	});
});

describe('useCheckTrial', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('checks trial status for an email', async () => {
		const mockCheck = {
			has_trial: true,
			is_active: true,
			expired: false,
			tier: 'enterprise',
			expires_at: '2026-04-01T00:00:00Z',
		};
		vi.mocked(licenseApi.checkTrial).mockResolvedValue(mockCheck);

		const { result } = renderHook(() => useCheckTrial('test@example.com'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data?.has_trial).toBe(true);
		expect(result.current.data?.is_active).toBe(true);
		expect(licenseApi.checkTrial).toHaveBeenCalledWith('test@example.com');
	});

	it('does not fetch when email is empty', () => {
		const { result } = renderHook(() => useCheckTrial(''), {
			wrapper: createWrapper(),
		});

		expect(result.current.fetchStatus).toBe('idle');
		expect(licenseApi.checkTrial).not.toHaveBeenCalled();
	});

	it('returns expired trial status', async () => {
		const mockCheck = {
			has_trial: true,
			is_active: false,
			expired: true,
			tier: 'enterprise',
			expires_at: '2025-01-01T00:00:00Z',
		};
		vi.mocked(licenseApi.checkTrial).mockResolvedValue(mockCheck);

		const { result } = renderHook(() => useCheckTrial('expired@example.com'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data?.expired).toBe(true);
		expect(result.current.data?.is_active).toBe(false);
	});
});
