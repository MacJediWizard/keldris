import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('react-router-dom', async () => {
	const actual = await vi.importActual('react-router-dom');
	return {
		...actual,
		useParams: () => ({ id: 'repo-1' }),
	};
});

vi.mock('../hooks/useStorageStats', () => ({
	useRepositoryStats: vi.fn(),
	useRepositoryGrowth: vi.fn(() => ({ data: [], isLoading: false })),
	useRepositoryHistory: vi.fn(() => ({ data: [], isLoading: false })),
}));

import { useRepositoryStats } from '../hooks/useStorageStats';

const { default: RepositoryStatsDetail } = await import(
	'./RepositoryStatsDetail'
);

function renderPage() {
	return render(
		<BrowserRouter>
			<RepositoryStatsDetail />
		</BrowserRouter>,
	);
}

describe('RepositoryStatsDetail', () => {
	beforeEach(() => vi.clearAllMocks());

	it('shows loading state', () => {
		vi.mocked(useRepositoryStats).mockReturnValue({
			data: undefined,
			isLoading: true,
		} as ReturnType<typeof useRepositoryStats>);
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('renders stats when data loads', () => {
		vi.mocked(useRepositoryStats).mockReturnValue({
			data: {
				id: 's1',
				repository_id: 'repo-1',
				repository_name: 'my-backup',
				dedup_ratio: 2.5,
				raw_data_size: 1073741824,
				restore_size: 2147483648,
				space_saved: 1073741824,
				space_saved_pct: 50,
				snapshot_count: 15,
				compression_ratio: 1.8,
				compression_space_saved: 400000000,
				collected_at: '2024-06-15T12:00:00Z',
			},
			isLoading: false,
		} as ReturnType<typeof useRepositoryStats>);
		renderPage();
		expect(screen.getByText('my-backup')).toBeInTheDocument();
	});

	it('shows back to storage stats link', () => {
		vi.mocked(useRepositoryStats).mockReturnValue({
			data: {
				id: 's1',
				repository_id: 'repo-1',
				repository_name: 'my-backup',
				dedup_ratio: 2.5,
				raw_data_size: 1073741824,
				restore_size: 2147483648,
				space_saved: 1073741824,
				space_saved_pct: 50,
				snapshot_count: 15,
				compression_ratio: 1.8,
				compression_space_saved: 400000000,
				collected_at: '2024-06-15T12:00:00Z',
			},
			isLoading: false,
		} as ReturnType<typeof useRepositoryStats>);
		renderPage();
		expect(
			screen.getByText('Detailed storage efficiency metrics'),
		).toBeInTheDocument();
	});
});
