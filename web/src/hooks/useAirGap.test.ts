import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import { useAirGapStatus, useUploadLicense } from './useAirGap';

vi.mock('../lib/api', () => ({
	airGapApi: {
		getStatus: vi.fn(),
		uploadLicense: vi.fn(),
	},
}));

import { airGapApi } from '../lib/api';

const mockStatus = {
	enabled: true,
	disabled_features: [
		{ name: 'auto_update', reason: 'Requires internet access' },
	],
	license: {
		customer_id: 'cust-123',
		tier: 'enterprise',
		expires_at: '2025-12-31T00:00:00Z',
		issued_at: '2024-01-01T00:00:00Z',
		valid: true,
	},
};

describe('useAirGapStatus', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches air-gap status', async () => {
		vi.mocked(airGapApi.getStatus).mockResolvedValue(mockStatus);

		const { result } = renderHook(() => useAirGapStatus(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockStatus);
		expect(airGapApi.getStatus).toHaveBeenCalledOnce();
	});

	it('handles error state', async () => {
		vi.mocked(airGapApi.getStatus).mockRejectedValue(
			new Error('Network error'),
		);

		const { result } = renderHook(() => useAirGapStatus(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isError).toBe(true));
		expect(result.current.error).toBeDefined();
	});
});

describe('useUploadLicense', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('uploads a license', async () => {
		const mockLicenseInfo = {
			customer_id: 'cust-123',
			tier: 'enterprise',
			expires_at: '2025-12-31T00:00:00Z',
			issued_at: '2024-01-01T00:00:00Z',
			valid: true,
		};
		vi.mocked(airGapApi.uploadLicense).mockResolvedValue(mockLicenseInfo);

		const { result } = renderHook(() => useUploadLicense(), {
			wrapper: createWrapper(),
		});

		const buffer = new ArrayBuffer(8);
		result.current.mutate(buffer);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(airGapApi.uploadLicense).toHaveBeenCalledWith(buffer);
	});

	it('handles upload error', async () => {
		vi.mocked(airGapApi.uploadLicense).mockRejectedValue(
			new Error('Invalid license'),
		);

		const { result } = renderHook(() => useUploadLicense(), {
			wrapper: createWrapper(),
		});

		const buffer = new ArrayBuffer(8);
		result.current.mutate(buffer);

		await waitFor(() => expect(result.current.isError).toBe(true));
	});
});
