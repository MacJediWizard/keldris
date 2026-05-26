import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useApplyBackupHookTemplate,
	useBackupHookTemplate,
	useBackupHookTemplates,
	useBuiltInBackupHookTemplates,
	useCreateBackupHookTemplate,
	useDeleteBackupHookTemplate,
	useUpdateBackupHookTemplate,
} from './useBackupHookTemplates';

vi.mock('../lib/api', () => ({
	backupHookTemplatesApi: {
		list: vi.fn(),
		listBuiltIn: vi.fn(),
		get: vi.fn(),
		create: vi.fn(),
		update: vi.fn(),
		delete: vi.fn(),
		apply: vi.fn(),
	},
}));

import { backupHookTemplatesApi } from '../lib/api';

describe('useBackupHookTemplates', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches templates list', async () => {
		const mockTemplates = [{ id: 't-1', name: 'Postgres' }];
		vi.mocked(backupHookTemplatesApi.list).mockResolvedValue(mockTemplates);

		const { result } = renderHook(() => useBackupHookTemplates(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockTemplates);
		expect(backupHookTemplatesApi.list).toHaveBeenCalledWith(undefined);
	});

	it('passes filter params', async () => {
		vi.mocked(backupHookTemplatesApi.list).mockResolvedValue([]);

		renderHook(() => useBackupHookTemplates({ service_type: 'postgres' }), {
			wrapper: createWrapper(),
		});

		await waitFor(() => {
			expect(backupHookTemplatesApi.list).toHaveBeenCalledWith({
				service_type: 'postgres',
			});
		});
	});
});

describe('useBuiltInBackupHookTemplates', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches built-in templates', async () => {
		vi.mocked(backupHookTemplatesApi.listBuiltIn).mockResolvedValue([]);

		const { result } = renderHook(() => useBuiltInBackupHookTemplates(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(backupHookTemplatesApi.listBuiltIn).toHaveBeenCalledOnce();
	});
});

describe('useBackupHookTemplate', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches a single template', async () => {
		vi.mocked(backupHookTemplatesApi.get).mockResolvedValue({ id: 't-1' });

		const { result } = renderHook(() => useBackupHookTemplate('t-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(backupHookTemplatesApi.get).toHaveBeenCalledWith('t-1');
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useBackupHookTemplate(''), {
			wrapper: createWrapper(),
		});

		expect(backupHookTemplatesApi.get).not.toHaveBeenCalled();
	});
});

describe('useCreateBackupHookTemplate', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('creates a template', async () => {
		vi.mocked(backupHookTemplatesApi.create).mockResolvedValue({ id: 'new' });

		const { result } = renderHook(() => useCreateBackupHookTemplate(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ name: 'new' } as never);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(backupHookTemplatesApi.create).toHaveBeenCalled();
	});
});

describe('useUpdateBackupHookTemplate', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('updates a template', async () => {
		vi.mocked(backupHookTemplatesApi.update).mockResolvedValue({ id: 't-1' });

		const { result } = renderHook(() => useUpdateBackupHookTemplate(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 't-1', data: { name: 'renamed' } as never });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(backupHookTemplatesApi.update).toHaveBeenCalledWith('t-1', {
			name: 'renamed',
		});
	});
});

describe('useDeleteBackupHookTemplate', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('deletes a template', async () => {
		vi.mocked(backupHookTemplatesApi.delete).mockResolvedValue({
			message: 'Deleted',
		});

		const { result } = renderHook(() => useDeleteBackupHookTemplate(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('t-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(backupHookTemplatesApi.delete).toHaveBeenCalledWith('t-1');
	});
});

describe('useApplyBackupHookTemplate', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('applies a template', async () => {
		vi.mocked(backupHookTemplatesApi.apply).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useApplyBackupHookTemplate(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			templateId: 't-1',
			data: { schedule_id: 's-1' } as never,
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(backupHookTemplatesApi.apply).toHaveBeenCalledWith('t-1', {
			schedule_id: 's-1',
		});
	});
});
