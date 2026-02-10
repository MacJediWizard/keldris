import { render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useMaintenance', () => ({
	useActiveMaintenance: vi.fn(),
}));

import { useActiveMaintenance } from '../../hooks/useMaintenance';
import { MaintenanceBanner } from './MaintenanceBanner';

describe('MaintenanceBanner', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		vi.useFakeTimers();
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	it('renders nothing when no maintenance data', () => {
		vi.mocked(useActiveMaintenance).mockReturnValue({
			data: undefined,
		} as ReturnType<typeof useActiveMaintenance>);
		const { container } = render(<MaintenanceBanner />);
		expect(container.innerHTML).toBe('');
	});

	it('renders nothing when no active or upcoming maintenance', () => {
		vi.mocked(useActiveMaintenance).mockReturnValue({
			data: { active: null, upcoming: null },
		} as ReturnType<typeof useActiveMaintenance>);
		const { container } = render(<MaintenanceBanner />);
		expect(container.innerHTML).toBe('');
	});

	it('shows active maintenance banner with amber background', () => {
		const futureDate = new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString(); // 2 hours from now
		vi.mocked(useActiveMaintenance).mockReturnValue({
			data: {
				active: {
					id: 'mw-1',
					title: 'Database Migration',
					message: 'Upgrading DB',
					starts_at: '2024-01-01T00:00:00Z',
					ends_at: futureDate,
				},
				upcoming: null,
			},
		} as ReturnType<typeof useActiveMaintenance>);
		render(<MaintenanceBanner />);
		expect(screen.getByText(/Maintenance in progress/)).toBeInTheDocument();
		expect(screen.getByText('Database Migration')).toBeInTheDocument();
		expect(screen.getByText('- Upgrading DB')).toBeInTheDocument();
		expect(screen.getByText(/Ends in/)).toBeInTheDocument();
	});

	it('shows upcoming maintenance banner with blue background', () => {
		const futureDate = new Date(Date.now() + 30 * 60 * 1000).toISOString(); // 30 minutes from now
		vi.mocked(useActiveMaintenance).mockReturnValue({
			data: {
				active: null,
				upcoming: {
					id: 'mw-2',
					title: 'Server Restart',
					starts_at: futureDate,
					ends_at: new Date(Date.now() + 90 * 60 * 1000).toISOString(),
				},
			},
		} as ReturnType<typeof useActiveMaintenance>);
		render(<MaintenanceBanner />);
		expect(screen.getByText(/Scheduled maintenance/)).toBeInTheDocument();
		expect(screen.getByText('Server Restart')).toBeInTheDocument();
		expect(screen.getByText(/Starts in/)).toBeInTheDocument();
	});

	it('displays time left in hours and minutes format', () => {
		const futureDate = new Date(
			Date.now() + 3 * 60 * 60 * 1000 + 15 * 60 * 1000,
		).toISOString(); // 3h 15m from now
		vi.mocked(useActiveMaintenance).mockReturnValue({
			data: {
				active: {
					id: 'mw-1',
					title: 'Update',
					starts_at: '2024-01-01T00:00:00Z',
					ends_at: futureDate,
				},
				upcoming: null,
			},
		} as ReturnType<typeof useActiveMaintenance>);
		render(<MaintenanceBanner />);
		expect(screen.getByText(/Ends in 3h 15m/)).toBeInTheDocument();
	});

	it('displays time left in minutes and seconds format', () => {
		const futureDate = new Date(
			Date.now() + 5 * 60 * 1000 + 30 * 1000,
		).toISOString(); // 5m 30s from now
		vi.mocked(useActiveMaintenance).mockReturnValue({
			data: {
				active: {
					id: 'mw-1',
					title: 'Quick Fix',
					starts_at: '2024-01-01T00:00:00Z',
					ends_at: futureDate,
				},
				upcoming: null,
			},
		} as ReturnType<typeof useActiveMaintenance>);
		render(<MaintenanceBanner />);
		expect(screen.getByText(/Ends in 5m 30s/)).toBeInTheDocument();
	});

	it('displays time left in seconds format', () => {
		const futureDate = new Date(Date.now() + 45 * 1000).toISOString(); // 45s from now
		vi.mocked(useActiveMaintenance).mockReturnValue({
			data: {
				active: {
					id: 'mw-1',
					title: 'Almost Done',
					starts_at: '2024-01-01T00:00:00Z',
					ends_at: futureDate,
				},
				upcoming: null,
			},
		} as ReturnType<typeof useActiveMaintenance>);
		render(<MaintenanceBanner />);
		expect(screen.getByText(/Ends in 45s/)).toBeInTheDocument();
	});

	it('hides message text when no message', () => {
		const futureDate = new Date(Date.now() + 60 * 60 * 1000).toISOString();
		vi.mocked(useActiveMaintenance).mockReturnValue({
			data: {
				active: {
					id: 'mw-1',
					title: 'Silent Update',
					starts_at: '2024-01-01T00:00:00Z',
					ends_at: futureDate,
				},
				upcoming: null,
			},
		} as ReturnType<typeof useActiveMaintenance>);
		render(<MaintenanceBanner />);
		expect(screen.queryByText(/^-/)).not.toBeInTheDocument();
	});

	it('prefers active over upcoming maintenance', () => {
		const futureDate = new Date(Date.now() + 60 * 60 * 1000).toISOString();
		vi.mocked(useActiveMaintenance).mockReturnValue({
			data: {
				active: {
					id: 'mw-1',
					title: 'Active Window',
					starts_at: '2024-01-01T00:00:00Z',
					ends_at: futureDate,
				},
				upcoming: {
					id: 'mw-2',
					title: 'Upcoming Window',
					starts_at: futureDate,
					ends_at: new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString(),
				},
			},
		} as ReturnType<typeof useActiveMaintenance>);
		render(<MaintenanceBanner />);
		expect(screen.getByText('Active Window')).toBeInTheDocument();
		expect(screen.getByText(/Maintenance in progress/)).toBeInTheDocument();
	});
});
