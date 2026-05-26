import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useImportPreview,
	useImportRepository,
	useVerifyImportAccess,
} from './useRepositoryImport';

function mockFetch(data: unknown, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

describe('useVerifyImportAccess', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('verifies repository import access', async () => {
		const fetchFn = mockFetch({ accessible: true });

		const { result } = renderHook(() => useVerifyImportAccess(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			backend: 's3',
			config: { bucket: 'b' },
			password: 'pw',
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/repositories/import/verify',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useImportPreview', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('previews a repository import', async () => {
		const fetchFn = mockFetch({ snapshots: [] });

		const { result } = renderHook(() => useImportPreview(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			backend: 's3',
			config: { bucket: 'b' },
			password: 'pw',
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/repositories/import/preview',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useImportRepository', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('imports a repository', async () => {
		const fetchFn = mockFetch({ id: 'repo-1' });

		const { result } = renderHook(() => useImportRepository(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			name: 'imported',
			backend: 's3',
			config: { bucket: 'b' },
			password: 'pw',
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/repositories/import',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});
