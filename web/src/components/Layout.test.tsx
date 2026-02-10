import { render, screen } from '@testing-library/react';
import { type Mock, beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';

vi.mock('react-router-dom', async () => {
	const actual = await vi.importActual<typeof import('react-router-dom')>(
		'react-router-dom',
	);
	return {
		...actual,
		Outlet: () => <div data-testid="outlet">Outlet Content</div>,
		useNavigate: vi.fn().mockReturnValue(vi.fn()),
	};
});

vi.mock('../hooks/useAuth', () => ({
	useMe: vi.fn(),
	useLogout: vi.fn().mockReturnValue({ mutate: vi.fn() }),
}));

vi.mock('../hooks/useAlerts', () => ({
	useAlertCount: vi.fn(),
}));

vi.mock('../hooks/useOnboarding', () => ({
	useOnboardingStatus: vi.fn(),
}));

vi.mock('../hooks/useOrganizations', () => ({
	useOrganizations: vi.fn(),
	useSwitchOrganization: vi.fn().mockReturnValue({ mutate: vi.fn() }),
}));

vi.mock('../hooks/useLocale', () => ({
	useLocale: vi.fn().mockReturnValue({
		t: (key: string, params?: Record<string, string>) => {
			const translations: Record<string, string> = {
				'common.appName': 'Keldris',
				'common.tagline': 'Keeper of your data',
				'common.loading': 'Loading...',
				'common.signOut': 'Sign Out',
				'common.selectOrg': 'Select Organization',
				'common.version': `v${params?.version ?? '0.0.1'}`,
				'nav.dashboard': 'Dashboard',
				'nav.agents': 'Agents',
				'nav.repositories': 'Repositories',
				'nav.schedules': 'Schedules',
				'nav.backups': 'Backups',
				'nav.restore': 'Restore',
				'nav.alerts': 'Alerts',
				'nav.notifications': 'Notifications',
				'nav.auditLogs': 'Audit Logs',
				'nav.storageStats': 'Storage Stats',
				'nav.costs': 'Costs',
				'nav.organization': 'Organization',
				'nav.members': 'Members',
				'nav.settings': 'Settings',
				'org.organizations': 'Organizations',
				'org.createOrganization': 'Create Organization',
			};
			return translations[key] || key;
		},
		formatRelativeTime: (d: string) => d || 'Never',
	}),
}));

vi.mock('./features/LanguageSelector', () => ({
	LanguageSelector: () => (
		<div data-testid="language-selector">Language Selector</div>
	),
}));

import { useNavigate } from 'react-router-dom';
import { useAlertCount } from '../hooks/useAlerts';
import { useMe } from '../hooks/useAuth';
import { useOnboardingStatus } from '../hooks/useOnboarding';
import { useOrganizations } from '../hooks/useOrganizations';
import { Layout } from './Layout';

const mockUseMe = useMe as Mock;
const mockUseAlertCount = useAlertCount as Mock;
const mockUseOnboardingStatus = useOnboardingStatus as Mock;
const mockUseOrganizations = useOrganizations as Mock;
const mockUseNavigate = useNavigate as Mock;

function renderLayout(initialRoute = '/') {
	return render(
		<MemoryRouter initialEntries={[initialRoute]}>
			<Layout />
		</MemoryRouter>,
	);
}

function setupDefaultMocks(overrides?: {
	user?: Record<string, unknown> | null;
	isLoading?: boolean;
	isError?: boolean;
	alertCount?: number | undefined;
	onboardingStatus?: Record<string, unknown> | null;
	onboardingLoading?: boolean;
	organizations?: Array<Record<string, unknown>> | null;
	orgsLoading?: boolean;
}) {
	mockUseMe.mockReturnValue({
		data: overrides?.user ?? {
			id: 'user-1',
			email: 'admin@example.com',
			name: 'Admin User',
			current_org_id: 'org-1',
			current_org_role: 'admin',
		},
		isLoading: overrides?.isLoading ?? false,
		isError: overrides?.isError ?? false,
	});

	mockUseAlertCount.mockReturnValue({
		data: overrides?.alertCount ?? 0,
	});

	mockUseOnboardingStatus.mockReturnValue({
		data: overrides?.onboardingStatus ?? {
			needs_onboarding: false,
			current_step: 'complete',
			completed_steps: [],
			skipped: false,
			is_complete: true,
		},
		isLoading: overrides?.onboardingLoading ?? false,
	});

	mockUseOrganizations.mockReturnValue({
		data: overrides?.organizations ?? [
			{ id: 'org-1', name: 'Acme Corp', slug: 'acme', role: 'admin' },
		],
		isLoading: overrides?.orgsLoading ?? false,
	});
}

describe('Layout', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		mockUseNavigate.mockReturnValue(vi.fn());
		setupDefaultMocks();
	});

	it('shows loading screen when auth is loading', () => {
		setupDefaultMocks({ isLoading: true });
		renderLayout();
		expect(screen.getByText('Loading...')).toBeInTheDocument();
		expect(screen.queryByText('Dashboard')).not.toBeInTheDocument();
	});

	it('shows loading screen on auth error', () => {
		setupDefaultMocks({ isError: true });
		renderLayout();
		expect(screen.getByText('Loading...')).toBeInTheDocument();
	});

	it('shows loading screen when onboarding is loading', () => {
		setupDefaultMocks({ onboardingLoading: true });
		renderLayout();
		expect(screen.getByText('Loading...')).toBeInTheDocument();
	});

	it('renders sidebar navigation items', () => {
		renderLayout();
		expect(screen.getByText('Dashboard')).toBeInTheDocument();
		expect(screen.getByText('Agents')).toBeInTheDocument();
		expect(screen.getByText('Repositories')).toBeInTheDocument();
		expect(screen.getByText('Schedules')).toBeInTheDocument();
		expect(screen.getByText('Backups')).toBeInTheDocument();
		expect(screen.getByText('Restore')).toBeInTheDocument();
		expect(screen.getByText('Alerts')).toBeInTheDocument();
		expect(screen.getByText('Notifications')).toBeInTheDocument();
		expect(screen.getByText('Audit Logs')).toBeInTheDocument();
		expect(screen.getByText('Storage Stats')).toBeInTheDocument();
		expect(screen.getByText('Costs')).toBeInTheDocument();
	});

	it('renders admin navigation for admin users', () => {
		renderLayout();
		expect(screen.getByText('Organization')).toBeInTheDocument();
		expect(screen.getByText('Members')).toBeInTheDocument();
		expect(screen.getByText('Settings')).toBeInTheDocument();
		expect(screen.getByText('SSO Group Sync')).toBeInTheDocument();
	});

	it('renders admin navigation for owner users', () => {
		setupDefaultMocks({
			user: {
				id: 'user-1',
				email: 'owner@example.com',
				name: 'Owner User',
				current_org_id: 'org-1',
				current_org_role: 'owner',
			},
		});
		renderLayout();
		expect(screen.getByText('Organization')).toBeInTheDocument();
		expect(screen.getByText('Members')).toBeInTheDocument();
	});

	it('hides admin navigation for non-admin users', () => {
		setupDefaultMocks({
			user: {
				id: 'user-2',
				email: 'member@example.com',
				name: 'Regular User',
				current_org_id: 'org-1',
				current_org_role: 'member',
			},
		});
		renderLayout();
		expect(screen.queryByText('Organization')).not.toBeInTheDocument();
		expect(screen.queryByText('Members')).not.toBeInTheDocument();
		expect(screen.queryByText('Settings')).not.toBeInTheDocument();
		expect(screen.queryByText('SSO Group Sync')).not.toBeInTheDocument();
	});

	it('renders header with user initial', () => {
		renderLayout();
		expect(screen.getByText('A')).toBeInTheDocument();
	});

	it('renders the outlet for nested routes', () => {
		renderLayout();
		expect(screen.getByTestId('outlet')).toBeInTheDocument();
	});

	it('renders the language selector', () => {
		renderLayout();
		expect(screen.getByTestId('language-selector')).toBeInTheDocument();
	});

	it('renders app name and tagline', () => {
		renderLayout();
		expect(screen.getByText('Keldris')).toBeInTheDocument();
		expect(screen.getByText('Keeper of your data')).toBeInTheDocument();
	});

	it('renders version in sidebar footer', () => {
		renderLayout();
		expect(screen.getByText('v0.0.1')).toBeInTheDocument();
	});

	it('shows alert badge with count', () => {
		setupDefaultMocks({ alertCount: 5 });
		renderLayout();
		expect(screen.getByText('5')).toBeInTheDocument();
	});

	it('shows 99+ when alert count exceeds 99', () => {
		setupDefaultMocks({ alertCount: 150 });
		renderLayout();
		expect(screen.getByText('99+')).toBeInTheDocument();
	});

	it('hides alert badge when count is 0', () => {
		setupDefaultMocks({ alertCount: 0 });
		renderLayout();
		expect(screen.queryByText('0')).not.toBeInTheDocument();
	});

	it('redirects to onboarding when needed', () => {
		const navigateFn = vi.fn();
		mockUseNavigate.mockReturnValue(navigateFn);
		setupDefaultMocks({
			onboardingStatus: {
				needs_onboarding: true,
				current_step: 'org',
				completed_steps: [],
				skipped: false,
				is_complete: false,
			},
		});
		renderLayout();
		expect(navigateFn).toHaveBeenCalledWith('/onboarding');
	});

	it('does not redirect to onboarding when already on onboarding page', () => {
		const navigateFn = vi.fn();
		mockUseNavigate.mockReturnValue(navigateFn);
		setupDefaultMocks({
			onboardingStatus: {
				needs_onboarding: true,
				current_step: 'org',
				completed_steps: [],
				skipped: false,
				is_complete: false,
			},
		});
		renderLayout('/onboarding');
		expect(navigateFn).not.toHaveBeenCalled();
	});

	it('does not redirect when onboarding is not needed', () => {
		const navigateFn = vi.fn();
		mockUseNavigate.mockReturnValue(navigateFn);
		renderLayout();
		expect(navigateFn).not.toHaveBeenCalled();
	});

	it('highlights active navigation item on root path', () => {
		renderLayout('/');
		const dashboardLink = screen.getByText('Dashboard').closest('a');
		expect(dashboardLink).toHaveClass('bg-indigo-600');
		const agentsLink = screen.getByText('Agents').closest('a');
		expect(agentsLink).not.toHaveClass('bg-indigo-600');
	});

	it('highlights active navigation item on agents path', () => {
		renderLayout('/agents');
		const agentsLink = screen.getByText('Agents').closest('a');
		expect(agentsLink).toHaveClass('bg-indigo-600');
	});

	it('renders OrgSwitcher with current org name', () => {
		setupDefaultMocks({
			organizations: [
				{ id: 'org-1', name: 'Acme Corp', slug: 'acme', role: 'admin' },
				{ id: 'org-2', name: 'Beta Inc', slug: 'beta', role: 'member' },
			],
		});
		renderLayout();
		expect(screen.getByText('Acme Corp')).toBeInTheDocument();
	});

	it('hides OrgSwitcher when no organizations', () => {
		setupDefaultMocks({ organizations: [] });
		renderLayout();
		expect(screen.queryByText('Select Organization')).not.toBeInTheDocument();
	});

	it('uses email initial when user has no name', () => {
		setupDefaultMocks({
			user: {
				id: 'user-1',
				email: 'test@example.com',
				current_org_id: 'org-1',
				current_org_role: 'member',
			},
		});
		renderLayout();
		expect(screen.getByText('T')).toBeInTheDocument();
	});
});
