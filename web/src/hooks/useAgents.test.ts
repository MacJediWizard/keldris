import { renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
	useAgents,
	useAgent,
	useCreateAgent,
	useDeleteAgent,
	useRotateAgentApiKey,
	useRevokeAgentApiKey,
	useAgentStats,
	useAgentBackups,
	useAgentSchedules,
	useAgentHealthHistory,
	useFleetHealth,
} from './useAgents';
import { createWrapper } from '../test/helpers';

vi.mock('../lib/api', () => ({
	agentsApi: {
		list: vi.fn(),
		get: vi.fn(),
		create: vi.fn(),
		delete: vi.fn(),
		rotateApiKey: vi.fn(),
		revokeApiKey: vi.fn(),
		getStats: vi.fn(),
		getBackups: vi.fn(),
		getSchedules: vi.fn(),
		getHealthHistory: vi.fn(),
		getFleetHealth: vi.fn(),
	},
	schedulesApi: {
		run: vi.fn(),
	},
}));

import { agentsApi } from '../lib/api';

const mockAgents = [
	{
		id: 'agent-1',
		hostname: 'server-1',
		status: 'active',
		org_id: 'org-1',
		created_at: '2024-01-01T00:00:00Z',
		updated_at: '2024-01-01T00:00:00Z',
	},
	{
		id: 'agent-2',
		hostname: 'server-2',
		status: 'offline',
		org_id: 'org-1',
		created_at: '2024-01-01T00:00:00Z',
		updated_at: '2024-01-01T00:00:00Z',
	},
];

describe('useAgents', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches agents list', async () => {
		vi.mocked(agentsApi.list).mockResolvedValue(mockAgents);

		const { result } = renderHook(() => useAgents(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockAgents);
		expect(agentsApi.list).toHaveBeenCalledOnce();
	});

	it('handles error state', async () => {
		vi.mocked(agentsApi.list).mockRejectedValue(new Error('Network error'));

		const { result } = renderHook(() => useAgents(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isError).toBe(true));
		expect(result.current.error).toBeDefined();
	});
});

describe('useAgent', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches a single agent', async () => {
		vi.mocked(agentsApi.get).mockResolvedValue(mockAgents[0]);

		const { result } = renderHook(() => useAgent('agent-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockAgents[0]);
		expect(agentsApi.get).toHaveBeenCalledWith('agent-1');
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useAgent(''), {
			wrapper: createWrapper(),
		});

		expect(agentsApi.get).not.toHaveBeenCalled();
	});
});

describe('useCreateAgent', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('creates an agent', async () => {
		const mockResponse = {
			id: 'new-agent',
			api_key: 'test-key',
			hostname: 'new-host',
		};
		vi.mocked(agentsApi.create).mockResolvedValue(mockResponse);

		const { result } = renderHook(() => useCreateAgent(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ hostname: 'new-host' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(agentsApi.create).toHaveBeenCalledWith({ hostname: 'new-host' });
	});
});

describe('useDeleteAgent', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('deletes an agent', async () => {
		vi.mocked(agentsApi.delete).mockResolvedValue({ message: 'Deleted' });

		const { result } = renderHook(() => useDeleteAgent(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('agent-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(agentsApi.delete).toHaveBeenCalledWith('agent-1');
	});
});

describe('useRotateAgentApiKey', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('rotates API key', async () => {
		vi.mocked(agentsApi.rotateApiKey).mockResolvedValue({
			api_key: 'new-key',
		});

		const { result } = renderHook(() => useRotateAgentApiKey(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('agent-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(agentsApi.rotateApiKey).toHaveBeenCalledWith('agent-1');
	});
});

describe('useRevokeAgentApiKey', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('revokes API key', async () => {
		vi.mocked(agentsApi.revokeApiKey).mockResolvedValue({
			message: 'Revoked',
		});

		const { result } = renderHook(() => useRevokeAgentApiKey(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('agent-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(agentsApi.revokeApiKey).toHaveBeenCalledWith('agent-1');
	});
});

describe('useAgentStats', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches agent stats', async () => {
		const mockStats = { total_backups: 10, success_rate: 95 };
		vi.mocked(agentsApi.getStats).mockResolvedValue(mockStats);

		const { result } = renderHook(() => useAgentStats('agent-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(agentsApi.getStats).toHaveBeenCalledWith('agent-1');
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useAgentStats(''), {
			wrapper: createWrapper(),
		});

		expect(agentsApi.getStats).not.toHaveBeenCalled();
	});
});

describe('useAgentBackups', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches agent backups', async () => {
		const mockBackups = { backups: [], total: 0 };
		vi.mocked(agentsApi.getBackups).mockResolvedValue(mockBackups);

		const { result } = renderHook(() => useAgentBackups('agent-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(agentsApi.getBackups).toHaveBeenCalledWith('agent-1');
	});
});

describe('useAgentSchedules', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches agent schedules', async () => {
		const mockSchedules = { schedules: [] };
		vi.mocked(agentsApi.getSchedules).mockResolvedValue(mockSchedules);

		const { result } = renderHook(() => useAgentSchedules('agent-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(agentsApi.getSchedules).toHaveBeenCalledWith('agent-1');
	});
});

describe('useAgentHealthHistory', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches health history with default limit', async () => {
		vi.mocked(agentsApi.getHealthHistory).mockResolvedValue({ history: [] });

		const { result } = renderHook(() => useAgentHealthHistory('agent-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(agentsApi.getHealthHistory).toHaveBeenCalledWith('agent-1', 100);
	});

	it('fetches health history with custom limit', async () => {
		vi.mocked(agentsApi.getHealthHistory).mockResolvedValue({ history: [] });

		const { result } = renderHook(() => useAgentHealthHistory('agent-1', 50), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(agentsApi.getHealthHistory).toHaveBeenCalledWith('agent-1', 50);
	});
});

describe('useFleetHealth', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches fleet health', async () => {
		const mockFleetHealth = {
			total: 10,
			healthy: 8,
			warning: 1,
			critical: 1,
		};
		vi.mocked(agentsApi.getFleetHealth).mockResolvedValue(mockFleetHealth);

		const { result } = renderHook(() => useFleetHealth(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockFleetHealth);
	});
});
