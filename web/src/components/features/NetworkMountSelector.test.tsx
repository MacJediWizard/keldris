import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { NetworkMount } from '../../lib/types';
import { NetworkMountSelector } from './NetworkMountSelector';

const mounts: NetworkMount[] = [
	{
		path: '/mnt/backup',
		type: 'nfs',
		remote: '192.168.1.100:/backup',
		status: 'connected',
	},
	{
		path: '/mnt/archive',
		type: 'cifs',
		remote: '//server/archive',
		status: 'connected',
	},
	{
		path: '/mnt/offsite',
		type: 'nfs',
		remote: '10.0.0.5:/offsite',
		status: 'disconnected',
	},
];

describe('NetworkMountSelector', () => {
	it('renders connected mounts as selectable', () => {
		render(
			<NetworkMountSelector
				mounts={mounts}
				selectedPaths={[]}
				onPathsChange={vi.fn()}
			/>,
		);
		expect(screen.getByText('/mnt/backup')).toBeInTheDocument();
		expect(screen.getByText('/mnt/archive')).toBeInTheDocument();
	});

	it('shows unavailable mounts section', () => {
		render(
			<NetworkMountSelector
				mounts={mounts}
				selectedPaths={[]}
				onPathsChange={vi.fn()}
			/>,
		);
		expect(screen.getByText('Unavailable mounts:')).toBeInTheDocument();
		expect(screen.getByText('/mnt/offsite')).toBeInTheDocument();
	});

	it('shows empty message when no mounts', () => {
		render(
			<NetworkMountSelector
				mounts={[]}
				selectedPaths={[]}
				onPathsChange={vi.fn()}
			/>,
		);
		expect(
			screen.getByText('No network mounts detected on this agent.'),
		).toBeInTheDocument();
	});

	it('toggles mount selection', () => {
		const onPathsChange = vi.fn();
		render(
			<NetworkMountSelector
				mounts={mounts}
				selectedPaths={[]}
				onPathsChange={onPathsChange}
			/>,
		);
		const checkboxes = screen.getAllByRole('checkbox');
		fireEvent.click(checkboxes[0]);
		expect(onPathsChange).toHaveBeenCalledWith(['/mnt/backup']);
	});

	it('deselects a selected mount', () => {
		const onPathsChange = vi.fn();
		render(
			<NetworkMountSelector
				mounts={mounts}
				selectedPaths={['/mnt/backup']}
				onPathsChange={onPathsChange}
			/>,
		);
		const checkboxes = screen.getAllByRole('checkbox');
		fireEvent.click(checkboxes[0]);
		expect(onPathsChange).toHaveBeenCalledWith([]);
	});

	it('renders checkboxes as checked for selected paths', () => {
		render(
			<NetworkMountSelector
				mounts={mounts}
				selectedPaths={['/mnt/backup']}
				onPathsChange={vi.fn()}
			/>,
		);
		const checkboxes = screen.getAllByRole('checkbox');
		expect(checkboxes[0]).toBeChecked();
		expect(checkboxes[1]).not.toBeChecked();
	});
});
