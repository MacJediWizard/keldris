import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { Form } from './Form';

describe('Form', () => {
	it('renders children', () => {
		render(
			<Form>
				<input type="text" />
			</Form>,
		);
		expect(screen.getByRole('textbox')).toBeInTheDocument();
	});

	it('applies spacing class', () => {
		const { container } = render(<Form>Content</Form>);
		expect(container.querySelector('form')).toHaveClass('space-y-4');
	});

	it('fires onSubmit handler', () => {
		const onSubmit = vi.fn((e) => e.preventDefault());
		render(
			<Form onSubmit={onSubmit}>
				<button type="submit">Submit</button>
			</Form>,
		);
		fireEvent.click(screen.getByText('Submit'));
		expect(onSubmit).toHaveBeenCalled();
	});

	it('accepts custom className', () => {
		const { container } = render(<Form className="mt-6">Content</Form>);
		expect(container.querySelector('form')).toHaveClass('mt-6');
	});
});
