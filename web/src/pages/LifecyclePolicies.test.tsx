import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { LifecyclePolicies } from './LifecyclePolicies';

const mockCreateMutateAsync = vi.fn();
const mockDeleteMutateAsync = vi.fn();
const mockUpdateMutateAsync = vi.fn();
const mockDryRunMutateAsync = vi.fn();

vi.mock('../hooks/useAuth', () => ({
	useMe: vi.fn(),
}));

vi.mock('../hooks/usePlanLimits', () => ({
	usePlanLimits: vi.fn(),
}));

vi.mock('../hooks/useLifecyclePolicies', () => ({
	useLifecyclePolicies: vi.fn(),
	useOrgLifecycleDeletions: vi.fn(),
	useCreateLifecyclePolicy: () => ({
		mutateAsync: mockCreateMutateAsync,
		isPending: false,
	}),
	useDeleteLifecyclePolicy: () => ({
		mutateAsync: mockDeleteMutateAsync,
		isPending: false,
	}),
	useUpdateLifecyclePolicy: () => ({
		mutateAsync: mockUpdateMutateAsync,
		isPending: false,
	}),
	useLifecycleDryRun: () => ({
		mutateAsync: mockDryRunMutateAsync,
		isPending: false,
	}),
}));

vi.mock('../components/features/UpgradePrompt', () => ({
	UpgradePrompt: ({ feature }: { feature: string }) => (
		<div data-testid="upgrade-prompt">Upgrade required for {feature}</div>
	),
}));

import { useMe } from '../hooks/useAuth';
import {
	useLifecyclePolicies,
	useOrgLifecycleDeletions,
} from '../hooks/useLifecyclePolicies';
import { usePlanLimits } from '../hooks/usePlanLimits';

function renderPage() {
	return render(
		<BrowserRouter>
			<LifecyclePolicies />
		</BrowserRouter>,
	);
}

function setupDefaultMocks(overrides?: {
	user?: Record<string, unknown>;
	policies?: unknown[] | undefined;
	policiesLoading?: boolean;
	policiesError?: boolean;
	deletions?: unknown[] | undefined;
	hasFeature?: boolean;
	limitsLoading?: boolean;
}) {
	vi.mocked(useMe).mockReturnValue({
		data: overrides?.user ?? {
			id: 'user-1',
			email: 'admin@example.com',
			name: 'Admin User',
			current_org_id: 'org-1',
			current_org_role: 'admin',
		},
	} as ReturnType<typeof useMe>);

	vi.mocked(usePlanLimits).mockReturnValue({
		isLoading: overrides?.limitsLoading ?? false,
		hasFeature: () => overrides?.hasFeature ?? true,
		planType: 'professional',
		limits: {},
		features: {},
		usage: { agentCount: 0, storageUsedBytes: 0 },
		canAddAgents: () => true,
		canAddStorage: () => true,
		getAgentLimitRemaining: () => undefined,
		getStorageLimitRemaining: () => undefined,
		isAtAgentLimit: () => false,
		isAtStorageLimit: () => false,
		getUpgradePlanFor: () => 'professional',
	} as ReturnType<typeof usePlanLimits>);

	vi.mocked(useLifecyclePolicies).mockReturnValue({
		data: overrides?.policies ?? [],
		isLoading: overrides?.policiesLoading ?? false,
		isError: overrides?.policiesError ?? false,
		refetch: vi.fn(),
	} as ReturnType<typeof useLifecyclePolicies>);

	vi.mocked(useOrgLifecycleDeletions).mockReturnValue({
		data: overrides?.deletions ?? undefined,
	} as ReturnType<typeof useOrgLifecycleDeletions>);
}

const mockPolicy = {
	id: 'policy-1',
	name: 'Standard Retention',
	description: 'Default retention policy for all data',
	status: 'active' as const,
	rules: [
		{ level: 'public' as const, retention: { min_days: 30, max_days: 90 } },
		{
			level: 'confidential' as const,
			retention: { min_days: 180, max_days: 365 },
		},
	],
	deletion_count: 42,
	bytes_reclaimed: 1073741824,
	created_by: 'user-1',
	created_at: '2024-01-01T00:00:00Z',
	updated_at: '2024-06-01T00:00:00Z',
};

