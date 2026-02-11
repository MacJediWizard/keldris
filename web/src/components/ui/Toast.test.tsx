import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { Toast } from './Toast';

describe('Toast', () => {
	it('renders the message', () => {
		render(<Toast message="Operation successful" />);
		expect(screen.getByText('Operation successful')).toBeInTheDocument();
	});

	it('has alert role', () => {
		render(<Toast message="Test" />);
		expect(screen.getByRole('alert')).toBeInTheDocument();
	});

	it('applies success variant', () => {
		render(<Toast variant="success" message="Success" />);
		expect(screen.getByRole('alert')).toHaveClass('bg-green-50');
	});

	it('applies error variant', () => {
		render(<Toast variant="error" message="Error" />);
		expect(screen.getByRole('alert')).toHaveClass('bg-red-50');
	});

	it('applies warning variant', () => {
		render(<Toast variant="warning" message="Warning" />);
		expect(screen.getByRole('alert')).toHaveClass('bg-yellow-50');
	});

	it('applies info variant by default', () => {
		render(<Toast message="Info" />);
		expect(screen.getByRole('alert')).toHaveClass('bg-blue-50');
	});

	it('shows close button when onClose is provided', () => {
		render(<Toast message="Test" onClose={vi.fn()} />);
		expect(screen.getByLabelText('Dismiss')).toBeInTheDocument();
	});

	it('does not show close button without onClose', () => {
		render(<Toast message="Test" />);
		expect(screen.queryByLabelText('Dismiss')).not.toBeInTheDocument();
	});

	it('calls onClose when dismiss is clicked', () => {
		const onClose = vi.fn();
		render(<Toast message="Test" onClose={onClose} />);
		fireEvent.click(screen.getByLabelText('Dismiss'));
		expect(onClose).toHaveBeenCalledTimes(1);
	});

	it('auto-dismisses after duration', () => {
		vi.useFakeTimers();
		const onClose = vi.fn();
		render(<Toast message="Test" onClose={onClose} duration={3000} />);
		vi.advanceTimersByTime(3000);
		expect(onClose).toHaveBeenCalledTimes(1);
		vi.useRealTimers();
	});
});
