import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useAgents', () => ({
	useAgents: vi.fn().mockReturnValue({
		data: [
			{
				id: 'agent-1',
				hostname: 'docker-host-1',
				status: 'active',
				created_at: '2024-01-01T00:00:00Z',
			},
		],
		isLoading: false,
		error: null,
	}),
}));

vi.mock('../hooks/useDockerBackup', () => ({
	useDockerContainers: vi.fn().mockReturnValue({
		data: undefined,
		isLoading: false,
		error: null,
	}),
	useDockerVolumes: vi.fn().mockReturnValue({
		data: undefined,
		isLoading: false,
		error: null,
	}),
	useDockerDaemonStatus: vi.fn().mockReturnValue({
		data: undefined,
		isLoading: false,
		error: null,
	}),
	useTriggerDockerBackup: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
		isSuccess: false,
		isError: false,
	}),
}));

vi.mock('../hooks/useLocale', () => ({
	useLocale: vi.fn().mockReturnValue({
		t: (key: string) => {
			const translations: Record<string, string> = {
				'dockerBackup.title': 'Docker Backup',
				'dockerBackup.subtitle': 'Back up Docker containers and volumes',
				'dockerBackup.selectAgent': 'Agent',
				'dockerBackup.chooseAgent': 'Select an agent...',
				'dockerBackup.repositoryId': 'Repository ID',
				'dockerBackup.repositoryIdPlaceholder': 'Enter repository ID',
				'dockerBackup.containers': 'Containers',
				'dockerBackup.volumes': 'Volumes',
				'dockerBackup.image': 'Image',
				'dockerBackup.driver': 'Driver',
				'dockerBackup.size': 'Size',
				'dockerBackup.select': 'Select',
				'dockerBackup.triggerBackup': 'Start Backup',
				'dockerBackup.daemonRunning': 'Docker is running',
				'dockerBackup.daemonStopped': 'Docker is not running',
				'common.name': 'Name',
				'common.status': 'Status',
			};
			return translations[key] || key;
		},
		formatBytes: (bytes: number) => `${bytes} B`,
	}),
}));

import DockerBackup from './DockerBackup';

describe('DockerBackup page', () => {
	it('renders the title', () => {
		renderWithProviders(<DockerBackup />);
		expect(screen.getByText('Docker Backup')).toBeInTheDocument();
	});

	it('renders the subtitle', () => {
		renderWithProviders(<DockerBackup />);
		expect(
			screen.getByText('Back up Docker containers and volumes'),
		).toBeInTheDocument();
	});

	it('renders the agent selector', () => {
		renderWithProviders(<DockerBackup />);
		expect(screen.getByLabelText('Agent')).toBeInTheDocument();
	});

	it('renders agent options in the selector', () => {
		renderWithProviders(<DockerBackup />);
		expect(screen.getByText('docker-host-1')).toBeInTheDocument();
	});

	it('renders the repository ID input', () => {
		renderWithProviders(<DockerBackup />);
		expect(screen.getByLabelText('Repository ID')).toBeInTheDocument();
	});
});
