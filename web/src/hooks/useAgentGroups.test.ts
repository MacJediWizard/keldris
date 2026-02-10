import { renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
	useAgentGroups,
	useAgentGroup,
	useCreateAgentGroup,
	useDeleteAgentGroup,
	useAgentGroupMembers,
	useAddAgentToGroup,
	useRemoveAgentFromGroup,
	useAgentsWithGroups,
} from './useAgentGroups';
import { createWrapper } from '../test/helpers';

vi.mock('../lib/api', () => ({
	agentGroupsApi: {
		list: vi.fn(),
		get: vi.fn(),
		create: vi.fn(),
		update: vi.fn(),
		delete: vi.fn(),
		listMembers: vi.fn(),
		addAgent: vi.fn(),
		removeAgent: vi.fn(),
	},
	agentsApi: {
		listWithGroups: vi.fn(),
	},
}));

import { agentGroupsApi, agentsApi } from '../lib/api';

describe('useAgentGroups', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches agent groups', async () => {
		vi.mocked(agentGroupsApi.list).mockResolvedValue([{ id: 'g1', name: 'Production' }]);
		const { result } = renderHook(() => useAgentGroups(), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toHaveLength(1);
	});
});

describe('useAgentGroup', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches a single group', async () => {
		vi.mocked(agentGroupsApi.get).mockResolvedValue({ id: 'g1', name: 'Production' });
		const { result } = renderHook(() => useAgentGroup('g1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(agentGroupsApi.get).toHaveBeenCalledWith('g1');
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useAgentGroup(''), { wrapper: createWrapper() });
		expect(agentGroupsApi.get).not.toHaveBeenCalled();
	});
});

describe('useCreateAgentGroup', () => {
	beforeEach(() => vi.clearAllMocks());

	it('creates a group', async () => {
		vi.mocked(agentGroupsApi.create).mockResolvedValue({ id: 'g1', name: 'New' });
		const { result } = renderHook(() => useCreateAgentGroup(), { wrapper: createWrapper() });
		result.current.mutate({ name: 'New', description: 'Test' });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteAgentGroup', () => {
	beforeEach(() => vi.clearAllMocks());

	it('deletes a group', async () => {
		vi.mocked(agentGroupsApi.delete).mockResolvedValue({ message: 'Deleted' });
		const { result } = renderHook(() => useDeleteAgentGroup(), { wrapper: createWrapper() });
		result.current.mutate('g1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useAgentGroupMembers', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches group members', async () => {
		vi.mocked(agentGroupsApi.listMembers).mockResolvedValue([]);
		const { result } = renderHook(() => useAgentGroupMembers('g1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(agentGroupsApi.listMembers).toHaveBeenCalledWith('g1');
	});

	it('does not fetch when groupId is empty', () => {
		renderHook(() => useAgentGroupMembers(''), { wrapper: createWrapper() });
		expect(agentGroupsApi.listMembers).not.toHaveBeenCalled();
	});
});

describe('useAddAgentToGroup', () => {
	beforeEach(() => vi.clearAllMocks());

	it('adds an agent to a group', async () => {
		vi.mocked(agentGroupsApi.addAgent).mockResolvedValue({ message: 'Added' });
		const { result } = renderHook(() => useAddAgentToGroup(), { wrapper: createWrapper() });
		result.current.mutate({ groupId: 'g1', data: { agent_id: 'a1' } });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useRemoveAgentFromGroup', () => {
	beforeEach(() => vi.clearAllMocks());

	it('removes an agent from a group', async () => {
		vi.mocked(agentGroupsApi.removeAgent).mockResolvedValue({ message: 'Removed' });
		const { result } = renderHook(() => useRemoveAgentFromGroup(), { wrapper: createWrapper() });
		result.current.mutate({ groupId: 'g1', agentId: 'a1' });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useAgentsWithGroups', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches agents with groups', async () => {
		vi.mocked(agentsApi.listWithGroups).mockResolvedValue([]);
		const { result } = renderHook(() => useAgentsWithGroups(), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
