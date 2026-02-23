import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { agentGroupsApi, agentsApi } from '../lib/api';
import type {
	AddAgentToGroupRequest,
	CreateAgentGroupRequest,
	UpdateAgentGroupRequest,
} from '../lib/types';

export function useAgentGroups() {
	return useQuery({
		queryKey: ['agentGroups'],
		queryFn: agentGroupsApi.list,
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useAgentGroup(id: string) {
	return useQuery({
		queryKey: ['agentGroups', id],
		queryFn: () => agentGroupsApi.get(id),
		enabled: !!id,
	});
}

export function useCreateAgentGroup() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateAgentGroupRequest) => agentGroupsApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['agentGroups'] });
		},
	});
}

export function useUpdateAgentGroup() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ id, data }: { id: string; data: UpdateAgentGroupRequest }) =>
			agentGroupsApi.update(id, data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['agentGroups'] });
		},
	});
}

export function useDeleteAgentGroup() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => agentGroupsApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['agentGroups'] });
		},
	});
}

export function useAgentGroupMembers(groupId: string) {
	return useQuery({
		queryKey: ['agentGroups', groupId, 'members'],
		queryFn: () => agentGroupsApi.listMembers(groupId),
		enabled: !!groupId,
	});
}

export function useAddAgentToGroup() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			groupId,
			data,
		}: {
			groupId: string;
			data: AddAgentToGroupRequest;
		}) => agentGroupsApi.addAgent(groupId, data),
		onSuccess: (_, variables) => {
			queryClient.invalidateQueries({ queryKey: ['agentGroups'] });
			queryClient.invalidateQueries({
				queryKey: ['agentGroups', variables.groupId, 'members'],
			});
			queryClient.invalidateQueries({ queryKey: ['agentsWithGroups'] });
		},
	});
}

export function useRemoveAgentFromGroup() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ groupId, agentId }: { groupId: string; agentId: string }) =>
			agentGroupsApi.removeAgent(groupId, agentId),
		onSuccess: (_, variables) => {
			queryClient.invalidateQueries({ queryKey: ['agentGroups'] });
			queryClient.invalidateQueries({
				queryKey: ['agentGroups', variables.groupId, 'members'],
			});
			queryClient.invalidateQueries({ queryKey: ['agentsWithGroups'] });
		},
	});
}

export function useAgentsWithGroups() {
	return useQuery({
		queryKey: ['agentsWithGroups'],
		queryFn: agentsApi.listWithGroups,
		staleTime: 30 * 1000, // 30 seconds
	});
}
