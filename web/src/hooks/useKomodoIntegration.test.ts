import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCreateKomodoIntegration,
	useDeleteKomodoIntegration,
	useDiscoverKomodoContainers,
	useKomodoContainer,
	useKomodoContainers,
	useKomodoIntegration,
	useKomodoIntegrations,
	useKomodoStack,
	useKomodoStacks,
	useKomodoWebhookEvents,
	useSyncKomodoIntegration,
	useTestKomodoConnection,
	useUpdateKomodoContainer,
	useUpdateKomodoIntegration,
} from './useKomodoIntegration';

vi.mock('../lib/api', () => ({
	komodoApi: {
		listIntegrations: vi.fn(),
		getIntegration: vi.fn(),
		createIntegration: vi.fn(),
		updateIntegration: vi.fn(),
		deleteIntegration: vi.fn(),
		testConnection: vi.fn(),
		syncIntegration: vi.fn(),
		discoverContainers: vi.fn(),
		listContainers: vi.fn(),
		getContainer: vi.fn(),
		updateContainer: vi.fn(),
		listStacks: vi.fn(),
		getStack: vi.fn(),
		listWebhookEvents: vi.fn(),
	},
}));

import { komodoApi } from '../lib/api';

describe('useKomodoIntegrations', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('lists integrations', async () => {
		vi.mocked(komodoApi.listIntegrations).mockResolvedValue([]);

		const { result } = renderHook(() => useKomodoIntegrations(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(komodoApi.listIntegrations).toHaveBeenCalledOnce();
	});
});

describe('useKomodoIntegration', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches one integration', async () => {
		vi.mocked(komodoApi.getIntegration).mockResolvedValue({ id: 'k-1' });

		const { result } = renderHook(() => useKomodoIntegration('k-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(komodoApi.getIntegration).toHaveBeenCalledWith('k-1');
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useKomodoIntegration(''), { wrapper: createWrapper() });
		expect(komodoApi.getIntegration).not.toHaveBeenCalled();
	});
});

describe('useCreateKomodoIntegration', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('creates integration', async () => {
		vi.mocked(komodoApi.createIntegration).mockResolvedValue({ id: 'new' });

		const { result } = renderHook(() => useCreateKomodoIntegration(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ name: 'k' } as never);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(komodoApi.createIntegration).toHaveBeenCalled();
	});
});

describe('useUpdateKomodoIntegration', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('updates integration', async () => {
		vi.mocked(komodoApi.updateIntegration).mockResolvedValue({ id: 'k-1' });

		const { result } = renderHook(() => useUpdateKomodoIntegration(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 'k-1', data: { name: 'x' } as never });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(komodoApi.updateIntegration).toHaveBeenCalledWith('k-1', {
			name: 'x',
		});
	});
});

describe('useDeleteKomodoIntegration', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('deletes integration', async () => {
		vi.mocked(komodoApi.deleteIntegration).mockResolvedValue({
			message: 'Deleted',
		});

		const { result } = renderHook(() => useDeleteKomodoIntegration(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('k-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(komodoApi.deleteIntegration).toHaveBeenCalledWith('k-1');
	});
});

describe('useTestKomodoConnection', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('tests connection', async () => {
		vi.mocked(komodoApi.testConnection).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useTestKomodoConnection(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('k-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(komodoApi.testConnection).toHaveBeenCalledWith('k-1');
	});
});

describe('useSyncKomodoIntegration', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('syncs integration', async () => {
		vi.mocked(komodoApi.syncIntegration).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useSyncKomodoIntegration(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('k-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(komodoApi.syncIntegration).toHaveBeenCalledWith('k-1');
	});
});

describe('useDiscoverKomodoContainers', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('returns disabled query until refetched', () => {
		const { result } = renderHook(() => useDiscoverKomodoContainers('k-1'), {
			wrapper: createWrapper(),
		});

		expect(result.current.isFetching).toBe(false);
		expect(komodoApi.discoverContainers).not.toHaveBeenCalled();
	});
});

describe('useKomodoContainers', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('lists containers', async () => {
		vi.mocked(komodoApi.listContainers).mockResolvedValue([]);

		const { result } = renderHook(() => useKomodoContainers(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(komodoApi.listContainers).toHaveBeenCalledOnce();
	});
});

describe('useKomodoContainer', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches a container', async () => {
		vi.mocked(komodoApi.getContainer).mockResolvedValue({ id: 'c-1' });

		const { result } = renderHook(() => useKomodoContainer('c-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(komodoApi.getContainer).toHaveBeenCalledWith('c-1');
	});
});

describe('useUpdateKomodoContainer', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('updates a container', async () => {
		vi.mocked(komodoApi.updateContainer).mockResolvedValue({ id: 'c-1' });

		const { result } = renderHook(() => useUpdateKomodoContainer(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 'c-1', data: { enabled: true } as never });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(komodoApi.updateContainer).toHaveBeenCalledWith('c-1', {
			enabled: true,
		});
	});
});

describe('useKomodoStacks', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('lists stacks', async () => {
		vi.mocked(komodoApi.listStacks).mockResolvedValue([]);

		const { result } = renderHook(() => useKomodoStacks(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(komodoApi.listStacks).toHaveBeenCalledOnce();
	});
});

describe('useKomodoStack', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches a stack', async () => {
		vi.mocked(komodoApi.getStack).mockResolvedValue({ id: 's-1' });

		const { result } = renderHook(() => useKomodoStack('s-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(komodoApi.getStack).toHaveBeenCalledWith('s-1');
	});
});

describe('useKomodoWebhookEvents', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('lists webhook events', async () => {
		vi.mocked(komodoApi.listWebhookEvents).mockResolvedValue([]);

		const { result } = renderHook(() => useKomodoWebhookEvents(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(komodoApi.listWebhookEvents).toHaveBeenCalledOnce();
	});
});
