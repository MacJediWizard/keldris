import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { costAlertsApi, costsApi, pricingApi } from '../lib/api';
import type {
	CreateCostAlertRequest,
	CreateStoragePricingRequest,
	UpdateCostAlertRequest,
	UpdateStoragePricingRequest,
} from '../lib/types';

// Cost Summary hooks
export function useCostSummary() {
	return useQuery({
		queryKey: ['costs', 'summary'],
		queryFn: costsApi.getSummary,
		staleTime: 60 * 1000, // 1 minute
	});
}

export function useRepositoryCosts() {
	return useQuery({
		queryKey: ['costs', 'repositories'],
		queryFn: costsApi.listRepositoryCosts,
		staleTime: 60 * 1000,
	});
}

export function useRepositoryCost(id: string) {
	return useQuery({
		queryKey: ['costs', 'repositories', id],
		queryFn: () => costsApi.getRepositoryCost(id),
		enabled: !!id,
		staleTime: 60 * 1000,
	});
}

export function useCostForecast(days = 30) {
	return useQuery({
		queryKey: ['costs', 'forecast', days],
		queryFn: () => costsApi.getForecast(days),
		staleTime: 60 * 1000,
	});
}

export function useCostHistory(days = 30) {
	return useQuery({
		queryKey: ['costs', 'history', days],
		queryFn: () => costsApi.getHistory(days),
		staleTime: 60 * 1000,
	});
}

// Pricing hooks
export function usePricing() {
	return useQuery({
		queryKey: ['pricing'],
		queryFn: pricingApi.list,
		staleTime: 5 * 60 * 1000, // 5 minutes
	});
}

export function useDefaultPricing() {
	return useQuery({
		queryKey: ['pricing', 'defaults'],
		queryFn: pricingApi.getDefaults,
		staleTime: 10 * 60 * 1000, // 10 minutes
	});
}

export function useCreatePricing() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateStoragePricingRequest) => pricingApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['pricing'] });
			queryClient.invalidateQueries({ queryKey: ['costs'] });
		},
	});
}

export function useUpdatePricing() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data: UpdateStoragePricingRequest;
		}) => pricingApi.update(id, data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['pricing'] });
			queryClient.invalidateQueries({ queryKey: ['costs'] });
		},
	});
}

export function useDeletePricing() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => pricingApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['pricing'] });
			queryClient.invalidateQueries({ queryKey: ['costs'] });
		},
	});
}

// Cost Alerts hooks
export function useCostAlerts() {
	return useQuery({
		queryKey: ['cost-alerts'],
		queryFn: costAlertsApi.list,
		staleTime: 60 * 1000,
	});
}

export function useCostAlert(id: string) {
	return useQuery({
		queryKey: ['cost-alerts', id],
		queryFn: () => costAlertsApi.get(id),
		enabled: !!id,
		staleTime: 60 * 1000,
	});
}

export function useCreateCostAlert() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateCostAlertRequest) => costAlertsApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['cost-alerts'] });
		},
	});
}

export function useUpdateCostAlert() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ id, data }: { id: string; data: UpdateCostAlertRequest }) =>
			costAlertsApi.update(id, data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['cost-alerts'] });
		},
	});
}

export function useDeleteCostAlert() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => costAlertsApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['cost-alerts'] });
		},
	});
}
