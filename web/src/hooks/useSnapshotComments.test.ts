import { renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
	useSnapshotComments,
	useCreateSnapshotComment,
	useDeleteSnapshotComment,
} from './useSnapshotComments';
import { createWrapper } from '../test/helpers';

vi.mock('../lib/api', () => ({
	snapshotCommentsApi: {
		list: vi.fn(),
		create: vi.fn(),
		delete: vi.fn(),
	},
}));

import { snapshotCommentsApi } from '../lib/api';

describe('useSnapshotComments', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches comments', async () => {
		vi.mocked(snapshotCommentsApi.list).mockResolvedValue([]);
		const { result } = renderHook(() => useSnapshotComments('s1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(snapshotCommentsApi.list).toHaveBeenCalledWith('s1');
	});
});

describe('useCreateSnapshotComment', () => {
	beforeEach(() => vi.clearAllMocks());

	it('creates a comment', async () => {
		vi.mocked(snapshotCommentsApi.create).mockResolvedValue({ id: 'c1' });
		const { result } = renderHook(() => useCreateSnapshotComment('s1'), { wrapper: createWrapper() });
		result.current.mutate({ content: 'Test comment' });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(snapshotCommentsApi.create).toHaveBeenCalledWith('s1', { content: 'Test comment' });
	});
});

describe('useDeleteSnapshotComment', () => {
	beforeEach(() => vi.clearAllMocks());

	it('deletes a comment', async () => {
		vi.mocked(snapshotCommentsApi.delete).mockResolvedValue({ message: 'Deleted' });
		const { result } = renderHook(() => useDeleteSnapshotComment('s1'), { wrapper: createWrapper() });
		result.current.mutate('c1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(snapshotCommentsApi.delete).toHaveBeenCalledWith('c1');
	});
});
