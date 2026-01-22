import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { excludePatternsApi } from '../lib/api';
import type {
	BuiltInPattern,
	CategoryInfo,
	CreateExcludePatternRequest,
	UpdateExcludePatternRequest,
} from '../lib/types';

export function useExcludePatterns(category?: string) {
	return useQuery({
		queryKey: ['exclude-patterns', { category }],
		queryFn: () => excludePatternsApi.list(category),
		staleTime: 30 * 1000,
	});
}

export function useExcludePattern(id: string) {
	return useQuery({
		queryKey: ['exclude-patterns', id],
		queryFn: () => excludePatternsApi.get(id),
		enabled: !!id,
	});
}

export function useExcludePatternsLibrary() {
	return useQuery<BuiltInPattern[]>({
		queryKey: ['exclude-patterns-library'],
		queryFn: () => excludePatternsApi.getLibrary(),
		staleTime: 60 * 60 * 1000, // Cache for 1 hour since library is static
	});
}

export function useExcludePatternCategories() {
	return useQuery<CategoryInfo[]>({
		queryKey: ['exclude-patterns-categories'],
		queryFn: () => excludePatternsApi.getCategories(),
		staleTime: 60 * 60 * 1000, // Cache for 1 hour since categories are static
	});
}

export function useCreateExcludePattern() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateExcludePatternRequest) =>
			excludePatternsApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['exclude-patterns'] });
		},
	});
}

export function useUpdateExcludePattern() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data: UpdateExcludePatternRequest;
		}) => excludePatternsApi.update(id, data),
		onSuccess: (_data, variables) => {
			queryClient.invalidateQueries({ queryKey: ['exclude-patterns'] });
			queryClient.invalidateQueries({
				queryKey: ['exclude-patterns', variables.id],
			});
		},
	});
}

export function useDeleteExcludePattern() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => excludePatternsApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['exclude-patterns'] });
		},
	});
}
