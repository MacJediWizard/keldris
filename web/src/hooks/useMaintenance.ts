import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { maintenanceApi } from '../lib/api';
import type {
	CreateMaintenanceWindowRequest,
	UpdateMaintenanceWindowRequest,
} from '../lib/types';

export function useMaintenanceWindows() {
	return useQuery({
		queryKey: ['maintenance-windows'],
		queryFn: () => maintenanceApi.list(),
		staleTime: 30 * 1000,
	});
}

export function useMaintenanceWindow(id: string) {
	return useQuery({
		queryKey: ['maintenance-windows', id],
		queryFn: () => maintenanceApi.get(id),
		enabled: !!id,
	});
}

export function useActiveMaintenance() {
	return useQuery({
		queryKey: ['maintenance', 'active'],
		queryFn: () => maintenanceApi.getActive(),
		refetchInterval: 60 * 1000, // Refresh every minute for countdown updates
		staleTime: 30 * 1000,
	});
}

export function useCreateMaintenanceWindow() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateMaintenanceWindowRequest) =>
			maintenanceApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['maintenance-windows'] });
			queryClient.invalidateQueries({ queryKey: ['maintenance', 'active'] });
		},
	});
}

export function useUpdateMaintenanceWindow() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data: UpdateMaintenanceWindowRequest;
		}) => maintenanceApi.update(id, data),
		onSuccess: (_, { id }) => {
			queryClient.invalidateQueries({ queryKey: ['maintenance-windows'] });
			queryClient.invalidateQueries({ queryKey: ['maintenance-windows', id] });
			queryClient.invalidateQueries({ queryKey: ['maintenance', 'active'] });
		},
	});
}

export function useDeleteMaintenanceWindow() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => maintenanceApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['maintenance-windows'] });
			queryClient.invalidateQueries({ queryKey: ['maintenance', 'active'] });
		},
	});
}

export function useEmergencyOverride() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ id, override }: { id: string; override: boolean }) =>
			maintenanceApi.emergencyOverride(id, override),
		onSuccess: (_, { id }) => {
			queryClient.invalidateQueries({ queryKey: ['maintenance-windows'] });
			queryClient.invalidateQueries({ queryKey: ['maintenance-windows', id] });
			queryClient.invalidateQueries({ queryKey: ['maintenance', 'active'] });
		},
	});
}
