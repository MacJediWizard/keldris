import { screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useAuth', () => ({
	useMe: vi.fn(),
}));

vi.mock('../hooks/useIPAllowlists', () => ({
	useIPAllowlists: vi.fn(),
	useIPAllowlistSettings: vi.fn(),
	useIPBlockedAttempts: vi.fn(),
	useCreateIPAllowlist: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
	useUpdateIPAllowlist: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
	useDeleteIPAllowlist: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
	useUpdateIPAllowlistSettings: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
	}),
}));

import { useMe } from '../hooks/useAuth';
import {
	useIPAllowlistSettings,
	useIPAllowlists,
	useIPBlockedAttempts,
} from '../hooks/useIPAllowlists';
import { IPAllowlistSettings } from './IPAllowlistSettings';

const adminUser = {
	id: 'user-1',
	current_org_id: 'org-1',
	current_org_role: 'admin',
};

function setupMocks(overrides?: {
	user?: Record<string, unknown>;
	allowlists?: unknown[] | undefined;
	allowlistsLoading?: boolean;
	settings?: Record<string, unknown> | undefined;
	settingsLoading?: boolean;
	attempts?: Record<string, unknown> | undefined;
	attemptsLoading?: boolean;
}) {
	vi.mocked(useMe).mockReturnValue({
		data: overrides?.user ?? adminUser,
	} as ReturnType<typeof useMe>);
	vi.mocked(useIPAllowlists).mockReturnValue({
		data: overrides?.allowlists ?? [],
		isLoading: overrides?.allowlistsLoading ?? false,
	} as ReturnType<typeof useIPAllowlists>);
	vi.mocked(useIPAllowlistSettings).mockReturnValue({
		data: overrides?.settings ?? {
			enabled: false,
			enforce_for_ui: true,
			enforce_for_agent: true,
			allow_admin_bypass: true,
		},
		isLoading: overrides?.settingsLoading ?? false,
	} as ReturnType<typeof useIPAllowlistSettings>);
	vi.mocked(useIPBlockedAttempts).mockReturnValue({
		data: overrides?.attempts ?? { attempts: [], total_count: 0 },
		isLoading: overrides?.attemptsLoading ?? false,
	} as ReturnType<typeof useIPBlockedAttempts>);
}

describe('IPAllowlistSettings', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		setupMocks();
		renderWithProviders(<IPAllowlistSettings />);
		expect(screen.getByText('IP Allowlist Settings')).toBeInTheDocument();
	});

	it('renders subtitle', () => {
		setupMocks();
		renderWithProviders(<IPAllowlistSettings />);
		expect(
			screen.getByText(
				'Restrict access to your organization by IP address or CIDR range',
			),
		).toBeInTheDocument();
	});

	it('shows loading skeletons', () => {
		setupMocks({ allowlistsLoading: true, settingsLoading: true });
		renderWithProviders(<IPAllowlistSettings />);
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows empty allowlists message', () => {
		setupMocks({ allowlists: [] });
		renderWithProviders(<IPAllowlistSettings />);
		expect(screen.getByText('No IP ranges configured yet')).toBeInTheDocument();
	});

	it('shows empty blocked attempts message', () => {
		setupMocks();
		renderWithProviders(<IPAllowlistSettings />);
		expect(
			screen.getByText('No blocked attempts recorded'),
		).toBeInTheDocument();
	});

	it('renders allowlist entries', () => {
		setupMocks({
			allowlists: [
				{
					id: 'al-1',
					cidr: '10.0.0.0/8',
					description: 'Internal network',
					type: 'both',
					enabled: true,
				},
			],
		});
		renderWithProviders(<IPAllowlistSettings />);
		expect(screen.getByText('10.0.0.0/8')).toBeInTheDocument();
		expect(screen.getByText('Internal network')).toBeInTheDocument();
	});

	it('hides Add IP Range button for non-admin', () => {
		setupMocks({
			user: { ...adminUser, current_org_role: 'member' },
			allowlists: [],
		});
		renderWithProviders(<IPAllowlistSettings />);
		expect(screen.queryByText('Add IP Range')).not.toBeInTheDocument();
	});
});
