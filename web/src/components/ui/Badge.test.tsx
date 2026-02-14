import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Badge } from './Badge';

describe('Badge', () => {
	it('renders children', () => {
		render(<Badge>Active</Badge>);
		expect(screen.getByText('Active')).toBeInTheDocument();
	});

	it('applies default variant by default', () => {
		render(<Badge>Default</Badge>);
		const badge = screen.getByText('Default');
		expect(badge).toHaveClass('bg-gray-100', 'text-gray-800');
	});

	it('applies success variant', () => {
		render(<Badge variant="success">Success</Badge>);
		const badge = screen.getByText('Success');
		expect(badge).toHaveClass('bg-green-100', 'text-green-800');
	});

	it('applies warning variant', () => {
		render(<Badge variant="warning">Warning</Badge>);
		const badge = screen.getByText('Warning');
		expect(badge).toHaveClass('bg-yellow-100', 'text-yellow-800');
	});

	it('applies error variant', () => {
		render(<Badge variant="error">Error</Badge>);
		const badge = screen.getByText('Error');
		expect(badge).toHaveClass('bg-red-100', 'text-red-800');
	});

	it('applies info variant', () => {
		render(<Badge variant="info">Info</Badge>);
		const badge = screen.getByText('Info');
		expect(badge).toHaveClass('bg-blue-100', 'text-blue-800');
	});

	it('applies small size by default', () => {
		render(<Badge>Small</Badge>);
		const badge = screen.getByText('Small');
		expect(badge).toHaveClass('px-2', 'py-0.5', 'text-xs');
	});

	it('applies medium size', () => {
		render(<Badge size="md">Medium</Badge>);
		const badge = screen.getByText('Medium');
		expect(badge).toHaveClass('px-2.5', 'py-0.5', 'text-sm');
	});

	it('renders as a span element', () => {
		const { container } = render(<Badge>Test</Badge>);
		expect(container.querySelector('span')).toBeInTheDocument();
	});

	it('has rounded-full class', () => {
		render(<Badge>Rounded</Badge>);
		const badge = screen.getByText('Rounded');
		expect(badge).toHaveClass('rounded-full');
	});
});
