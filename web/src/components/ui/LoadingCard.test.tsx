import { render } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { LoadingCard } from './LoadingCard';

describe('LoadingCard', () => {
	it('renders default stat variant', () => {
		const { container } = render(<LoadingCard />);
		expect(container.firstChild).toBeDefined();
		expect(container.querySelector('.animate-pulse')).not.toBeNull();
	});

	it('renders each preset variant without crashing', () => {
		const variants = [
			'stat',
			'stat-sm',
			'alert',
			'template',
			'repo',
			'sla',
			'health',
		] as const;
		for (const variant of variants) {
			const { container } = render(<LoadingCard variant={variant} />);
			expect(container.firstChild).toBeDefined();
		}
	});

	it('renders children when provided', () => {
		const { getByText } = render(
			<LoadingCard>
				<span>custom skeleton</span>
			</LoadingCard>,
		);
		expect(getByText('custom skeleton')).toBeDefined();
	});

	it('applies custom className', () => {
		const { container } = render(<LoadingCard className="my-extra" />);
		expect(
			(container.firstChild as HTMLElement).className.includes('my-extra'),
		).toBe(true);
	});
});
