import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('react-router-dom', async () => {
	const actual = await vi.importActual('react-router-dom');
	return {
		...actual,
		useParams: () => ({ id: 'agent-1' }),
	};
});

vi.mock('../hooks/useAgents', () => ({
	useAgent: vi.fn(),
	useAgentStats: vi.fn(() => ({ data: undefined })),
	useAgentBackups: vi.fn(() => ({ data: [], isLoading: false })),
	useAgentSchedules: vi.fn(() => ({ data: [], isLoading: false })),
	useAgentHealthHistory: vi.fn(() => ({ data: [] })),
	useDeleteAgent: () => ({ mutateAsync: vi.fn() }),
	useRotateAgentApiKey: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useRevokeAgentApiKey: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useRunSchedule: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

import { useAgent } from '../hooks/useAgents';

const { default: AgentDetails } = await import('./AgentDetails');

function renderPage() {
	return render(
		<BrowserRouter>
			<AgentDetails />
		</BrowserRouter>,
	);
}

describe('AgentDetails', () => {
	beforeEach(() => vi.clearAllMocks());

	it('shows loading state', () => {
		vi.mocked(useAgent).mockReturnValue({
			data: undefined,
			isLoading: true,
			isError: false,
		} as ReturnType<typeof useAgent>);
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows error state', () => {
		vi.mocked(useAgent).mockReturnValue({
			data: undefined,
			isLoading: false,
			isError: true,
		} as ReturnType<typeof useAgent>);
		renderPage();
		expect(screen.getByText('Agent not found')).toBeInTheDocument();
	});

	it('renders agent details', () => {
		vi.mocked(useAgent).mockReturnValue({
			data: {
				id: 'agent-1',
				hostname: 'production-server',
				status: 'active',
				last_seen: '2024-01-01T00:00:00Z',
				created_at: '2024-01-01T00:00:00Z',
				os_info: { os: 'linux', arch: 'amd64', version: 'Ubuntu 22.04' },
			},
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAgent>);
		renderPage();
		expect(screen.getAllByText('production-server').length).toBeGreaterThan(0);
	});

	it('shows tabs', () => {
		vi.mocked(useAgent).mockReturnValue({
			data: {
				id: 'agent-1',
				hostname: 'server',
				status: 'active',
				last_seen: '2024-01-01T00:00:00Z',
				created_at: '2024-01-01T00:00:00Z',
			},
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAgent>);
		renderPage();
		expect(screen.getByText('Overview')).toBeInTheDocument();
		expect(screen.getByText(/Backup History/)).toBeInTheDocument();
		expect(screen.getAllByText(/Schedules/).length).toBeGreaterThan(0);
		expect(screen.getByText('Health')).toBeInTheDocument();
	});

	it('shows back to agents link', () => {
		vi.mocked(useAgent).mockReturnValue({
			data: {
				id: 'agent-1',
				hostname: 'server',
				status: 'active',
				last_seen: '2024-01-01T00:00:00Z',
				created_at: '2024-01-01T00:00:00Z',
			},
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAgent>);
		renderPage();
		const backLink = document.querySelector('a[href="/agents"]');
		expect(backLink).not.toBeNull();
	});

	it('shows agent status badge', () => {
		vi.mocked(useAgent).mockReturnValue({
			data: {
				id: 'agent-1',
				hostname: 'server',
				status: 'active',
				last_seen: '2024-01-01T00:00:00Z',
				created_at: '2024-01-01T00:00:00Z',
			},
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAgent>);
		renderPage();
		expect(screen.getByText('active')).toBeInTheDocument();
	});

	it('shows OS info when available', () => {
		vi.mocked(useAgent).mockReturnValue({
			data: {
				id: 'agent-1',
				hostname: 'server',
				status: 'active',
				last_seen: '2024-01-01T00:00:00Z',
				created_at: '2024-01-01T00:00:00Z',
				os_info: { os: 'linux', arch: 'amd64', version: 'Ubuntu 22.04' },
			},
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAgent>);
		renderPage();
		expect(screen.getAllByText(/linux/).length).toBeGreaterThan(0);
		expect(screen.getAllByText(/amd64/).length).toBeGreaterThan(0);
	});

	it('shows offline status', () => {
		vi.mocked(useAgent).mockReturnValue({
			data: {
				id: 'agent-1',
				hostname: 'offline-server',
				status: 'offline',
				last_seen: '2024-01-01T00:00:00Z',
				created_at: '2024-01-01T00:00:00Z',
			},
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAgent>);
		renderPage();
		expect(screen.getByText('offline')).toBeInTheDocument();
	});

	it('shows backup history tab', async () => {
		const user = (await import('@testing-library/user-event')).default.setup();
		vi.mocked(useAgent).mockReturnValue({
			data: {
				id: 'agent-1',
				hostname: 'server',
				status: 'active',
				last_seen: '2024-01-01T00:00:00Z',
				created_at: '2024-01-01T00:00:00Z',
			},
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAgent>);
		renderPage();
		const tabs = screen.getAllByText(/Backup History/);
		await user.click(tabs[0]);
		expect(screen.getAllByText(/Backup History/).length).toBeGreaterThan(0);
	});

	it('shows health tab content', async () => {
		const user = (await import('@testing-library/user-event')).default.setup();
		vi.mocked(useAgent).mockReturnValue({
			data: {
				id: 'agent-1',
				hostname: 'server',
				status: 'active',
				last_seen: '2024-01-01T00:00:00Z',
				created_at: '2024-01-01T00:00:00Z',
			},
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAgent>);
		renderPage();
		await user.click(screen.getByText('Health'));
		expect(screen.getByText('Health')).toBeInTheDocument();
	});

	it('shows delete agent action', () => {
		vi.mocked(useAgent).mockReturnValue({
			data: {
				id: 'agent-1',
				hostname: 'server',
				status: 'active',
				last_seen: '2024-01-01T00:00:00Z',
				created_at: '2024-01-01T00:00:00Z',
			},
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAgent>);
		renderPage();
		expect(screen.getByText(/Delete/)).toBeInTheDocument();
	});
});
