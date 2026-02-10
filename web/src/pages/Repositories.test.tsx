import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../hooks/useRepositories', () => ({
	useRepositories: vi.fn(),
	useCreateRepository: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
	useDeleteRepository: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useTestRepository: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useTestConnection: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useRecoverRepositoryKey: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock('../hooks/useVerifications', () => ({
	useVerificationStatus: () => ({ data: undefined }),
	useTriggerVerification: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

import { useRepositories } from '../hooks/useRepositories';

const { default: Repositories } = await import('./Repositories');

function renderPage() {
	return render(
		<BrowserRouter>
			<Repositories />
		</BrowserRouter>,
	);
}

describe('Repositories', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		vi.mocked(useRepositories).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useRepositories>);
		renderPage();
		expect(screen.getByText('Repositories')).toBeInTheDocument();
	});

	it('shows loading state', () => {
		vi.mocked(useRepositories).mockReturnValue({
			data: undefined,
			isLoading: true,
			isError: false,
		} as ReturnType<typeof useRepositories>);
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows empty state', () => {
		vi.mocked(useRepositories).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useRepositories>);
		renderPage();
		expect(screen.getByText('No repositories configured')).toBeInTheDocument();
	});

	it('renders repository cards', () => {
		vi.mocked(useRepositories).mockReturnValue({
			data: [
				{
					id: 'r1',
					name: 'my-backups',
					type: 'local',
					path: '/backups',
					status: 'active',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useRepositories>);
		renderPage();
		expect(screen.getByText('my-backups')).toBeInTheDocument();
	});

	it('shows error state', () => {
		vi.mocked(useRepositories).mockReturnValue({
			data: undefined,
			isLoading: false,
			isError: true,
		} as ReturnType<typeof useRepositories>);
		renderPage();
		expect(screen.getByText('Failed to load repositories')).toBeInTheDocument();
	});

	it('shows add repository button', () => {
		vi.mocked(useRepositories).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useRepositories>);
		renderPage();
		expect(screen.getByText('Add Repository')).toBeInTheDocument();
	});

	it('renders multiple repository cards', () => {
		vi.mocked(useRepositories).mockReturnValue({
			data: [
				{
					id: 'r1',
					name: 'local-backup',
					type: 'local',
					path: '/backups/local',
					status: 'active',
					created_at: '2024-01-01T00:00:00Z',
				},
				{
					id: 'r2',
					name: 's3-backup',
					type: 's3',
					path: 's3:mybucket',
					status: 'active',
					created_at: '2024-02-01T00:00:00Z',
				},
				{
					id: 'r3',
					name: 'sftp-backup',
					type: 'sftp',
					path: 'sftp:server:/backups',
					status: 'inactive',
					created_at: '2024-03-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useRepositories>);
		renderPage();
		expect(screen.getByText('local-backup')).toBeInTheDocument();
		expect(screen.getByText('s3-backup')).toBeInTheDocument();
		expect(screen.getByText('sftp-backup')).toBeInTheDocument();
	});

	it('shows repository type badges', () => {
		vi.mocked(useRepositories).mockReturnValue({
			data: [
				{
					id: 'r1',
					name: 'local-backup',
					type: 'local',
					path: '/backups',
					status: 'active',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useRepositories>);
		renderPage();
		expect(screen.getAllByText('Local').length).toBeGreaterThan(0);
	});

	it('opens add repository modal', async () => {
		const user = (await import('@testing-library/user-event')).default.setup();
		vi.mocked(useRepositories).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useRepositories>);
		renderPage();
		await user.click(screen.getByText('Add Repository'));
		expect(screen.getAllByText('Add Repository').length).toBeGreaterThan(1);
	});

	it('shows repository type selector in modal', async () => {
		const user = (await import('@testing-library/user-event')).default.setup();
		vi.mocked(useRepositories).mockReturnValue({
			data: [],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useRepositories>);
		renderPage();
		await user.click(screen.getByText('Add Repository'));
		expect(screen.getByText('Local Filesystem')).toBeInTheDocument();
		expect(screen.getByText('Test Connection')).toBeInTheDocument();
	});

	it('shows delete and test buttons on repository cards', () => {
		vi.mocked(useRepositories).mockReturnValue({
			data: [
				{
					id: 'r1',
					name: 'my-backup',
					type: 'local',
					path: '/backups',
					status: 'active',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useRepositories>);
		renderPage();
		expect(screen.getByText(/Test/)).toBeInTheDocument();
		expect(screen.getByText(/Delete/)).toBeInTheDocument();
	});

	it('shows integrity section on repository cards', () => {
		vi.mocked(useRepositories).mockReturnValue({
			data: [
				{
					id: 'r1',
					name: 'local-backup',
					type: 'local',
					path: '/backups/data',
					status: 'active',
					created_at: '2024-01-01T00:00:00Z',
				},
			],
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useRepositories>);
		renderPage();
		expect(screen.getByText('Integrity')).toBeInTheDocument();
		expect(screen.getByText('Not verified')).toBeInTheDocument();
	});
});
