import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useTrial', () => ({
	useTrialStatus: vi.fn(),
	useStartTrial: vi.fn(),
}));

vi.mock('react-i18next', () => ({
	useTranslation: () => ({
		t: (
			_key: string,
			defaultValue: string,
			vars?: Record<string, string | number>,
		) => {
			if (!vars) return defaultValue;
			return defaultValue.replace(/\{\{(\w+)\}\}/g, (_m, k) =>
				String(vars[k] ?? ''),
			);
		},
	}),
}));

import { useStartTrial, useTrialStatus } from '../../hooks/useTrial';
import { TrialBanner } from './TrialBanner';

function setTrial(data: unknown, isLoading = false, isError = false) {
	vi.mocked(useTrialStatus).mockReturnValue({
		data,
		isLoading,
		isError,
	} as never);
	vi.mocked(useStartTrial).mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
		isError: false,
	} as never);
}

describe('TrialBanner', () => {
	it('renders nothing while loading', () => {
		setTrial(undefined, true);
		const { container } = render(<TrialBanner />);
		expect(container.firstChild).toBeNull();
	});

	it('renders nothing on error', () => {
		setTrial(undefined, false, true);
		const { container } = render(<TrialBanner />);
		expect(container.firstChild).toBeNull();
	});

	it('renders nothing when no trial info', () => {
		setTrial(undefined);
		const { container } = render(<TrialBanner />);
		expect(container.firstChild).toBeNull();
	});

	it('renders nothing when already on paid plan', () => {
		setTrial({ plan_tier: 'pro', trial_status: 'none' });
		const { container } = render(<TrialBanner />);
		expect(container.firstChild).toBeNull();
	});

	it('renders nothing when trial is converted', () => {
		setTrial({ plan_tier: 'free', trial_status: 'converted' });
		const { container } = render(<TrialBanner />);
		expect(container.firstChild).toBeNull();
	});

	it('renders expired trial banner', () => {
		setTrial({ plan_tier: 'free', trial_status: 'expired' });
		render(<TrialBanner />);
		expect(screen.getByText('Pro Trial Expired')).toBeDefined();
	});

	it('renders active trial banner with days remaining', () => {
		setTrial({
			plan_tier: 'free',
			trial_status: 'active',
			is_trial_active: true,
			days_remaining: 21,
		});
		render(<TrialBanner />);
		expect(screen.getByText('Pro trial: 21 days remaining')).toBeDefined();
	});

	it('renders start trial prompt when trial_status=none', () => {
		setTrial({
			plan_tier: 'free',
			trial_status: 'none',
			is_trial_active: false,
		});
		render(<TrialBanner />);
		expect(screen.getByText('Try Pro features free for 30 days')).toBeDefined();
	});
});
