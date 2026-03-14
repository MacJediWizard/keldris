import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import Webhooks from './Webhooks';

const mockCreateMutateAsync = vi.fn();
const mockUpdateMutateAsync = vi.fn();
const mockDeleteMutateAsync = vi.fn();
const mockTestMutateAsync = vi.fn();
const mockRetryMutate = vi.fn();

vi.mock('../hooks/useWebhooks', () => ({
	useWebhookEndpoints: vi.fn(),
	useWebhookEventTypes: vi.fn(),
	useWebhookEndpointDeliveries: vi.fn(),
	useCreateWebhookEndpoint: () => ({
		mutateAsync: mockCreateMutateAsync,
		isPending: false,
	}),
	useUpdateWebhookEndpoint: () => ({
		mutateAsync: mockUpdateMutateAsync,
		isPending: false,
	}),
	useDeleteWebhookEndpoint: () => ({
		mutateAsync: mockDeleteMutateAsync,
		isPending: false,
	}),
	useTestWebhookEndpoint: () => ({
		mutateAsync: mockTestMutateAsync,
		isPending: false,
	}),
	useRetryWebhookDelivery: () => ({
		mutate: mockRetryMutate,
		isPending: false,
	}),
}));

import {
	useWebhookEndpointDeliveries,
	useWebhookEndpoints,
	useWebhookEventTypes,
} from '../hooks/useWebhooks';

function renderPage() {
	return render(
		<BrowserRouter>
			<Webhooks />
		</BrowserRouter>,
	);
}

function setupDefaultMocks(overrides?: {
	endpoints?: unknown[] | undefined;
	endpointsLoading?: boolean;
	eventTypes?: string[] | undefined;
	deliveries?: unknown[] | undefined;
	deliveriesLoading?: boolean;
}) {
	vi.mocked(useWebhookEndpoints).mockReturnValue({
		data: overrides?.endpoints ?? [],
		isLoading: overrides?.endpointsLoading ?? false,
	} as ReturnType<typeof useWebhookEndpoints>);

	vi.mocked(useWebhookEventTypes).mockReturnValue({
		data: overrides?.eventTypes
			? { event_types: overrides.eventTypes }
			: undefined,
	} as ReturnType<typeof useWebhookEventTypes>);

	vi.mocked(useWebhookEndpointDeliveries).mockReturnValue({
		data: overrides?.deliveries
			? { deliveries: overrides.deliveries }
			: undefined,
		isLoading: overrides?.deliveriesLoading ?? false,
	} as ReturnType<typeof useWebhookEndpointDeliveries>);
}

const mockEndpoint = {
	id: 'ep-1',
	org_id: 'org-1',
	name: 'Slack Integration',
	url: 'https://hooks.slack.com/services/T00/B00/xxx',
	enabled: true,
	event_types: ['backup.completed', 'backup.failed'] as const,
	retry_count: 3,
	timeout_seconds: 30,
	created_at: '2024-01-01T00:00:00Z',
	updated_at: '2024-06-01T00:00:00Z',
};

const mockEndpointDisabled = {
	id: 'ep-2',
	org_id: 'org-1',
	name: 'PagerDuty Alerts',
	url: 'https://events.pagerduty.com/v2/enqueue',
	enabled: false,
	event_types: ['alert.triggered', 'agent.offline'] as const,
	retry_count: 5,
	timeout_seconds: 15,
	created_at: '2024-02-01T00:00:00Z',
	updated_at: '2024-05-01T00:00:00Z',
};

const mockEndpointManyEvents = {
	id: 'ep-3',
	org_id: 'org-1',
	name: 'Event Logger',
	url: 'https://example.com/webhooks',
	enabled: true,
	event_types: [
		'backup.started',
		'backup.completed',
		'backup.failed',
		'agent.online',
	] as const,
	retry_count: 3,
	timeout_seconds: 30,
	created_at: '2024-03-01T00:00:00Z',
	updated_at: '2024-03-01T00:00:00Z',
};

