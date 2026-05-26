import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCreateTemplate,
	useDeleteTemplate,
	useExportAgent,
	useExportBundle,
	useExportRepository,
	useExportSchedule,
	useImportConfig,
	useTemplate,
	useTemplates,
	useUpdateTemplate,
	useUseTemplate,
	useValidateImport,
} from './useConfigExport';

vi.mock('../lib/api', () => ({
	configExportApi: {
		exportAgent: vi.fn(),
		exportSchedule: vi.fn(),
		exportRepository: vi.fn(),
		exportBundle: vi.fn(),
		importConfig: vi.fn(),
		validateImport: vi.fn(),
	},
	templatesApi: {
		list: vi.fn(),
		get: vi.fn(),
		create: vi.fn(),
		update: vi.fn(),
		delete: vi.fn(),
		use: vi.fn(),
	},
}));

import { configExportApi, templatesApi } from '../lib/api';

describe('useExportAgent', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('exports an agent', async () => {
		vi.mocked(configExportApi.exportAgent).mockResolvedValue('config');

		const { result } = renderHook(() => useExportAgent(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 'a-1', format: 'json' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(configExportApi.exportAgent).toHaveBeenCalledWith('a-1', 'json');
	});
});

describe('useExportSchedule', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('exports a schedule', async () => {
		vi.mocked(configExportApi.exportSchedule).mockResolvedValue('config');

		const { result } = renderHook(() => useExportSchedule(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 's-1' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(configExportApi.exportSchedule).toHaveBeenCalledWith(
			's-1',
			undefined,
		);
	});
});

describe('useExportRepository', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('exports a repository', async () => {
		vi.mocked(configExportApi.exportRepository).mockResolvedValue('config');

		const { result } = renderHook(() => useExportRepository(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 'r-1' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(configExportApi.exportRepository).toHaveBeenCalledWith(
			'r-1',
			undefined,
		);
	});
});

describe('useExportBundle', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('exports a bundle', async () => {
		vi.mocked(configExportApi.exportBundle).mockResolvedValue('bundle');

		const { result } = renderHook(() => useExportBundle(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ agent_ids: ['a-1'] } as never);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(configExportApi.exportBundle).toHaveBeenCalled();
	});
});

describe('useImportConfig', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('imports config', async () => {
		vi.mocked(configExportApi.importConfig).mockResolvedValue({
			success: true,
			imported: { agent_count: 0, schedule_count: 0, repository_count: 0 },
		} as never);

		const { result } = renderHook(() => useImportConfig(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ config: '{}' } as never);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(configExportApi.importConfig).toHaveBeenCalled();
	});
});

describe('useValidateImport', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('validates config', async () => {
		vi.mocked(configExportApi.validateImport).mockResolvedValue({
			valid: true,
		} as never);

		const { result } = renderHook(() => useValidateImport(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ config: '{}' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useTemplates', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('lists templates', async () => {
		vi.mocked(templatesApi.list).mockResolvedValue([]);

		const { result } = renderHook(() => useTemplates(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(templatesApi.list).toHaveBeenCalledOnce();
	});
});

describe('useTemplate', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches a template', async () => {
		vi.mocked(templatesApi.get).mockResolvedValue({ id: 't-1' });

		const { result } = renderHook(() => useTemplate('t-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(templatesApi.get).toHaveBeenCalledWith('t-1');
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useTemplate(''), { wrapper: createWrapper() });
		expect(templatesApi.get).not.toHaveBeenCalled();
	});
});

describe('useCreateTemplate', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('creates a template', async () => {
		vi.mocked(templatesApi.create).mockResolvedValue({ id: 'new' });

		const { result } = renderHook(() => useCreateTemplate(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ name: 'new' } as never);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(templatesApi.create).toHaveBeenCalled();
	});
});

describe('useUpdateTemplate', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('updates a template', async () => {
		vi.mocked(templatesApi.update).mockResolvedValue({ id: 't-1' });

		const { result } = renderHook(() => useUpdateTemplate(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 't-1', data: { name: 'x' } as never });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(templatesApi.update).toHaveBeenCalledWith('t-1', { name: 'x' });
	});
});

describe('useDeleteTemplate', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('deletes a template', async () => {
		vi.mocked(templatesApi.delete).mockResolvedValue({ message: 'Deleted' });

		const { result } = renderHook(() => useDeleteTemplate(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('t-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(templatesApi.delete).toHaveBeenCalledWith('t-1');
	});
});

describe('useUseTemplate', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('uses a template', async () => {
		vi.mocked(templatesApi.use).mockResolvedValue({
			success: true,
			imported: { agent_count: 0, schedule_count: 0, repository_count: 0 },
		} as never);

		const { result } = renderHook(() => useUseTemplate(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 't-1' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(templatesApi.use).toHaveBeenCalledWith('t-1', undefined);
	});
});
