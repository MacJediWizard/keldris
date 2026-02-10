import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { BrowserRouter } from 'react-router-dom';

vi.mock('../hooks/useReports', () => ({
	useReportSchedules: vi.fn(),
	useReportHistory: vi.fn(() => ({ data: [], isLoading: false })),
	useCreateReportSchedule: () => ({ mutateAsync: vi.fn(), isPending: false, isError: false }),
	useUpdateReportSchedule: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useDeleteReportSchedule: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useSendReport: () => ({ mutateAsync: vi.fn(), isPending: false }),
	usePreviewReport: vi.fn(() => ({ data: undefined, isLoading: false })),
}));

vi.mock('../hooks/useNotifications', () => ({
	useNotificationChannels: () => ({ data: [], isLoading: false }),
}));

import { useReportSchedules } from '../hooks/useReports';

const { default: Reports } = await import('./Reports');

function renderPage() {
	return render(
		<BrowserRouter>
			<Reports />
		</BrowserRouter>,
	);
}

describe('Reports', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		vi.mocked(useReportSchedules).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useReportSchedules>);
		renderPage();
		expect(screen.getByText('Email Reports')).toBeInTheDocument();
	});

	it('shows loading state', () => {
		vi.mocked(useReportSchedules).mockReturnValue({ data: undefined, isLoading: true, isError: false } as ReturnType<typeof useReportSchedules>);
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows empty state', () => {
		vi.mocked(useReportSchedules).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useReportSchedules>);
		renderPage();
		expect(screen.getByText(/No report schedules configured/)).toBeInTheDocument();
	});

	it('renders report schedules', () => {
		vi.mocked(useReportSchedules).mockReturnValue({
			data: [
				{ id: '1', name: 'Weekly Report', frequency: 'weekly', timezone: 'UTC', recipients: ['admin@example.com'], enabled: true, created_at: '2024-01-01T00:00:00Z' },
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useReportSchedules>);
		renderPage();
		expect(screen.getByText('Weekly Report')).toBeInTheDocument();
	});

	it('shows add schedule button', () => {
		vi.mocked(useReportSchedules).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useReportSchedules>);
		renderPage();
		expect(screen.getByText('Create Schedule')).toBeInTheDocument();
	});

	it('shows tabs', () => {
		vi.mocked(useReportSchedules).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useReportSchedules>);
		renderPage();
		expect(screen.getByText('Schedules')).toBeInTheDocument();
		expect(screen.getByText('History')).toBeInTheDocument();
	});

	it('shows schedule table headers', () => {
		vi.mocked(useReportSchedules).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useReportSchedules>);
		renderPage();
		expect(screen.getByText('Schedule')).toBeInTheDocument();
		expect(screen.getByText('Frequency')).toBeInTheDocument();
	});

	it('renders schedule details in table', () => {
		vi.mocked(useReportSchedules).mockReturnValue({
			data: [
				{ id: '1', name: 'Daily Summary', frequency: 'daily', timezone: 'US/Eastern', recipients: ['admin@example.com', 'team@example.com'], enabled: true, created_at: '2024-01-01T00:00:00Z', updated_at: '2024-01-01T00:00:00Z', org_id: 'org-1' },
				{ id: '2', name: 'Monthly Report', frequency: 'monthly', timezone: 'UTC', recipients: ['boss@example.com'], enabled: false, created_at: '2024-02-01T00:00:00Z', updated_at: '2024-02-01T00:00:00Z', org_id: 'org-1' },
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useReportSchedules>);
		renderPage();
		expect(screen.getByText('Daily Summary')).toBeInTheDocument();
		expect(screen.getByText('Monthly Report')).toBeInTheDocument();
		expect(screen.getByText('daily')).toBeInTheDocument();
		expect(screen.getByText('monthly')).toBeInTheDocument();
	});

	it('shows action buttons for schedules', () => {
		vi.mocked(useReportSchedules).mockReturnValue({
			data: [
				{ id: '1', name: 'Weekly Report', frequency: 'weekly', timezone: 'UTC', recipients: ['admin@example.com'], enabled: true, created_at: '2024-01-01T00:00:00Z', updated_at: '2024-01-01T00:00:00Z', org_id: 'org-1' },
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useReportSchedules>);
		renderPage();
		expect(screen.getByText('Send Now')).toBeInTheDocument();
	});

	it('switches to History tab', async () => {
		const user = (await import('@testing-library/user-event')).default.setup();
		vi.mocked(useReportSchedules).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useReportSchedules>);
		renderPage();
		await user.click(screen.getByText('History'));
		expect(screen.getByText(/No reports have been sent/)).toBeInTheDocument();
	});

	it('shows history entries on History tab', async () => {
		const user = (await import('@testing-library/user-event')).default.setup();
		vi.mocked(useReportSchedules).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useReportSchedules>);

		const { useReportHistory } = await import('../hooks/useReports');
		vi.mocked(useReportHistory).mockReturnValue({
			data: [
				{ id: 'h1', org_id: 'org-1', report_type: 'weekly', period_start: '2024-01-01T00:00:00Z', period_end: '2024-01-07T00:00:00Z', recipients: ['admin@example.com'], status: 'sent' as const, sent_at: '2024-01-07T12:00:00Z', created_at: '2024-01-07T12:00:00Z' },
			],
			isLoading: false,
		} as ReturnType<typeof useReportHistory>);

		renderPage();
		await user.click(screen.getByText('History'));
		expect(screen.getByText('sent')).toBeInTheDocument();
	});

	it('opens create schedule modal', async () => {
		const user = (await import('@testing-library/user-event')).default.setup();
		vi.mocked(useReportSchedules).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useReportSchedules>);
		renderPage();
		await user.click(screen.getByText('Create Schedule'));
		expect(screen.getByText('Create Report Schedule')).toBeInTheDocument();
		expect(screen.getByLabelText('Schedule Name')).toBeInTheDocument();
	});

	it('shows subtitle', () => {
		vi.mocked(useReportSchedules).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useReportSchedules>);
		renderPage();
		expect(screen.getByText(/Schedule automated backup summary reports/)).toBeInTheDocument();
	});
});
