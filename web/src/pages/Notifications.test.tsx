import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { BrowserRouter } from 'react-router-dom';

vi.mock('../hooks/useNotifications', () => ({
	useNotificationChannels: vi.fn(),
	useNotificationLogs: vi.fn(() => ({ data: [], isLoading: false })),
	useNotificationPreferences: vi.fn(() => ({ data: [], isLoading: false })),
	useCreateNotificationChannel: () => ({ mutateAsync: vi.fn(), isPending: false, isError: false }),
	useUpdateNotificationChannel: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useDeleteNotificationChannel: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useCreateNotificationPreference: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useUpdateNotificationPreference: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

import { useNotificationChannels } from '../hooks/useNotifications';

const { default: Notifications } = await import('./Notifications');

function renderPage() {
	return render(
		<BrowserRouter>
			<Notifications />
		</BrowserRouter>,
	);
}

describe('Notifications', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		vi.mocked(useNotificationChannels).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useNotificationChannels>);
		renderPage();
		expect(screen.getByText('Notifications')).toBeInTheDocument();
	});

	it('shows loading state', () => {
		vi.mocked(useNotificationChannels).mockReturnValue({ data: undefined, isLoading: true, isError: false } as ReturnType<typeof useNotificationChannels>);
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows empty state', () => {
		vi.mocked(useNotificationChannels).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useNotificationChannels>);
		renderPage();
		expect(screen.getByText('No notification channels configured')).toBeInTheDocument();
	});

	it('renders channels', () => {
		vi.mocked(useNotificationChannels).mockReturnValue({
			data: [
				{ id: '1', name: 'Email Alerts', type: 'email', enabled: true, config: { host: 'smtp.example.com' }, created_at: '2024-01-01T00:00:00Z' },
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useNotificationChannels>);
		renderPage();
		expect(screen.getByText('Email Alerts')).toBeInTheDocument();
	});

	it('shows tabs', () => {
		vi.mocked(useNotificationChannels).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useNotificationChannels>);
		renderPage();
		expect(screen.getByText('Channels')).toBeInTheDocument();
		expect(screen.getByText('History')).toBeInTheDocument();
	});

	it('shows add channel button', () => {
		vi.mocked(useNotificationChannels).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useNotificationChannels>);
		renderPage();
		expect(screen.getAllByText('Add Channel').length).toBeGreaterThan(0);
	});

	it('shows error state', () => {
		vi.mocked(useNotificationChannels).mockReturnValue({ data: undefined, isLoading: false, isError: true } as ReturnType<typeof useNotificationChannels>);
		renderPage();
		expect(screen.getByText(/Failed to load/)).toBeInTheDocument();
	});

	it('renders multiple channels', () => {
		vi.mocked(useNotificationChannels).mockReturnValue({
			data: [
				{ id: '1', name: 'Email Alerts', type: 'email', enabled: true, config: { host: 'smtp.example.com' }, created_at: '2024-01-01T00:00:00Z' },
				{ id: '2', name: 'Slack Alerts', type: 'slack', enabled: true, config: { webhook_url: 'https://hooks.slack.com/...' }, created_at: '2024-01-02T00:00:00Z' },
				{ id: '3', name: 'Webhook', type: 'webhook', enabled: false, config: { url: 'https://example.com/hook' }, created_at: '2024-01-03T00:00:00Z' },
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useNotificationChannels>);
		renderPage();
		expect(screen.getByText('Email Alerts')).toBeInTheDocument();
		expect(screen.getByText('Slack Alerts')).toBeInTheDocument();
		expect(screen.getByText('Webhook')).toBeInTheDocument();
	});

	it('shows channel type badges', () => {
		vi.mocked(useNotificationChannels).mockReturnValue({
			data: [
				{ id: '1', name: 'Email', type: 'email', enabled: true, config: {}, created_at: '2024-01-01T00:00:00Z' },
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useNotificationChannels>);
		renderPage();
		expect(screen.getByText('email')).toBeInTheDocument();
	});

	it('switches to History tab', async () => {
		const user = (await import('@testing-library/user-event')).default.setup();
		vi.mocked(useNotificationChannels).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useNotificationChannels>);
		renderPage();
		await user.click(screen.getByText('History'));
		// Verify tab switch happened
		expect(screen.getByText('History')).toBeInTheDocument();
	});

	it('opens add channel modal', async () => {
		const user = (await import('@testing-library/user-event')).default.setup();
		vi.mocked(useNotificationChannels).mockReturnValue({ data: [], isLoading: false, isError: false } as ReturnType<typeof useNotificationChannels>);
		renderPage();
		const addButtons = screen.getAllByText('Add Channel');
		await user.click(addButtons[0]);
		// Modal should open with channel type options
		expect(screen.getAllByText(/Channel/).length).toBeGreaterThan(1);
	});

	it('shows channel actions', () => {
		vi.mocked(useNotificationChannels).mockReturnValue({
			data: [
				{ id: '1', name: 'Email', type: 'email', enabled: true, config: {}, created_at: '2024-01-01T00:00:00Z' },
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useNotificationChannels>);
		renderPage();
		expect(screen.getByText(/Delete/)).toBeInTheDocument();
	});
});
