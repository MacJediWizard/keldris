import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../hooks/usePolicies', () => ({
	usePolicies: vi.fn(),
	useCreatePolicy: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
	useDeletePolicy: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useApplyPolicy: () => ({ mutateAsync: vi.fn(), isPending: false }),
	usePolicySchedules: vi.fn(() => ({ data: [], isLoading: false })),
}));

vi.mock('../hooks/useAgents', () => ({
	useAgents: () => ({
		data: [{ id: 'a1', hostname: 'server-1' }],
		isLoading: false,
	}),
}));

vi.mock('../hooks/useRepositories', () => ({
	useRepositories: () => ({
		data: [{ id: 'r1', name: 'repo-1' }],
		isLoading: false,
	}),
}));

import { usePolicies } from '../hooks/usePolicies';

// Dynamic import of component
const { default: Policies } = await import('./Policies');

function renderPage() {
	return render(
		<BrowserRouter>
			<Policies />
		</BrowserRouter>,
	);
}

describe('Policies', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		vi.mocked(usePolicies).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof usePolicies>);
		renderPage();
		expect(screen.getByText('Policies')).toBeInTheDocument();
	});

	it('shows loading state', () => {
		vi.mocked(usePolicies).mockReturnValue({
			data: undefined,
			isLoading: true,
			isError: false,
		} as ReturnType<typeof usePolicies>);
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows error state', () => {
		vi.mocked(usePolicies).mockReturnValue({
			data: undefined,
			isLoading: false,
			isError: true,
		} as ReturnType<typeof usePolicies>);
		renderPage();
		expect(screen.getByText('Failed to load policies')).toBeInTheDocument();
	});

	it('shows empty state', () => {
		vi.mocked(usePolicies).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof usePolicies>);
		renderPage();
		expect(screen.getByText('No policies created yet')).toBeInTheDocument();
	});

	it('renders policies', () => {
		vi.mocked(usePolicies).mockReturnValue({
			data: [
				{
					id: '1',
					name: 'Daily Backup',
					description: 'Runs daily',
					paths: ['/home'],
					excludes: [],
					retention: { keep_last: 7 },
					cron: '0 0 * * *',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof usePolicies>);
		renderPage();
		expect(screen.getByText('Daily Backup')).toBeInTheDocument();
	});

	it('shows create button', () => {
		vi.mocked(usePolicies).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof usePolicies>);
		renderPage();
		expect(screen.getByText('Create Policy')).toBeInTheDocument();
	});

	it('opens create modal', async () => {
		const user = userEvent.setup();
		vi.mocked(usePolicies).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof usePolicies>);
		renderPage();
		await user.click(screen.getByText('Create Policy'));
		expect(screen.getAllByText('Create Policy').length).toBeGreaterThan(1);
	});

	it('shows subtitle', () => {
		vi.mocked(usePolicies).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof usePolicies>);
		renderPage();
		expect(
			screen.getByText('Create reusable backup configuration templates'),
		).toBeInTheDocument();
	});

	it('shows table headers', () => {
		vi.mocked(usePolicies).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof usePolicies>);
		renderPage();
		expect(screen.getByText('Name')).toBeInTheDocument();
		expect(screen.getByText('Paths')).toBeInTheDocument();
		expect(screen.getByText('Created')).toBeInTheDocument();
	});

	it('renders multiple policies', () => {
		vi.mocked(usePolicies).mockReturnValue({
			data: [
				{
					id: '1',
					name: 'Daily Backup',
					description: 'Runs daily',
					paths: ['/home'],
					excludes: [],
					retention: { keep_last: 7 },
					cron: '0 0 * * *',
					created_at: '2024-01-01T00:00:00Z',
				},
				{
					id: '2',
					name: 'Weekly Archive',
					description: 'Weekly',
					paths: ['/data'],
					excludes: [],
					retention: { keep_last: 4 },
					cron: '0 0 * * 0',
					created_at: '2024-02-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof usePolicies>);
		renderPage();
		expect(screen.getByText('Daily Backup')).toBeInTheDocument();
		expect(screen.getByText('Weekly Archive')).toBeInTheDocument();
	});

	it('shows cron expression for policies', () => {
		vi.mocked(usePolicies).mockReturnValue({
			data: [
				{
					id: '1',
					name: 'Test Policy',
					description: '',
					paths: ['/home'],
					excludes: [],
					retention: {},
					cron: '0 0 * * *',
					cron_expression: '0 0 * * *',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof usePolicies>);
		renderPage();
		expect(screen.getByText('0 0 * * *')).toBeInTheDocument();
	});

	it('shows path count for policies', () => {
		vi.mocked(usePolicies).mockReturnValue({
			data: [
				{
					id: '1',
					name: 'Path Policy',
					description: '',
					paths: ['/home', '/data'],
					excludes: [],
					retention: {},
					cron: '0 0 * * *',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof usePolicies>);
		renderPage();
		expect(screen.getByText('2 path(s)')).toBeInTheDocument();
	});

	it('shows empty description in empty state', () => {
		vi.mocked(usePolicies).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof usePolicies>);
		renderPage();
		expect(
			screen.getByText(
				/Create a policy to define reusable backup configurations/,
			),
		).toBeInTheDocument();
	});

	it('shows create modal form fields', async () => {
		const user = userEvent.setup();
		vi.mocked(usePolicies).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof usePolicies>);
		renderPage();
		await user.click(screen.getByText('Create Policy'));
		expect(screen.getByLabelText('Description')).toBeInTheDocument();
		expect(screen.getByLabelText('Default Paths')).toBeInTheDocument();
	});

	it('shows actions menu trigger for policies', () => {
		vi.mocked(usePolicies).mockReturnValue({
			data: [
				{
					id: '1',
					name: 'Test',
					description: '',
					paths: ['/home'],
					excludes: [],
					retention: {},
					cron: '0 0 * * *',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof usePolicies>);
		renderPage();
		const actionButtons = document.querySelectorAll('svg[viewBox="0 0 20 20"]');
		expect(actionButtons.length).toBeGreaterThan(0);
	});
});
