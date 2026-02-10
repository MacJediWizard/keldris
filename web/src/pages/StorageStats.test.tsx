import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { StorageStats } from './StorageStats';

vi.mock('../hooks/useStorageStats', () => ({
	useStorageStatsSummary: vi.fn(),
	useRepositoryStatsList: vi.fn(),
	useStorageGrowth: vi.fn(),
}));

import {
	useRepositoryStatsList,
	useStorageGrowth,
	useStorageStatsSummary,
} from '../hooks/useStorageStats';

function renderPage() {
	return render(
		<BrowserRouter>
			<StorageStats />
		</BrowserRouter>,
	);
}

describe('StorageStats', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		vi.mocked(useStorageStatsSummary).mockReturnValue({
			data: undefined,
			isLoading: false,
		} as ReturnType<typeof useStorageStatsSummary>);
		vi.mocked(useRepositoryStatsList).mockReturnValue({
			data: undefined,
			isLoading: false,
		} as ReturnType<typeof useRepositoryStatsList>);
		vi.mocked(useStorageGrowth).mockReturnValue({
			data: undefined,
			isLoading: false,
		} as ReturnType<typeof useStorageGrowth>);
		renderPage();
		expect(screen.getByText('Storage Statistics')).toBeInTheDocument();
	});

	it('shows loading state', () => {
		vi.mocked(useStorageStatsSummary).mockReturnValue({
			data: undefined,
			isLoading: true,
		} as ReturnType<typeof useStorageStatsSummary>);
		vi.mocked(useRepositoryStatsList).mockReturnValue({
			data: undefined,
			isLoading: true,
		} as ReturnType<typeof useRepositoryStatsList>);
		vi.mocked(useStorageGrowth).mockReturnValue({
			data: undefined,
			isLoading: true,
		} as ReturnType<typeof useStorageGrowth>);
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('renders summary cards with data', () => {
		vi.mocked(useStorageStatsSummary).mockReturnValue({
			data: {
				avg_dedup_ratio: 2.5,
				total_space_saved: 5368709120,
				total_restore_size: 10737418240,
				total_raw_size: 5368709120,
				total_snapshots: 42,
				repository_count: 3,
			},
			isLoading: false,
		} as ReturnType<typeof useStorageStatsSummary>);
		vi.mocked(useRepositoryStatsList).mockReturnValue({
			data: [],
			isLoading: false,
		} as ReturnType<typeof useRepositoryStatsList>);
		vi.mocked(useStorageGrowth).mockReturnValue({
			data: [],
			isLoading: false,
		} as ReturnType<typeof useStorageGrowth>);
		renderPage();
		expect(screen.getByText('Average Dedup Ratio')).toBeInTheDocument();
		expect(screen.getByText('Total Space Saved')).toBeInTheDocument();
		expect(screen.getByText('Actual Storage Used')).toBeInTheDocument();
		expect(screen.getByText('Total Snapshots')).toBeInTheDocument();
		expect(screen.getByText('42')).toBeInTheDocument();
	});

	it('renders repository stats table', () => {
		vi.mocked(useStorageStatsSummary).mockReturnValue({
			data: undefined,
			isLoading: false,
		} as ReturnType<typeof useStorageStatsSummary>);
		vi.mocked(useRepositoryStatsList).mockReturnValue({
			data: [
				{
					id: '1',
					repository_id: 'r1',
					repository_name: 'my-repo',
					dedup_ratio: 2.0,
					space_saved: 1073741824,
					space_saved_pct: 50,
					raw_data_size: 1073741824,
					restore_size: 2147483648,
					snapshot_count: 10,
					collected_at: '2024-06-15T12:00:00Z',
				},
			],
			isLoading: false,
		} as ReturnType<typeof useRepositoryStatsList>);
		vi.mocked(useStorageGrowth).mockReturnValue({
			data: [],
			isLoading: false,
		} as ReturnType<typeof useStorageGrowth>);
		renderPage();
		expect(screen.getByText('my-repo')).toBeInTheDocument();
		expect(screen.getByText('Repository Statistics')).toBeInTheDocument();
	});

	it('shows empty state for no repo stats', () => {
		vi.mocked(useStorageStatsSummary).mockReturnValue({
			data: undefined,
			isLoading: false,
		} as ReturnType<typeof useStorageStatsSummary>);
		vi.mocked(useRepositoryStatsList).mockReturnValue({
			data: [],
			isLoading: false,
		} as ReturnType<typeof useRepositoryStatsList>);
		vi.mocked(useStorageGrowth).mockReturnValue({
			data: [],
			isLoading: false,
		} as ReturnType<typeof useStorageGrowth>);
		renderPage();
		expect(
			screen.getByText('No repository statistics yet'),
		).toBeInTheDocument();
	});

	it('renders storage growth chart', () => {
		vi.mocked(useStorageStatsSummary).mockReturnValue({
			data: undefined,
			isLoading: false,
		} as ReturnType<typeof useStorageStatsSummary>);
		vi.mocked(useRepositoryStatsList).mockReturnValue({
			data: [],
			isLoading: false,
		} as ReturnType<typeof useRepositoryStatsList>);
		vi.mocked(useStorageGrowth).mockReturnValue({
			data: [
				{ date: '2024-06-01', raw_data_size: 500000, restore_size: 1000000 },
				{ date: '2024-06-02', raw_data_size: 600000, restore_size: 1100000 },
			],
			isLoading: false,
		} as ReturnType<typeof useStorageGrowth>);
		renderPage();
		expect(screen.getByText('Storage Growth')).toBeInTheDocument();
		expect(screen.getByText('Actual storage')).toBeInTheDocument();
		expect(screen.getByText('Original data')).toBeInTheDocument();
	});
});
