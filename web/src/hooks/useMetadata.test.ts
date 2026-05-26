import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCreateMetadataSchema,
	useDeleteMetadataSchema,
	useMetadataEntityTypes,
	useMetadataFieldTypes,
	useMetadataSchema,
	useMetadataSchemas,
	useMetadataSearch,
	useUpdateAgentMetadata,
	useUpdateMetadataSchema,
	useUpdateRepositoryMetadata,
	useUpdateScheduleMetadata,
} from './useMetadata';

function mockFetch(data: unknown, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

describe('useMetadataSchemas', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches schemas for an entity type', async () => {
		const fetchFn = mockFetch({
			schemas: [{ id: 's1', entity_type: 'agent' }],
		});

		const { result } = renderHook(() => useMetadataSchemas('agent'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/metadata/schemas?entity_type=agent',
			expect.objectContaining({ credentials: 'include' }),
		);
	});
});

describe('useMetadataSchema', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches a single schema', async () => {
		mockFetch({ id: 's1', entity_type: 'agent' });

		const { result } = renderHook(() => useMetadataSchema('s1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual({ id: 's1', entity_type: 'agent' });
	});

	it('does not fetch when id is empty', () => {
		const fetchFn = mockFetch({});

		renderHook(() => useMetadataSchema(''), { wrapper: createWrapper() });

		expect(fetchFn).not.toHaveBeenCalled();
	});
});

describe('useMetadataFieldTypes', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches field types', async () => {
		const fetchFn = mockFetch({ field_types: [] });

		const { result } = renderHook(() => useMetadataFieldTypes(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/metadata/schemas/types',
			expect.objectContaining({ credentials: 'include' }),
		);
	});
});

describe('useMetadataEntityTypes', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches entity types', async () => {
		const fetchFn = mockFetch({ entity_types: [] });

		const { result } = renderHook(() => useMetadataEntityTypes(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/metadata/schemas/entities',
			expect.objectContaining({ credentials: 'include' }),
		);
	});
});

describe('useCreateMetadataSchema', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('creates a schema', async () => {
		const fetchFn = mockFetch({ id: 's2', entity_type: 'agent' });

		const { result } = renderHook(() => useCreateMetadataSchema(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			entity_type: 'agent',
			name: 'new',
		} as Parameters<typeof result.current.mutate>[0]);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/metadata/schemas',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useUpdateMetadataSchema', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('updates a schema', async () => {
		const fetchFn = mockFetch({ id: 's1', entity_type: 'agent' });

		const { result } = renderHook(() => useUpdateMetadataSchema(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			id: 's1',
			data: { name: 'updated' } as Parameters<
				typeof result.current.mutate
			>[0]['data'],
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/metadata/schemas/s1',
			expect.objectContaining({ method: 'PUT' }),
		);
	});
});

describe('useDeleteMetadataSchema', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('deletes a schema', async () => {
		const fetchFn = mockFetch({ message: 'deleted' });

		const { result } = renderHook(() => useDeleteMetadataSchema(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 's1', entityType: 'agent' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/metadata/schemas/s1',
			expect.objectContaining({ method: 'DELETE' }),
		);
	});
});

describe('useUpdateAgentMetadata', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('updates agent metadata', async () => {
		const fetchFn = mockFetch({ message: 'ok' });

		const { result } = renderHook(() => useUpdateAgentMetadata(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			agentId: 'a1',
			data: { metadata: {} } as Parameters<
				typeof result.current.mutate
			>[0]['data'],
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/agents/a1/metadata',
			expect.objectContaining({ method: 'PUT' }),
		);
	});
});

describe('useUpdateRepositoryMetadata', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('updates repository metadata', async () => {
		const fetchFn = mockFetch({ message: 'ok' });

		const { result } = renderHook(() => useUpdateRepositoryMetadata(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			repositoryId: 'r1',
			data: { metadata: {} } as Parameters<
				typeof result.current.mutate
			>[0]['data'],
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/repositories/r1/metadata',
			expect.objectContaining({ method: 'PUT' }),
		);
	});
});

describe('useUpdateScheduleMetadata', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('updates schedule metadata', async () => {
		const fetchFn = mockFetch({ message: 'ok' });

		const { result } = renderHook(() => useUpdateScheduleMetadata(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			scheduleId: 'sc1',
			data: { metadata: {} } as Parameters<
				typeof result.current.mutate
			>[0]['data'],
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/schedules/sc1/metadata',
			expect.objectContaining({ method: 'PUT' }),
		);
	});
});

describe('useMetadataSearch', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('searches metadata', async () => {
		const fetchFn = mockFetch({ results: [] });

		const { result } = renderHook(
			() => useMetadataSearch('agent', 'env', 'prod'),
			{ wrapper: createWrapper() },
		);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/metadata/search?entity_type=agent&key=env&value=prod',
			expect.objectContaining({ credentials: 'include' }),
		);
	});

	it('does not fetch when disabled', () => {
		const fetchFn = mockFetch({});

		renderHook(() => useMetadataSearch('agent', '', ''), {
			wrapper: createWrapper(),
		});

		expect(fetchFn).not.toHaveBeenCalled();
	});
});
