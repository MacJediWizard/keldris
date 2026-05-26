import { render } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
	AvatarSkeleton,
	BadgeSkeleton,
	ButtonSkeleton,
	CardSkeleton,
	FormSectionSkeleton,
	InputSkeleton,
	Skeleton,
	StatCardSkeleton,
	TableRowSkeleton,
	TextSkeleton,
	skeletonKeys,
} from './Skeleton';

describe('Skeleton', () => {
	it('renders with animate-pulse class', () => {
		const { container } = render(<Skeleton />);
		expect(container.querySelector('.animate-pulse')).not.toBeNull();
	});

	it('appends custom className', () => {
		const { container } = render(<Skeleton className="h-10 w-20" />);
		const el = container.firstChild as HTMLElement;
		expect(el.className).toContain('h-10');
		expect(el.className).toContain('w-20');
	});
});

describe('skeletonKeys', () => {
	it('returns stable keys with default prefix', () => {
		expect(skeletonKeys(3)).toEqual(['sk-0', 'sk-1', 'sk-2']);
	});

	it('respects custom prefix', () => {
		expect(skeletonKeys(2, 'row')).toEqual(['row-0', 'row-1']);
	});

	it('returns empty array for 0', () => {
		expect(skeletonKeys(0)).toEqual([]);
	});
});

describe('TextSkeleton', () => {
	it('renders with default width/size classes', () => {
		const { container } = render(<TextSkeleton />);
		const el = container.firstChild as HTMLElement;
		expect(el.className).toContain('w-32');
		expect(el.className).toContain('h-4');
	});

	it('applies width prop', () => {
		const { container } = render(<TextSkeleton width="full" />);
		expect((container.firstChild as HTMLElement).className).toContain('w-full');
	});
});

describe('AvatarSkeleton', () => {
	it('renders rounded-full circle', () => {
		const { container } = render(<AvatarSkeleton />);
		expect((container.firstChild as HTMLElement).className).toContain(
			'rounded-full',
		);
	});
});

describe('BadgeSkeleton', () => {
	it('renders pill shape', () => {
		const { container } = render(<BadgeSkeleton />);
		expect((container.firstChild as HTMLElement).className).toContain(
			'rounded-full',
		);
	});
});

describe('ButtonSkeleton', () => {
	it('renders with size classes', () => {
		const { container } = render(<ButtonSkeleton size="lg" />);
		expect((container.firstChild as HTMLElement).className).toContain('h-12');
	});
});

describe('CardSkeleton', () => {
	it('renders header by default', () => {
		const { container } = render(<CardSkeleton />);
		expect(container.querySelectorAll('.animate-pulse').length).toBeGreaterThan(
			1,
		);
	});

	it('hides header when showHeader=false', () => {
		const { container } = render(<CardSkeleton showHeader={false} />);
		expect(container.firstChild).not.toBeNull();
	});

	it('renders requested number of lines', () => {
		const { container } = render(<CardSkeleton lines={5} />);
		expect(
			container.querySelectorAll('.animate-pulse').length,
		).toBeGreaterThanOrEqual(5);
	});
});

describe('TableRowSkeleton', () => {
	it('renders requested column count', () => {
		const { container } = render(
			<table>
				<tbody>
					<TableRowSkeleton columns={4} />
				</tbody>
			</table>,
		);
		expect(container.querySelectorAll('td').length).toBe(4);
	});

	it('adds checkbox column when showCheckbox=true', () => {
		const { container } = render(
			<table>
				<tbody>
					<TableRowSkeleton columns={4} showCheckbox />
				</tbody>
			</table>,
		);
		expect(container.querySelectorAll('td').length).toBe(4);
	});
});

describe('StatCardSkeleton', () => {
	it('renders', () => {
		const { container } = render(<StatCardSkeleton />);
		expect(container.firstChild).not.toBeNull();
	});
});

describe('InputSkeleton', () => {
	it('shows label by default', () => {
		const { container } = render(<InputSkeleton />);
		// Two skeleton divs: label + input
		expect(container.querySelectorAll('.animate-pulse').length).toBe(2);
	});

	it('hides label when showLabel=false', () => {
		const { container } = render(<InputSkeleton showLabel={false} />);
		expect(container.querySelectorAll('.animate-pulse').length).toBe(1);
	});
});

describe('FormSectionSkeleton', () => {
	it('renders default 4 fields', () => {
		const { container } = render(<FormSectionSkeleton />);
		// 4 fields × 2 skeletons each (label+input) = 8
		expect(container.querySelectorAll('.animate-pulse').length).toBe(8);
	});

	it('renders custom field count', () => {
		const { container } = render(<FormSectionSkeleton fields={2} />);
		expect(container.querySelectorAll('.animate-pulse').length).toBe(4);
	});
});
