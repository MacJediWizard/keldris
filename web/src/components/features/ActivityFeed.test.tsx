import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useActivity', () => ({
	useRecentActivity: vi.fn(),
	useActivityFeed: vi.fn(() => ({ events: [], isConnected: false })),
}));

vi.mock('../../hooks/useLocale', () => ({
	useLocale: () => ({
		t: (_key: string, defaultValue: string) => defaultValue,
		locale: 'en',
		formatRelativeTime: (_d: unknown) => 'just now',
		formatDate: (_d: unknown) => 'date',
		formatTime: (_d: unknown) => 'time',
		formatDateTime: (_d: unknown) => 'datetime',
	}),
}));

import { useRecentActivity } from '../../hooks/useActivity';
import { ActivityFeedFull, ActivityFeedWidget } from './ActivityFeed';

function setEvents(events: unknown[], isLoading = false) {
	vi.mocked(useRecentActivity).mockReturnValue({
		data: events,
		isLoading,
	} as never);
}

function withRouter(ui: React.ReactNode) {
	return <MemoryRouter>{ui}</MemoryRouter>;
}

describe('ActivityFeedWidget', () => {
	it('renders header', () => {
		setEvents([]);
		render(withRouter(<ActivityFeedWidget enableRealtime={false} />));
		expect(screen.getByText('Activity Feed')).toBeDefined();
	});

	it('renders events', () => {
		setEvents([
			{
				id: '1',
				event_type: 'backup.completed',
				category: 'backup',
				description: 'Backup of /var/www completed',
				timestamp: new Date().toISOString(),
			},
		]);
		render(withRouter(<ActivityFeedWidget enableRealtime={false} />));
		expect(screen.getByText('Backup of /var/www completed')).toBeDefined();
	});
});

describe('ActivityFeedFull', () => {
	it('renders', () => {
		setEvents([]);
		const { container } = render(withRouter(<ActivityFeedFull />));
		expect(container.firstChild).not.toBeNull();
	});
});
