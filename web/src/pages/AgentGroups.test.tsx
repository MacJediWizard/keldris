import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../hooks/useAgentGroups', () => ({
	useAgentGroups: vi.fn(),
	useCreateAgentGroup: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
	useUpdateAgentGroup: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
	useDeleteAgentGroup: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useAgentGroupMembers: vi.fn(() => ({ data: [] })),
	useAddAgentToGroup: () => ({ mutateAsync: vi.fn() }),
	useRemoveAgentFromGroup: () => ({ mutateAsync: vi.fn() }),
}));

vi.mock('../hooks/useAgents', () => ({
	useAgents: () => ({ data: [{ id: 'a1', hostname: 'server-1' }] }),
}));

import { useAgentGroups } from '../hooks/useAgentGroups';

const { default: AgentGroups } = await import('./AgentGroups');

function renderPage() {
	return render(
		<BrowserRouter>
			<AgentGroups />
		</BrowserRouter>,
	);
}

describe('AgentGroups', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		vi.mocked(useAgentGroups).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAgentGroups>);
		renderPage();
		expect(screen.getByText('Agent Groups')).toBeInTheDocument();
	});

	it('shows loading state', () => {
		vi.mocked(useAgentGroups).mockReturnValue({
			data: undefined,
			isLoading: true,
			isError: false,
		} as ReturnType<typeof useAgentGroups>);
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows empty state', () => {
		vi.mocked(useAgentGroups).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAgentGroups>);
		renderPage();
		expect(screen.getByText('No agent groups')).toBeInTheDocument();
	});

	it('renders groups', () => {
		vi.mocked(useAgentGroups).mockReturnValue({
			data: [
				{
					id: 'g1',
					name: 'Production',
					description: 'Prod servers',
					color: '#ef4444',
					member_count: 3,
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAgentGroups>);
		renderPage();
		expect(screen.getByText('Production')).toBeInTheDocument();
		expect(screen.getByText('Prod servers')).toBeInTheDocument();
	});

	it('shows error state', () => {
		vi.mocked(useAgentGroups).mockReturnValue({
			data: undefined,
			isLoading: false,
			isError: true,
		} as ReturnType<typeof useAgentGroups>);
		renderPage();
		expect(screen.getByText('Failed to load groups')).toBeInTheDocument();
	});

	it('shows create button', () => {
		vi.mocked(useAgentGroups).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAgentGroups>);
		renderPage();
		expect(screen.getByText('Create Group')).toBeInTheDocument();
	});

	it('opens create modal', async () => {
		const user = userEvent.setup();
		vi.mocked(useAgentGroups).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAgentGroups>);
		renderPage();
		await user.click(screen.getByText('Create Group'));
		expect(screen.getByText('Create Agent Group')).toBeInTheDocument();
	});
});
