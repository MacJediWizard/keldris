import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { notificationsApi } from '../lib/api';
import type {
	CreateNotificationChannelRequest,
	CreateNotificationPreferenceRequest,
	UpdateNotificationChannelRequest,
	UpdateNotificationPreferenceRequest,
} from '../lib/types';

// Channels hooks
export function useNotificationChannels() {
	return useQuery({
		queryKey: ['notification-channels'],
		queryFn: notificationsApi.listChannels,
		staleTime: 30 * 1000,
	});
}

export function useNotificationChannel(id: string) {
	return useQuery({
		queryKey: ['notification-channels', id],
		queryFn: () => notificationsApi.getChannel(id),
		enabled: !!id,
	});
}

export function useCreateNotificationChannel() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateNotificationChannelRequest) =>
			notificationsApi.createChannel(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['notification-channels'] });
		},
	});
}

export function useUpdateNotificationChannel() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data: UpdateNotificationChannelRequest;
		}) => notificationsApi.updateChannel(id, data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['notification-channels'] });
		},
	});
}

export function useDeleteNotificationChannel() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => notificationsApi.deleteChannel(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['notification-channels'] });
		},
	});
}

// Preferences hooks
export function useNotificationPreferences() {
	return useQuery({
		queryKey: ['notification-preferences'],
		queryFn: notificationsApi.listPreferences,
		staleTime: 30 * 1000,
	});
}

export function useCreateNotificationPreference() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateNotificationPreferenceRequest) =>
			notificationsApi.createPreference(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['notification-preferences'] });
			queryClient.invalidateQueries({ queryKey: ['notification-channels'] });
		},
	});
}

export function useUpdateNotificationPreference() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data: UpdateNotificationPreferenceRequest;
		}) => notificationsApi.updatePreference(id, data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['notification-preferences'] });
			queryClient.invalidateQueries({ queryKey: ['notification-channels'] });
		},
	});
}

export function useDeleteNotificationPreference() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => notificationsApi.deletePreference(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['notification-preferences'] });
			queryClient.invalidateQueries({ queryKey: ['notification-channels'] });
		},
	});
}

// Logs hooks
export function useNotificationLogs() {
	return useQuery({
		queryKey: ['notification-logs'],
		queryFn: notificationsApi.listLogs,
		staleTime: 30 * 1000,
	});
}
