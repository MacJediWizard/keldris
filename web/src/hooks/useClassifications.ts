import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { classificationsApi } from '../lib/api';
import type {
	CreatePathClassificationRuleRequest,
	SetScheduleClassificationRequest,
	UpdatePathClassificationRuleRequest,
} from '../lib/types';

// Classification levels reference
export function useClassificationLevels() {
	return useQuery({
		queryKey: ['classification-levels'],
		queryFn: () => classificationsApi.getLevels(),
		staleTime: 60 * 60 * 1000, // 1 hour (reference data rarely changes)
	});
}

// Data types reference
export function useClassificationDataTypes() {
	return useQuery({
		queryKey: ['classification-data-types'],
		queryFn: () => classificationsApi.getDataTypes(),
		staleTime: 60 * 60 * 1000, // 1 hour
	});
}

// Default rules
export function useDefaultClassificationRules() {
	return useQuery({
		queryKey: ['classification-default-rules'],
		queryFn: () => classificationsApi.getDefaultRules(),
		staleTime: 60 * 60 * 1000, // 1 hour
	});
}

// Classification rules
export function useClassificationRules() {
	return useQuery({
		queryKey: ['classification-rules'],
		queryFn: () => classificationsApi.listRules(),
		staleTime: 60 * 1000, // 1 minute
	});
}

export function useClassificationRule(id: string) {
	return useQuery({
		queryKey: ['classification-rules', id],
		queryFn: () => classificationsApi.getRule(id),
		enabled: !!id,
	});
}

export function useCreateClassificationRule() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreatePathClassificationRuleRequest) =>
			classificationsApi.createRule(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['classification-rules'] });
		},
	});
}

export function useUpdateClassificationRule() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data: UpdatePathClassificationRuleRequest;
		}) => classificationsApi.updateRule(id, data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['classification-rules'] });
		},
	});
}

export function useDeleteClassificationRule() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => classificationsApi.deleteRule(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['classification-rules'] });
		},
	});
}

// Schedule classifications
export function useScheduleClassifications(level?: string) {
	return useQuery({
		queryKey: ['schedule-classifications', level],
		queryFn: () => classificationsApi.listScheduleClassifications(level),
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useScheduleClassification(scheduleId: string) {
	return useQuery({
		queryKey: ['schedule-classifications', scheduleId],
		queryFn: () => classificationsApi.getScheduleClassification(scheduleId),
		enabled: !!scheduleId,
	});
}

export function useSetScheduleClassification() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			scheduleId,
			data,
		}: {
			scheduleId: string;
			data: SetScheduleClassificationRequest;
		}) => classificationsApi.setScheduleClassification(scheduleId, data),
		onSuccess: (_data, variables) => {
			queryClient.invalidateQueries({ queryKey: ['schedule-classifications'] });
			queryClient.invalidateQueries({ queryKey: ['schedules'] });
			queryClient.invalidateQueries({
				queryKey: ['schedule-classifications', variables.scheduleId],
			});
		},
	});
}

export function useAutoClassifySchedule() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (scheduleId: string) =>
			classificationsApi.autoClassifySchedule(scheduleId),
		onSuccess: (_data, scheduleId) => {
			queryClient.invalidateQueries({ queryKey: ['schedule-classifications'] });
			queryClient.invalidateQueries({ queryKey: ['schedules'] });
			queryClient.invalidateQueries({
				queryKey: ['schedule-classifications', scheduleId],
			});
		},
	});
}

// Backup classifications
export function useBackupsByClassification(level: string) {
	return useQuery({
		queryKey: ['backups-by-classification', level],
		queryFn: () => classificationsApi.listBackupsByClassification(level),
		enabled: !!level,
		staleTime: 30 * 1000, // 30 seconds
	});
}

// Summary and reports
export function useClassificationSummary() {
	return useQuery({
		queryKey: ['classification-summary'],
		queryFn: () => classificationsApi.getSummary(),
		staleTime: 60 * 1000, // 1 minute
	});
}

export function useComplianceReport() {
	return useQuery({
		queryKey: ['compliance-report'],
		queryFn: () => classificationsApi.getComplianceReport(),
		staleTime: 60 * 1000, // 1 minute
	});
}
