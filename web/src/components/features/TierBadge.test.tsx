import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { TierBadge } from './TierBadge';

describe('TierBadge', () => {
	it('renders Free label for free tier', () => {
		render(<TierBadge tier="free" />);
		expect(screen.getByText('Free')).toBeDefined();
	});

	it('renders Pro label for pro tier', () => {
		render(<TierBadge tier="pro" />);
		expect(screen.getByText('Pro')).toBeDefined();
	});

	it('renders Professional label for professional tier', () => {
		render(<TierBadge tier="professional" />);
		expect(screen.getByText('Professional')).toBeDefined();
	});

	it('renders Enterprise label for enterprise tier', () => {
		render(<TierBadge tier="enterprise" />);
		expect(screen.getByText('Enterprise')).toBeDefined();
	});

	it('applies enterprise tier purple style', () => {
		const { container } = render(<TierBadge tier="enterprise" />);
		expect((container.firstChild as HTMLElement).className).toContain(
			'bg-purple-100',
		);
	});

	it('appends custom className', () => {
		const { container } = render(
			<TierBadge tier="free" className="extra-class" />,
		);
		expect((container.firstChild as HTMLElement).className).toContain(
			'extra-class',
		);
	});
});
