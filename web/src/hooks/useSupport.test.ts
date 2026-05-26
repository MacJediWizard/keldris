import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import { useGenerateSupportBundle } from './useSupport';

function mockFetchBlob(blob: Blob, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		blob: () => Promise.resolve(blob),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

describe('useGenerateSupportBundle', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
		vi.spyOn(window.URL, 'createObjectURL').mockReturnValue('blob:mock');
		vi.spyOn(window.URL, 'revokeObjectURL').mockImplementation(() => {});
		vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {});
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('generates a support bundle', async () => {
		const blob = new Blob(['zip-bytes']);
		const fetchFn = mockFetchBlob(blob);

		const { result } = renderHook(() => useGenerateSupportBundle(), {
			wrapper: createWrapper(),
		});

		result.current.mutate();

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/support/bundle',
			expect.objectContaining({ method: 'POST' }),
		);
		expect(result.current.data?.filename).toMatch(/keldris-support-bundle-/);
	});

	it('handles error', async () => {
		const fn = vi.fn().mockResolvedValue({
			ok: false,
			status: 500,
			blob: () => Promise.resolve(new Blob()),
		} as unknown as Response);
		vi.stubGlobal('fetch', fn);

		const { result } = renderHook(() => useGenerateSupportBundle(), {
			wrapper: createWrapper(),
		});

		result.current.mutate();

		await waitFor(() => expect(result.current.isError).toBe(true));
	});
});
