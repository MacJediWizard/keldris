import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { LoadingSpinner } from './LoadingSpinner';

describe('LoadingSpinner', () => {
	it('renders a spinner container', () => {
		const { container } = render(<LoadingSpinner />);
		const wrapper = container.firstElementChild;
		expect(wrapper).toBeInTheDocument();
		expect(wrapper).toHaveClass('flex', 'items-center', 'justify-center');
	});

	it('renders the spinner element with animation class', () => {
		const { container } = render(<LoadingSpinner />);
		const spinner = container.querySelector('.animate-spin');
		expect(spinner).toBeInTheDocument();
		expect(spinner).toHaveClass('rounded-full', 'border-4');
	});

	it('has minimum height', () => {
		const { container } = render(<LoadingSpinner />);
		const wrapper = container.firstElementChild;
		expect(wrapper).toHaveClass('min-h-[200px]');
	});
});
