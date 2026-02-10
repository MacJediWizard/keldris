import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { BrowserRouter } from 'react-router-dom';

vi.mock('../hooks/useAgents', () => ({
	useAgents: () => ({ data: [{ id: 'a1', hostname: 'server-1' }] }),
}));

vi.mock('../hooks/useRepositories', () => ({
	useRepositories: () => ({ data: [{ id: 'r1', name: 'repo-1' }] }),
}));

vi.mock('../hooks/useSnapshots', () => ({
	useSnapshots: vi.fn(),
	useSnapshotFiles: vi.fn(() => ({ data: undefined, isLoading: false })),
}));

vi.mock('../hooks/useRestore', () => ({
	useRestores: vi.fn(() => ({ data: [], isLoading: false })),
	useCreateRestore: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock('../hooks/useSnapshotComments', () => ({
	useSnapshotComments: () => ({ data: [] }),
}));

vi.mock('../components/features/SnapshotComments', () => ({
	SnapshotComments: () => null,
}));

import { useSnapshots } from '../hooks/useSnapshots';

const { default: Restore } = await import('./Restore');

function renderPage() {
	return render(
		<BrowserRouter>
			<Restore />
		</BrowserRouter>,
	);
}

describe('Restore', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		vi.mocked(useSnapshots).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useSnapshots>);
		renderPage();
		expect(screen.getByText('Restore')).toBeInTheDocument();
	});

	it('shows tabs', () => {
		vi.mocked(useSnapshots).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useSnapshots>);
		renderPage();
		expect(screen.getByText('Snapshots')).toBeInTheDocument();
		expect(screen.getByText('Restore Jobs')).toBeInTheDocument();
	});

	it('shows loading state', () => {
		vi.mocked(useSnapshots).mockReturnValue({ data: undefined, isLoading: true, isError: false } as ReturnType<typeof useSnapshots>);
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows empty state', () => {
		vi.mocked(useSnapshots).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useSnapshots>);
		renderPage();
		expect(screen.getByText('No snapshots found')).toBeInTheDocument();
	});

	it('shows error state for snapshots', () => {
		vi.mocked(useSnapshots).mockReturnValue({ data: undefined, isLoading: false, isError: true } as ReturnType<typeof useSnapshots>);
		renderPage();
		expect(screen.getByText('Failed to load snapshots')).toBeInTheDocument();
	});

	it('shows subtitle', () => {
		vi.mocked(useSnapshots).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useSnapshots>);
		renderPage();
		expect(screen.getByText('Browse snapshots and restore files')).toBeInTheDocument();
	});

	it('shows file history link', () => {
		vi.mocked(useSnapshots).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useSnapshots>);
		renderPage();
		const link = document.querySelector('a[href="/file-history"]');
		expect(link).not.toBeNull();
	});

	it('shows filter dropdowns', () => {
		vi.mocked(useSnapshots).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useSnapshots>);
		renderPage();
		expect(screen.getByText('All Agents')).toBeInTheDocument();
		expect(screen.getByText('All Repositories')).toBeInTheDocument();
	});

	it('shows agent options in filter', () => {
		vi.mocked(useSnapshots).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useSnapshots>);
		renderPage();
		expect(screen.getByText('server-1')).toBeInTheDocument();
	});

	it('renders snapshot rows', () => {
		vi.mocked(useSnapshots).mockReturnValue({
			data: [
				{ id: 'snap-123abc', agent_id: 'a1', repository_id: 'r1', short_id: 'snap-12', time: '2024-01-01T00:00:00Z', hostname: 'server-1', paths: ['/home'], tags: [], tree: 'abc' },
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSnapshots>);
		renderPage();
		expect(screen.getByText('snap-12')).toBeInTheDocument();
	});

	it('switches to Restore Jobs tab', async () => {
		const user = (await import('@testing-library/user-event')).default.setup();
		vi.mocked(useSnapshots).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useSnapshots>);
		renderPage();
		await user.click(screen.getByText('Restore Jobs'));
		expect(screen.getByText('No restore jobs')).toBeInTheDocument();
	});

	it('shows compare button disabled with no selection', () => {
		vi.mocked(useSnapshots).mockReturnValue({
			data: [
				{ id: 'snap-1', agent_id: 'a1', repository_id: 'r1', short_id: 's1', time: '2024-01-01T00:00:00Z', hostname: 'server-1', paths: ['/home'], tags: [], tree: 'a' },
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSnapshots>);
		renderPage();
		const compareBtns = screen.getAllByText('Compare');
		expect(compareBtns.length).toBeGreaterThan(0);
	});

	it('shows empty state message for snapshots', () => {
		vi.mocked(useSnapshots).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useSnapshots>);
		renderPage();
		expect(screen.getByText(/Snapshots will appear here once backups complete/)).toBeInTheDocument();
	});
});