describe('Webhooks', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders page title', () => {
		setupDefaultMocks();
		renderPage();
		expect(screen.getByText('Webhooks')).toBeInTheDocument();
	});

	it('renders subtitle', () => {
		setupDefaultMocks();
		renderPage();
		expect(
			screen.getByText(
				'Configure outbound webhooks to receive real-time event notifications',
			),
		).toBeInTheDocument();
	});

	it('shows Add Endpoint button', () => {
		setupDefaultMocks();
		renderPage();
		expect(screen.getByText('Add Endpoint')).toBeInTheDocument();
	});

	it('shows loading skeleton rows when endpoints are loading', () => {
		setupDefaultMocks({ endpointsLoading: true });
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows empty state when no endpoints exist', () => {
		setupDefaultMocks({ endpoints: [] });
		renderPage();
		expect(
			screen.getByText('No webhook endpoints configured'),
		).toBeInTheDocument();
		expect(
			screen.getByText(
				'Add an endpoint to receive real-time event notifications',
			),
		).toBeInTheDocument();
	});

	it('renders endpoint data in table', () => {
		setupDefaultMocks({ endpoints: [mockEndpoint] });
		renderPage();
		expect(screen.getByText('Slack Integration')).toBeInTheDocument();
		expect(
			screen.getByText('https://hooks.slack.com/services/T00/B00/xxx'),
		).toBeInTheDocument();
	});

	it('renders multiple endpoints', () => {
		setupDefaultMocks({
			endpoints: [mockEndpoint, mockEndpointDisabled],
		});
		renderPage();
		expect(screen.getByText('Slack Integration')).toBeInTheDocument();
		expect(screen.getByText('PagerDuty Alerts')).toBeInTheDocument();
	});

	it('shows event type labels for endpoints', () => {
		setupDefaultMocks({ endpoints: [mockEndpoint] });
		renderPage();
		expect(screen.getByText('Backup Completed')).toBeInTheDocument();
		expect(screen.getByText('Backup Failed')).toBeInTheDocument();
	});

	it('truncates event types when more than 2', () => {
		setupDefaultMocks({ endpoints: [mockEndpointManyEvents] });
		renderPage();
		expect(screen.getByText('+2 more')).toBeInTheDocument();
	});

	it('shows table headers', () => {
		setupDefaultMocks({ endpoints: [mockEndpoint] });
		renderPage();
		expect(screen.getByText('Name')).toBeInTheDocument();
		expect(screen.getByText('URL')).toBeInTheDocument();
		expect(screen.getByText('Events')).toBeInTheDocument();
		expect(screen.getByText('Status')).toBeInTheDocument();
		expect(screen.getByText('Actions')).toBeInTheDocument();
	});

	it('shows action buttons for endpoints', () => {
		setupDefaultMocks({ endpoints: [mockEndpoint] });
		renderPage();
		expect(screen.getByText('Test')).toBeInTheDocument();
		expect(screen.getByText('Log')).toBeInTheDocument();
		expect(screen.getByText('Delete')).toBeInTheDocument();
	});

	it('calls delete when Delete is clicked and confirmed', async () => {
		const user = userEvent.setup();
		setupDefaultMocks({ endpoints: [mockEndpoint] });
		renderPage();
		await user.click(screen.getByText('Delete'));
		expect(mockDeleteMutateAsync).toHaveBeenCalledWith('ep-1');
	});

	it('calls toggle enabled when status toggle is clicked', async () => {
		const user = userEvent.setup();
		setupDefaultMocks({ endpoints: [mockEndpoint] });
		renderPage();
		const toggleButtons = document.querySelectorAll(
			'button.relative.inline-flex',
		);
		expect(toggleButtons.length).toBe(1);
		await user.click(toggleButtons[0] as HTMLElement);
		expect(mockUpdateMutateAsync).toHaveBeenCalledWith({
			id: 'ep-1',
			data: { enabled: false },
		});
	});

	it('calls toggle to enable when disabled endpoint toggle is clicked', async () => {
		const user = userEvent.setup();
		setupDefaultMocks({ endpoints: [mockEndpointDisabled] });
		renderPage();
		const toggleButtons = document.querySelectorAll(
			'button.relative.inline-flex',
		);
		await user.click(toggleButtons[0] as HTMLElement);
		expect(mockUpdateMutateAsync).toHaveBeenCalledWith({
			id: 'ep-2',
			data: { enabled: true },
		});
	});

	it('opens add endpoint modal on button click', async () => {
		const user = userEvent.setup();
		setupDefaultMocks();
		renderPage();
		await user.click(screen.getByText('Add Endpoint'));
		expect(screen.getByText('Add Webhook Endpoint')).toBeInTheDocument();
	});

	it('shows add endpoint form fields', async () => {
		const user = userEvent.setup();
		setupDefaultMocks();
		renderPage();
		await user.click(screen.getByText('Add Endpoint'));
		expect(screen.getByLabelText('Name')).toBeInTheDocument();
		expect(screen.getByLabelText('Endpoint URL')).toBeInTheDocument();
		expect(screen.getByLabelText('Signing Secret')).toBeInTheDocument();
		expect(screen.getByLabelText('Retry Count')).toBeInTheDocument();
		expect(screen.getByLabelText('Timeout (seconds)')).toBeInTheDocument();
	});

	it('shows event type checkboxes in add modal', async () => {
		const user = userEvent.setup();
		setupDefaultMocks();
		renderPage();
		await user.click(screen.getByText('Add Endpoint'));
		expect(screen.getByText('Event Types')).toBeInTheDocument();
		expect(screen.getByText('Backup Started')).toBeInTheDocument();
		expect(screen.getByText('Backup Completed')).toBeInTheDocument();
		expect(screen.getByText('Agent Online')).toBeInTheDocument();
	});

	it('shows Generate button for secret', async () => {
		const user = userEvent.setup();
		setupDefaultMocks();
		renderPage();
		await user.click(screen.getByText('Add Endpoint'));
		expect(screen.getByText('Generate')).toBeInTheDocument();
	});

	it('shows cancel button in add modal', async () => {
		const user = userEvent.setup();
		setupDefaultMocks();
		renderPage();
		await user.click(screen.getByText('Add Endpoint'));
		expect(screen.getByText('Cancel')).toBeInTheDocument();
	});

	it('shows HMAC info text in add modal', async () => {
		const user = userEvent.setup();
		setupDefaultMocks();
		renderPage();
		await user.click(screen.getByText('Add Endpoint'));
		expect(
			screen.getByText('Used to sign webhook payloads with HMAC-SHA256'),
		).toBeInTheDocument();
	});

	it('opens delivery log modal on Log click', async () => {
		const user = userEvent.setup();
		setupDefaultMocks({ endpoints: [mockEndpoint] });
		renderPage();
		await user.click(screen.getByText('Log'));
		expect(
			screen.getByText('Delivery Log: Slack Integration'),
		).toBeInTheDocument();
	});

	it('shows loading state in delivery log modal', async () => {
		const user = userEvent.setup();
		setupDefaultMocks({
			endpoints: [mockEndpoint],
			deliveriesLoading: true,
		});
		renderPage();
		await user.click(screen.getByText('Log'));
		expect(screen.getByText('Loading...')).toBeInTheDocument();
	});

	it('shows empty state in delivery log modal', async () => {
		const user = userEvent.setup();
		setupDefaultMocks({
			endpoints: [mockEndpoint],
			deliveries: [],
		});
		renderPage();
		await user.click(screen.getByText('Log'));
		expect(screen.getByText('No deliveries yet')).toBeInTheDocument();
	});

	it('shows delivery log data in modal', async () => {
		const user = userEvent.setup();
		setupDefaultMocks({
			endpoints: [mockEndpoint],
			deliveries: [
				{
					id: 'del-1',
					org_id: 'org-1',
					endpoint_id: 'ep-1',
					event_type: 'backup.completed',
					payload: {},
					response_status: 200,
					attempt_number: 1,
					max_attempts: 3,
					status: 'delivered',
					created_at: '2024-06-01T12:00:00Z',
				},
				{
					id: 'del-2',
					org_id: 'org-1',
					endpoint_id: 'ep-1',
					event_type: 'backup.failed',
					payload: {},
					attempt_number: 3,
					max_attempts: 3,
					status: 'failed',
					error_message: 'Connection timeout',
					created_at: '2024-06-01T13:00:00Z',
				},
			],
		});
		renderPage();
		await user.click(screen.getByText('Log'));
		// "Backup Completed" appears in both endpoint events column and delivery log
		expect(
			screen.getAllByText('Backup Completed').length,
		).toBeGreaterThanOrEqual(2);
		expect(screen.getAllByText('Backup Failed').length).toBeGreaterThanOrEqual(
			2,
		);
		expect(screen.getByText('200')).toBeInTheDocument();
		expect(screen.getByText('Connection timeout')).toBeInTheDocument();
		expect(screen.getByText('1/3')).toBeInTheDocument();
		expect(screen.getByText('3/3')).toBeInTheDocument();
	});

	it('shows Retry button for failed deliveries', async () => {
		const user = userEvent.setup();
		setupDefaultMocks({
			endpoints: [mockEndpoint],
			deliveries: [
				{
					id: 'del-2',
					org_id: 'org-1',
					endpoint_id: 'ep-1',
					event_type: 'backup.failed',
					payload: {},
					attempt_number: 3,
					max_attempts: 3,
					status: 'failed',
					error_message: 'Timeout',
					created_at: '2024-06-01T13:00:00Z',
				},
			],
		});
		renderPage();
		await user.click(screen.getByText('Log'));
		expect(screen.getByText('Retry')).toBeInTheDocument();
	});

	it('calls retryDelivery on Retry click', async () => {
		const user = userEvent.setup();
		setupDefaultMocks({
			endpoints: [mockEndpoint],
			deliveries: [
				{
					id: 'del-2',
					org_id: 'org-1',
					endpoint_id: 'ep-1',
					event_type: 'backup.failed',
					payload: {},
					attempt_number: 3,
					max_attempts: 3,
					status: 'failed',
					error_message: 'Timeout',
					created_at: '2024-06-01T13:00:00Z',
				},
			],
		});
		renderPage();
		await user.click(screen.getByText('Log'));
		await user.click(screen.getByText('Retry'));
		expect(mockRetryMutate).toHaveBeenCalledWith('del-2');
	});

	it('does not show Retry button for delivered items', async () => {
		const user = userEvent.setup();
		setupDefaultMocks({
			endpoints: [mockEndpoint],
			deliveries: [
				{
					id: 'del-1',
					org_id: 'org-1',
					endpoint_id: 'ep-1',
					event_type: 'backup.completed',
					payload: {},
					response_status: 200,
					attempt_number: 1,
					max_attempts: 3,
					status: 'delivered',
					created_at: '2024-06-01T12:00:00Z',
				},
			],
		});
		renderPage();
		await user.click(screen.getByText('Log'));
		expect(screen.queryByText('Retry')).not.toBeInTheDocument();
	});

	it('shows delivery log table headers', async () => {
		const user = userEvent.setup();
		setupDefaultMocks({
			endpoints: [mockEndpoint],
			deliveries: [
				{
					id: 'del-1',
					org_id: 'org-1',
					endpoint_id: 'ep-1',
					event_type: 'backup.completed',
					payload: {},
					response_status: 200,
					attempt_number: 1,
					max_attempts: 3,
					status: 'delivered',
					created_at: '2024-06-01T12:00:00Z',
				},
			],
		});
		renderPage();
		await user.click(screen.getByText('Log'));
		expect(screen.getByText('Event')).toBeInTheDocument();
		expect(screen.getByText('Response')).toBeInTheDocument();
		expect(screen.getByText('Attempts')).toBeInTheDocument();
		expect(screen.getByText('Time')).toBeInTheDocument();
	});

	it('renders webhook signature verification section', () => {
		setupDefaultMocks();
		renderPage();
		expect(
			screen.getByText('Webhook Signature Verification'),
		).toBeInTheDocument();
		expect(screen.getByText('X-Keldris-Signature-256')).toBeInTheDocument();
	});

	it('shows verification code example', () => {
		setupDefaultMocks();
		renderPage();
		expect(
			screen.getByText(/verifyWebhook/, { exact: false }),
		).toBeInTheDocument();
	});

	it('shows status badge styles for delivered delivery', async () => {
		const user = userEvent.setup();
		setupDefaultMocks({
			endpoints: [mockEndpoint],
			deliveries: [
				{
					id: 'del-1',
					org_id: 'org-1',
					endpoint_id: 'ep-1',
					event_type: 'backup.completed',
					payload: {},
					response_status: 200,
					attempt_number: 1,
					max_attempts: 3,
					status: 'delivered',
					created_at: '2024-06-01T12:00:00Z',
				},
			],
		});
		renderPage();
		await user.click(screen.getByText('Log'));
		const badge = screen.getByText('delivered');
		expect(badge).toHaveClass('rounded-full');
	});

	it('shows error message for failed delivery without response status', async () => {
		const user = userEvent.setup();
		setupDefaultMocks({
			endpoints: [mockEndpoint],
			deliveries: [
				{
					id: 'del-3',
					org_id: 'org-1',
					endpoint_id: 'ep-1',
					event_type: 'agent.offline',
					payload: {},
					attempt_number: 1,
					max_attempts: 3,
					status: 'failed',
					error_message: 'DNS resolution failed',
					created_at: '2024-06-01T14:00:00Z',
				},
			],
		});
		renderPage();
		await user.click(screen.getByText('Log'));
		expect(screen.getByText('DNS resolution failed')).toBeInTheDocument();
	});

	it('shows dash for delivery with no response status or error', async () => {
		const user = userEvent.setup();
		setupDefaultMocks({
			endpoints: [mockEndpoint],
			deliveries: [
				{
					id: 'del-4',
					org_id: 'org-1',
					endpoint_id: 'ep-1',
					event_type: 'backup.started',
					payload: {},
					attempt_number: 1,
					max_attempts: 3,
					status: 'pending',
					created_at: '2024-06-01T15:00:00Z',
				},
			],
		});
		renderPage();
		await user.click(screen.getByText('Log'));
		expect(screen.getByText('-')).toBeInTheDocument();
	});
});
