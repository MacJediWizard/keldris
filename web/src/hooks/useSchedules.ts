import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { schedulesApi } from '../lib/api';
import type {
	CreateScheduleRequest,
	UpdateScheduleRequest,
} from '../lib/types';

export function useSchedules(agentId?: string) {
	return useQuery({
		queryKey: ['schedules', { agentId }],
		queryFn: () => schedulesApi.list(agentId),
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useSchedule(id: string) {
	return useQuery({
		queryKey: ['schedules', id],
		queryFn: () => schedulesApi.get(id),
		enabled: !!id,
	});
}

export function useCreateSchedule() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateScheduleRequest) => schedulesApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['schedules'] });
		},
	});
}

export function useUpdateSchedule() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ id, data }: { id: string; data: UpdateScheduleRequest }) =>
			schedulesApi.update(id, data),
		onSuccess: (_, { id }) => {
			queryClient.invalidateQueries({ queryKey: ['schedules'] });
			queryClient.invalidateQueries({ queryKey: ['schedules', id] });
		},
	});
}

export function useDeleteSchedule() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => schedulesApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['schedules'] });
		},
	});
}

export function useRunSchedule() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => schedulesApi.run(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['backups'] });
		},
	});
}
