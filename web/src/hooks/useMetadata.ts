import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { metadataApi } from '../lib/api';
import type {
	CreateMetadataSchemaRequest,
	MetadataEntityType,
	MetadataSchema,
	UpdateEntityMetadataRequest,
	UpdateMetadataSchemaRequest,
} from '../lib/types';

// Query keys
export const metadataKeys = {
	all: ['metadata'] as const,
	schemas: (entityType: MetadataEntityType) =>
		[...metadataKeys.all, 'schemas', entityType] as const,
	schema: (id: string) => [...metadataKeys.all, 'schema', id] as const,
	fieldTypes: () => [...metadataKeys.all, 'fieldTypes'] as const,
	entityTypes: () => [...metadataKeys.all, 'entityTypes'] as const,
};

// Hooks for metadata schemas
export function useMetadataSchemas(entityType: MetadataEntityType) {
	return useQuery({
		queryKey: metadataKeys.schemas(entityType),
		queryFn: () => metadataApi.listSchemas(entityType),
	});
}

export function useMetadataSchema(id: string) {
	return useQuery({
		queryKey: metadataKeys.schema(id),
		queryFn: () => metadataApi.getSchema(id),
		enabled: !!id,
	});
}

export function useMetadataFieldTypes() {
	return useQuery({
		queryKey: metadataKeys.fieldTypes(),
		queryFn: () => metadataApi.getFieldTypes(),
		staleTime: 1000 * 60 * 60, // 1 hour - these don't change
	});
}

export function useMetadataEntityTypes() {
	return useQuery({
		queryKey: metadataKeys.entityTypes(),
		queryFn: () => metadataApi.getEntityTypes(),
		staleTime: 1000 * 60 * 60, // 1 hour - these don't change
	});
}

export function useCreateMetadataSchema() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (data: CreateMetadataSchemaRequest) =>
			metadataApi.createSchema(data),
		onSuccess: (schema: MetadataSchema) => {
			queryClient.invalidateQueries({
				queryKey: metadataKeys.schemas(schema.entity_type),
			});
		},
	});
}

export function useUpdateMetadataSchema() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data: UpdateMetadataSchemaRequest;
		}) => metadataApi.updateSchema(id, data),
		onSuccess: (schema: MetadataSchema) => {
			queryClient.invalidateQueries({
				queryKey: metadataKeys.schemas(schema.entity_type),
			});
			queryClient.invalidateQueries({
				queryKey: metadataKeys.schema(schema.id),
			});
		},
	});
}

export function useDeleteMetadataSchema() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: ({ id, entityType }: { id: string; entityType: MetadataEntityType }) =>
			metadataApi.deleteSchema(id),
		onSuccess: (_, { entityType }) => {
			queryClient.invalidateQueries({
				queryKey: metadataKeys.schemas(entityType),
			});
		},
	});
}

// Hooks for entity metadata
export function useUpdateAgentMetadata() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: ({
			agentId,
			data,
		}: {
			agentId: string;
			data: UpdateEntityMetadataRequest;
		}) => metadataApi.updateAgentMetadata(agentId, data),
		onSuccess: (_, { agentId }) => {
			queryClient.invalidateQueries({ queryKey: ['agents'] });
			queryClient.invalidateQueries({ queryKey: ['agent', agentId] });
		},
	});
}

export function useUpdateRepositoryMetadata() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: ({
			repositoryId,
			data,
		}: {
			repositoryId: string;
			data: UpdateEntityMetadataRequest;
		}) => metadataApi.updateRepositoryMetadata(repositoryId, data),
		onSuccess: (_, { repositoryId }) => {
			queryClient.invalidateQueries({ queryKey: ['repositories'] });
			queryClient.invalidateQueries({ queryKey: ['repository', repositoryId] });
		},
	});
}

export function useUpdateScheduleMetadata() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: ({
			scheduleId,
			data,
		}: {
			scheduleId: string;
			data: UpdateEntityMetadataRequest;
		}) => metadataApi.updateScheduleMetadata(scheduleId, data),
		onSuccess: (_, { scheduleId }) => {
			queryClient.invalidateQueries({ queryKey: ['schedules'] });
			queryClient.invalidateQueries({ queryKey: ['schedule', scheduleId] });
		},
	});
}

export function useMetadataSearch(
	entityType: MetadataEntityType,
	key: string,
	value: string,
	enabled = true,
) {
	return useQuery({
		queryKey: [...metadataKeys.all, 'search', entityType, key, value],
		queryFn: () => metadataApi.search(entityType, key, value),
		enabled: enabled && !!key && !!value,
	});
}
