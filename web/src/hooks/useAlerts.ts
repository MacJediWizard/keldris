import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { alertRulesApi, alertsApi } from '../lib/api';
import type {
	CreateAlertRuleRequest,
	UpdateAlertRuleRequest,
} from '../lib/types';

// Alert hooks

export function useAlerts() {
	return useQuery({
		queryKey: ['alerts'],
		queryFn: alertsApi.list,
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useActiveAlerts() {
	return useQuery({
		queryKey: ['alerts', 'active'],
		queryFn: alertsApi.listActive,
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useAlertCount() {
	return useQuery({
		queryKey: ['alerts', 'count'],
		queryFn: alertsApi.count,
		staleTime: 10 * 1000, // 10 seconds - refresh more frequently for badge
		refetchInterval: 60 * 1000, // Refetch every minute
	});
}

export function useAlert(id: string) {
	return useQuery({
		queryKey: ['alerts', id],
		queryFn: () => alertsApi.get(id),
		enabled: !!id,
	});
}

export function useAcknowledgeAlert() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => alertsApi.acknowledge(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['alerts'] });
		},
	});
}

export function useResolveAlert() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => alertsApi.resolve(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['alerts'] });
		},
	});
}

// Alert Rule hooks

export function useAlertRules() {
	return useQuery({
		queryKey: ['alertRules'],
		queryFn: alertRulesApi.list,
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useAlertRule(id: string) {
	return useQuery({
		queryKey: ['alertRules', id],
		queryFn: () => alertRulesApi.get(id),
		enabled: !!id,
	});
}

export function useCreateAlertRule() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateAlertRuleRequest) => alertRulesApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['alertRules'] });
		},
	});
}

export function useUpdateAlertRule() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ id, data }: { id: string; data: UpdateAlertRuleRequest }) =>
			alertRulesApi.update(id, data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['alertRules'] });
		},
	});
}

export function useDeleteAlertRule() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => alertRulesApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['alertRules'] });
		},
	});
}
