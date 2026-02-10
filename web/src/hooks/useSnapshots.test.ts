import { renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
	useSnapshots,
	useSnapshot,
	useSnapshotFiles,
	useSnapshotCompare,
} from './useSnapshots';
import { createWrapper } from '../test/helpers';

vi.mock('../lib/api', () => ({
	snapshotsApi: {
		list: vi.fn(),
		get: vi.fn(),
		listFiles: vi.fn(),
		compare: vi.fn(),
	},
}));

import { snapshotsApi } from '../lib/api';

describe('useSnapshots', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches snapshots', async () => {
		vi.mocked(snapshotsApi.list).mockResolvedValue([]);
		const { result } = renderHook(() => useSnapshots(), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('fetches with filter', async () => {
		vi.mocked(snapshotsApi.list).mockResolvedValue([]);
		const filter = { agent_id: 'a1' };
		const { result } = renderHook(() => useSnapshots(filter), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(snapshotsApi.list).toHaveBeenCalledWith(filter);
	});
});

describe('useSnapshot', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches a snapshot', async () => {
		vi.mocked(snapshotsApi.get).mockResolvedValue({ id: 's1' });
		const { result } = renderHook(() => useSnapshot('s1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useSnapshot(''), { wrapper: createWrapper() });
		expect(snapshotsApi.get).not.toHaveBeenCalled();
	});
});

describe('useSnapshotFiles', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches files', async () => {
		vi.mocked(snapshotsApi.listFiles).mockResolvedValue({ files: [] });
		const { result } = renderHook(() => useSnapshotFiles('s1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when snapshotId is empty', () => {
		renderHook(() => useSnapshotFiles(''), { wrapper: createWrapper() });
		expect(snapshotsApi.listFiles).not.toHaveBeenCalled();
	});
});

describe('useSnapshotCompare', () => {
	beforeEach(() => vi.clearAllMocks());

	it('compares snapshots', async () => {
		vi.mocked(snapshotsApi.compare).mockResolvedValue({ diff: [] });
		const { result } = renderHook(() => useSnapshotCompare('s1', 's2'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not compare when either id is empty', () => {
		renderHook(() => useSnapshotCompare('', 's2'), { wrapper: createWrapper() });
		expect(snapshotsApi.compare).not.toHaveBeenCalled();
	});
});
