import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useAgents', () => ({
	useAgents: vi.fn().mockReturnValue({
		data: [
			{
				id: '1',
				hostname: 'server-1',
				status: 'active',
				last_seen: '2024-06-15T12:00:00Z',
				os_info: { os: 'linux', arch: 'amd64', version: 'Ubuntu 22.04' },
				created_at: '2024-01-01T00:00:00Z',
			},
			{
				id: '2',
				hostname: 'server-2',
				status: 'offline',
				last_seen: '2024-06-10T12:00:00Z',
				os_info: { os: 'darwin', arch: 'arm64', version: 'macOS 14' },
				created_at: '2024-01-01T00:00:00Z',
			},
		],
		isLoading: false,
		error: null,
	}),
	useCreateAgent: vi.fn().mockReturnValue({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
	useDeleteAgent: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
	useRotateAgentApiKey: vi.fn().mockReturnValue({
		mutateAsync: vi.fn(),
		isPending: false,
	}),
	useRevokeAgentApiKey: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
}));

vi.mock('../hooks/useLocale', () => ({
	useLocale: vi.fn().mockReturnValue({
		t: (key: string) => {
			const translations: Record<string, string> = {
				'agents.title': 'Agents',
				'agents.subtitle': 'Manage your backup agents',
				'agents.registerAgent': 'Register Agent',
				'agents.registerNewAgent': 'Register New Agent',
				'agents.hostname': 'Hostname',
				'agents.hostnamePlaceholder': 'Enter hostname',
				'agents.status': 'Status',
				'agents.lastSeen': 'Last Seen',
				'agents.os': 'OS',
				'agents.actions': 'Actions',
				'agents.failedToCreate': 'Failed to register agent',
				'agents.noAgents': 'No agents registered yet',
				'agents.getStarted': 'Get started by registering an agent',
				'common.cancel': 'Cancel',
				'common.delete': 'Delete',
				'common.confirm': 'Confirm',
			};
			return translations[key] || key;
		},
		formatRelativeTime: (d: string) => d || 'Never',
	}),
}));

// Import after mocks
import Agents from './Agents';

describe('Agents page', () => {
	it('renders the agents title', () => {
		renderWithProviders(<Agents />);
		expect(screen.getByText('Agents')).toBeInTheDocument();
	});

	it('renders the subtitle', () => {
		renderWithProviders(<Agents />);
		expect(
			screen.getByText('Manage your backup agents'),
		).toBeInTheDocument();
	});

	it('renders agent hostnames in the table', () => {
		renderWithProviders(<Agents />);
		expect(screen.getByText('server-1')).toBeInTheDocument();
		expect(screen.getByText('server-2')).toBeInTheDocument();
	});

	it('renders the register agent button', () => {
		renderWithProviders(<Agents />);
		expect(screen.getByText('Register Agent')).toBeInTheDocument();
	});

	it('shows agent status indicators', () => {
		renderWithProviders(<Agents />);
		expect(screen.getByText('active')).toBeInTheDocument();
		expect(screen.getByText('offline')).toBeInTheDocument();
	});
});
