import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { UpgradeLimitWarning, UpgradePrompt } from './UpgradePrompt';

function withRouter(ui: React.ReactNode) {
	return <MemoryRouter>{ui}</MemoryRouter>;
}

describe('UpgradePrompt', () => {
	it('renders banner variant by default', () => {
		render(withRouter(<UpgradePrompt feature="custom_branding" />));
		expect(screen.getByText(/Unlock|requires/)).toBeDefined();
	});

	it('renders inline variant', () => {
		render(
			withRouter(<UpgradePrompt feature="custom_branding" variant="inline" />),
		);
		expect(screen.getByText(/requires/)).toBeDefined();
	});

	it('renders card variant with benefits', () => {
		render(
			withRouter(<UpgradePrompt feature="custom_branding" variant="card" />),
		);
		expect(screen.getByRole('heading')).toBeDefined();
	});

	it('renders modal variant', () => {
		render(
			withRouter(<UpgradePrompt feature="custom_branding" variant="modal" />),
		);
		expect(screen.getByRole('heading')).toBeDefined();
	});

	it('fires onDismiss when modal close clicked', () => {
		const onDismiss = vi.fn();
		render(
			withRouter(
				<UpgradePrompt
					feature="custom_branding"
					variant="modal"
					onDismiss={onDismiss}
				/>,
			),
		);
		screen.getByRole('button', { name: 'Close' }).click();
		expect(onDismiss).toHaveBeenCalledOnce();
	});

	it('renders unknown feature with fallback label', () => {
		render(
			withRouter(
				<UpgradePrompt
					feature={'completely_unknown' as 'custom_branding'}
					variant="inline"
				/>,
			),
		);
		expect(screen.getByText(/Completely Unknown/)).toBeDefined();
	});
});

describe('UpgradeLimitWarning', () => {
	it('renders nothing when below 80% usage', () => {
		const { container } = render(
			withRouter(
				<UpgradeLimitWarning type="agents" current={10} limit={100} />,
			),
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders amber warning when near limit (80-99%)', () => {
		render(
			withRouter(
				<UpgradeLimitWarning type="agents" current={85} limit={100} />,
			),
		);
		expect(screen.getByText(/Approaching/)).toBeDefined();
	});

	it('renders red warning when at limit (>=100%)', () => {
		render(
			withRouter(
				<UpgradeLimitWarning type="agents" current={100} limit={100} />,
			),
		);
		expect(screen.getByText(/limit reached/)).toBeDefined();
	});

	it('formats storage values in GB/TB', () => {
		const oneGB = 1024 * 1024 * 1024;
		render(
			withRouter(
				<UpgradeLimitWarning
					type="storage"
					current={oneGB * 80}
					limit={oneGB * 100}
				/>,
			),
		);
		expect(screen.getByText(/80.0 GB/)).toBeDefined();
	});
});
