import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useDockerRegistries', () => ({
	useDockerRegistries: vi.fn().mockReturnValue({
		data: [
			{
				id: 'r1',
				name: 'My Registry',
				type: 'dockerhub',
				url: 'https://registry.example.com',
				is_default: true,
				enabled: true,
				health_status: 'healthy',
				last_health_check: '2024-01-01T00:00:00Z',
				last_health_error: '',
				credentials_expires_at: null,
			},
		],
		isLoading: false,
		isError: false,
	}),
	useDockerRegistryTypes: vi.fn().mockReturnValue({
		data: [],
	}),
	useExpiringCredentials: vi.fn().mockReturnValue({
		data: { registries: [], warning_days: 30 },
	}),
	useCreateDockerRegistry: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
	useUpdateDockerRegistry: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
	useDeleteDockerRegistry: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
	useLoginDockerRegistry: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
	useLoginAllDockerRegistries: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
	useHealthCheckDockerRegistry: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
	useHealthCheckAllDockerRegistries: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
	useRotateDockerRegistryCredentials: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
	useSetDefaultDockerRegistry: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
}));

import { DockerRegistries } from './DockerRegistries';

describe('DockerRegistries page', () => {
	it('renders the title', () => {
		renderWithProviders(<DockerRegistries />);
		expect(screen.getByText('Docker Registries')).toBeInTheDocument();
	});

	it('renders registry cards from data', () => {
		renderWithProviders(<DockerRegistries />);
		expect(screen.getByText('My Registry')).toBeInTheDocument();
	});
});
