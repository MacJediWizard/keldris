import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { Alerts } from './Alerts';

const mockMutate = vi.fn();

vi.mock('../hooks/useAlerts', () => ({
	useAlerts: vi.fn(),
	useAcknowledgeAlert: () => ({ mutate: mockMutate, isPending: false }),
	useResolveAlert: () => ({ mutate: mockMutate, isPending: false }),
}));

import { useAlerts } from '../hooks/useAlerts';

function renderPage() {
	return render(
		<BrowserRouter>
			<Alerts />
		</BrowserRouter>,
	);
}

describe('Alerts', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title and subtitle', () => {
		vi.mocked(useAlerts).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAlerts>);
		renderPage();
		expect(screen.getByText('Alerts')).toBeInTheDocument();
		expect(
			screen.getByText('Monitor and manage system alerts'),
		).toBeInTheDocument();
	});

	it('shows loading state', () => {
		vi.mocked(useAlerts).mockReturnValue({
			data: undefined,
			isLoading: true,
			isError: false,
		} as ReturnType<typeof useAlerts>);
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows error state', () => {
		vi.mocked(useAlerts).mockReturnValue({
			data: undefined,
			isLoading: false,
			isError: true,
		} as ReturnType<typeof useAlerts>);
		renderPage();
		expect(screen.getByText('Failed to load alerts')).toBeInTheDocument();
	});

	it('shows empty state', () => {
		vi.mocked(useAlerts).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAlerts>);
		renderPage();
		expect(screen.getByText('No alerts')).toBeInTheDocument();
		expect(
			screen.getByText('Everything is running smoothly'),
		).toBeInTheDocument();
	});

	it('renders alert cards with data', () => {
		vi.mocked(useAlerts).mockReturnValue({
			data: [
				{
					id: '1',
					title: 'Disk Full',
					message: 'Disk is 95% full',
					severity: 'critical',
					status: 'active',
					type: 'storage',
					created_at: '2024-01-01T00:00:00Z',
				},
				{
					id: '2',
					title: 'Slow Backup',
					message: 'Backup took too long',
					severity: 'warning',
					status: 'acknowledged',
					type: 'backup',
					created_at: '2024-01-02T00:00:00Z',
					acknowledged_at: '2024-01-02T01:00:00Z',
				},
				{
					id: '3',
					title: 'Info Alert',
					message: 'System updated',
					severity: 'info',
					status: 'resolved',
					type: 'system',
					created_at: '2024-01-03T00:00:00Z',
					resolved_at: '2024-01-03T01:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAlerts>);
		renderPage();
		expect(screen.getByText('Disk Full')).toBeInTheDocument();
		expect(screen.getByText('Slow Backup')).toBeInTheDocument();
		expect(screen.getByText('Info Alert')).toBeInTheDocument();
	});

	it('shows status count labels', () => {
		vi.mocked(useAlerts).mockReturnValue({
			data: [
				{
					id: '1',
					title: 'A',
					message: 'm',
					severity: 'critical',
					status: 'active',
					type: 'storage',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAlerts>);
		renderPage();
		expect(screen.getAllByText('Active').length).toBeGreaterThan(0);
		expect(screen.getAllByText('Acknowledged').length).toBeGreaterThan(0);
		expect(screen.getAllByText('Resolved').length).toBeGreaterThan(0);
	});

	it('shows acknowledge and resolve buttons for active alerts', () => {
		vi.mocked(useAlerts).mockReturnValue({
			data: [
				{
					id: '1',
					title: 'Active Alert',
					message: 'msg',
					severity: 'critical',
					status: 'active',
					type: 'storage',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAlerts>);
		renderPage();
		expect(screen.getByText('Acknowledge')).toBeInTheDocument();
		expect(screen.getByText('Resolve')).toBeInTheDocument();
	});

	it('calls acknowledge on click', async () => {
		const user = userEvent.setup();
		vi.mocked(useAlerts).mockReturnValue({
			data: [
				{
					id: '1',
					title: 'Active Alert',
					message: 'msg',
					severity: 'critical',
					status: 'active',
					type: 'storage',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAlerts>);
		renderPage();
		await user.click(screen.getByText('Acknowledge'));
		expect(mockMutate).toHaveBeenCalledWith('1');
	});

	it('filters by severity', async () => {
		const user = userEvent.setup();
		vi.mocked(useAlerts).mockReturnValue({
			data: [
				{
					id: '1',
					title: 'Critical Alert',
					message: 'msg',
					severity: 'critical',
					status: 'active',
					type: 'storage',
					created_at: '2024-01-01T00:00:00Z',
				},
				{
					id: '2',
					title: 'Info Alert',
					message: 'msg',
					severity: 'info',
					status: 'active',
					type: 'system',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAlerts>);
		renderPage();
		const selects = screen.getAllByRole('combobox');
		const severitySelect = selects[1];
		await user.selectOptions(severitySelect, 'critical');
		expect(screen.getByText('Critical Alert')).toBeInTheDocument();
		expect(screen.queryByText('Info Alert')).not.toBeInTheDocument();
	});
});
