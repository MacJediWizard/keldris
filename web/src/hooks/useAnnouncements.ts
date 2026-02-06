import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { announcementsApi } from '../lib/api';
import type {
	CreateAnnouncementRequest,
	UpdateAnnouncementRequest,
} from '../lib/types';

export function useAnnouncements() {
	return useQuery({
		queryKey: ['announcements'],
		queryFn: () => announcementsApi.list(),
		staleTime: 30 * 1000,
	});
}

export function useAnnouncement(id: string) {
	return useQuery({
		queryKey: ['announcements', id],
		queryFn: () => announcementsApi.get(id),
		enabled: !!id,
	});
}

export function useActiveAnnouncements() {
	return useQuery({
		queryKey: ['announcements', 'active'],
		queryFn: () => announcementsApi.getActive(),
		refetchInterval: 60 * 1000, // Refresh every minute
		staleTime: 30 * 1000,
	});
}

export function useCreateAnnouncement() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateAnnouncementRequest) =>
			announcementsApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['announcements'] });
			queryClient.invalidateQueries({ queryKey: ['announcements', 'active'] });
		},
	});
}

export function useUpdateAnnouncement() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data: UpdateAnnouncementRequest;
		}) => announcementsApi.update(id, data),
		onSuccess: (_, { id }) => {
			queryClient.invalidateQueries({ queryKey: ['announcements'] });
			queryClient.invalidateQueries({ queryKey: ['announcements', id] });
			queryClient.invalidateQueries({ queryKey: ['announcements', 'active'] });
		},
	});
}

export function useDeleteAnnouncement() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => announcementsApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['announcements'] });
			queryClient.invalidateQueries({ queryKey: ['announcements', 'active'] });
		},
	});
}

export function useDismissAnnouncement() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => announcementsApi.dismiss(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['announcements', 'active'] });
		},
	});
}
