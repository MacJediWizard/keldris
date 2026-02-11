import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Card, CardContent, CardFooter, CardHeader } from './Card';

describe('Card', () => {
	it('renders children', () => {
		render(<Card>Card content</Card>);
		expect(screen.getByText('Card content')).toBeInTheDocument();
	});

	it('applies card styling', () => {
		const { container } = render(<Card>Content</Card>);
		expect(container.firstElementChild).toHaveClass(
			'rounded-lg',
			'border',
			'bg-white',
			'shadow-sm',
		);
	});

	it('accepts custom className', () => {
		const { container } = render(<Card className="mt-4">Content</Card>);
		expect(container.firstElementChild).toHaveClass('mt-4');
	});
});

describe('CardHeader', () => {
	it('renders children', () => {
		render(<CardHeader>Header</CardHeader>);
		expect(screen.getByText('Header')).toBeInTheDocument();
	});

	it('has border-bottom', () => {
		const { container } = render(<CardHeader>Header</CardHeader>);
		expect(container.firstElementChild).toHaveClass('border-b');
	});
});

describe('CardContent', () => {
	it('renders children', () => {
		render(<CardContent>Body</CardContent>);
		expect(screen.getByText('Body')).toBeInTheDocument();
	});

	it('has padding', () => {
		const { container } = render(<CardContent>Body</CardContent>);
		expect(container.firstElementChild).toHaveClass('px-6', 'py-4');
	});
});

describe('CardFooter', () => {
	it('renders children', () => {
		render(<CardFooter>Footer</CardFooter>);
		expect(screen.getByText('Footer')).toBeInTheDocument();
	});

	it('has border-top', () => {
		const { container } = render(<CardFooter>Footer</CardFooter>);
		expect(container.firstElementChild).toHaveClass('border-t');
	});
});
