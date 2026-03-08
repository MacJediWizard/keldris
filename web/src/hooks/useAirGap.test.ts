import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import { useAirGapStatus, useUploadLicense } from './useAirGap';

const mockStatus = {
	airgap_mode: true,
	disable_update_checker: true,
	disable_telemetry: true,
	disable_external_links: false,
	license_valid: true,
};

function mockFetch(data: unknown, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

describe('useAirGapStatus', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches air-gap status', async () => {
		const fetchFn = mockFetch(mockStatus);

		const { result } = renderHook(() => useAirGapStatus(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockStatus);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/public/airgap/status');
	});

	it('returns default status on 404', async () => {
		mockFetch({}, false, 404);

		const { result } = renderHook(() => useAirGapStatus(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual({
			airgap_mode: false,
			disable_update_checker: false,
			disable_telemetry: false,
			disable_external_links: false,
			license_valid: true,
		});
	});

	it('handles error state', async () => {
		mockFetch({ error: 'Server error' }, false, 500);

		const { result } = renderHook(() => useAirGapStatus(), {
			wrapper: createWrapper(),
		});

		// The hook sets retry: 1 which overrides the test QueryClient default,
		// so wait long enough for the retry to complete.
		await waitFor(() => expect(result.current.isError).toBe(true), {
			timeout: 5000,
		});
		expect(result.current.error).toBeDefined();
	});
});

describe('useUploadLicense', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('uploads a license', async () => {
		const mockLicenseInfo = {
			valid: true,
			type: 'enterprise',
			organization: 'Test Corp',
			expires_at: '2025-12-31T00:00:00Z',
			airgap_mode: true,
		};
		const fetchFn = mockFetch(mockLicenseInfo);

		const { result } = renderHook(() => useUploadLicense(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('license-data-string');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/airgap/license',
			expect.objectContaining({
				method: 'POST',
				credentials: 'include',
				body: JSON.stringify({ license: 'license-data-string' }),
			}),
		);
	});

	it('handles upload error', async () => {
		mockFetch({ error: 'Invalid license' }, false, 400);

		const { result } = renderHook(() => useUploadLicense(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('bad-license-data');

		await waitFor(() => expect(result.current.isError).toBe(true));
	});
});
