import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { SLATracking } from './SLATracking';

const mockDeleteMutate = vi.fn();

vi.mock('../hooks/useSLAPolicies', () => ({
	useSLAPolicies: vi.fn(),
	useSLAPolicyStatus: () => ({
		data: {
			policy_id: '1',
			current_rpo_hours: 2.5,
			current_rto_hours: 2.5,
			success_rate: 99.8,
			compliant: true,
			calculated_at: '2024-01-01T00:00:00Z',
		},
		isLoading: false,
	}),
	useSLAPolicyHistory: () => ({ data: [], isLoading: false }),
	useCreateSLAPolicy: () => ({
		mutate: vi.fn(),
		isPending: false,
		isError: false,
	}),
	useDeleteSLAPolicy: () => ({
		mutate: mockDeleteMutate,
		isPending: false,
	}),
}));

import { useSLAPolicies } from '../hooks/useSLAPolicies';

function renderPage() {
	return render(
		<BrowserRouter>
			<SLATracking />
		</BrowserRouter>,
	);
}

describe('SLATracking', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title and subtitle', () => {
		vi.mocked(useSLAPolicies).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSLAPolicies>);
		renderPage();
		expect(screen.getByText('SLA Tracking')).toBeInTheDocument();
		expect(
			screen.getByText('Monitor backup service level agreements'),
		).toBeInTheDocument();
	});

	it('shows loading state', () => {
		vi.mocked(useSLAPolicies).mockReturnValue({
			data: undefined,
			isLoading: true,
			isError: false,
		} as ReturnType<typeof useSLAPolicies>);
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows error state', () => {
		vi.mocked(useSLAPolicies).mockReturnValue({
			data: undefined,
			isLoading: false,
			isError: true,
		} as ReturnType<typeof useSLAPolicies>);
		renderPage();
		expect(
			screen.getByText('Failed to load SLA policies'),
		).toBeInTheDocument();
	});

	it('shows empty state', () => {
		vi.mocked(useSLAPolicies).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSLAPolicies>);
		renderPage();
		expect(screen.getByText('No SLA policies')).toBeInTheDocument();
	});

	it('renders policy cards with data', () => {
		vi.mocked(useSLAPolicies).mockReturnValue({
			data: [
				{
					id: '1',
					org_id: 'org-1',
					name: 'Gold SLA',
					target_rpo_hours: 24,
					target_rto_hours: 4,
					target_success_rate: 99.5,
					enabled: true,
					created_at: '2024-01-01T00:00:00Z',
					updated_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSLAPolicies>);
		renderPage();
		expect(screen.getByText('Gold SLA')).toBeInTheDocument();
		expect(screen.getByText('Compliant')).toBeInTheDocument();
	});

	it('shows create policy form on button click', async () => {
		const user = userEvent.setup();
		vi.mocked(useSLAPolicies).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSLAPolicies>);
		renderPage();
		await user.click(screen.getByText('Create Policy'));
		expect(screen.getByText('Create SLA Policy')).toBeInTheDocument();
		expect(screen.getByLabelText('Policy Name')).toBeInTheDocument();
	});

	it('calls delete on click', async () => {
		const user = userEvent.setup();
		vi.mocked(useSLAPolicies).mockReturnValue({
			data: [
				{
					id: '1',
					org_id: 'org-1',
					name: 'Gold SLA',
					target_rpo_hours: 24,
					target_rto_hours: 4,
					target_success_rate: 99.5,
					enabled: true,
					created_at: '2024-01-01T00:00:00Z',
					updated_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSLAPolicies>);
		renderPage();
		await user.click(screen.getByText('Delete'));
		expect(mockDeleteMutate).toHaveBeenCalledWith('1');
	});
});
