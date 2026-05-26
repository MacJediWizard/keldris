import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useActivateLicense,
	useAdminLicense,
	useAdminLicenses,
	useAdminRevokeLicense,
	useAdminUpdateLicense,
	useCurrentLicense,
	useLicenseHistory,
	useLicensePurchaseUrl,
	useLicenseWarnings,
	useValidateLicense,
} from './useLicenses';

function mockFetch(data: unknown, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

describe('useCurrentLicense', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches the current license info', async () => {
		const mockInfo = { tier: 'enterprise', valid: true };
		const fetchFn = mockFetch(mockInfo);

		const { result } = renderHook(() => useCurrentLicense(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockInfo);
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/license',
			expect.objectContaining({ credentials: 'include' }),
		);
	});
});

describe('useLicenseWarnings', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches license warnings', async () => {
		const mockWarnings = { warnings: { limits: [{ name: 'agents' }] } };
		mockFetch(mockWarnings);

		const { result } = renderHook(() => useLicenseWarnings(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockWarnings);
	});

	it('returns default warnings shape on error', async () => {
		mockFetch({ error: 'fail' }, false, 500);

		const { result } = renderHook(() => useLicenseWarnings(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual({ warnings: { limits: [] } });
	});
});

describe('useLicenseHistory', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches license history', async () => {
		const fetchFn = mockFetch({ history: [] });

		const { result } = renderHook(() => useLicenseHistory(25, 0), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/licenses/history?limit=25&offset=0',
			expect.objectContaining({ credentials: 'include' }),
		);
	});
});

describe('useValidateLicense', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('validates a license key', async () => {
		const fetchFn = mockFetch({ valid: true });

		const { result } = renderHook(() => useValidateLicense(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('abc-123');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/licenses/validate',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useActivateLicense', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('activates a license', async () => {
		const fetchFn = mockFetch({ license: { id: 'lic-1' } });

		const { result } = renderHook(() => useActivateLicense(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ license_key: 'abc-123' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/licenses/activate',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useLicensePurchaseUrl', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches the purchase URL', async () => {
		mockFetch({ url: 'https://buy.example.com' });

		const { result } = renderHook(() => useLicensePurchaseUrl(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual({ url: 'https://buy.example.com' });
	});

	it('returns empty url on error', async () => {
		mockFetch({ error: 'fail' }, false, 500);

		const { result } = renderHook(() => useLicensePurchaseUrl(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual({ url: '' });
	});
});

describe('useAdminLicenses', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches admin licenses list', async () => {
		const fetchFn = mockFetch({ licenses: [] });

		const { result } = renderHook(() => useAdminLicenses(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/admin/licenses',
			expect.objectContaining({ credentials: 'include' }),
		);
	});
});

describe('useAdminLicense', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('does not fetch when id is empty', () => {
		const fetchFn = mockFetch({});

		renderHook(() => useAdminLicense(''), {
			wrapper: createWrapper(),
		});

		expect(fetchFn).not.toHaveBeenCalled();
	});
});

describe('useAdminUpdateLicense', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('updates an admin license', async () => {
		const fetchFn = mockFetch({ license: { id: 'lic-1' } });

		const { result } = renderHook(() => useAdminUpdateLicense(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 'lic-1', data: { status: 'active' } });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/admin/licenses/lic-1',
			expect.objectContaining({ method: 'PUT' }),
		);
	});
});

describe('useAdminRevokeLicense', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('revokes an admin license', async () => {
		const fetchFn = mockFetch({ message: 'revoked' });

		const { result } = renderHook(() => useAdminRevokeLicense(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('lic-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/admin/licenses/lic-1',
			expect.objectContaining({ method: 'DELETE' }),
		);
	});
});
