import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { type BulkAction, BulkActionButton, BulkActions } from './BulkActions';

const actions: BulkAction[] = [
	{ id: 'delete', label: 'Delete', variant: 'danger' },
	{ id: 'export', label: 'Export' },
	{ id: 'disabled-action', label: 'Disabled', disabled: true },
];

describe('BulkActions', () => {
	it('renders label and toggles menu open', () => {
		render(<BulkActions actions={actions} onAction={() => {}} />);
		const trigger = screen.getByRole('button', { name: /Actions/i });
		expect(trigger).toBeDefined();
		fireEvent.click(trigger);
		expect(screen.getByRole('button', { name: 'Delete' })).toBeDefined();
		expect(screen.getByRole('button', { name: 'Export' })).toBeDefined();
	});

	it('uses custom label', () => {
		render(
			<BulkActions actions={actions} onAction={() => {}} label="Do Stuff" />,
		);
		expect(screen.getByRole('button', { name: /Do Stuff/i })).toBeDefined();
	});

	it('fires onAction with action id when clicked', () => {
		const onAction = vi.fn();
		render(<BulkActions actions={actions} onAction={onAction} />);
		fireEvent.click(screen.getByRole('button', { name: /Actions/i }));
		fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
		expect(onAction).toHaveBeenCalledWith('delete');
	});

	it('disables trigger when disabled=true', () => {
		render(<BulkActions actions={actions} onAction={() => {}} disabled />);
		expect(screen.getByRole('button', { name: /Actions/i })).toBeDisabled();
	});

	it('disables actions marked disabled', () => {
		render(<BulkActions actions={actions} onAction={() => {}} />);
		fireEvent.click(screen.getByRole('button', { name: /Actions/i }));
		expect(screen.getByRole('button', { name: 'Disabled' })).toBeDisabled();
	});
});

describe('BulkActionButton', () => {
	it('renders label and fires onClick', () => {
		const onClick = vi.fn();
		render(<BulkActionButton label="Run" onClick={onClick} />);
		const btn = screen.getByRole('button', { name: 'Run' });
		btn.click();
		expect(onClick).toHaveBeenCalledOnce();
	});

	it('renders danger variant', () => {
		render(
			<BulkActionButton label="Delete" onClick={() => {}} variant="danger" />,
		);
		const btn = screen.getByRole('button', { name: 'Delete' });
		expect(btn.className).toContain('bg-red-600');
	});

	it('renders primary variant', () => {
		render(
			<BulkActionButton label="Save" onClick={() => {}} variant="primary" />,
		);
		const btn = screen.getByRole('button', { name: 'Save' });
		expect(btn.className).toContain('bg-indigo-600');
	});

	it('disables button when disabled=true', () => {
		render(<BulkActionButton label="Run" onClick={() => {}} disabled />);
		expect(screen.getByRole('button', { name: 'Run' })).toBeDisabled();
	});
});
