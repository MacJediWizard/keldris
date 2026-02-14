import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { DRTests } from './DRTests';

const mockCancelMutate = vi.fn();

vi.mock('../hooks/useDRTests', () => ({
	useDRTests: vi.fn(),
	useCancelDRTest: () => ({ mutate: mockCancelMutate, isPending: false }),
}));

import { useDRTests } from '../hooks/useDRTests';

function renderPage() {
	return render(
		<BrowserRouter>
			<DRTests />
		</BrowserRouter>,
	);
}

describe('DRTests', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title and subtitle', () => {
		vi.mocked(useDRTests).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useDRTests>);
		renderPage();
		expect(screen.getByText('DR Tests')).toBeInTheDocument();
		expect(
			screen.getByText('Disaster recovery test results and history'),
		).toBeInTheDocument();
	});

	it('shows loading state', () => {
		vi.mocked(useDRTests).mockReturnValue({
			data: undefined,
			isLoading: true,
			isError: false,
		} as ReturnType<typeof useDRTests>);
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows error state', () => {
		vi.mocked(useDRTests).mockReturnValue({
			data: undefined,
			isLoading: false,
			isError: true,
		} as ReturnType<typeof useDRTests>);
		renderPage();
		expect(screen.getByText('Failed to load DR tests')).toBeInTheDocument();
	});

	it('shows empty state', () => {
		vi.mocked(useDRTests).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useDRTests>);
		renderPage();
		expect(screen.getByText('No DR tests')).toBeInTheDocument();
		expect(
			screen.getByText('Run a test from the DR Runbooks page to get started'),
		).toBeInTheDocument();
	});

	it('renders test rows with data', () => {
		vi.mocked(useDRTests).mockReturnValue({
			data: [
				{
					id: '1',
					org_id: 'org-1',
					runbook_id: 'rb-1',
					runbook_name: 'Prod DB Recovery',
					status: 'passed',
					started_at: '2024-01-01T00:00:00Z',
					completed_at: '2024-01-01T00:30:00Z',
					created_at: '2024-01-01T00:00:00Z',
				},
				{
					id: '2',
					org_id: 'org-1',
					runbook_id: 'rb-2',
					runbook_name: 'File Server Recovery',
					status: 'failed',
					started_at: '2024-01-02T00:00:00Z',
					completed_at: '2024-01-02T01:00:00Z',
					created_at: '2024-01-02T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useDRTests>);
		renderPage();
		expect(screen.getByText('Prod DB Recovery')).toBeInTheDocument();
		expect(screen.getByText('File Server Recovery')).toBeInTheDocument();
	});

	it('shows status count labels', () => {
		vi.mocked(useDRTests).mockReturnValue({
			data: [
				{
					id: '1',
					org_id: 'org-1',
					runbook_id: 'rb-1',
					status: 'passed',
					created_at: '2024-01-01T00:00:00Z',
				},
				{
					id: '2',
					org_id: 'org-1',
					runbook_id: 'rb-2',
					status: 'failed',
					created_at: '2024-01-02T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useDRTests>);
		renderPage();
		expect(screen.getAllByText('Passed').length).toBeGreaterThan(0);
		expect(screen.getAllByText('Failed').length).toBeGreaterThan(0);
		expect(screen.getAllByText('Running').length).toBeGreaterThan(0);
	});

	it('shows cancel button for running tests', () => {
		vi.mocked(useDRTests).mockReturnValue({
			data: [
				{
					id: '1',
					org_id: 'org-1',
					runbook_id: 'rb-1',
					runbook_name: 'Running Test',
					status: 'running',
					started_at: '2024-01-01T00:00:00Z',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useDRTests>);
		renderPage();
		expect(screen.getByText('Cancel')).toBeInTheDocument();
	});

	it('calls cancel on click', async () => {
		const user = userEvent.setup();
		vi.mocked(useDRTests).mockReturnValue({
			data: [
				{
					id: 'test-1',
					org_id: 'org-1',
					runbook_id: 'rb-1',
					runbook_name: 'Running Test',
					status: 'running',
					started_at: '2024-01-01T00:00:00Z',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useDRTests>);
		renderPage();
		await user.click(screen.getByText('Cancel'));
		expect(mockCancelMutate).toHaveBeenCalledWith({ id: 'test-1' });
	});

	it('does not show cancel button for completed tests', () => {
		vi.mocked(useDRTests).mockReturnValue({
			data: [
				{
					id: '1',
					org_id: 'org-1',
					runbook_id: 'rb-1',
					runbook_name: 'Done Test',
					status: 'passed',
					started_at: '2024-01-01T00:00:00Z',
					completed_at: '2024-01-01T00:30:00Z',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useDRTests>);
		renderPage();
		expect(screen.queryByText('Cancel')).not.toBeInTheDocument();
	});

	it('filters by status dropdown', async () => {
		const user = userEvent.setup();
		vi.mocked(useDRTests).mockReturnValue({
			data: [
				{
					id: '1',
					org_id: 'org-1',
					runbook_id: 'rb-1',
					runbook_name: 'Passed Test',
					status: 'passed',
					created_at: '2024-01-01T00:00:00Z',
				},
				{
					id: '2',
					org_id: 'org-1',
					runbook_id: 'rb-2',
					runbook_name: 'Failed Test',
					status: 'failed',
					created_at: '2024-01-02T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useDRTests>);
		renderPage();
		const selects = screen.getAllByRole('combobox');
		const statusSelect = selects[0];
		await user.selectOptions(statusSelect, 'failed');
		expect(useDRTests).toHaveBeenCalledWith({ status: 'failed' });
	});

	it('shows RTO and RPO metrics', () => {
		vi.mocked(useDRTests).mockReturnValue({
			data: [
				{
					id: '1',
					org_id: 'org-1',
					runbook_id: 'rb-1',
					runbook_name: 'Metrics Test',
					status: 'passed',
					actual_rto_minutes: 15,
					actual_rpo_minutes: 5,
					started_at: '2024-01-01T00:00:00Z',
					completed_at: '2024-01-01T00:15:00Z',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useDRTests>);
		renderPage();
		expect(screen.getByText('RTO: 15m')).toBeInTheDocument();
		expect(screen.getByText('RPO: 5m')).toBeInTheDocument();
	});

	it('links runbook name to dr-runbooks page', () => {
		vi.mocked(useDRTests).mockReturnValue({
			data: [
				{
					id: '1',
					org_id: 'org-1',
					runbook_id: 'rb-1',
					runbook_name: 'Linked Runbook',
					status: 'passed',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useDRTests>);
		renderPage();
		const link = screen.getByText('Linked Runbook');
		expect(link.closest('a')).toHaveAttribute('href', '/dr-runbooks');
	});
});
