import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { downtimeApi, uptimeApi, downtimeAlertsApi } from '../lib/api';
import type {
	CreateDowntimeEventRequest,
	UpdateDowntimeEventRequest,
	ResolveDowntimeEventRequest,
	CreateDowntimeAlertRequest,
	UpdateDowntimeAlertRequest,
} from '../lib/types';

// Downtime Event hooks

export function useDowntimeEvents(limit = 100, offset = 0) {
	return useQuery({
		queryKey: ['downtime', 'events', limit, offset],
		queryFn: () => downtimeApi.list(limit, offset),
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useActiveDowntime() {
	return useQuery({
		queryKey: ['downtime', 'active'],
		queryFn: downtimeApi.listActive,
		staleTime: 30 * 1000, // 30 seconds
		refetchInterval: 60 * 1000, // Refetch every minute
	});
}

export function useUptimeSummary() {
	return useQuery({
		queryKey: ['downtime', 'summary'],
		queryFn: downtimeApi.getSummary,
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useDowntimeEvent(id: string) {
	return useQuery({
		queryKey: ['downtime', 'event', id],
		queryFn: () => downtimeApi.get(id),
		enabled: !!id,
	});
}

export function useCreateDowntimeEvent() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateDowntimeEventRequest) => downtimeApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['downtime'] });
		},
	});
}

export function useUpdateDowntimeEvent() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data: UpdateDowntimeEventRequest;
		}) => downtimeApi.update(id, data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['downtime'] });
		},
	});
}

export function useResolveDowntimeEvent() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data?: ResolveDowntimeEventRequest;
		}) => downtimeApi.resolve(id, data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['downtime'] });
		},
	});
}

export function useDeleteDowntimeEvent() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => downtimeApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['downtime'] });
		},
	});
}

// Uptime Badge hooks

export function useUptimeBadges() {
	return useQuery({
		queryKey: ['uptime', 'badges'],
		queryFn: uptimeApi.getBadges,
		staleTime: 5 * 60 * 1000, // 5 minutes
	});
}

export function useRefreshUptimeBadges() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: uptimeApi.refreshBadges,
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['uptime', 'badges'] });
			queryClient.invalidateQueries({ queryKey: ['downtime', 'summary'] });
		},
	});
}

export function useMonthlyUptimeReport(year: number, month: number) {
	return useQuery({
		queryKey: ['uptime', 'report', year, month],
		queryFn: () => uptimeApi.getMonthlyReport(year, month),
		enabled: year > 0 && month > 0 && month <= 12,
		staleTime: 5 * 60 * 1000, // 5 minutes
	});
}

// Downtime Alert hooks

export function useDowntimeAlerts() {
	return useQuery({
		queryKey: ['downtime', 'alerts'],
		queryFn: downtimeAlertsApi.list,
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useDowntimeAlert(id: string) {
	return useQuery({
		queryKey: ['downtime', 'alert', id],
		queryFn: () => downtimeAlertsApi.get(id),
		enabled: !!id,
	});
}

export function useCreateDowntimeAlert() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateDowntimeAlertRequest) =>
			downtimeAlertsApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['downtime', 'alerts'] });
		},
	});
}

export function useUpdateDowntimeAlert() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data: UpdateDowntimeAlertRequest;
		}) => downtimeAlertsApi.update(id, data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['downtime', 'alerts'] });
		},
	});
}

export function useDeleteDowntimeAlert() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => downtimeAlertsApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['downtime', 'alerts'] });
		},
	});
}
