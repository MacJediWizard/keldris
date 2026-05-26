import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import {
	ColdRestoreStatusBadge,
	StorageTierBadge,
	StorageTierSelect,
	TierCostIndicator,
} from './StorageTierBadge';

describe('StorageTierBadge', () => {
	it('renders Hot label', () => {
		render(<StorageTierBadge tier="hot" />);
		expect(screen.getByText('Hot')).toBeDefined();
	});

	it('renders Warm label', () => {
		render(<StorageTierBadge tier="warm" />);
		expect(screen.getByText('Warm')).toBeDefined();
	});

	it('renders Cold label', () => {
		render(<StorageTierBadge tier="cold" />);
		expect(screen.getByText('Cold')).toBeDefined();
	});

	it('renders Archive label', () => {
		render(<StorageTierBadge tier="archive" />);
		expect(screen.getByText('Archive')).toBeDefined();
	});

	it('shows age when showAge=true', () => {
		render(<StorageTierBadge tier="cold" ageDays={42} showAge />);
		expect(screen.getByText('(42d)')).toBeDefined();
	});

	it('hides age by default', () => {
		render(<StorageTierBadge tier="cold" ageDays={42} />);
		expect(screen.queryByText('(42d)')).toBeNull();
	});
});

describe('StorageTierSelect', () => {
	it('renders 4 tier options by default', () => {
		render(<StorageTierSelect value="hot" onChange={() => {}} />);
		expect(screen.getAllByRole('option')).toHaveLength(4);
	});

	it('excludes specified tiers', () => {
		render(
			<StorageTierSelect
				value="hot"
				onChange={() => {}}
				excludeTiers={['archive']}
			/>,
		);
		expect(screen.getAllByRole('option')).toHaveLength(3);
		expect(screen.queryByRole('option', { name: 'Archive' })).toBeNull();
	});

	it('fires onChange when selection changes', () => {
		const onChange = vi.fn();
		render(<StorageTierSelect value="hot" onChange={onChange} />);
		fireEvent.change(screen.getByRole('combobox'), {
			target: { value: 'archive' },
		});
		expect(onChange).toHaveBeenCalledWith('archive');
	});
});

describe('TierCostIndicator', () => {
	it('renders tier badge + cost', () => {
		render(<TierCostIndicator tier="hot" monthlyCost={4.5} />);
		expect(screen.getByText('Hot')).toBeDefined();
		expect(screen.getByText('$4.50/mo')).toBeDefined();
	});

	it('renders <$0.01 when cost very small', () => {
		render(<TierCostIndicator tier="archive" monthlyCost={0.001} />);
		expect(screen.getByText('<$0.01/mo')).toBeDefined();
	});
});

describe('ColdRestoreStatusBadge', () => {
	it('renders Pending status', () => {
		render(<ColdRestoreStatusBadge status="pending" />);
		expect(screen.getByText('Pending')).toBeDefined();
	});

	it('renders Warming status with animated dot', () => {
		const { container } = render(<ColdRestoreStatusBadge status="warming" />);
		expect(screen.getByText('Warming')).toBeDefined();
		expect(container.querySelector('.animate-pulse')).not.toBeNull();
	});

	it('renders ETA when warming with estimatedReadyAt', () => {
		const future = new Date(Date.now() + 30 * 60_000).toISOString();
		render(
			<ColdRestoreStatusBadge status="warming" estimatedReadyAt={future} />,
		);
		expect(screen.getByText(/30m|29m|31m/)).toBeDefined();
	});

	it('falls back to Pending for unknown status', () => {
		render(<ColdRestoreStatusBadge status="bogus" />);
		expect(screen.getByText('Pending')).toBeDefined();
	});
});
