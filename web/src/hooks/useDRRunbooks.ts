import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { drRunbooksApi } from '../lib/api';
import type {
	CreateDRRunbookRequest,
	CreateDRTestScheduleRequest,
	UpdateDRRunbookRequest,
} from '../lib/types';

export function useDRRunbooks() {
	return useQuery({
		queryKey: ['dr-runbooks'],
		queryFn: () => drRunbooksApi.list(),
		staleTime: 30 * 1000,
	});
}

export function useDRRunbook(id: string) {
	return useQuery({
		queryKey: ['dr-runbooks', id],
		queryFn: () => drRunbooksApi.get(id),
		enabled: !!id,
	});
}

export function useDRStatus() {
	return useQuery({
		queryKey: ['dr-status'],
		queryFn: async () => {
			try {
				return await drRunbooksApi.getStatus();
			} catch {
				return {
					active_runbooks: 0,
					total_runbooks: 0,
					tested_runbooks: 0,
					untested_runbooks: 0,
					overdue_runbooks: 0,
					tests_last_30_days: 0,
					pass_rate: 0,
					last_test_at: null,
					next_test_at: null,
					upcoming_tests: [],
				};
			}
		},
		staleTime: 30 * 1000,
	});
}

export function useCreateDRRunbook() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateDRRunbookRequest) => drRunbooksApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['dr-runbooks'] });
			queryClient.invalidateQueries({ queryKey: ['dr-status'] });
		},
	});
}

export function useUpdateDRRunbook() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ id, data }: { id: string; data: UpdateDRRunbookRequest }) =>
			drRunbooksApi.update(id, data),
		onSuccess: (_, { id }) => {
			queryClient.invalidateQueries({ queryKey: ['dr-runbooks'] });
			queryClient.invalidateQueries({ queryKey: ['dr-runbooks', id] });
		},
	});
}

export function useDeleteDRRunbook() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => drRunbooksApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['dr-runbooks'] });
			queryClient.invalidateQueries({ queryKey: ['dr-status'] });
		},
	});
}

export function useActivateDRRunbook() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => drRunbooksApi.activate(id),
		onSuccess: (_, id) => {
			queryClient.invalidateQueries({ queryKey: ['dr-runbooks'] });
			queryClient.invalidateQueries({ queryKey: ['dr-runbooks', id] });
			queryClient.invalidateQueries({ queryKey: ['dr-status'] });
		},
	});
}

export function useArchiveDRRunbook() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => drRunbooksApi.archive(id),
		onSuccess: (_, id) => {
			queryClient.invalidateQueries({ queryKey: ['dr-runbooks'] });
			queryClient.invalidateQueries({ queryKey: ['dr-runbooks', id] });
			queryClient.invalidateQueries({ queryKey: ['dr-status'] });
		},
	});
}

export function useRenderDRRunbook(id: string) {
	return useQuery({
		queryKey: ['dr-runbooks', id, 'render'],
		queryFn: () => drRunbooksApi.render(id),
		enabled: !!id,
	});
}

export function useGenerateDRRunbook() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (scheduleId: string) =>
			drRunbooksApi.generateFromSchedule(scheduleId),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['dr-runbooks'] });
			queryClient.invalidateQueries({ queryKey: ['dr-status'] });
		},
	});
}

export function useDRTestSchedules(runbookId: string) {
	return useQuery({
		queryKey: ['dr-runbooks', runbookId, 'test-schedules'],
		queryFn: () => drRunbooksApi.listTestSchedules(runbookId),
		enabled: !!runbookId,
	});
}

export function useCreateDRTestSchedule() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			runbookId,
			data,
		}: {
			runbookId: string;
			data: CreateDRTestScheduleRequest;
		}) => drRunbooksApi.createTestSchedule(runbookId, data),
		onSuccess: (_, { runbookId }) => {
			queryClient.invalidateQueries({
				queryKey: ['dr-runbooks', runbookId, 'test-schedules'],
			});
			queryClient.invalidateQueries({ queryKey: ['dr-status'] });
		},
	});
}
