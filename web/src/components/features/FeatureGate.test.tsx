import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useLicenses', () => ({
	useCurrentLicense: vi.fn(),
	useLicensePurchaseUrl: vi.fn(() => ({ data: undefined })),
}));

import {
	useCurrentLicense,
	useLicensePurchaseUrl,
} from '../../hooks/useLicenses';
import {
	FeatureCheck,
	FeatureDisabledButton,
	FeatureGate,
	ProBadge,
} from './FeatureGate';

function setLicense(license: unknown, isLoading = false) {
	vi.mocked(useCurrentLicense).mockReturnValue({
		data: license,
		isLoading,
	} as never);
	vi.mocked(useLicensePurchaseUrl).mockReturnValue({
		data: undefined,
	} as never);
}

function withRouter(ui: React.ReactNode) {
	return <MemoryRouter>{ui}</MemoryRouter>;
}

describe('FeatureGate', () => {
	it('renders children while loading', () => {
		setLicense(undefined, true);
		render(
			withRouter(
				<FeatureGate feature="sso">
					<span>gated content</span>
				</FeatureGate>,
			),
		);
		expect(screen.getByText('gated content')).toBeDefined();
	});

	it('renders children when feature is in license', () => {
		setLicense({ features: ['oidc'], limits: {} });
		render(
			withRouter(
				<FeatureGate feature="sso">
					<span>gated content</span>
				</FeatureGate>,
			),
		);
		expect(screen.getByText('gated content')).toBeDefined();
	});

	it('renders default upgrade prompt when feature missing', () => {
		setLicense({ features: [], limits: {} });
		render(
			withRouter(
				<FeatureGate feature="sso">
					<span>gated content</span>
				</FeatureGate>,
			),
		);
		expect(screen.getByText('Pro Feature')).toBeDefined();
		expect(screen.getByText('Single Sign-On')).toBeDefined();
	});

	it('renders custom fallback when provided', () => {
		setLicense({ features: [], limits: {} });
		render(
			withRouter(
				<FeatureGate feature="sso" fallback={<span>custom fallback</span>}>
					<span>gated</span>
				</FeatureGate>,
			),
		);
		expect(screen.getByText('custom fallback')).toBeDefined();
		expect(screen.queryByText('Pro Feature')).toBeNull();
	});
});

describe('ProBadge', () => {
	it('renders nothing while loading', () => {
		setLicense(undefined, true);
		const { container } = render(<ProBadge feature="sso" />);
		expect(container.firstChild).toBeNull();
	});

	it('renders nothing when feature available', () => {
		setLicense({ features: ['oidc'], limits: {} });
		const { container } = render(<ProBadge feature="sso" />);
		expect(container.firstChild).toBeNull();
	});

	it('renders Pro badge when feature missing', () => {
		setLicense({ features: [], limits: {} });
		render(<ProBadge feature="sso" />);
		expect(screen.getByText('Pro')).toBeDefined();
	});

	it('renders feature label when showLabel=true', () => {
		setLicense({ features: [], limits: {} });
		render(<ProBadge feature="sso" showLabel />);
		expect(screen.getByText('Single Sign-On')).toBeDefined();
	});
});

describe('FeatureDisabledButton', () => {
	it('renders enabled button when feature available', () => {
		setLicense({ features: ['oidc'], limits: {} });
		render(
			<FeatureDisabledButton feature="sso">Click me</FeatureDisabledButton>,
		);
		const btn = screen.getByRole('button', { name: /Click me/ });
		expect(btn).not.toBeDisabled();
	});

	it('renders disabled button when feature missing', () => {
		setLicense({ features: [], limits: {} });
		render(
			<FeatureDisabledButton feature="sso">Click me</FeatureDisabledButton>,
		);
		const btn = screen.getByRole('button', { name: /Click me/ });
		expect(btn).toBeDisabled();
	});
});

describe('FeatureCheck', () => {
	it('passes hasFeature=true when feature available', () => {
		setLicense({ features: ['oidc'], limits: {} });
		render(
			<FeatureCheck feature="sso">
				{(has, label) => (
					<span>
						{label}: {has ? 'yes' : 'no'}
					</span>
				)}
			</FeatureCheck>,
		);
		expect(screen.getByText('Single Sign-On: yes')).toBeDefined();
	});

	it('passes hasFeature=false when feature missing', () => {
		setLicense({ features: [], limits: {} });
		render(
			<FeatureCheck feature="sso">
				{(has, label) => (
					<span>
						{label}: {has ? 'yes' : 'no'}
					</span>
				)}
			</FeatureCheck>,
		);
		expect(screen.getByText('Single Sign-On: no')).toBeDefined();
	});
});
