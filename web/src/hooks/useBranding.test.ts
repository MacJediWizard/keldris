import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import { useBranding, useResetBranding, useUpdateBranding } from './useBranding';

vi.mock('../lib/api', () => ({
	brandingApi: {
		get: vi.fn(),
		update: vi.fn(),
		reset: vi.fn(),
	},
}));

import { brandingApi } from '../lib/api';

const mockBranding = {
	id: 'branding-1',
	org_id: 'org-1',
	product_name: 'MyBackup',
	logo_url: 'https://example.com/logo.png',
	favicon_url: 'https://example.com/favicon.ico',
	primary_color: '#FF0000',
	secondary_color: '#00FF00',
	support_url: 'https://support.example.com',
	custom_css: '',
};

describe('useBranding', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches branding settings', async () => {
		vi.mocked(brandingApi.get).mockResolvedValue(mockBranding);

		const { result } = renderHook(() => useBranding(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockBranding);
		expect(brandingApi.get).toHaveBeenCalledOnce();
	});

	it('does not retry on 402 errors', async () => {
		const error = Object.assign(new Error('Payment required'), {
			status: 402,
		});
		vi.mocked(brandingApi.get).mockRejectedValue(error);

		const { result } = renderHook(() => useBranding(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isError).toBe(true));
		expect(brandingApi.get).toHaveBeenCalledOnce();
	});
});

describe('useUpdateBranding', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('updates branding settings', async () => {
		const updatedBranding = { ...mockBranding, product_name: 'Updated' };
		vi.mocked(brandingApi.update).mockResolvedValue(updatedBranding);

		const { result } = renderHook(() => useUpdateBranding(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ product_name: 'Updated' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(brandingApi.update).toHaveBeenCalledWith({
			product_name: 'Updated',
		});
	});

	it('handles update error', async () => {
		vi.mocked(brandingApi.update).mockRejectedValue(
			new Error('Update failed'),
		);

		const { result } = renderHook(() => useUpdateBranding(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ product_name: 'Fail' });

		await waitFor(() => expect(result.current.isError).toBe(true));
	});
});

describe('useResetBranding', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('resets branding to defaults', async () => {
		vi.mocked(brandingApi.reset).mockResolvedValue({ message: 'Reset' });

		const { result } = renderHook(() => useResetBranding(), {
			wrapper: createWrapper(),
		});

		result.current.mutate();

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(brandingApi.reset).toHaveBeenCalledOnce();
	});
});