const mockPolicyDraft = {
	id: 'policy-2',
	name: 'Compliance Policy',
	description: 'Strict retention for regulated data',
	status: 'draft' as const,
	rules: [
		{
			level: 'restricted' as const,
			retention: { min_days: 365, max_days: 0 },
		},
	],
	deletion_count: 0,
	bytes_reclaimed: 0,
	created_by: 'user-1',
	created_at: '2024-03-01T00:00:00Z',
	updated_at: '2024-03-01T00:00:00Z',
};

describe('LifecyclePolicies', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders page title', () => {
		setupDefaultMocks();
		renderPage();
		expect(screen.getByText('Lifecycle Policies')).toBeInTheDocument();
	});

	it('renders subtitle when admin with feature', () => {
		setupDefaultMocks();
		renderPage();
		expect(
			screen.getByText('Automated snapshot retention with compliance controls'),
		).toBeInTheDocument();
	});

	it('shows loading spinner when plan limits are loading', () => {
		setupDefaultMocks({ limitsLoading: true });
		renderPage();
		const spinner = document.querySelector('.animate-spin');
		expect(spinner).toBeInTheDocument();
	});

	it('shows upgrade prompt when feature is not available', () => {
		setupDefaultMocks({ hasFeature: false });
		renderPage();
		expect(screen.getByTestId('upgrade-prompt')).toBeInTheDocument();
		expect(
			screen.getByText('Upgrade required for lifecycle_policies'),
		).toBeInTheDocument();
	});

	it('shows access denied for non-admin users', () => {
		setupDefaultMocks({
			user: {
				id: 'user-2',
				email: 'viewer@example.com',
				name: 'Viewer User',
				current_org_id: 'org-1',
				current_org_role: 'member',
			},
		});
		renderPage();
		expect(screen.getByText('Access Denied')).toBeInTheDocument();
		expect(
			screen.getByText(
				'You must be an administrator to manage lifecycle policies.',
			),
		).toBeInTheDocument();
	});

	it('allows owner users to access the page', () => {
		setupDefaultMocks({
			user: {
				id: 'user-3',
				email: 'owner@example.com',
				name: 'Owner User',
				current_org_id: 'org-1',
				current_org_role: 'owner',
			},
		});
		renderPage();
		expect(screen.queryByText('Access Denied')).not.toBeInTheDocument();
		expect(screen.getAllByText('Create Policy').length).toBeGreaterThan(0);
	});

	it('shows loading skeleton rows when policies are loading', () => {
		setupDefaultMocks({ policiesLoading: true });
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows error state when policies fail to load', () => {
		setupDefaultMocks({ policiesError: true });
		renderPage();
		expect(
			screen.getByText('Failed to load lifecycle policies'),
		).toBeInTheDocument();
		expect(
			screen.getByText('Please try refreshing the page'),
		).toBeInTheDocument();
	});

	it('shows empty state when no policies exist', () => {
		setupDefaultMocks({ policies: [] });
		renderPage();
		expect(screen.getByText('No Lifecycle Policies')).toBeInTheDocument();
		expect(
			screen.getByText(
				'Create a lifecycle policy to automate snapshot retention.',
			),
		).toBeInTheDocument();
	});

	it('shows Create Policy button in empty state', () => {
		setupDefaultMocks({ policies: [] });
		renderPage();
		const buttons = screen.getAllByText('Create Policy');
		expect(buttons.length).toBeGreaterThan(0);
	});

	it('renders policy data in table', () => {
		setupDefaultMocks({ policies: [mockPolicy] });
		renderPage();
		expect(screen.getByText('Standard Retention')).toBeInTheDocument();
		expect(
			screen.getByText('Default retention policy for all data'),
		).toBeInTheDocument();
		expect(screen.getByText('active')).toBeInTheDocument();
		expect(screen.getByText('2 rule(s)')).toBeInTheDocument();
		expect(screen.getByText('42 deleted')).toBeInTheDocument();
	});

	it('renders multiple policies', () => {
		setupDefaultMocks({ policies: [mockPolicy, mockPolicyDraft] });
		renderPage();
		expect(screen.getByText('Standard Retention')).toBeInTheDocument();
		expect(screen.getByText('Compliance Policy')).toBeInTheDocument();
		expect(screen.getByText('active')).toBeInTheDocument();
		expect(screen.getByText('draft')).toBeInTheDocument();
	});

	it('shows classification levels in rule summary', () => {
		setupDefaultMocks({ policies: [mockPolicy] });
		renderPage();
		expect(screen.getByText('public, confidential')).toBeInTheDocument();
	});

	it('shows action buttons for policies', () => {
		setupDefaultMocks({ policies: [mockPolicy] });
		renderPage();
		expect(screen.getByText('Dry Run')).toBeInTheDocument();
		expect(screen.getByText('Disable')).toBeInTheDocument();
		expect(screen.getByText('Delete')).toBeInTheDocument();
	});

	it('shows Activate button for non-active policies', () => {
		setupDefaultMocks({ policies: [mockPolicyDraft] });
		renderPage();
		expect(screen.getByText('Activate')).toBeInTheDocument();
	});

	it('shows Disable button for active policies', () => {
		setupDefaultMocks({ policies: [mockPolicy] });
		renderPage();
		expect(screen.getByText('Disable')).toBeInTheDocument();
	});

	it('shows Refresh button', () => {
		setupDefaultMocks();
		renderPage();
		expect(screen.getByText('Refresh')).toBeInTheDocument();
	});

	it('shows delete confirmation on Delete click', async () => {
		const user = userEvent.setup();
		setupDefaultMocks({ policies: [mockPolicy] });
		renderPage();
		await user.click(screen.getByText('Delete'));
		expect(screen.getByText('Confirm')).toBeInTheDocument();
		expect(screen.getByText('Cancel')).toBeInTheDocument();
	});

	it('calls deletePolicy on confirm', async () => {
		const user = userEvent.setup();
		setupDefaultMocks({ policies: [mockPolicy] });
		renderPage();
		await user.click(screen.getByText('Delete'));
		await user.click(screen.getByText('Confirm'));
		expect(mockDeleteMutateAsync).toHaveBeenCalledWith('policy-1');
	});

	it('cancels delete on cancel click', async () => {
		const user = userEvent.setup();
		setupDefaultMocks({ policies: [mockPolicy] });
		renderPage();
		await user.click(screen.getByText('Delete'));
		await user.click(screen.getByText('Cancel'));
		expect(mockDeleteMutateAsync).not.toHaveBeenCalled();
		expect(screen.queryByText('Confirm')).not.toBeInTheDocument();
	});

	it('calls toggleStatus when Disable is clicked', async () => {
		const user = userEvent.setup();
		setupDefaultMocks({ policies: [mockPolicy] });
		renderPage();
		await user.click(screen.getByText('Disable'));
		expect(mockUpdateMutateAsync).toHaveBeenCalledWith({
			id: 'policy-1',
			data: { status: 'disabled' },
		});
	});

	it('calls toggleStatus when Activate is clicked', async () => {
		const user = userEvent.setup();
		setupDefaultMocks({ policies: [mockPolicyDraft] });
		renderPage();
		await user.click(screen.getByText('Activate'));
		expect(mockUpdateMutateAsync).toHaveBeenCalledWith({
			id: 'policy-2',
			data: { status: 'active' },
		});
	});

	it('opens create policy modal on button click', async () => {
		const user = userEvent.setup();
		setupDefaultMocks({ policies: [] });
		renderPage();
		const createButtons = screen.getAllByText('Create Policy');
		await user.click(createButtons[0]);
		expect(screen.getByText('Create Lifecycle Policy')).toBeInTheDocument();
	});

	it('shows create policy form fields', async () => {
		const user = userEvent.setup();
		setupDefaultMocks();
		renderPage();
		await user.click(screen.getAllByText('Create Policy')[0]);
		expect(screen.getByLabelText('Name')).toBeInTheDocument();
		expect(screen.getByLabelText('Description')).toBeInTheDocument();
		expect(screen.getByLabelText('Status')).toBeInTheDocument();
	});

	it('shows default retention rule in create modal', async () => {
		const user = userEvent.setup();
		setupDefaultMocks();
		renderPage();
		await user.click(screen.getAllByText('Create Policy')[0]);
		expect(
			screen.getByText('Retention Rules by Classification'),
		).toBeInTheDocument();
		expect(screen.getByText('Remove')).toBeInTheDocument();
	});

	it('shows Add Rule button in create modal', async () => {
		const user = userEvent.setup();
		setupDefaultMocks();
		renderPage();
		await user.click(screen.getAllByText('Create Policy')[0]);
		expect(screen.getByText('+ Add Rule')).toBeInTheDocument();
	});

	it('shows cancel button in create modal', async () => {
		const user = userEvent.setup();
		setupDefaultMocks();
		renderPage();
		await user.click(screen.getAllByText('Create Policy')[0]);
		expect(screen.getByText('Cancel')).toBeInTheDocument();
	});

	it('opens dry run modal on Dry Run click', async () => {
		const user = userEvent.setup();
		setupDefaultMocks({ policies: [mockPolicy] });
		renderPage();
		await user.click(screen.getByText('Dry Run'));
		expect(screen.getByText('Dry Run: Standard Retention')).toBeInTheDocument();
		expect(screen.getByText('Run Preview')).toBeInTheDocument();
	});

	it('shows dry run description text', async () => {
		const user = userEvent.setup();
		setupDefaultMocks({ policies: [mockPolicy] });
		renderPage();
		await user.click(screen.getByText('Dry Run'));
		expect(
			screen.getByText(
				'Preview what snapshots would be affected by this policy without making any changes.',
			),
		).toBeInTheDocument();
	});

	it('renders recent deletions section when deletions exist', () => {
		setupDefaultMocks({
			policies: [mockPolicy],
			deletions: [
				{
					id: 'del-1',
					org_id: 'org-1',
					policy_id: 'policy-1',
					snapshot_id:
						'abc123def456abc123def456abc123def456abc123def456abc123def456abc1',
					repository_id: 'repo-1',
					reason: 'Exceeded maximum retention of 90 days',
					size_bytes: 524288000,
					deleted_by: 'system',
					deleted_at: '2024-06-15T12:00:00Z',
				},
			],
		});
		renderPage();
		expect(screen.getByText('Recent Deletions')).toBeInTheDocument();
		expect(
			screen.getByText('Exceeded maximum retention of 90 days'),
		).toBeInTheDocument();
	});

	it('does not render recent deletions when none exist', () => {
		setupDefaultMocks({ policies: [mockPolicy], deletions: [] });
		renderPage();
		expect(screen.queryByText('Recent Deletions')).not.toBeInTheDocument();
	});

	it('renders info card about lifecycle policies', () => {
		setupDefaultMocks();
		renderPage();
		expect(screen.getByText('About Lifecycle Policies')).toBeInTheDocument();
	});

	it('shows table headers when policies exist', () => {
		setupDefaultMocks({ policies: [mockPolicy] });
		renderPage();
		expect(screen.getByText('Name')).toBeInTheDocument();
		expect(screen.getByText('Status')).toBeInTheDocument();
		expect(screen.getByText('Rules')).toBeInTheDocument();
		expect(screen.getByText('Stats')).toBeInTheDocument();
		expect(screen.getByText('Actions')).toBeInTheDocument();
	});

	it('shows table headers during loading', () => {
		setupDefaultMocks({ policiesLoading: true });
		renderPage();
		expect(screen.getByText('Name')).toBeInTheDocument();
		expect(screen.getByText('Status')).toBeInTheDocument();
	});

	it('submits create form with entered data', async () => {
		const user = userEvent.setup();
		mockCreateMutateAsync.mockResolvedValue({});
		setupDefaultMocks({ policies: [mockPolicy] });
		renderPage();
		await user.click(screen.getByText('Create Policy'));
		await user.type(screen.getByLabelText('Name'), 'Test Policy');
		await user.type(screen.getByLabelText('Description'), 'A test policy');
		const submitButton = screen
			.getAllByRole('button', { name: 'Create Policy' })
			.find((btn) => btn.getAttribute('type') === 'submit');
		expect(submitButton).toBeDefined();
		// biome-ignore lint/style/noNonNullAssertion: submitButton is asserted above
		await user.click(submitButton!);
		expect(mockCreateMutateAsync).toHaveBeenCalledWith({
			name: 'Test Policy',
			description: 'A test policy',
			status: 'draft',
			rules: [
				{
					level: 'public',
					retention: { min_days: 30, max_days: 90 },
				},
			],
		});
	});

	it('shows info card bullet points', () => {
		setupDefaultMocks();
		renderPage();
		expect(
			screen.getByText(/Minimum retention/, { exact: false }),
		).toBeInTheDocument();
		expect(
			screen.getByText(/Maximum retention/, { exact: false }),
		).toBeInTheDocument();
		expect(
			screen.getByText(/legal hold/, { exact: false }),
		).toBeInTheDocument();
		expect(screen.getByText(/dry run/, { exact: false })).toBeInTheDocument();
	});
});
