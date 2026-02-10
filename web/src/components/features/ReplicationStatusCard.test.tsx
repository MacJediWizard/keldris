import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { ReplicationStatus, Repository } from '../../lib/types';
import { ReplicationStatusCard } from './ReplicationStatusCard';

const repositories: Repository[] = [
	{
		id: 'repo-1',
		name: 'Primary Repo',
		type: 'local',
		org_id: 'org-1',
		created_at: '2024-01-01T00:00:00Z',
		updated_at: '2024-01-01T00:00:00Z',
	} as Repository,
	{
		id: 'repo-2',
		name: 'Secondary Repo',
		type: 's3',
		org_id: 'org-1',
		created_at: '2024-01-01T00:00:00Z',
		updated_at: '2024-01-01T00:00:00Z',
	} as Repository,
];

const statuses: ReplicationStatus[] = [
	{
		id: 'rep-1',
		schedule_id: 'sched-1',
		source_repository_id: 'repo-1',
		target_repository_id: 'repo-2',
		status: 'synced',
		last_sync_at: '2024-06-15T12:00:00Z',
		created_at: '2024-01-01T00:00:00Z',
		updated_at: '2024-06-15T12:00:00Z',
	},
];

describe('ReplicationStatusCard', () => {
	it('shows empty state when no statuses', () => {
		render(<ReplicationStatusCard statuses={[]} repositories={repositories} />);
		expect(screen.getByText(/No replication configured/)).toBeInTheDocument();
	});

	it('renders replication status with repo names', () => {
		render(
			<ReplicationStatusCard statuses={statuses} repositories={repositories} />,
		);
		expect(screen.getByText('Replication Status')).toBeInTheDocument();
		expect(screen.getByText('Primary Repo')).toBeInTheDocument();
		expect(screen.getByText('Secondary Repo')).toBeInTheDocument();
	});

	it('shows synced status badge', () => {
		render(
			<ReplicationStatusCard statuses={statuses} repositories={repositories} />,
		);
		expect(screen.getByText('Synced')).toBeInTheDocument();
	});

	it('shows last sync time', () => {
		render(
			<ReplicationStatusCard statuses={statuses} repositories={repositories} />,
		);
		expect(screen.getByText(/Last synced:/)).toBeInTheDocument();
	});

	it('shows error message when present', () => {
		const failedStatuses: ReplicationStatus[] = [
			{
				...statuses[0],
				status: 'failed',
				error_message: 'Connection timed out',
			},
		];
		render(
			<ReplicationStatusCard
				statuses={failedStatuses}
				repositories={repositories}
			/>,
		);
		expect(screen.getByText('Connection timed out')).toBeInTheDocument();
		expect(screen.getByText('Failed')).toBeInTheDocument();
	});

	it('shows "Unknown" for missing repo names', () => {
		const unknownStatuses: ReplicationStatus[] = [
			{
				...statuses[0],
				source_repository_id: 'nonexistent',
				target_repository_id: 'also-nonexistent',
			},
		];
		render(
			<ReplicationStatusCard
				statuses={unknownStatuses}
				repositories={repositories}
			/>,
		);
		const unknowns = screen.getAllByText('Unknown');
		expect(unknowns).toHaveLength(2);
	});
});
