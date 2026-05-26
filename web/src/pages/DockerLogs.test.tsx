import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useAgents', () => ({
	useAgents: vi.fn().mockReturnValue({
		data: [{ id: 'a1', hostname: 'docker-host-1' }],
	}),
}));

vi.mock('../hooks/useDockerLogs', () => ({
	useDockerLogBackups: vi.fn().mockReturnValue({
		data: {
			backups: [
				{
					id: 'b1',
					agent_id: 'a1',
					container_id: 'abcdef123456',
					container_name: 'web-server',
					status: 'completed',
					original_size: 1024,
					compressed_size: 512,
					compressed: true,
					line_count: 42,
					start_time: '2024-01-01T00:00:00Z',
					end_time: '2024-01-01T01:00:00Z',
					created_at: '2024-01-01T01:00:00Z',
				},
			],
			total_count: 1,
		},
		isLoading: false,
		isError: false,
		refetch: vi.fn(),
	}),
	useDeleteDockerLogBackup: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
	useDockerLogView: vi.fn().mockReturnValue({
		data: undefined,
		isLoading: false,
	}),
	useDockerLogDownload: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
	useDockerLogSettings: vi.fn().mockReturnValue({
		data: undefined,
		isLoading: false,
	}),
	useDockerLogStorageStats: vi.fn().mockReturnValue({
		data: undefined,
		isLoading: false,
	}),
	useUpdateDockerLogSettings: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
	useApplyDockerLogRetention: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
}));

import { DockerLogs } from './DockerLogs';

describe('DockerLogs page', () => {
	it('renders the title', () => {
		renderWithProviders(<DockerLogs />);
		expect(screen.getByText('Docker Container Logs')).toBeInTheDocument();
	});

	it('renders backup rows from data', () => {
		renderWithProviders(<DockerLogs />);
		expect(screen.getByText('web-server')).toBeInTheDocument();
	});
});
