import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { DryRunResponse } from '../../lib/types';
import { DryRunResultsModal } from './DryRunResultsModal';

function makeResults(overrides: Partial<DryRunResponse> = {}): DryRunResponse {
	return {
		total_files: 42,
		total_size: 1024 * 1024,
		new_files: 10,
		changed_files: 5,
		message: 'Dry run complete',
		files_to_backup: [
			{ path: '/a/file.txt', type: 'file', action: 'new', size: 1000 },
			{ path: '/a/dir', type: 'dir', action: 'changed', size: 0 },
		],
		excluded_files: [{ path: '*.tmp', reason: 'excluded by pattern' }],
		...overrides,
	} as DryRunResponse;
}

describe('DryRunResultsModal', () => {
	it('renders nothing when isOpen=false', () => {
		const { container } = render(
			<DryRunResultsModal
				isOpen={false}
				onClose={() => {}}
				results={null}
				isLoading={false}
				error={null}
			/>,
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders loading state', () => {
		render(
			<DryRunResultsModal
				isOpen
				onClose={() => {}}
				results={null}
				isLoading
				error={null}
			/>,
		);
		expect(screen.getByText('Running dry run simulation...')).toBeDefined();
	});

	it('renders error state', () => {
		render(
			<DryRunResultsModal
				isOpen
				onClose={() => {}}
				results={null}
				isLoading={false}
				error={new Error('something failed')}
			/>,
		);
		expect(screen.getByText('Dry run failed')).toBeDefined();
		expect(screen.getByText('something failed')).toBeDefined();
	});

	it('renders results with summary stats', () => {
		render(
			<DryRunResultsModal
				isOpen
				onClose={() => {}}
				results={makeResults()}
				isLoading={false}
				error={null}
			/>,
		);
		expect(screen.getByText('42')).toBeDefined();
		expect(screen.getByText('Total Files')).toBeDefined();
		expect(screen.getByText('Dry run complete')).toBeDefined();
		expect(screen.getByText('/a/file.txt')).toBeDefined();
	});

	it('switches to excluded tab', () => {
		render(
			<DryRunResultsModal
				isOpen
				onClose={() => {}}
				results={makeResults()}
				isLoading={false}
				error={null}
			/>,
		);
		fireEvent.click(screen.getByRole('button', { name: /Excluded/ }));
		expect(screen.getByText('*.tmp')).toBeDefined();
		expect(screen.getByText('excluded by pattern')).toBeDefined();
	});

	it('fires onClose when close button clicked', () => {
		const onClose = vi.fn();
		render(
			<DryRunResultsModal
				isOpen
				onClose={onClose}
				results={makeResults()}
				isLoading={false}
				error={null}
			/>,
		);
		const closeButtons = screen.getAllByRole('button', { name: 'Close' });
		fireEvent.click(closeButtons[0]);
		expect(onClose).toHaveBeenCalledOnce();
	});

	it('shows empty state when no files to backup', () => {
		render(
			<DryRunResultsModal
				isOpen
				onClose={() => {}}
				results={makeResults({ files_to_backup: [] })}
				isLoading={false}
				error={null}
			/>,
		);
		expect(screen.getByText('No files would be backed up.')).toBeDefined();
	});
});
