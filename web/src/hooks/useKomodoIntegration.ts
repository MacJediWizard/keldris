import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { komodoApi } from '../lib/api';
import type {
	CreateKomodoIntegrationRequest,
	UpdateKomodoContainerRequest,
	UpdateKomodoIntegrationRequest,
} from '../lib/types';

// Query Keys
export const komodoKeys = {
	all: ['komodo'] as const,
	integrations: () => [...komodoKeys.all, 'integrations'] as const,
	integration: (id: string) => [...komodoKeys.integrations(), id] as const,
	containers: () => [...komodoKeys.all, 'containers'] as const,
	container: (id: string) => [...komodoKeys.containers(), id] as const,
	stacks: () => [...komodoKeys.all, 'stacks'] as const,
	stack: (id: string) => [...komodoKeys.stacks(), id] as const,
	events: () => [...komodoKeys.all, 'events'] as const,
	discovery: (id: string) =>
		[...komodoKeys.integrations(), id, 'discovery'] as const,
};

// Integration Hooks
export function useKomodoIntegrations() {
	return useQuery({
		queryKey: komodoKeys.integrations(),
		queryFn: komodoApi.listIntegrations,
		staleTime: 30 * 1000,
	});
}

export function useKomodoIntegration(id: string) {
	return useQuery({
		queryKey: komodoKeys.integration(id),
		queryFn: () => komodoApi.getIntegration(id),
		enabled: !!id,
		staleTime: 30 * 1000,
	});
}

export function useCreateKomodoIntegration() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (data: CreateKomodoIntegrationRequest) =>
			komodoApi.createIntegration(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: komodoKeys.integrations() });
		},
	});
}

export function useUpdateKomodoIntegration() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: ({
			id,
			data,
		}: { id: string; data: UpdateKomodoIntegrationRequest }) =>
			komodoApi.updateIntegration(id, data),
		}: {
			id: string;
			data: UpdateKomodoIntegrationRequest;
		}) => komodoApi.updateIntegration(id, data),
		onSuccess: (_, variables) => {
			queryClient.invalidateQueries({ queryKey: komodoKeys.integrations() });
			queryClient.invalidateQueries({
				queryKey: komodoKeys.integration(variables.id),
			});
		},
	});
}

export function useDeleteKomodoIntegration() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (id: string) => komodoApi.deleteIntegration(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: komodoKeys.integrations() });
			queryClient.invalidateQueries({ queryKey: komodoKeys.containers() });
			queryClient.invalidateQueries({ queryKey: komodoKeys.stacks() });
		},
	});
}

export function useTestKomodoConnection() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (id: string) => komodoApi.testConnection(id),
		onSuccess: (_, id) => {
			queryClient.invalidateQueries({ queryKey: komodoKeys.integration(id) });
			queryClient.invalidateQueries({ queryKey: komodoKeys.integrations() });
		},
	});
}

export function useSyncKomodoIntegration() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (id: string) => komodoApi.syncIntegration(id),
		onSuccess: (_, id) => {
			queryClient.invalidateQueries({ queryKey: komodoKeys.integration(id) });
			queryClient.invalidateQueries({ queryKey: komodoKeys.integrations() });
			queryClient.invalidateQueries({ queryKey: komodoKeys.containers() });
			queryClient.invalidateQueries({ queryKey: komodoKeys.stacks() });
		},
	});
}

export function useDiscoverKomodoContainers(id: string) {
	return useQuery({
		queryKey: komodoKeys.discovery(id),
		queryFn: () => komodoApi.discoverContainers(id),
		enabled: false, // Only fetch when explicitly requested
		staleTime: 0,
	});
}

// Container Hooks
export function useKomodoContainers() {
	return useQuery({
		queryKey: komodoKeys.containers(),
		queryFn: komodoApi.listContainers,
		staleTime: 30 * 1000,
	});
}

export function useKomodoContainer(id: string) {
	return useQuery({
		queryKey: komodoKeys.container(id),
		queryFn: () => komodoApi.getContainer(id),
		enabled: !!id,
		staleTime: 30 * 1000,
	});
}

export function useUpdateKomodoContainer() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: ({
			id,
			data,
		}: { id: string; data: UpdateKomodoContainerRequest }) =>
			komodoApi.updateContainer(id, data),
		}: {
			id: string;
			data: UpdateKomodoContainerRequest;
		}) => komodoApi.updateContainer(id, data),
		onSuccess: (_, variables) => {
			queryClient.invalidateQueries({ queryKey: komodoKeys.containers() });
			queryClient.invalidateQueries({
				queryKey: komodoKeys.container(variables.id),
			});
		},
	});
}

// Stack Hooks
export function useKomodoStacks() {
	return useQuery({
		queryKey: komodoKeys.stacks(),
		queryFn: komodoApi.listStacks,
		staleTime: 30 * 1000,
	});
}

export function useKomodoStack(id: string) {
	return useQuery({
		queryKey: komodoKeys.stack(id),
		queryFn: () => komodoApi.getStack(id),
		enabled: !!id,
		staleTime: 30 * 1000,
	});
}

// Webhook Events Hook
export function useKomodoWebhookEvents() {
	return useQuery({
		queryKey: komodoKeys.events(),
		queryFn: komodoApi.listWebhookEvents,
		staleTime: 10 * 1000,
	});
}
