import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { webhooksApi } from '../lib/api';
import type {
	CreateWebhookEndpointRequest,
	TestWebhookRequest,
	UpdateWebhookEndpointRequest,
} from '../lib/types';

// Event types hook
export function useWebhookEventTypes() {
	return useQuery({
		queryKey: ['webhook-event-types'],
		queryFn: webhooksApi.listEventTypes,
		staleTime: 60 * 60 * 1000, // 1 hour - event types rarely change
	});
}

// Endpoints hooks
export function useWebhookEndpoints() {
	return useQuery({
		queryKey: ['webhook-endpoints'],
		queryFn: webhooksApi.listEndpoints,
		staleTime: 30 * 1000,
	});
}

export function useWebhookEndpoint(id: string) {
	return useQuery({
		queryKey: ['webhook-endpoints', id],
		queryFn: () => webhooksApi.getEndpoint(id),
		enabled: !!id,
	});
}

export function useCreateWebhookEndpoint() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateWebhookEndpointRequest) =>
			webhooksApi.createEndpoint(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['webhook-endpoints'] });
		},
	});
}

export function useUpdateWebhookEndpoint() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data: UpdateWebhookEndpointRequest;
		}) => webhooksApi.updateEndpoint(id, data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['webhook-endpoints'] });
		},
	});
}

export function useDeleteWebhookEndpoint() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => webhooksApi.deleteEndpoint(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['webhook-endpoints'] });
		},
	});
}

export function useTestWebhookEndpoint() {
	return useMutation({
		mutationFn: ({ id, data }: { id: string; data?: TestWebhookRequest }) =>
			webhooksApi.testEndpoint(id, data),
	});
}

// Deliveries hooks
export function useWebhookDeliveries(limit = 50, offset = 0) {
	return useQuery({
		queryKey: ['webhook-deliveries', { limit, offset }],
		queryFn: () => webhooksApi.listDeliveries(limit, offset),
		staleTime: 10 * 1000,
	});
}

export function useWebhookEndpointDeliveries(
	endpointId: string,
	limit = 50,
	offset = 0,
) {
	return useQuery({
		queryKey: ['webhook-deliveries', endpointId, { limit, offset }],
		queryFn: () =>
			webhooksApi.listEndpointDeliveries(endpointId, limit, offset),
		enabled: !!endpointId,
		staleTime: 10 * 1000,
	});
}

export function useWebhookDelivery(id: string) {
	return useQuery({
		queryKey: ['webhook-deliveries', id],
		queryFn: () => webhooksApi.getDelivery(id),
		enabled: !!id,
	});
}

export function useRetryWebhookDelivery() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => webhooksApi.retryDelivery(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['webhook-deliveries'] });
		},
	});
}
