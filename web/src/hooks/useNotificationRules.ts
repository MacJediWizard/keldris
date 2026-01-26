import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { notificationRulesApi } from '../lib/api';
import type {
	CreateNotificationRuleRequest,
	TestNotificationRuleRequest,
	UpdateNotificationRuleRequest,
} from '../lib/types';

// Rules hooks
export function useNotificationRules() {
	return useQuery({
		queryKey: ['notification-rules'],
		queryFn: notificationRulesApi.list,
		staleTime: 30 * 1000,
	});
}

export function useNotificationRule(id: string) {
	return useQuery({
		queryKey: ['notification-rules', id],
		queryFn: () => notificationRulesApi.get(id),
		enabled: !!id,
	});
}

export function useCreateNotificationRule() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateNotificationRuleRequest) =>
			notificationRulesApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['notification-rules'] });
		},
	});
}

export function useUpdateNotificationRule() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data: UpdateNotificationRuleRequest;
		}) => notificationRulesApi.update(id, data),
		onSuccess: (_, variables) => {
			queryClient.invalidateQueries({ queryKey: ['notification-rules'] });
			queryClient.invalidateQueries({
				queryKey: ['notification-rules', variables.id],
			});
		},
	});
}

export function useDeleteNotificationRule() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => notificationRulesApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['notification-rules'] });
		},
	});
}

export function useTestNotificationRule() {
	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data?: TestNotificationRuleRequest;
		}) => notificationRulesApi.test(id, data),
	});
}

// Rule events hooks
export function useNotificationRuleEvents(ruleId: string) {
	return useQuery({
		queryKey: ['notification-rules', ruleId, 'events'],
		queryFn: () => notificationRulesApi.listEvents(ruleId),
		enabled: !!ruleId,
		staleTime: 30 * 1000,
	});
}

// Rule executions hooks
export function useNotificationRuleExecutions(ruleId: string) {
	return useQuery({
		queryKey: ['notification-rules', ruleId, 'executions'],
		queryFn: () => notificationRulesApi.listExecutions(ruleId),
		enabled: !!ruleId,
		staleTime: 30 * 1000,
	});
}
