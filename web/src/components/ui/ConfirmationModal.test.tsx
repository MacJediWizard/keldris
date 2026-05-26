import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { ConfirmationModal } from './ConfirmationModal';

const baseProps = {
	isOpen: true,
	onClose: vi.fn(),
	onConfirm: vi.fn(),
	title: 'Delete item',
	message: 'Are you sure?',
};

describe('ConfirmationModal', () => {
	it('renders nothing when closed', () => {
		render(<ConfirmationModal {...baseProps} isOpen={false} />);
		expect(screen.queryByText('Delete item')).toBeNull();
	});

	it('renders title and message when open', () => {
		render(<ConfirmationModal {...baseProps} />);
		expect(screen.getByText('Delete item')).toBeDefined();
		expect(screen.getByText('Are you sure?')).toBeDefined();
	});

	it('fires onConfirm when confirm button clicked', () => {
		const onConfirm = vi.fn();
		render(<ConfirmationModal {...baseProps} onConfirm={onConfirm} />);
		screen.getByRole('button', { name: 'Confirm' }).click();
		expect(onConfirm).toHaveBeenCalledOnce();
	});

	it('fires onClose when cancel button clicked', () => {
		const onClose = vi.fn();
		render(<ConfirmationModal {...baseProps} onClose={onClose} />);
		screen.getByRole('button', { name: 'Cancel' }).click();
		expect(onClose).toHaveBeenCalledOnce();
	});

	it('uses custom labels when provided', () => {
		render(
			<ConfirmationModal
				{...baseProps}
				confirmLabel="Delete Forever"
				cancelLabel="Keep It"
			/>,
		);
		expect(
			screen.getByRole('button', { name: 'Delete Forever' }),
		).toBeDefined();
		expect(screen.getByRole('button', { name: 'Keep It' })).toBeDefined();
	});

	it('shows item count when provided', () => {
		render(<ConfirmationModal {...baseProps} itemCount={5} />);
		expect(screen.getByText('This action will affect 5 items.')).toBeDefined();
	});

	it('uses singular form when itemCount is 1', () => {
		render(<ConfirmationModal {...baseProps} itemCount={1} />);
		expect(screen.getByText('This action will affect 1 item.')).toBeDefined();
	});

	it('shows Processing label when loading and disables buttons', () => {
		render(<ConfirmationModal {...baseProps} isLoading />);
		expect(screen.getByText('Processing...')).toBeDefined();
		expect(screen.getByRole('button', { name: 'Cancel' })).toBeDisabled();
	});
});
