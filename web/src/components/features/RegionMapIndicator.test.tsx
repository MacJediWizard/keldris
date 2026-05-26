import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type {
	GeoRegion,
	GeoReplicationConfig,
	GeoReplicationStatusType,
} from '../../lib/types';
import { RegionMapIndicator } from './RegionMapIndicator';

function region(overrides: Partial<GeoRegion> = {}): GeoRegion {
	return {
		code: 'us-east-1',
		name: 'us-east-1',
		display_name: 'US East',
		latitude: 0,
		longitude: 0,
		...overrides,
	};
}

function config(
	overrides: Partial<GeoReplicationConfig> = {},
): GeoReplicationConfig {
	return {
		id: 'c1',
		source_repository_id: 's1',
		target_repository_id: 't1',
		source_region: region({ code: 'us-east-1' }),
		target_region: region({ code: 'eu-west-1', display_name: 'EU West' }),
		enabled: true,
		status: 'synced' as GeoReplicationStatusType,
		max_lag_snapshots: 0,
		max_lag_duration_hours: 0,
		alert_on_lag: false,
		created_at: '',
		updated_at: '',
		...overrides,
	};
}

describe('RegionMapIndicator', () => {
	it('renders empty message when no active configs', () => {
		render(<RegionMapIndicator configs={[]} />);
		expect(screen.getByText('No geo-replication configured')).toBeDefined();
	});

	it('renders empty message when all configs are disabled', () => {
		render(<RegionMapIndicator configs={[config({ enabled: false })]} />);
		expect(screen.getByText('No geo-replication configured')).toBeDefined();
	});

	it('renders compact summary when compact=true', () => {
		render(
			<RegionMapIndicator
				compact
				configs={[
					config({ id: 'a', status: 'synced' }),
					config({ id: 'b', status: 'syncing' }),
					config({ id: 'c', status: 'failed' }),
				]}
			/>,
		);
		expect(screen.getByText('1 synced')).toBeDefined();
		expect(screen.getByText('1 syncing')).toBeDefined();
		expect(screen.getByText('1 failed')).toBeDefined();
	});

	it('renders full map title and legend when not compact', () => {
		render(<RegionMapIndicator configs={[config()]} />);
		expect(screen.getByText('Geo-Replication Map')).toBeDefined();
		expect(screen.getByText('Synced')).toBeDefined();
		expect(screen.getByText('Pending')).toBeDefined();
	});
});
