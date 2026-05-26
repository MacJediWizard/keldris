import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCreateDockerRegistry,
	useDeleteDockerRegistry,
	useDockerRegistries,
	useDockerRegistry,
	useDockerRegistryTypes,
	useExpiringCredentials,
	useHealthCheckAllDockerRegistries,
	useHealthCheckDockerRegistry,
	useLoginAllDockerRegistries,
	useLoginDockerRegistry,
	useRotateDockerRegistryCredentials,
	useSetDefaultDockerRegistry,
	useUpdateDockerRegistry,
} from './useDockerRegistries';

vi.mock('../lib/api', () => ({
	dockerRegistriesApi: {
		list: vi.fn(),
		get: vi.fn(),
		getTypes: vi.fn(),
		getExpiringCredentials: vi.fn(),
		create: vi.fn(),
		update: vi.fn(),
		delete: vi.fn(),
		login: vi.fn(),
		loginAll: vi.fn(),
		healthCheck: vi.fn(),
		healthCheckAll: vi.fn(),
		rotateCredentials: vi.fn(),
		setDefault: vi.fn(),
	},
}));

import { dockerRegistriesApi } from '../lib/api';

describe('useDockerRegistries', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('lists registries', async () => {
		vi.mocked(dockerRegistriesApi.list).mockResolvedValue([]);

		const { result } = renderHook(() => useDockerRegistries(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerRegistriesApi.list).toHaveBeenCalledOnce();
	});
});

describe('useDockerRegistry', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches a registry', async () => {
		vi.mocked(dockerRegistriesApi.get).mockResolvedValue({ id: 'r-1' });

		const { result } = renderHook(() => useDockerRegistry('r-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerRegistriesApi.get).toHaveBeenCalledWith('r-1');
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useDockerRegistry(''), { wrapper: createWrapper() });
		expect(dockerRegistriesApi.get).not.toHaveBeenCalled();
	});
});

describe('useDockerRegistryTypes', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches registry types', async () => {
		vi.mocked(dockerRegistriesApi.getTypes).mockResolvedValue([]);

		const { result } = renderHook(() => useDockerRegistryTypes(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerRegistriesApi.getTypes).toHaveBeenCalledOnce();
	});
});

describe('useExpiringCredentials', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches expiring credentials', async () => {
		vi.mocked(dockerRegistriesApi.getExpiringCredentials).mockResolvedValue([]);

		const { result } = renderHook(() => useExpiringCredentials(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerRegistriesApi.getExpiringCredentials).toHaveBeenCalledOnce();
	});
});

describe('useCreateDockerRegistry', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('creates a registry', async () => {
		vi.mocked(dockerRegistriesApi.create).mockResolvedValue({ id: 'new' });

		const { result } = renderHook(() => useCreateDockerRegistry(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ name: 'docker' } as never);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerRegistriesApi.create).toHaveBeenCalled();
	});
});

describe('useUpdateDockerRegistry', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('updates a registry', async () => {
		vi.mocked(dockerRegistriesApi.update).mockResolvedValue({ id: 'r-1' });

		const { result } = renderHook(() => useUpdateDockerRegistry(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 'r-1', data: { name: 'x' } as never });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerRegistriesApi.update).toHaveBeenCalledWith('r-1', {
			name: 'x',
		});
	});
});

describe('useDeleteDockerRegistry', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('deletes a registry', async () => {
		vi.mocked(dockerRegistriesApi.delete).mockResolvedValue({
			message: 'Deleted',
		});

		const { result } = renderHook(() => useDeleteDockerRegistry(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('r-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerRegistriesApi.delete).toHaveBeenCalledWith('r-1');
	});
});

describe('useLoginDockerRegistry', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('logs into a registry', async () => {
		vi.mocked(dockerRegistriesApi.login).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useLoginDockerRegistry(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('r-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerRegistriesApi.login).toHaveBeenCalledWith('r-1');
	});
});

describe('useLoginAllDockerRegistries', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('logs into all registries', async () => {
		vi.mocked(dockerRegistriesApi.loginAll).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useLoginAllDockerRegistries(), {
			wrapper: createWrapper(),
		});

		result.current.mutate();

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerRegistriesApi.loginAll).toHaveBeenCalledOnce();
	});
});

describe('useHealthCheckDockerRegistry', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('runs health check', async () => {
		vi.mocked(dockerRegistriesApi.healthCheck).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useHealthCheckDockerRegistry(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('r-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerRegistriesApi.healthCheck).toHaveBeenCalledWith('r-1');
	});
});

describe('useHealthCheckAllDockerRegistries', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('runs health check on all', async () => {
		vi.mocked(dockerRegistriesApi.healthCheckAll).mockResolvedValue({
			ok: true,
		});

		const { result } = renderHook(() => useHealthCheckAllDockerRegistries(), {
			wrapper: createWrapper(),
		});

		result.current.mutate();

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerRegistriesApi.healthCheckAll).toHaveBeenCalledOnce();
	});
});

describe('useRotateDockerRegistryCredentials', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('rotates credentials', async () => {
		vi.mocked(dockerRegistriesApi.rotateCredentials).mockResolvedValue({
			ok: true,
		});

		const { result } = renderHook(() => useRotateDockerRegistryCredentials(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 'r-1', data: { token: 'x' } as never });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerRegistriesApi.rotateCredentials).toHaveBeenCalledWith('r-1', {
			token: 'x',
		});
	});
});

describe('useSetDefaultDockerRegistry', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('sets default registry', async () => {
		vi.mocked(dockerRegistriesApi.setDefault).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useSetDefaultDockerRegistry(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('r-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerRegistriesApi.setDefault).toHaveBeenCalledWith('r-1');
	});
});
