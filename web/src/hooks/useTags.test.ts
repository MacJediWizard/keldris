import { renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
	useTags,
	useTag,
	useCreateTag,
	useDeleteTag,
	useBackupTags,
	useSetBackupTags,
} from './useTags';
import { createWrapper } from '../test/helpers';

vi.mock('../lib/api', () => ({
	tagsApi: {
		list: vi.fn(),
		get: vi.fn(),
		create: vi.fn(),
		update: vi.fn(),
		delete: vi.fn(),
		getBackupTags: vi.fn(),
		setBackupTags: vi.fn(),
	},
}));

import { tagsApi } from '../lib/api';

const mockTags = [
	{ id: 'tag-1', name: 'production', color: '#ff0000', org_id: 'org-1' },
	{ id: 'tag-2', name: 'staging', color: '#00ff00', org_id: 'org-1' },
];

describe('useTags', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches tags', async () => {
		vi.mocked(tagsApi.list).mockResolvedValue(mockTags);

		const { result } = renderHook(() => useTags(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockTags);
	});
});

describe('useTag', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches a single tag', async () => {
		vi.mocked(tagsApi.get).mockResolvedValue(mockTags[0]);

		const { result } = renderHook(() => useTag('tag-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(tagsApi.get).toHaveBeenCalledWith('tag-1');
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useTag(''), {
			wrapper: createWrapper(),
		});
		expect(tagsApi.get).not.toHaveBeenCalled();
	});
});

describe('useCreateTag', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('creates a tag', async () => {
		vi.mocked(tagsApi.create).mockResolvedValue(mockTags[0]);

		const { result } = renderHook(() => useCreateTag(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ name: 'production', color: '#ff0000' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteTag', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('deletes a tag', async () => {
		vi.mocked(tagsApi.delete).mockResolvedValue({ message: 'Deleted' });

		const { result } = renderHook(() => useDeleteTag(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('tag-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(tagsApi.delete).toHaveBeenCalledWith('tag-1');
	});
});

describe('useBackupTags', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches backup tags', async () => {
		vi.mocked(tagsApi.getBackupTags).mockResolvedValue(mockTags);

		const { result } = renderHook(() => useBackupTags('backup-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(tagsApi.getBackupTags).toHaveBeenCalledWith('backup-1');
	});

	it('does not fetch when backupId is empty', () => {
		renderHook(() => useBackupTags(''), {
			wrapper: createWrapper(),
		});
		expect(tagsApi.getBackupTags).not.toHaveBeenCalled();
	});
});

describe('useSetBackupTags', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('sets backup tags', async () => {
		vi.mocked(tagsApi.setBackupTags).mockResolvedValue(mockTags);

		const { result } = renderHook(() => useSetBackupTags(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			backupId: 'backup-1',
			data: { tag_ids: ['tag-1', 'tag-2'] },
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(tagsApi.setBackupTags).toHaveBeenCalledWith('backup-1', {
			tag_ids: ['tag-1', 'tag-2'],
		});
	});
});
