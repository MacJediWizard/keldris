import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCreateDockerStack,
	useDeleteDockerStack,
	useDeleteDockerStackBackup,
	useDiscoverDockerStacks,
	useDockerStack,
	useDockerStackBackup,
	useDockerStackBackups,
	useDockerStackRestore,
	useDockerStacks,
	useRestoreDockerStack,
	useTriggerDockerStackBackup,
	useUpdateDockerStack,
} from './useDockerStacks';

function mockFetch(data: unknown, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

beforeEach(() => {
	vi.restoreAllMocks();
});

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('useDockerStacks', () => {
	it('fetches docker stacks', async () => {
		mockFetch({ stacks: [{ id: 's1', name: 'web' }] });
		const { result } = renderHook(() => useDockerStacks(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual([{ id: 's1', name: 'web' }]);
	});
});

describe('useDockerStack', () => {
	it('fetches a single docker stack', async () => {
		mockFetch({ id: 's1', name: 'web' });
		const { result } = renderHook(() => useDockerStack('s1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual({ id: 's1', name: 'web' });
	});

	it('does not fetch when id is empty', () => {
		const fetchFn = mockFetch({});
		renderHook(() => useDockerStack(''), { wrapper: createWrapper() });
		expect(fetchFn).not.toHaveBeenCalled();
	});
});

describe('useCreateDockerStack', () => {
	it('creates a docker stack', async () => {
		mockFetch({ id: 's1', name: 'web' });
		const { result } = renderHook(() => useCreateDockerStack(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ name: 'web', agent_id: 'a1' } as never);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useUpdateDockerStack', () => {
	it('updates a docker stack', async () => {
		mockFetch({ id: 's1', name: 'web' });
		const { result } = renderHook(() => useUpdateDockerStack(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ id: 's1', data: { name: 'web2' } as never });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteDockerStack', () => {
	it('deletes a docker stack', async () => {
		mockFetch({ message: 'Deleted' });
		const { result } = renderHook(() => useDeleteDockerStack(), {
			wrapper: createWrapper(),
		});
		result.current.mutate('s1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useTriggerDockerStackBackup', () => {
	it('triggers a backup', async () => {
		mockFetch({ backup_id: 'b1' });
		const { result } = renderHook(() => useTriggerDockerStackBackup(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ id: 's1' });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDockerStackBackups', () => {
	it('fetches backups for a stack', async () => {
		mockFetch({ backups: [] });
		const { result } = renderHook(() => useDockerStackBackups('s1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDockerStackBackup', () => {
	it('fetches a single backup', async () => {
		mockFetch({ id: 'b1' });
		const { result } = renderHook(() => useDockerStackBackup('b1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteDockerStackBackup', () => {
	it('deletes a backup', async () => {
		mockFetch({ message: 'Deleted' });
		const { result } = renderHook(() => useDeleteDockerStackBackup(), {
			wrapper: createWrapper(),
		});
		result.current.mutate('b1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useRestoreDockerStack', () => {
	it('restores a backup', async () => {
		mockFetch({ id: 'r1' });
		const { result } = renderHook(() => useRestoreDockerStack(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({
			backupId: 'b1',
			data: { agent_id: 'a1' } as never,
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDockerStackRestore', () => {
	it('fetches a restore', async () => {
		mockFetch({ id: 'r1', status: 'completed' });
		const { result } = renderHook(() => useDockerStackRestore('r1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDiscoverDockerStacks', () => {
	it('discovers stacks', async () => {
		mockFetch({ stacks: [] });
		const { result } = renderHook(() => useDiscoverDockerStacks(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ agent_id: 'a1' } as never);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
