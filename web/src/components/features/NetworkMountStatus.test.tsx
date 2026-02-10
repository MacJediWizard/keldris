import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { NetworkMountStatus } from './NetworkMountStatus';
import type { NetworkMount } from '../../lib/types';

const connectedMount: NetworkMount = {
	path: '/mnt/backup',
	type: 'nfs',
	remote: '192.168.1.100:/backup',
	status: 'connected',
};

const staleMount: NetworkMount = {
	path: '/mnt/archive',
	type: 'cifs',
	remote: '//server/archive',
	status: 'stale',
};

const disconnectedMount: NetworkMount = {
	path: '/mnt/offsite',
	type: 'nfs',
	remote: '10.0.0.5:/offsite',
	status: 'disconnected',
};

describe('NetworkMountStatus', () => {
	it('returns null for empty mounts array', () => {
		const { container } = render(<NetworkMountStatus mounts={[]} />);
		expect(container.firstChild).toBeNull();
	});

	it('returns null for undefined mounts', () => {
		const { container } = render(
			<NetworkMountStatus mounts={undefined as unknown as NetworkMount[]} />,
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders mount details in full mode', () => {
		render(<NetworkMountStatus mounts={[connectedMount]} />);
		expect(screen.getByText('Network Mounts')).toBeInTheDocument();
		expect(screen.getByText('/mnt/backup')).toBeInTheDocument();
		expect(screen.getByText('NFS - 192.168.1.100:/backup')).toBeInTheDocument();
		expect(screen.getByText('connected')).toBeInTheDocument();
	});

	it('renders compact mode with summary counts', () => {
		render(
			<NetworkMountStatus
				mounts={[connectedMount, staleMount, disconnectedMount]}
				compact
			/>,
		);
		expect(screen.getByText('3 mounts')).toBeInTheDocument();
		expect(screen.getByText('1 connected')).toBeInTheDocument();
		expect(screen.getByText('2 unavailable')).toBeInTheDocument();
	});

	it('renders compact singular mount label', () => {
		render(<NetworkMountStatus mounts={[connectedMount]} compact />);
		expect(screen.getByText('1 mount')).toBeInTheDocument();
	});

	it('renders multiple mounts in full mode', () => {
		render(
			<NetworkMountStatus
				mounts={[connectedMount, staleMount, disconnectedMount]}
			/>,
		);
		expect(screen.getByText('/mnt/backup')).toBeInTheDocument();
		expect(screen.getByText('/mnt/archive')).toBeInTheDocument();
		expect(screen.getByText('/mnt/offsite')).toBeInTheDocument();
	});

	it('shows correct status colors', () => {
		render(
			<NetworkMountStatus mounts={[connectedMount, disconnectedMount]} />,
		);
		expect(screen.getByText('connected')).toBeInTheDocument();
		expect(screen.getByText('disconnected')).toBeInTheDocument();
	});
});
