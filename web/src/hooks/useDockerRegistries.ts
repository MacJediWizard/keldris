import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { dockerRegistriesApi } from '../lib/api';
import type {
	CreateDockerRegistryRequest,
	RotateCredentialsRequest,
	UpdateDockerRegistryRequest,
} from '../lib/types';

export function useDockerRegistries() {
	return useQuery({
		queryKey: ['docker-registries'],
		queryFn: () => dockerRegistriesApi.list(),
		staleTime: 30 * 1000,
	});
}

export function useDockerRegistry(id: string) {
	return useQuery({
		queryKey: ['docker-registries', id],
		queryFn: () => dockerRegistriesApi.get(id),
		enabled: !!id,
	});
}

export function useDockerRegistryTypes() {
	return useQuery({
		queryKey: ['docker-registry-types'],
		queryFn: () => dockerRegistriesApi.getTypes(),
		staleTime: 60 * 60 * 1000, // 1 hour - types rarely change
	});
}

export function useExpiringCredentials() {
	return useQuery({
		queryKey: ['docker-registries-expiring'],
		queryFn: () => dockerRegistriesApi.getExpiringCredentials(),
		staleTime: 5 * 60 * 1000, // 5 minutes
	});
}

export function useCreateDockerRegistry() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateDockerRegistryRequest) =>
			dockerRegistriesApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['docker-registries'] });
		},
	});
}

export function useUpdateDockerRegistry() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: { id: string; data: UpdateDockerRegistryRequest }) =>
			dockerRegistriesApi.update(id, data),
		onSuccess: (_, { id }) => {
			queryClient.invalidateQueries({ queryKey: ['docker-registries'] });
			queryClient.invalidateQueries({ queryKey: ['docker-registries', id] });
		},
	});
}

export function useDeleteDockerRegistry() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => dockerRegistriesApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['docker-registries'] });
		},
	});
}

export function useLoginDockerRegistry() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => dockerRegistriesApi.login(id),
		onSuccess: (_, id) => {
			queryClient.invalidateQueries({ queryKey: ['docker-registries', id] });
		},
	});
}

export function useLoginAllDockerRegistries() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: () => dockerRegistriesApi.loginAll(),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['docker-registries'] });
		},
	});
}

export function useHealthCheckDockerRegistry() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => dockerRegistriesApi.healthCheck(id),
		onSuccess: (_, id) => {
			queryClient.invalidateQueries({ queryKey: ['docker-registries', id] });
			queryClient.invalidateQueries({ queryKey: ['docker-registries'] });
		},
	});
}

export function useHealthCheckAllDockerRegistries() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: () => dockerRegistriesApi.healthCheckAll(),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['docker-registries'] });
		},
	});
}

export function useRotateDockerRegistryCredentials() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			id,
			data,
		}: { id: string; data: RotateCredentialsRequest }) =>
			dockerRegistriesApi.rotateCredentials(id, data),
		onSuccess: (_, { id }) => {
			queryClient.invalidateQueries({ queryKey: ['docker-registries'] });
			queryClient.invalidateQueries({ queryKey: ['docker-registries', id] });
			queryClient.invalidateQueries({ queryKey: ['docker-registries-expiring'] });
		},
	});
}

export function useSetDefaultDockerRegistry() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => dockerRegistriesApi.setDefault(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['docker-registries'] });
		},
	});
}
