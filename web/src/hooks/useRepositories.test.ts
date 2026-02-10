import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCreateRepository,
	useDeleteRepository,
	useRepositories,
	useRepository,
	useTestConnection,
	useTestRepository,
} from './useRepositories';

vi.mock('../lib/api', () => ({
	repositoriesApi: {
		list: vi.fn(),
		get: vi.fn(),
		create: vi.fn(),
		update: vi.fn(),
		delete: vi.fn(),
		test: vi.fn(),
		testConnection: vi.fn(),
		recoverKey: vi.fn(),
	},
}));

import { repositoriesApi } from '../lib/api';

const mockRepos = [
	{
		id: 'repo-1',
		name: 'Local Backup',
		type: 'local',
		org_id: 'org-1',
		created_at: '2024-01-01T00:00:00Z',
		updated_at: '2024-01-01T00:00:00Z',
	},
];

describe('useRepositories', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches repositories', async () => {
		vi.mocked(repositoriesApi.list).mockResolvedValue(mockRepos);

		const { result } = renderHook(() => useRepositories(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockRepos);
	});
});

describe('useRepository', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches a single repository', async () => {
		vi.mocked(repositoriesApi.get).mockResolvedValue(mockRepos[0]);

		const { result } = renderHook(() => useRepository('repo-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(repositoriesApi.get).toHaveBeenCalledWith('repo-1');
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useRepository(''), {
			wrapper: createWrapper(),
		});
		expect(repositoriesApi.get).not.toHaveBeenCalled();
	});
});

describe('useCreateRepository', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('creates a repository', async () => {
		vi.mocked(repositoriesApi.create).mockResolvedValue({
			id: 'new-repo',
			repository_password: 'pwd',
		});

		const { result } = renderHook(() => useCreateRepository(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			name: 'New Repo',
			type: 'local',
			path: '/backup',
		} as Parameters<typeof repositoriesApi.create>[0]);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteRepository', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('deletes a repository', async () => {
		vi.mocked(repositoriesApi.delete).mockResolvedValue({
			message: 'Deleted',
		});

		const { result } = renderHook(() => useDeleteRepository(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('repo-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(repositoriesApi.delete).toHaveBeenCalledWith('repo-1');
	});
});

describe('useTestRepository', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('tests a repository', async () => {
		vi.mocked(repositoriesApi.test).mockResolvedValue({ success: true });

		const { result } = renderHook(() => useTestRepository(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('repo-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(repositoriesApi.test).toHaveBeenCalledWith('repo-1');
	});
});

describe('useTestConnection', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('tests a connection', async () => {
		vi.mocked(repositoriesApi.testConnection).mockResolvedValue({
			success: true,
		});

		const { result } = renderHook(() => useTestConnection(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ type: 's3', path: 's3:bucket/path' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
