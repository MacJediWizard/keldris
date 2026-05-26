import { screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useAuth', () => ({
	useMe: vi.fn(),
}));

vi.mock('../hooks/usePlanLimits', () => ({
	usePlanLimits: vi.fn(),
}));

vi.mock('../hooks/useLegalHolds', () => ({
	useLegalHolds: vi.fn(),
	useDeleteLegalHold: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock('../components/features/UpgradePrompt', () => ({
	UpgradePrompt: ({ feature }: { feature: string }) => (
		<div data-testid="upgrade-prompt">Upgrade required for {feature}</div>
	),
}));

import { useMe } from '../hooks/useAuth';
import { useLegalHolds } from '../hooks/useLegalHolds';
import { usePlanLimits } from '../hooks/usePlanLimits';
import { LegalHolds } from './LegalHolds';

const adminUser = {
	id: 'user-1',
	current_org_id: 'org-1',
	current_org_role: 'admin',
};

function setupMocks(overrides?: {
	user?: Record<string, unknown>;
	holds?: unknown[] | undefined;
	holdsLoading?: boolean;
	holdsError?: boolean;
	hasFeature?: boolean;
	limitsLoading?: boolean;
}) {
	vi.mocked(useMe).mockReturnValue({
		data: overrides?.user ?? adminUser,
	} as ReturnType<typeof useMe>);
	vi.mocked(usePlanLimits).mockReturnValue({
		isLoading: overrides?.limitsLoading ?? false,
		hasFeature: () => overrides?.hasFeature ?? true,
		planType: 'enterprise',
		limits: {},
		features: {},
		usage: { agentCount: 0, storageUsedBytes: 0 },
		canAddAgents: () => true,
		canAddStorage: () => true,
		getAgentLimitRemaining: () => undefined,
		getStorageLimitRemaining: () => undefined,
		isAtAgentLimit: () => false,
		isAtStorageLimit: () => false,
		getUpgradePlanFor: () => 'enterprise',
	} as ReturnType<typeof usePlanLimits>);
	vi.mocked(useLegalHolds).mockReturnValue({
		data: overrides?.holds ?? [],
		isLoading: overrides?.holdsLoading ?? false,
		isError: overrides?.holdsError ?? false,
		refetch: vi.fn(),
	} as ReturnType<typeof useLegalHolds>);
}

describe('LegalHolds', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title for admin', () => {
		setupMocks();
		renderWithProviders(<LegalHolds />);
		expect(screen.getByText('Legal Holds')).toBeInTheDocument();
	});

	it('shows upgrade prompt without legal_holds feature', () => {
		setupMocks({ hasFeature: false });
		renderWithProviders(<LegalHolds />);
		expect(screen.getByTestId('upgrade-prompt')).toBeInTheDocument();
	});

	it('shows access denied for non-admin', () => {
		setupMocks({
			user: { ...adminUser, current_org_role: 'member' },
		});
		renderWithProviders(<LegalHolds />);
		expect(screen.getByText('Access Denied')).toBeInTheDocument();
	});

	it('shows loading spinner when limits loading', () => {
		setupMocks({ limitsLoading: true });
		renderWithProviders(<LegalHolds />);
		const spinner = document.querySelector('.animate-spin');
		expect(spinner).toBeInTheDocument();
	});

	it('shows empty state', () => {
		setupMocks({ holds: [] });
		renderWithProviders(<LegalHolds />);
		expect(screen.getByText('No Legal Holds')).toBeInTheDocument();
	});

	it('shows error state', () => {
		setupMocks({ holdsError: true });
		renderWithProviders(<LegalHolds />);
		expect(screen.getByText('Failed to load legal holds')).toBeInTheDocument();
	});

	it('renders holds list', () => {
		setupMocks({
			holds: [
				{
					id: 'hold-1',
					snapshot_id: 'snap-abc-1234567890',
					reason: 'Litigation hold for case 2024-001',
					placed_by_name: 'Admin User',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
		});
		renderWithProviders(<LegalHolds />);
		expect(
			screen.getByText('Litigation hold for case 2024-001'),
		).toBeInTheDocument();
		expect(screen.getByText('Admin User')).toBeInTheDocument();
	});
});
