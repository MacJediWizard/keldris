import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useDockerContainers,
	useDockerDaemonStatus,
	useDockerVolumes,
	useTriggerDockerBackup,
} from './useDockerBackup';

vi.mock('../lib/api', () => ({
	dockerBackupApi: {
		listContainers: vi.fn(),
		listVolumes: vi.fn(),
		getDaemonStatus: vi.fn(),
		triggerBackup: vi.fn(),
	},
}));

import { dockerBackupApi } from '../lib/api';

const mockContainers = [
	{
		id: 'c1',
		name: 'app',
		image: 'nginx:latest',
		status: 'Up 2 hours',
		state: 'running',
		created: '2024-01-01T00:00:00Z',
		ports: ['80/tcp'],
	},
];

const mockVolumes = [
	{
		name: 'data',
		driver: 'local',
		mountpoint: '/var/lib/docker/volumes/data/_data',
		size_bytes: 1048576,
		created: '2024-01-01T00:00:00Z',
	},
];

const mockDaemonStatus = {
	available: true,
	version: '24.0.7',
	container_count: 5,
	volume_count: 3,
	server_os: 'linux',
	docker_root_dir: '/var/lib/docker',
	storage_driver: 'overlay2',
};

describe('useDockerContainers', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches containers for an agent', async () => {
		vi.mocked(dockerBackupApi.listContainers).mockResolvedValue(mockContainers);

		const { result } = renderHook(() => useDockerContainers('agent-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockContainers);
		expect(dockerBackupApi.listContainers).toHaveBeenCalledWith('agent-1');
	});

	it('does not fetch when agentId is empty', () => {
		renderHook(() => useDockerContainers(''), {
			wrapper: createWrapper(),
		});

		expect(dockerBackupApi.listContainers).not.toHaveBeenCalled();
	});

	it('handles error state', async () => {
		vi.mocked(dockerBackupApi.listContainers).mockRejectedValue(
			new Error('Network error'),
		);

		const { result } = renderHook(() => useDockerContainers('agent-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isError).toBe(true));
	});
});

describe('useDockerVolumes', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches volumes for an agent', async () => {
		vi.mocked(dockerBackupApi.listVolumes).mockResolvedValue(mockVolumes);

		const { result } = renderHook(() => useDockerVolumes('agent-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockVolumes);
		expect(dockerBackupApi.listVolumes).toHaveBeenCalledWith('agent-1');
	});

	it('does not fetch when agentId is empty', () => {
		renderHook(() => useDockerVolumes(''), {
			wrapper: createWrapper(),
		});

		expect(dockerBackupApi.listVolumes).not.toHaveBeenCalled();
	});
});

describe('useDockerDaemonStatus', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches daemon status for an agent', async () => {
		vi.mocked(dockerBackupApi.getDaemonStatus).mockResolvedValue(
			mockDaemonStatus,
		);

		const { result } = renderHook(() => useDockerDaemonStatus('agent-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockDaemonStatus);
		expect(dockerBackupApi.getDaemonStatus).toHaveBeenCalledWith('agent-1');
	});

	it('does not fetch when agentId is empty', () => {
		renderHook(() => useDockerDaemonStatus(''), {
			wrapper: createWrapper(),
		});

		expect(dockerBackupApi.getDaemonStatus).not.toHaveBeenCalled();
	});
});

describe('useTriggerDockerBackup', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('triggers a docker backup', async () => {
		const mockResponse = {
			id: 'backup-1',
			status: 'queued',
			created_at: '2024-01-01T00:00:00Z',
		};
		vi.mocked(dockerBackupApi.triggerBackup).mockResolvedValue(mockResponse);

		const { result } = renderHook(() => useTriggerDockerBackup(), {
			wrapper: createWrapper(),
		});

		const request = {
			agent_id: 'agent-1',
			repository_id: 'repo-1',
			container_ids: ['c1'],
			volume_names: ['v1'],
		};

		result.current.mutate(request);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(dockerBackupApi.triggerBackup).toHaveBeenCalledWith(request);
	});

	it('handles backup error', async () => {
		vi.mocked(dockerBackupApi.triggerBackup).mockRejectedValue(
			new Error('Backup failed'),
		);

		const { result } = renderHook(() => useTriggerDockerBackup(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			agent_id: 'agent-1',
			repository_id: 'repo-1',
			container_ids: ['c1'],
		});

		await waitFor(() => expect(result.current.isError).toBe(true));
	});
});
