import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { Input } from './Input';

describe('Input', () => {
	it('renders an input element', () => {
		render(<Input />);
		expect(screen.getByRole('textbox')).toBeInTheDocument();
	});

	it('renders with label', () => {
		render(<Input label="Email" />);
		expect(screen.getByLabelText('Email')).toBeInTheDocument();
	});

	it('shows error message', () => {
		render(<Input label="Email" error="Required field" />);
		expect(screen.getByText('Required field')).toBeInTheDocument();
	});

	it('sets aria-invalid when error is present', () => {
		render(<Input label="Email" error="Required" />);
		expect(screen.getByRole('textbox')).toHaveAttribute(
			'aria-invalid',
			'true',
		);
	});

	it('applies error styling', () => {
		render(<Input label="Email" error="Required" />);
		expect(screen.getByRole('textbox')).toHaveClass('border-red-300');
	});

	it('shows helper text', () => {
		render(<Input label="Email" helperText="Enter your email" />);
		expect(screen.getByText('Enter your email')).toBeInTheDocument();
	});

	it('does not show helper text when error is present', () => {
		render(
			<Input label="Email" error="Required" helperText="Enter your email" />,
		);
		expect(screen.queryByText('Enter your email')).not.toBeInTheDocument();
	});

	it('fires onChange handler', () => {
		const onChange = vi.fn();
		render(<Input onChange={onChange} />);
		fireEvent.change(screen.getByRole('textbox'), {
			target: { value: 'test' },
		});
		expect(onChange).toHaveBeenCalled();
	});

	it('uses custom id when provided', () => {
		render(<Input id="custom-id" label="Name" />);
		expect(screen.getByRole('textbox')).toHaveAttribute('id', 'custom-id');
	});
});
