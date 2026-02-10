import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { Maintenance } from './Maintenance';

vi.mock('../hooks/useAuth', () => ({
	useMe: vi.fn(),
}));

vi.mock('../hooks/useMaintenance', () => ({
	useMaintenanceWindows: vi.fn(),
	useCreateMaintenanceWindow: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
	}),
	useUpdateMaintenanceWindow: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
	}),
	useDeleteMaintenanceWindow: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
	}),
}));

import { useMe } from '../hooks/useAuth';
import { useMaintenanceWindows } from '../hooks/useMaintenance';

function renderPage() {
	return render(
		<BrowserRouter>
			<Maintenance />
		</BrowserRouter>,
	);
}

describe('Maintenance', () => {
	beforeEach(() => vi.clearAllMocks());

	it('shows admin-only message for non-admin users', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'member' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useMaintenanceWindows).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useMaintenanceWindows>);
		renderPage();
		expect(
			screen.getByText('Only administrators can manage maintenance windows.'),
		).toBeInTheDocument();
	});

	it('shows title for non-admin', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'member' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useMaintenanceWindows).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useMaintenanceWindows>);
		renderPage();
		expect(screen.getByText('Maintenance')).toBeInTheDocument();
	});

	it('shows subtitle for non-admin', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'member' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useMaintenanceWindows).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useMaintenanceWindows>);
		renderPage();
		expect(
			screen.getByText('Schedule maintenance windows to pause backups'),
		).toBeInTheDocument();
	});

	it('shows readonly message for readonly users', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'readonly' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useMaintenanceWindows).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useMaintenanceWindows>);
		renderPage();
		expect(
			screen.getByText('Only administrators can manage maintenance windows.'),
		).toBeInTheDocument();
	});

	it('shows loading state for admin', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'admin' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useMaintenanceWindows).mockReturnValue({
			data: undefined,
			isLoading: true,
			isError: false,
		} as ReturnType<typeof useMaintenanceWindows>);
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows error state', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'admin' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useMaintenanceWindows).mockReturnValue({
			data: undefined,
			isLoading: false,
			isError: true,
		} as ReturnType<typeof useMaintenanceWindows>);
		renderPage();
		expect(
			screen.getByText('Failed to load maintenance windows'),
		).toBeInTheDocument();
	});

	it('shows empty state for admin', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'admin' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useMaintenanceWindows).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useMaintenanceWindows>);
		renderPage();
		expect(screen.getByText('No maintenance windows')).toBeInTheDocument();
	});

	it('shows empty state help text', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'admin' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useMaintenanceWindows).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useMaintenanceWindows>);
		renderPage();
		expect(
			screen.getByText(
				'Schedule a maintenance window to pause backups during maintenance.',
			),
		).toBeInTheDocument();
	});

	it('renders maintenance windows', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'admin' },
		} as ReturnType<typeof useMe>);
		const futureDate = new Date(Date.now() + 86400000).toISOString();
		const futureEnd = new Date(Date.now() + 172800000).toISOString();
		vi.mocked(useMaintenanceWindows).mockReturnValue({
			data: [
				{
					id: '1',
					title: 'Server Upgrade',
					message: 'Upgrading servers',
					starts_at: futureDate,
					ends_at: futureEnd,
					notify_before_minutes: 60,
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useMaintenanceWindows>);
		renderPage();
		expect(screen.getByText('Server Upgrade')).toBeInTheDocument();
		expect(screen.getByText('Upgrading servers')).toBeInTheDocument();
		expect(screen.getByText('Upcoming')).toBeInTheDocument();
	});

	it('shows completed status for past windows', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'admin' },
		} as ReturnType<typeof useMe>);
		const pastDate = new Date(Date.now() - 172800000).toISOString();
		const pastEnd = new Date(Date.now() - 86400000).toISOString();
		vi.mocked(useMaintenanceWindows).mockReturnValue({
			data: [
				{
					id: '1',
					title: 'Past Maint',
					starts_at: pastDate,
					ends_at: pastEnd,
					notify_before_minutes: 30,
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useMaintenanceWindows>);
		renderPage();
		expect(screen.getByText('Completed')).toBeInTheDocument();
	});

	it('shows active status for current windows', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'admin' },
		} as ReturnType<typeof useMe>);
		const activeStart = new Date(Date.now() - 3600000).toISOString();
		const activeEnd = new Date(Date.now() + 3600000).toISOString();
		vi.mocked(useMaintenanceWindows).mockReturnValue({
			data: [
				{
					id: '1',
					title: 'Active Maint',
					starts_at: activeStart,
					ends_at: activeEnd,
					notify_before_minutes: 30,
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useMaintenanceWindows>);
		renderPage();
		expect(screen.getByText('Active')).toBeInTheDocument();
	});

	it('shows schedule button', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'admin' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useMaintenanceWindows).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useMaintenanceWindows>);
		renderPage();
		expect(screen.getByText('Schedule Maintenance')).toBeInTheDocument();
	});

	it('opens form when schedule button clicked', async () => {
		const user = userEvent.setup();
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'admin' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useMaintenanceWindows).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useMaintenanceWindows>);
		renderPage();
		await user.click(screen.getByText('Schedule Maintenance'));
		expect(screen.getByText('Schedule Maintenance Window')).toBeInTheDocument();
		expect(screen.getByLabelText('Title')).toBeInTheDocument();
	});

	it('shows form fields when form is open', async () => {
		const user = userEvent.setup();
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'admin' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useMaintenanceWindows).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useMaintenanceWindows>);
		renderPage();
		await user.click(screen.getByText('Schedule Maintenance'));
		expect(screen.getByLabelText('Message (optional)')).toBeInTheDocument();
		expect(screen.getByLabelText('Start Time')).toBeInTheDocument();
		expect(screen.getByLabelText('End Time')).toBeInTheDocument();
		expect(
			screen.getByLabelText('Notify Before (minutes)'),
		).toBeInTheDocument();
	});

	it('shows cancel and schedule buttons in form', async () => {
		const user = userEvent.setup();
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'admin' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useMaintenanceWindows).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useMaintenanceWindows>);
		renderPage();
		await user.click(screen.getByText('Schedule Maintenance'));
		expect(screen.getByText('Cancel')).toBeInTheDocument();
		expect(screen.getByText('Schedule')).toBeInTheDocument();
	});

	it('shows edit and delete buttons for windows', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'owner' },
		} as ReturnType<typeof useMe>);
		const futureDate = new Date(Date.now() + 86400000).toISOString();
		const futureEnd = new Date(Date.now() + 172800000).toISOString();
		vi.mocked(useMaintenanceWindows).mockReturnValue({
			data: [
				{
					id: '1',
					title: 'Maint',
					starts_at: futureDate,
					ends_at: futureEnd,
					notify_before_minutes: 30,
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useMaintenanceWindows>);
		renderPage();
		expect(screen.getByText('Edit')).toBeInTheDocument();
		expect(screen.getByText('Delete')).toBeInTheDocument();
	});

	it('shows multiple maintenance windows', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'admin' },
		} as ReturnType<typeof useMe>);
		const futureDate1 = new Date(Date.now() + 86400000).toISOString();
		const futureEnd1 = new Date(Date.now() + 172800000).toISOString();
		const futureDate2 = new Date(Date.now() + 259200000).toISOString();
		const futureEnd2 = new Date(Date.now() + 345600000).toISOString();
		vi.mocked(useMaintenanceWindows).mockReturnValue({
			data: [
				{
					id: '1',
					title: 'Server Upgrade',
					starts_at: futureDate1,
					ends_at: futureEnd1,
					notify_before_minutes: 60,
				},
				{
					id: '2',
					title: 'Database Migration',
					starts_at: futureDate2,
					ends_at: futureEnd2,
					notify_before_minutes: 30,
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useMaintenanceWindows>);
		renderPage();
		expect(screen.getByText('Server Upgrade')).toBeInTheDocument();
		expect(screen.getByText('Database Migration')).toBeInTheDocument();
	});

	it('shows notify before help text in form', async () => {
		const user = userEvent.setup();
		vi.mocked(useMe).mockReturnValue({
			data: { current_org_role: 'admin' },
		} as ReturnType<typeof useMe>);
		vi.mocked(useMaintenanceWindows).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useMaintenanceWindows>);
		renderPage();
		await user.click(screen.getByText('Schedule Maintenance'));
		expect(
			screen.getByText(
				'Send notification this many minutes before maintenance starts',
			),
		).toBeInTheDocument();
	});
});
