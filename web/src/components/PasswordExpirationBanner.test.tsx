import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../hooks/usePasswordPolicy', () => ({
	usePasswordExpiration: vi.fn(),
}));

vi.mock('./ChangePasswordForm', () => ({
	ChangePasswordForm: () => <div data-testid="change-password-form" />,
}));

import { usePasswordExpiration } from '../hooks/usePasswordPolicy';
import { PasswordExpirationBanner } from './PasswordExpirationBanner';

function setExpiration(
	data: unknown,
	{ isLoading = false }: { isLoading?: boolean } = {},
) {
	vi.mocked(usePasswordExpiration).mockReturnValue({
		data,
		isLoading,
	} as never);
}

describe('PasswordExpirationBanner', () => {
	it('renders nothing while loading', () => {
		setExpiration(undefined, { isLoading: true });
		const { container } = render(<PasswordExpirationBanner />);
		expect(container.firstChild).toBeNull();
	});

	it('renders nothing without expiration info', () => {
		setExpiration(undefined);
		const { container } = render(<PasswordExpirationBanner />);
		expect(container.firstChild).toBeNull();
	});

	it('renders nothing when password is fine', () => {
		setExpiration({
			is_expired: false,
			must_change_now: false,
			days_until_expiry: 365,
			warn_days_remaining: 7,
		});
		const { container } = render(<PasswordExpirationBanner />);
		expect(container.firstChild).toBeNull();
	});

	it('renders blocking modal when password expired', () => {
		setExpiration({
			is_expired: true,
			must_change_now: false,
			warn_days_remaining: 7,
		});
		render(<PasswordExpirationBanner />);
		expect(screen.getByText('Password Expired')).toBeDefined();
		expect(screen.getByTestId('change-password-form')).toBeDefined();
	});

	it('renders blocking modal when admin forced change', () => {
		setExpiration({
			is_expired: false,
			must_change_now: true,
			warn_days_remaining: 7,
		});
		render(<PasswordExpirationBanner />);
		expect(screen.getByText('Password Change Required')).toBeDefined();
	});

	it('renders warning banner when expiring soon', () => {
		setExpiration({
			is_expired: false,
			must_change_now: false,
			days_until_expiry: 3,
			warn_days_remaining: 7,
		});
		render(<PasswordExpirationBanner />);
		expect(
			screen.getByText(/Your password will expire in 3 days/),
		).toBeDefined();
		expect(screen.getByRole('button', { name: 'Change Now' })).toBeDefined();
	});
});
