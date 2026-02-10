import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../hooks/useDRRunbooks', () => ({
	useDRRunbooks: vi.fn(),
	useDRStatus: vi.fn(() => ({ data: undefined })),
	useCreateDRRunbook: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
	useDeleteDRRunbook: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useActivateDRRunbook: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useArchiveDRRunbook: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useGenerateDRRunbook: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useRenderDRRunbook: vi.fn(() => ({ data: undefined })),
}));

vi.mock('../hooks/useDRTests', () => ({
	useDRTests: vi.fn(() => ({ data: [] })),
	useRunDRTest: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock('../hooks/useSchedules', () => ({
	useSchedules: () => ({ data: [] }),
}));

import { useDRRunbooks } from '../hooks/useDRRunbooks';

const { default: DRRunbooks } = await import('./DRRunbooks');

function renderPage() {
	return render(
		<BrowserRouter>
			<DRRunbooks />
		</BrowserRouter>,
	);
}

describe('DRRunbooks', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		vi.mocked(useDRRunbooks).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useDRRunbooks>);
		renderPage();
		expect(screen.getByText('DR Runbooks')).toBeInTheDocument();
	});

	it('shows loading state', () => {
		vi.mocked(useDRRunbooks).mockReturnValue({
			data: undefined,
			isLoading: true,
			isError: false,
		} as ReturnType<typeof useDRRunbooks>);
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows empty state', () => {
		vi.mocked(useDRRunbooks).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useDRRunbooks>);
		renderPage();
		expect(screen.getByText('No DR runbooks configured')).toBeInTheDocument();
	});

	it('renders runbooks', () => {
		vi.mocked(useDRRunbooks).mockReturnValue({
			data: [
				{
					id: 'rb1',
					name: 'Disaster Recovery Plan',
					status: 'active',
					description: 'Main DR plan',
					steps: [],
					created_at: '2024-01-01T00:00:00Z',
					updated_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useDRRunbooks>);
		renderPage();
		expect(screen.getByText('Disaster Recovery Plan')).toBeInTheDocument();
	});

	it('shows error state', () => {
		vi.mocked(useDRRunbooks).mockReturnValue({
			data: undefined,
			isLoading: false,
			isError: true,
		} as ReturnType<typeof useDRRunbooks>);
		renderPage();
		expect(screen.getByText('Failed to load runbooks')).toBeInTheDocument();
	});

	it('shows create button', () => {
		vi.mocked(useDRRunbooks).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useDRRunbooks>);
		renderPage();
		expect(screen.getByText('Create Runbook')).toBeInTheDocument();
	});
});
