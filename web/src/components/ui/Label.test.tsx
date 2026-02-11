import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Label } from './Label';

describe('Label', () => {
	it('renders children text', () => {
		render(<Label>Username</Label>);
		expect(screen.getByText('Username')).toBeInTheDocument();
	});

	it('renders as a label element', () => {
		const { container } = render(<Label>Email</Label>);
		expect(container.querySelector('label')).toBeInTheDocument();
	});

	it('applies font-medium styling', () => {
		const { container } = render(<Label>Name</Label>);
		expect(container.querySelector('label')).toHaveClass(
			'text-sm',
			'font-medium',
		);
	});

	it('shows required indicator when required', () => {
		render(<Label required>Email</Label>);
		expect(screen.getByText('*')).toBeInTheDocument();
		expect(screen.getByText('*')).toHaveClass('text-red-500');
	});

	it('does not show required indicator by default', () => {
		render(<Label>Email</Label>);
		expect(screen.queryByText('*')).not.toBeInTheDocument();
	});

	it('passes htmlFor prop', () => {
		const { container } = render(<Label htmlFor="email-input">Email</Label>);
		expect(container.querySelector('label')).toHaveAttribute(
			'for',
			'email-input',
		);
	});

	it('accepts custom className', () => {
		const { container } = render(<Label className="mb-2">Name</Label>);
		expect(container.querySelector('label')).toHaveClass('mb-2');
	});
});
