import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { BrowserRouter } from 'react-router-dom';

vi.mock('../hooks/useBackups', () => ({
	useBackups: vi.fn(),
}));

vi.mock('../hooks/useAgents', () => ({
	useAgents: () => ({ data: [{ id: 'a1', hostname: 'server-1' }] }),
}));

vi.mock('../hooks/useRepositories', () => ({
	useRepositories: () => ({ data: [{ id: 'r1', name: 'repo-1' }] }),
}));

vi.mock('../hooks/useSchedules', () => ({
	useSchedules: () => ({ data: [] }),
}));

vi.mock('../hooks/useTags', () => ({
	useTags: () => ({ data: [{ id: 't1', name: 'prod', color: '#ef4444' }] }),
	useBackupTags: () => ({ data: [] }),
	useSetBackupTags: () => ({ mutateAsync: vi.fn() }),
}));

import { useBackups } from '../hooks/useBackups';

const { default: Backups } = await import('./Backups');

function renderPage() {
	return render(
		<BrowserRouter>
			<Backups />
		</BrowserRouter>,
	);
}

describe('Backups', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		vi.mocked(useBackups).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useBackups>);
		renderPage();
		expect(screen.getByText('Backups')).toBeInTheDocument();
	});

	it('shows loading state', () => {
		vi.mocked(useBackups).mockReturnValue({ data: undefined, isLoading: true, isError: false } as ReturnType<typeof useBackups>);
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows empty state', () => {
		vi.mocked(useBackups).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useBackups>);
		renderPage();
		expect(screen.getByText('No backups found')).toBeInTheDocument();
	});

	it('renders backup rows', () => {
		vi.mocked(useBackups).mockReturnValue({
			data: [
				{ id: 'b1', snapshot_id: 'snap123456', agent_id: 'a1', repository_id: 'r1', status: 'completed', size_bytes: 1048576, started_at: '2024-01-01T00:00:00Z', completed_at: '2024-01-01T00:01:00Z' },
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useBackups>);
		renderPage();
		expect(screen.getAllByText('server-1').length).toBeGreaterThan(0);
	});

	it('shows error state', () => {
		vi.mocked(useBackups).mockReturnValue({ data: undefined, isLoading: false, isError: true } as ReturnType<typeof useBackups>);
		renderPage();
		expect(screen.getByText('Failed to load backups')).toBeInTheDocument();
	});

	it('shows filter controls', () => {
		vi.mocked(useBackups).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useBackups>);
		renderPage();
		expect(screen.getByPlaceholderText('Search by snapshot ID...')).toBeInTheDocument();
	});

	it('shows subtitle', () => {
		vi.mocked(useBackups).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useBackups>);
		renderPage();
		expect(screen.getByText('View and manage backup snapshots')).toBeInTheDocument();
	});

	it('shows agent filter dropdown', () => {
		vi.mocked(useBackups).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useBackups>);
		renderPage();
		expect(screen.getByText('All Agents')).toBeInTheDocument();
		expect(screen.getByText('server-1')).toBeInTheDocument();
	});

	it('shows status filter dropdown', () => {
		vi.mocked(useBackups).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useBackups>);
		renderPage();
		expect(screen.getByText('All Status')).toBeInTheDocument();
		expect(screen.getByText('Completed')).toBeInTheDocument();
		expect(screen.getByText('Running')).toBeInTheDocument();
	});

	it('renders multiple backup rows', () => {
		vi.mocked(useBackups).mockReturnValue({
			data: [
				{ id: 'b1', snapshot_id: 'snap111', agent_id: 'a1', repository_id: 'r1', status: 'completed', size_bytes: 1048576, started_at: '2024-01-01T00:00:00Z', completed_at: '2024-01-01T00:01:00Z' },
				{ id: 'b2', snapshot_id: 'snap222', agent_id: 'a1', repository_id: 'r1', status: 'failed', size_bytes: 0, started_at: '2024-01-02T00:00:00Z', completed_at: '2024-01-02T00:00:30Z' },
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useBackups>);
		renderPage();
		expect(screen.getAllByText('server-1').length).toBeGreaterThan(0);
	});

	it('shows empty state message', () => {
		vi.mocked(useBackups).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useBackups>);
		renderPage();
		expect(screen.getByText(/Backups will appear here once agents start running/)).toBeInTheDocument();
	});

	it('shows table headers when backups exist', () => {
		vi.mocked(useBackups).mockReturnValue({
			data: [
				{ id: 'b1', snapshot_id: 'snap111', agent_id: 'a1', repository_id: 'r1', status: 'completed', size_bytes: 1048576, started_at: '2024-01-01T00:00:00Z', completed_at: '2024-01-01T00:01:00Z' },
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useBackups>);
		renderPage();
		expect(screen.getByText('Snapshot ID')).toBeInTheDocument();
	});

	it('shows tag filter chips', () => {
		vi.mocked(useBackups).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useBackups>);
		renderPage();
		expect(screen.getByText('prod')).toBeInTheDocument();
	});
});
