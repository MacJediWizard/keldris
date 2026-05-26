import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useImmutability', () => ({
	useSnapshotImmutabilityStatus: vi.fn(),
}));

import { useSnapshotImmutabilityStatus } from '../../hooks/useImmutability';
import {
	ImmutabilityDeleteWarning,
	LockIcon,
	SnapshotImmutabilityBadge,
} from './SnapshotImmutabilityBadge';

function setStatus(status: unknown, isLoading = false) {
	vi.mocked(useSnapshotImmutabilityStatus).mockReturnValue({
		data: status,
		isLoading,
	} as never);
}

describe('SnapshotImmutabilityBadge', () => {
	it('renders nothing while loading', () => {
		setStatus(undefined, true);
		const { container } = render(
			<SnapshotImmutabilityBadge snapshotId="snap" repositoryId="repo" />,
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders nothing when not locked', () => {
		setStatus({ is_locked: false });
		const { container } = render(
			<SnapshotImmutabilityBadge snapshotId="snap" repositoryId="repo" />,
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders badge when locked', () => {
		setStatus({
			is_locked: true,
			locked_until: '2026-12-31T00:00:00Z',
			remaining_days: 45,
		});
		const { container } = render(
			<SnapshotImmutabilityBadge snapshotId="snap" repositoryId="repo" />,
		);
		expect(container.querySelector('span.bg-amber-100')).not.toBeNull();
	});

	it('shows days when showDetails=true', () => {
		setStatus({ is_locked: true, remaining_days: 12 });
		render(
			<SnapshotImmutabilityBadge
				snapshotId="snap"
				repositoryId="repo"
				showDetails
			/>,
		);
		expect(screen.getByText('12d')).toBeDefined();
	});
});

describe('LockIcon', () => {
	it('renders nothing when isLocked=false', () => {
		const { container } = render(<LockIcon isLocked={false} />);
		expect(container.firstChild).toBeNull();
	});

	it('renders SVG when locked', () => {
		const { container } = render(<LockIcon isLocked />);
		expect(container.querySelector('svg')).not.toBeNull();
	});

	it('applies size class', () => {
		const { container } = render(<LockIcon isLocked size="lg" />);
		expect(container.querySelector('.w-5')).not.toBeNull();
	});
});

describe('ImmutabilityDeleteWarning', () => {
	it('renders nothing when isLocked=false', () => {
		const { container } = render(
			<ImmutabilityDeleteWarning isLocked={false} />,
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders warning copy when locked', () => {
		render(<ImmutabilityDeleteWarning isLocked />);
		expect(screen.getByText('Snapshot is immutable')).toBeDefined();
	});

	it('includes remaining days when provided', () => {
		render(<ImmutabilityDeleteWarning isLocked remainingDays={7} />);
		expect(screen.getByText(/7 days remaining/)).toBeDefined();
	});
});
