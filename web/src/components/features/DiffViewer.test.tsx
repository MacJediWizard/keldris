import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { FileDiffResponse } from '../../lib/types';
import { DiffViewer } from './DiffViewer';

function makeDiff(overrides: Partial<FileDiffResponse> = {}): FileDiffResponse {
	return {
		path: '/etc/foo.conf',
		change_type: 'modified',
		is_binary: false,
		unified_diff:
			'--- a/foo\n+++ b/foo\n@@ -1,2 +1,2 @@\n line one\n-removed\n+added\n',
		old_size: 100,
		new_size: 110,
		old_hash: 'abc',
		new_hash: 'def',
		snapshot_id_1: 'snap-1',
		snapshot_id_2: 'snap-2',
		...overrides,
	} as FileDiffResponse;
}

describe('DiffViewer', () => {
	it('renders path and unified diff content', () => {
		render(<DiffViewer diff={makeDiff()} />);
		expect(screen.getByText('/etc/foo.conf')).toBeDefined();
		expect(screen.getByText('modified')).toBeDefined();
		expect(screen.getByText('added')).toBeDefined();
		expect(screen.getByText('removed')).toBeDefined();
	});

	it('toggles to split view', () => {
		render(<DiffViewer diff={makeDiff()} />);
		fireEvent.click(screen.getByRole('button', { name: 'Split' }));
		// Split view shows snapshot id headers
		expect(screen.getByText('snap-1')).toBeDefined();
		expect(screen.getByText('snap-2')).toBeDefined();
	});

	it('shows binary file notice for binary diffs', () => {
		render(
			<DiffViewer
				diff={makeDiff({
					is_binary: true,
					unified_diff: '',
				})}
			/>,
		);
		expect(screen.getAllByText('Binary file').length).toBeGreaterThan(0);
		expect(
			screen.getByText('Content comparison is not available for binary files'),
		).toBeDefined();
	});

	it('shows identical message when binary hashes match', () => {
		render(
			<DiffViewer
				diff={makeDiff({
					is_binary: true,
					unified_diff: '',
					old_hash: 'same',
					new_hash: 'same',
				})}
			/>,
		);
		expect(screen.getByText('Files are identical')).toBeDefined();
	});

	it('shows "New file" message when change_type=added and empty diff', () => {
		render(
			<DiffViewer
				diff={makeDiff({
					change_type: 'added',
					unified_diff: '',
				})}
			/>,
		);
		expect(screen.getByText('New file')).toBeDefined();
	});

	it('shows "Deleted file" message when change_type=removed and empty diff', () => {
		render(
			<DiffViewer
				diff={makeDiff({
					change_type: 'removed',
					unified_diff: '',
				})}
			/>,
		);
		expect(screen.getByText('Deleted file')).toBeDefined();
	});
});
