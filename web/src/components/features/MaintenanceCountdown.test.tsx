import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useAuth', () => ({
	useMe: vi.fn(),
}));

vi.mock('../../hooks/useMaintenance', () => ({
	useActiveMaintenance: vi.fn(),
	useEmergencyOverride: vi.fn(),
}));

import { useMe } from '../../hooks/useAuth';
import {
	useActiveMaintenance,
	useEmergencyOverride,
} from '../../hooks/useMaintenance';
import { MaintenanceCountdown } from './MaintenanceCountdown';

const overrideMutate = vi.fn();

function setMocks({
	role = 'member',
	maintenance = null as unknown,
	overridePending = false,
}: {
	role?: string;
	maintenance?: unknown;
	overridePending?: boolean;
} = {}) {
	vi.mocked(useMe).mockReturnValue({
		data: { current_org_role: role },
	} as never);
	vi.mocked(useActiveMaintenance).mockReturnValue({
		data: maintenance,
	} as never);
	vi.mocked(useEmergencyOverride).mockReturnValue({
		mutate: overrideMutate,
		isPending: overridePending,
	} as never);
}

describe('MaintenanceCountdown', () => {
	it('renders nothing when no maintenance', () => {
		setMocks();
		const { container } = render(<MaintenanceCountdown />);
		expect(container.firstChild).toBeNull();
	});

	it('renders nothing when only countdown but no active/upcoming', () => {
		setMocks({
			maintenance: {
				active: null,
				upcoming: null,
				show_countdown: true,
				countdown_to: new Date(Date.now() + 60_000).toISOString(),
			},
		});
		const { container } = render(<MaintenanceCountdown />);
		expect(container.firstChild).toBeNull();
	});

	it('renders active maintenance banner', () => {
		setMocks({
			maintenance: {
				active: {
					id: 'm1',
					title: 'DB migration',
					message: 'Brief outage',
					ends_at: new Date(Date.now() + 60 * 60_000).toISOString(),
				},
				upcoming: null,
				read_only_mode: false,
			},
		});
		render(<MaintenanceCountdown />);
		expect(screen.getByText(/Maintenance in progress/)).toBeDefined();
		expect(screen.getByText('DB migration')).toBeDefined();
	});

	it('renders read-only banner with override button for admin', () => {
		setMocks({
			role: 'admin',
			maintenance: {
				active: {
					id: 'm1',
					title: 'Locked window',
					message: '',
					ends_at: new Date(Date.now() + 60 * 60_000).toISOString(),
				},
				upcoming: null,
				read_only_mode: true,
			},
		});
		render(<MaintenanceCountdown />);
		expect(screen.getByText(/Read-only mode active/)).toBeDefined();
		expect(
			screen.getByRole('button', { name: 'Emergency Override' }),
		).toBeDefined();
	});

	it('hides override button for non-admin', () => {
		setMocks({
			role: 'member',
			maintenance: {
				active: {
					id: 'm1',
					title: 'Locked window',
					message: '',
					ends_at: new Date(Date.now() + 60 * 60_000).toISOString(),
				},
				upcoming: null,
				read_only_mode: true,
			},
		});
		render(<MaintenanceCountdown />);
		expect(
			screen.queryByRole('button', { name: 'Emergency Override' }),
		).toBeNull();
	});

	it('renders upcoming maintenance banner', () => {
		setMocks({
			maintenance: {
				active: null,
				upcoming: {
					id: 'm1',
					title: 'Planned outage',
					message: 'Coming soon',
					starts_at: new Date(Date.now() + 60 * 60_000).toISOString(),
				},
				read_only_mode: false,
			},
		});
		render(<MaintenanceCountdown />);
		expect(screen.getByText(/Scheduled maintenance/)).toBeDefined();
		expect(screen.getByText('Planned outage')).toBeDefined();
	});
});
