import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { slaApi } from '../lib/api';
import type {
	AcknowledgeBreachRequest,
	AssignSLARequest,
	CreateSLADefinitionRequest,
	UpdateSLADefinitionRequest,
} from '../lib/types';

// SLA Definitions
export function useSLAs() {
	return useQuery({
		queryKey: ['slas'],
		queryFn: () => slaApi.list(),
		staleTime: 30 * 1000,
	});
}

export function useSLA(id: string) {
	return useQuery({
		queryKey: ['slas', id],
		queryFn: () => slaApi.get(id),
		enabled: !!id,
	});
}

export function useCreateSLA() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateSLADefinitionRequest) => slaApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['slas'] });
		},
	});
}

export function useUpdateSLA() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: { id: string; data: UpdateSLADefinitionRequest }) =>
			slaApi.update(id, data),
		onSuccess: (_, { id }) => {
			queryClient.invalidateQueries({ queryKey: ['slas'] });
			queryClient.invalidateQueries({ queryKey: ['slas', id] });
		},
	});
}

export function useDeleteSLA() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => slaApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['slas'] });
		},
	});
}

// SLA Assignments
export function useSLAAssignments(slaId: string) {
	return useQuery({
		queryKey: ['slas', slaId, 'assignments'],
		queryFn: () => slaApi.listAssignments(slaId),
		enabled: !!slaId,
	});
}

export function useCreateSLAAssignment() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ slaId, data }: { slaId: string; data: AssignSLARequest }) =>
			slaApi.createAssignment(slaId, data),
		onSuccess: (_, { slaId }) => {
			queryClient.invalidateQueries({ queryKey: ['slas'] });
			queryClient.invalidateQueries({ queryKey: ['slas', slaId, 'assignments'] });
		},
	});
}

export function useDeleteSLAAssignment() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			slaId,
			assignmentId,
		}: { slaId: string; assignmentId: string }) =>
			slaApi.deleteAssignment(slaId, assignmentId),
		onSuccess: (_, { slaId }) => {
			queryClient.invalidateQueries({ queryKey: ['slas'] });
			queryClient.invalidateQueries({ queryKey: ['slas', slaId, 'assignments'] });
		},
	});
}

// SLA Compliance
export function useSLACompliance(slaId: string) {
	return useQuery({
		queryKey: ['slas', slaId, 'compliance'],
		queryFn: () => slaApi.getCompliance(slaId),
		enabled: !!slaId,
	});
}

export function useOrgSLACompliance() {
	return useQuery({
		queryKey: ['sla-compliance'],
		queryFn: () => slaApi.listOrgCompliance(),
		staleTime: 60 * 1000,
	});
}

// SLA Breaches
export function useSLABreaches() {
	return useQuery({
		queryKey: ['sla-breaches'],
		queryFn: () => slaApi.listBreaches(),
		staleTime: 30 * 1000,
	});
}

export function useActiveSLABreaches() {
	return useQuery({
		queryKey: ['sla-breaches', 'active'],
		queryFn: () => slaApi.listActiveBreaches(),
		staleTime: 30 * 1000,
		refetchInterval: 60 * 1000, // Refresh every minute
	});
}

export function useSLABreachesBySLA(slaId: string) {
	return useQuery({
		queryKey: ['slas', slaId, 'breaches'],
		queryFn: () => slaApi.listBreachesBySLA(slaId),
		enabled: !!slaId,
	});
}

export function useSLABreach(id: string) {
	return useQuery({
		queryKey: ['sla-breaches', id],
		queryFn: () => slaApi.getBreach(id),
		enabled: !!id,
	});
}

export function useAcknowledgeBreach() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: { id: string; data?: AcknowledgeBreachRequest }) =>
			slaApi.acknowledgeBreach(id, data),
		onSuccess: (_, { id }) => {
			queryClient.invalidateQueries({ queryKey: ['sla-breaches'] });
			queryClient.invalidateQueries({ queryKey: ['sla-breaches', id] });
			queryClient.invalidateQueries({ queryKey: ['sla-dashboard'] });
		},
	});
}

export function useResolveBreach() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => slaApi.resolveBreach(id),
		onSuccess: (_, id) => {
			queryClient.invalidateQueries({ queryKey: ['sla-breaches'] });
			queryClient.invalidateQueries({ queryKey: ['sla-breaches', id] });
			queryClient.invalidateQueries({ queryKey: ['sla-dashboard'] });
		},
	});
}

// SLA Dashboard
export function useSLADashboard() {
	return useQuery({
		queryKey: ['sla-dashboard'],
		queryFn: () => slaApi.getDashboard(),
		staleTime: 30 * 1000,
		refetchInterval: 60 * 1000, // Refresh every minute
	});
}

// SLA Report
export function useSLAReport(month?: string) {
	return useQuery({
		queryKey: ['sla-report', month],
		queryFn: () => slaApi.getReport(month),
		staleTime: 5 * 60 * 1000, // 5 minutes
	});
}
