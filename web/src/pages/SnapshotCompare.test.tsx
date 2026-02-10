import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { BrowserRouter } from 'react-router-dom';

vi.mock('react-router-dom', async () => {
	const actual = await vi.importActual('react-router-dom');
	return {
		...actual,
		useSearchParams: () => [new URLSearchParams(), vi.fn()],
	};
});

vi.mock('../hooks/useAgents', () => ({
	useAgents: () => ({ data: [{ id: 'a1', hostname: 'server-1' }] }),
}));

vi.mock('../hooks/useRepositories', () => ({
	useRepositories: () => ({ data: [{ id: 'r1', name: 'repo-1' }] }),
}));

vi.mock('../hooks/useSnapshots', () => ({
	useSnapshots: vi.fn(() => ({ data: [], isLoading: false })),
	useSnapshotCompare: vi.fn(() => ({ data: undefined, isLoading: false })),
}));

const { default: SnapshotCompare } = await import('./SnapshotCompare');

function renderPage() {
	return render(
		<BrowserRouter>
			<SnapshotCompare />
		</BrowserRouter>,
	);
}

describe('SnapshotCompare', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		renderPage();
		expect(screen.getByText('Compare Snapshots')).toBeInTheDocument();
	});

	it('shows select snapshots heading', () => {
		renderPage();
		expect(screen.getByText('Select Snapshots')).toBeInTheDocument();
	});

	it('shows empty snapshots message', () => {
		renderPage();
		expect(screen.getByText(/No snapshots available/)).toBeInTheDocument();
	});
});
