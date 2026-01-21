import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { reportsApi } from '../lib/api';
import type {
	CreateReportScheduleRequest,
	ReportFrequency,
	UpdateReportScheduleRequest,
} from '../lib/types';

export function useReportSchedules() {
	return useQuery({
		queryKey: ['report-schedules'],
		queryFn: reportsApi.listSchedules,
		staleTime: 30 * 1000,
	});
}

export function useReportSchedule(id: string) {
	return useQuery({
		queryKey: ['report-schedules', id],
		queryFn: () => reportsApi.getSchedule(id),
		enabled: !!id,
	});
}

export function useCreateReportSchedule() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateReportScheduleRequest) =>
			reportsApi.createSchedule(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['report-schedules'] });
		},
	});
}

export function useUpdateReportSchedule() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data: UpdateReportScheduleRequest;
		}) => reportsApi.updateSchedule(id, data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['report-schedules'] });
		},
	});
}

export function useDeleteReportSchedule() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => reportsApi.deleteSchedule(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['report-schedules'] });
		},
	});
}

export function useSendReport() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ id, preview = false }: { id: string; preview?: boolean }) =>
			reportsApi.sendReport(id, preview),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['report-history'] });
		},
	});
}

export function usePreviewReport() {
	return useMutation({
		mutationFn: ({
			frequency,
			timezone = 'UTC',
		}: {
			frequency: ReportFrequency;
			timezone?: string;
		}) => reportsApi.previewReport(frequency, timezone),
	});
}

export function useReportHistory() {
	return useQuery({
		queryKey: ['report-history'],
		queryFn: reportsApi.listHistory,
		staleTime: 30 * 1000,
	});
}

export function useReportHistoryEntry(id: string) {
	return useQuery({
		queryKey: ['report-history', id],
		queryFn: () => reportsApi.getHistory(id),
		enabled: !!id,
	});
}
