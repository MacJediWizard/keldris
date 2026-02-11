import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ErrorMessage } from './ErrorMessage';

describe('ErrorMessage', () => {
	it('renders the error message', () => {
		render(<ErrorMessage message="Something went wrong" />);
		expect(screen.getByText('Something went wrong')).toBeInTheDocument();
	});

	it('has alert role', () => {
		render(<ErrorMessage message="Error" />);
		expect(screen.getByRole('alert')).toBeInTheDocument();
	});

	it('applies error styling', () => {
		const { container } = render(<ErrorMessage message="Error" />);
		expect(container.firstElementChild).toHaveClass(
			'bg-red-50',
			'border-red-200',
		);
	});

	it('renders error icon', () => {
		const { container } = render(<ErrorMessage message="Error" />);
		expect(container.querySelector('svg')).toBeInTheDocument();
	});
});
