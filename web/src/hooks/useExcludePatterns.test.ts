import { renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
	useExcludePatterns,
	useExcludePattern,
	useExcludePatternsLibrary,
	useExcludePatternCategories,
	useCreateExcludePattern,
	useDeleteExcludePattern,
} from './useExcludePatterns';
import { createWrapper } from '../test/helpers';

vi.mock('../lib/api', () => ({
	excludePatternsApi: {
		list: vi.fn(),
		get: vi.fn(),
		getLibrary: vi.fn(),
		getCategories: vi.fn(),
		create: vi.fn(),
		update: vi.fn(),
		delete: vi.fn(),
	},
}));

import { excludePatternsApi } from '../lib/api';

describe('useExcludePatterns', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches patterns', async () => {
		vi.mocked(excludePatternsApi.list).mockResolvedValue([]);
		const { result } = renderHook(() => useExcludePatterns(), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('fetches with category filter', async () => {
		vi.mocked(excludePatternsApi.list).mockResolvedValue([]);
		const { result } = renderHook(() => useExcludePatterns('os'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(excludePatternsApi.list).toHaveBeenCalledWith('os');
	});
});

describe('useExcludePattern', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches a pattern', async () => {
		vi.mocked(excludePatternsApi.get).mockResolvedValue({ id: 'ep1' });
		const { result } = renderHook(() => useExcludePattern('ep1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useExcludePattern(''), { wrapper: createWrapper() });
		expect(excludePatternsApi.get).not.toHaveBeenCalled();
	});
});

describe('useExcludePatternsLibrary', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches built-in patterns', async () => {
		vi.mocked(excludePatternsApi.getLibrary).mockResolvedValue([]);
		const { result } = renderHook(() => useExcludePatternsLibrary(), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useExcludePatternCategories', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches categories', async () => {
		vi.mocked(excludePatternsApi.getCategories).mockResolvedValue([]);
		const { result } = renderHook(() => useExcludePatternCategories(), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useCreateExcludePattern', () => {
	beforeEach(() => vi.clearAllMocks());

	it('creates a pattern', async () => {
		vi.mocked(excludePatternsApi.create).mockResolvedValue({ id: 'ep1' });
		const { result } = renderHook(() => useCreateExcludePattern(), { wrapper: createWrapper() });
		result.current.mutate({ pattern: '*.tmp', description: 'Temp files' } as Parameters<typeof excludePatternsApi.create>[0]);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteExcludePattern', () => {
	beforeEach(() => vi.clearAllMocks());

	it('deletes a pattern', async () => {
		vi.mocked(excludePatternsApi.delete).mockResolvedValue({ message: 'Deleted' });
		const { result } = renderHook(() => useDeleteExcludePattern(), { wrapper: createWrapper() });
		result.current.mutate('ep1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
