import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { repositoriesApi } from '../lib/api';
import type {
	CloneRepositoryRequest,
	CreateRepositoryRequest,
	TestConnectionRequest,
	CreateRepositoryRequest,
	UpdateRepositoryRequest,
} from '../lib/types';

export function useRepositories() {
	return useQuery({
		queryKey: ['repositories'],
		queryFn: repositoriesApi.list,
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useRepository(id: string) {
	return useQuery({
		queryKey: ['repositories', id],
		queryFn: () => repositoriesApi.get(id),
		enabled: !!id,
	});
}

export function useCreateRepository() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateRepositoryRequest) => repositoriesApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['repositories'] });
		},
	});
}

export function useUpdateRepository() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ id, data }: { id: string; data: UpdateRepositoryRequest }) =>
			repositoriesApi.update(id, data),
		onSuccess: (_, { id }) => {
			queryClient.invalidateQueries({ queryKey: ['repositories'] });
			queryClient.invalidateQueries({ queryKey: ['repositories', id] });
		},
	});
}

export function useDeleteRepository() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => repositoriesApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['repositories'] });
		},
	});
}

export function useTestRepository() {
	return useMutation({
		mutationFn: (id: string) => repositoriesApi.test(id),
	});
}

export function useTestConnection() {
	return useMutation({
		mutationFn: (data: TestConnectionRequest) =>
			repositoriesApi.testConnection(data),
	});
}

export function useRecoverRepositoryKey() {
	return useMutation({
		mutationFn: (id: string) => repositoriesApi.recoverKey(id),
	});
}

export function useCloneRepository() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ id, data }: { id: string; data: CloneRepositoryRequest }) =>
			repositoriesApi.clone(id, data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['repositories'] });
		},
	});
}
