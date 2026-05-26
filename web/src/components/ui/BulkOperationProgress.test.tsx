import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import {
	BulkOperationProgress,
	type BulkOperationResult,
} from './BulkOperationProgress';

describe('BulkOperationProgress', () => {
	it('renders nothing when isOpen=false', () => {
		const { container } = render(
			<BulkOperationProgress
				isOpen={false}
				onClose={() => {}}
				title="Deleting"
				total={5}
				completed={0}
				results={[]}
				isComplete={false}
			/>,
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders title and progress', () => {
		render(
			<BulkOperationProgress
				isOpen
				onClose={() => {}}
				title="Deleting agents"
				total={10}
				completed={3}
				results={[]}
				isComplete={false}
			/>,
		);
		expect(screen.getByText('Deleting agents')).toBeDefined();
		expect(screen.getByText('3 of 10')).toBeDefined();
		expect(
			screen.getByText('Please wait while the operation completes...'),
		).toBeDefined();
	});

	it('shows success/failure counts when complete', () => {
		const results: BulkOperationResult[] = [
			{ id: '1', success: true },
			{ id: '2', success: true },
			{ id: '3', success: false, error: 'Failed to delete' },
		];
		render(
			<BulkOperationProgress
				isOpen
				onClose={() => {}}
				title="Done"
				total={3}
				completed={3}
				results={results}
				isComplete
			/>,
		);
		expect(screen.getByText('2 succeeded')).toBeDefined();
		expect(screen.getByText('1 failed')).toBeDefined();
		expect(screen.getByText('Failed to delete')).toBeDefined();
	});

	it('shows Done button and fires onClose', () => {
		const onClose = vi.fn();
		render(
			<BulkOperationProgress
				isOpen
				onClose={onClose}
				title="Done"
				total={1}
				completed={1}
				results={[{ id: '1', success: true }]}
				isComplete
			/>,
		);
		screen.getByRole('button', { name: 'Done' }).click();
		expect(onClose).toHaveBeenCalledOnce();
	});

	it('handles zero total without crashing', () => {
		render(
			<BulkOperationProgress
				isOpen
				onClose={() => {}}
				title="Empty"
				total={0}
				completed={0}
				results={[]}
				isComplete
			/>,
		);
		expect(screen.getByText('0 of 0')).toBeDefined();
	});
});
