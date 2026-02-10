import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../hooks/useSchedules', () => ({
	useSchedules: vi.fn(),
	useCreateSchedule: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
	useUpdateSchedule: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useDeleteSchedule: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useRunSchedule: () => ({ mutateAsync: vi.fn(), isPending: false }),
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

vi.mock('../hooks/usePolicies', () => ({
	usePolicies: () => ({ data: [], isLoading: false }),
}));

vi.mock('../components/features/BackupScriptsEditor', () => ({
	BackupScriptsEditor: () => (
		<div data-testid="scripts-editor">Scripts Editor</div>
	),
}));

vi.mock('../components/features/MultiRepoSelector', () => ({
	MultiRepoSelector: () => <div data-testid="repo-selector">Repo Selector</div>,
}));

vi.mock('../components/features/PatternLibraryModal', () => ({
	PatternLibraryModal: () => null,
}));

import { useSchedules } from '../hooks/useSchedules';

const { default: Schedules } = await import('./Schedules');

function renderPage() {
	return render(
		<BrowserRouter>
			<Schedules />
		</BrowserRouter>,
	);
}

describe('Schedules', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		vi.mocked(useSchedules).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSchedules>);
		renderPage();
		expect(screen.getByText('Schedules')).toBeInTheDocument();
	});

	it('shows loading state', () => {
		vi.mocked(useSchedules).mockReturnValue({
			data: undefined,
			isLoading: true,
			isError: false,
		} as ReturnType<typeof useSchedules>);
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows empty state', () => {
		vi.mocked(useSchedules).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSchedules>);
		renderPage();
		expect(screen.getByText('No schedules configured')).toBeInTheDocument();
	});

	it('renders schedules', () => {
		vi.mocked(useSchedules).mockReturnValue({
			data: [
				{
					id: 's1',
					name: 'Daily Backup',
					agent_id: 'a1',
					cron_expression: '0 0 * * *',
					paths: ['/home'],
					enabled: true,
					next_run: '2024-01-02T00:00:00Z',
					last_run: '2024-01-01T00:00:00Z',
					repositories: [{ repository_id: 'r1', priority: 1, enabled: true }],
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSchedules>);
		renderPage();
		expect(screen.getByText('Daily Backup')).toBeInTheDocument();
	});

	it('shows error state', () => {
		vi.mocked(useSchedules).mockReturnValue({
			data: undefined,
			isLoading: false,
			isError: true,
		} as ReturnType<typeof useSchedules>);
		renderPage();
		expect(screen.getByText('Failed to load schedules')).toBeInTheDocument();
	});

	it('shows create button', () => {
		vi.mocked(useSchedules).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSchedules>);
		renderPage();
		expect(screen.getByText('Create Schedule')).toBeInTheDocument();
	});

	it('shows subtitle', () => {
		vi.mocked(useSchedules).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSchedules>);
		renderPage();
		expect(
			screen.getByText('Configure automated backup jobs'),
		).toBeInTheDocument();
	});

	it('shows search and filter controls', () => {
		vi.mocked(useSchedules).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSchedules>);
		renderPage();
		expect(
			screen.getByPlaceholderText('Search schedules...'),
		).toBeInTheDocument();
		expect(screen.getByText('All Status')).toBeInTheDocument();
	});

	it('renders multiple schedules', () => {
		vi.mocked(useSchedules).mockReturnValue({
			data: [
				{
					id: 's1',
					name: 'Daily Backup',
					agent_id: 'a1',
					cron_expression: '0 0 * * *',
					paths: ['/home'],
					enabled: true,
					next_run: '2024-01-02T00:00:00Z',
					last_run: null,
					repositories: [{ repository_id: 'r1', priority: 1, enabled: true }],
					created_at: '2024-01-01T00:00:00Z',
				},
				{
					id: 's2',
					name: 'Weekly Archive',
					agent_id: 'a1',
					cron_expression: '0 0 * * 0',
					paths: ['/data'],
					enabled: false,
					next_run: null,
					last_run: null,
					repositories: [],
					created_at: '2024-02-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSchedules>);
		renderPage();
		expect(screen.getByText('Daily Backup')).toBeInTheDocument();
		expect(screen.getByText('Weekly Archive')).toBeInTheDocument();
	});

	it('shows empty state help text', () => {
		vi.mocked(useSchedules).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSchedules>);
		renderPage();
		expect(
			screen.getByText(/Create a schedule to automate your backups/),
		).toBeInTheDocument();
	});

	it('shows action buttons for schedules', () => {
		vi.mocked(useSchedules).mockReturnValue({
			data: [
				{
					id: 's1',
					name: 'Daily Backup',
					agent_id: 'a1',
					cron_expression: '0 0 * * *',
					paths: ['/home'],
					enabled: true,
					next_run: '2024-01-02T00:00:00Z',
					last_run: null,
					repositories: [{ repository_id: 'r1', priority: 1, enabled: true }],
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSchedules>);
		renderPage();
		expect(screen.getByText('Run Now')).toBeInTheDocument();
	});

	it('opens create modal', async () => {
		const user = (await import('@testing-library/user-event')).default.setup();
		vi.mocked(useSchedules).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSchedules>);
		renderPage();
		await user.click(screen.getByText('Create Schedule'));
		expect(screen.getAllByText('Create Schedule').length).toBeGreaterThan(1);
	});

	it('shows schedule cron expression', () => {
		vi.mocked(useSchedules).mockReturnValue({
			data: [
				{
					id: 's1',
					name: 'Nightly',
					agent_id: 'a1',
					cron_expression: '0 2 * * *',
					paths: ['/home'],
					enabled: true,
					next_run: '2024-01-02T02:00:00Z',
					last_run: null,
					repositories: [],
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSchedules>);
		renderPage();
		expect(screen.getByText('0 2 * * *')).toBeInTheDocument();
	});

	it('shows filter options', () => {
		vi.mocked(useSchedules).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useSchedules>);
		renderPage();
		expect(screen.getByText('Active')).toBeInTheDocument();
		expect(screen.getByText('Paused')).toBeInTheDocument();
	});
});
