import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useContainerHook,
	useContainerHookExecutions,
	useContainerHookTemplates,
	useContainerHooks,
	useCreateContainerHook,
	useDeleteContainerHook,
	useUpdateContainerHook,
} from './useContainerHooks';

vi.mock('../lib/api', () => ({
	containerHooksApi: {
		list: vi.fn(),
		get: vi.fn(),
		create: vi.fn(),
		update: vi.fn(),
		delete: vi.fn(),
		listTemplates: vi.fn(),
		listExecutions: vi.fn(),
	},
}));

import { containerHooksApi } from '../lib/api';

describe('useContainerHooks', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches container hooks for a schedule', async () => {
		vi.mocked(containerHooksApi.list).mockResolvedValue([]);

		const { result } = renderHook(() => useContainerHooks('s-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(containerHooksApi.list).toHaveBeenCalledWith('s-1');
	});

	it('does not fetch when scheduleId is empty', () => {
		renderHook(() => useContainerHooks(''), { wrapper: createWrapper() });
		expect(containerHooksApi.list).not.toHaveBeenCalled();
	});
});

describe('useContainerHook', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches a single hook', async () => {
		vi.mocked(containerHooksApi.get).mockResolvedValue({ id: 'h-1' });

		const { result } = renderHook(() => useContainerHook('s-1', 'h-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(containerHooksApi.get).toHaveBeenCalledWith('s-1', 'h-1');
	});
});

describe('useCreateContainerHook', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('creates a hook', async () => {
		vi.mocked(containerHooksApi.create).mockResolvedValue({ id: 'new' });

		const { result } = renderHook(() => useCreateContainerHook(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			scheduleId: 's-1',
			data: { name: 'h' } as never,
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(containerHooksApi.create).toHaveBeenCalledWith('s-1', { name: 'h' });
	});
});

describe('useUpdateContainerHook', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('updates a hook', async () => {
		vi.mocked(containerHooksApi.update).mockResolvedValue({ id: 'h-1' });

		const { result } = renderHook(() => useUpdateContainerHook(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			scheduleId: 's-1',
			id: 'h-1',
			data: { name: 'x' } as never,
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(containerHooksApi.update).toHaveBeenCalledWith('s-1', 'h-1', {
			name: 'x',
		});
	});
});

describe('useDeleteContainerHook', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('deletes a hook', async () => {
		vi.mocked(containerHooksApi.delete).mockResolvedValue({
			message: 'Deleted',
		});

		const { result } = renderHook(() => useDeleteContainerHook(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ scheduleId: 's-1', id: 'h-1' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(containerHooksApi.delete).toHaveBeenCalledWith('s-1', 'h-1');
	});
});

describe('useContainerHookTemplates', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('lists templates', async () => {
		vi.mocked(containerHooksApi.listTemplates).mockResolvedValue([]);

		const { result } = renderHook(() => useContainerHookTemplates(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(containerHooksApi.listTemplates).toHaveBeenCalledOnce();
	});
});

describe('useContainerHookExecutions', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('lists executions for backup', async () => {
		vi.mocked(containerHooksApi.listExecutions).mockResolvedValue([]);

		const { result } = renderHook(() => useContainerHookExecutions('b-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(containerHooksApi.listExecutions).toHaveBeenCalledWith('b-1');
	});

	it('does not fetch when backupId is empty', () => {
		renderHook(() => useContainerHookExecutions(''), {
			wrapper: createWrapper(),
		});
		expect(containerHooksApi.listExecutions).not.toHaveBeenCalled();
	});
});
