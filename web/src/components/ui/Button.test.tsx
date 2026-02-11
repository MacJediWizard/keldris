import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { Button } from './Button';

describe('Button', () => {
	it('renders children', () => {
		render(<Button>Click me</Button>);
		expect(screen.getByText('Click me')).toBeInTheDocument();
	});

	it('applies primary variant by default', () => {
		render(<Button>Primary</Button>);
		const button = screen.getByRole('button');
		expect(button).toHaveClass('bg-indigo-600');
	});

	it('applies secondary variant', () => {
		render(<Button variant="secondary">Secondary</Button>);
		const button = screen.getByRole('button');
		expect(button).toHaveClass('bg-gray-100');
	});

	it('applies danger variant', () => {
		render(<Button variant="danger">Danger</Button>);
		const button = screen.getByRole('button');
		expect(button).toHaveClass('bg-red-600');
	});

	it('applies outline variant', () => {
		render(<Button variant="outline">Outline</Button>);
		const button = screen.getByRole('button');
		expect(button).toHaveClass('border-gray-300', 'bg-white');
	});

	it('applies small size', () => {
		render(<Button size="sm">Small</Button>);
		const button = screen.getByRole('button');
		expect(button).toHaveClass('px-3', 'py-1.5');
	});

	it('applies medium size by default', () => {
		render(<Button>Medium</Button>);
		const button = screen.getByRole('button');
		expect(button).toHaveClass('px-4', 'py-2');
	});

	it('applies large size', () => {
		render(<Button size="lg">Large</Button>);
		const button = screen.getByRole('button');
		expect(button).toHaveClass('px-6', 'py-3');
	});

	it('shows loading spinner when loading', () => {
		const { container } = render(<Button loading>Loading</Button>);
		expect(container.querySelector('.animate-spin')).toBeInTheDocument();
	});

	it('is disabled when loading', () => {
		render(<Button loading>Loading</Button>);
		expect(screen.getByRole('button')).toBeDisabled();
	});

	it('is disabled when disabled prop is true', () => {
		render(<Button disabled>Disabled</Button>);
		expect(screen.getByRole('button')).toBeDisabled();
	});

	it('fires onClick handler', () => {
		const onClick = vi.fn();
		render(<Button onClick={onClick}>Click</Button>);
		fireEvent.click(screen.getByRole('button'));
		expect(onClick).toHaveBeenCalledTimes(1);
	});

	it('does not fire onClick when disabled', () => {
		const onClick = vi.fn();
		render(
			<Button onClick={onClick} disabled>
				Click
			</Button>,
		);
		fireEvent.click(screen.getByRole('button'));
		expect(onClick).not.toHaveBeenCalled();
	});

	it('has type="button" by default', () => {
		render(<Button>Test</Button>);
		expect(screen.getByRole('button')).toHaveAttribute('type', 'button');
	});
});
