import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { GeoReplicationConfig, Repository } from '../../lib/types';
import { GeoReplicationStatusCard } from './GeoReplicationStatusCard';

const repositories: Repository[] = [
	{ id: 'r1', name: 'Primary Repo' } as Repository,
	{ id: 'r2', name: 'Secondary Repo' } as Repository,
];

function makeConfig(
	overrides: Partial<GeoReplicationConfig> = {},
): GeoReplicationConfig {
	return {
		id: 'gc-1',
		source_repository_id: 'r1',
		target_repository_id: 'r2',
		source_region: { display_name: 'us-east-1' },
		target_region: { display_name: 'us-west-2' },
		status: 'synced',
		enabled: true,
		max_lag_snapshots: 5,
		max_lag_duration_hours: 24,
		last_sync_at: new Date(Date.now() - 60_000).toISOString(),
		...overrides,
	} as GeoReplicationConfig;
}

describe('GeoReplicationStatusCard', () => {
	it('renders empty state when no configs', () => {
		render(
			<GeoReplicationStatusCard configs={[]} repositories={repositories} />,
		);
		expect(screen.getByText('Geo-Replication Status')).toBeDefined();
		expect(screen.getByText(/No geo-replication configured/)).toBeDefined();
	});

	it('renders config with source -> target names', () => {
		render(
			<GeoReplicationStatusCard
				configs={[makeConfig()]}
				repositories={repositories}
			/>,
		);
		expect(screen.getByText('Primary Repo')).toBeDefined();
		expect(screen.getByText('Secondary Repo')).toBeDefined();
		expect(screen.getByText('Synced')).toBeDefined();
	});

	it('falls back to Unknown when repo not found', () => {
		render(
			<GeoReplicationStatusCard
				configs={[makeConfig({ source_repository_id: 'missing' })]}
				repositories={repositories}
			/>,
		);
		expect(screen.getByText('Unknown')).toBeDefined();
	});

	it('fires onToggleEnabled when toggle button clicked', () => {
		const onToggle = vi.fn();
		render(
			<GeoReplicationStatusCard
				configs={[makeConfig()]}
				repositories={repositories}
				onToggleEnabled={onToggle}
			/>,
		);
		fireEvent.click(
			screen.getByRole('button', { name: 'Disable replication' }),
		);
		expect(onToggle).toHaveBeenCalledWith('gc-1', false);
	});

	it('fires onTriggerReplication when sync now clicked', () => {
		const onTrigger = vi.fn();
		render(
			<GeoReplicationStatusCard
				configs={[makeConfig()]}
				repositories={repositories}
				onTriggerReplication={onTrigger}
			/>,
		);
		fireEvent.click(screen.getByRole('button', { name: 'Trigger sync now' }));
		expect(onTrigger).toHaveBeenCalledWith('gc-1');
	});

	it('hides trigger button when status=syncing', () => {
		render(
			<GeoReplicationStatusCard
				configs={[makeConfig({ status: 'syncing' })]}
				repositories={repositories}
				onTriggerReplication={vi.fn()}
			/>,
		);
		expect(
			screen.queryByRole('button', { name: 'Trigger sync now' }),
		).toBeNull();
	});

	it('shows last_error when present', () => {
		render(
			<GeoReplicationStatusCard
				configs={[makeConfig({ last_error: 'Network timeout' })]}
				repositories={repositories}
			/>,
		);
		expect(screen.getByText('Network timeout')).toBeDefined();
	});
});
