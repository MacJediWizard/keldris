import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useReadOnlyMode', () => ({
	useReadOnlyMode: vi.fn(),
}));

import { useReadOnlyMode } from '../../hooks/useReadOnlyMode';
import { ReadOnlyBlocker, ReadOnlyDisabledButton } from './ReadOnlyBlocker';

function setReadOnly(isReadOnly: boolean, maintenanceTitle?: string) {
	vi.mocked(useReadOnlyMode).mockReturnValue({
		isReadOnly,
		maintenanceTitle,
	} as never);
}

describe('ReadOnlyBlocker', () => {
	it('renders children when not in read-only mode', () => {
		setReadOnly(false);
		render(
			<ReadOnlyBlocker>
				<span>inner</span>
			</ReadOnlyBlocker>,
		);
		expect(screen.getByText('inner')).toBeDefined();
	});

	it('renders fallback when in read-only mode and fallback provided', () => {
		setReadOnly(true);
		render(
			<ReadOnlyBlocker fallback={<span>fallback-node</span>}>
				<span>inner</span>
			</ReadOnlyBlocker>,
		);
		expect(screen.getByText('fallback-node')).toBeDefined();
		expect(screen.queryByText('inner')).toBeNull();
	});

	it('renders message when in read-only mode and no fallback', () => {
		setReadOnly(true, 'Quarterly maintenance');
		render(
			<ReadOnlyBlocker>
				<span>inner</span>
			</ReadOnlyBlocker>,
		);
		expect(screen.getByText(/Read-only mode/)).toBeDefined();
		expect(screen.getByText(/Quarterly maintenance/)).toBeDefined();
	});

	it('renders nothing when in read-only mode and showMessage=false', () => {
		setReadOnly(true);
		const { container } = render(
			<ReadOnlyBlocker showMessage={false}>
				<span>inner</span>
			</ReadOnlyBlocker>,
		);
		expect(container.firstChild).toBeNull();
	});
});

describe('ReadOnlyDisabledButton', () => {
	it('fires onClick when not read-only and not disabled', () => {
		setReadOnly(false);
		const onClick = vi.fn();
		render(
			<ReadOnlyDisabledButton onClick={onClick}>Click</ReadOnlyDisabledButton>,
		);
		screen.getByRole('button', { name: 'Click' }).click();
		expect(onClick).toHaveBeenCalledOnce();
	});

	it('disables button when read-only', () => {
		setReadOnly(true, 'Window 7');
		render(<ReadOnlyDisabledButton>Click</ReadOnlyDisabledButton>);
		const btn = screen.getByRole('button', { name: 'Click' });
		expect(btn).toBeDisabled();
		expect(btn.getAttribute('title')).toContain('Window 7');
	});

	it('respects disabled prop even when not read-only', () => {
		setReadOnly(false);
		render(
			<ReadOnlyDisabledButton disabled={true}>Click</ReadOnlyDisabledButton>,
		);
		expect(screen.getByRole('button', { name: 'Click' })).toBeDisabled();
	});
});
