import { screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useAuth', () => ({
	useMe: vi.fn(),
}));

vi.mock('../hooks/usePasswordPolicy', () => ({
	usePasswordPolicy: vi.fn(),
	useUpdatePasswordPolicy: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

import { useMe } from '../hooks/useAuth';
import { usePasswordPolicy } from '../hooks/usePasswordPolicy';
import { PasswordPolicies } from './PasswordPolicies';

const adminUser = {
	id: 'user-1',
	email: 'admin@example.com',
	current_org_id: 'org-1',
	current_org_role: 'admin',
};

const policyResponse = {
	policy: {
		min_length: 12,
		require_uppercase: true,
		require_lowercase: true,
		require_number: true,
		require_special: false,
		max_age_days: 90,
		history_count: 3,
	},
	requirements: {
		description: 'Minimum 12 characters with mixed case and numbers',
	},
};

describe('PasswordPolicies', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title for admin', () => {
		vi.mocked(useMe).mockReturnValue({ data: adminUser } as ReturnType<
			typeof useMe
		>);
		vi.mocked(usePasswordPolicy).mockReturnValue({
			data: policyResponse,
			isLoading: false,
			isError: false,
		} as ReturnType<typeof usePasswordPolicy>);
		renderWithProviders(<PasswordPolicies />);
		expect(screen.getByText('Password Policy')).toBeInTheDocument();
	});

	it('shows non-admin access restriction', () => {
		vi.mocked(useMe).mockReturnValue({
			data: { ...adminUser, current_org_role: 'member' },
		} as ReturnType<typeof useMe>);
		vi.mocked(usePasswordPolicy).mockReturnValue({
			data: undefined,
			isLoading: false,
			isError: false,
		} as ReturnType<typeof usePasswordPolicy>);
		renderWithProviders(<PasswordPolicies />);
		expect(
			screen.getByText('Only administrators can manage password policies.'),
		).toBeInTheDocument();
	});

	it('shows loading state', () => {
		vi.mocked(useMe).mockReturnValue({ data: adminUser } as ReturnType<
			typeof useMe
		>);
		vi.mocked(usePasswordPolicy).mockReturnValue({
			data: undefined,
			isLoading: true,
			isError: false,
		} as ReturnType<typeof usePasswordPolicy>);
		renderWithProviders(<PasswordPolicies />);
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows error state', () => {
		vi.mocked(useMe).mockReturnValue({ data: adminUser } as ReturnType<
			typeof useMe
		>);
		vi.mocked(usePasswordPolicy).mockReturnValue({
			data: undefined,
			isLoading: false,
			isError: true,
		} as ReturnType<typeof usePasswordPolicy>);
		renderWithProviders(<PasswordPolicies />);
		expect(
			screen.getByText('Failed to load password policy'),
		).toBeInTheDocument();
	});

	it('renders policy fields with data', () => {
		vi.mocked(useMe).mockReturnValue({ data: adminUser } as ReturnType<
			typeof useMe
		>);
		vi.mocked(usePasswordPolicy).mockReturnValue({
			data: policyResponse,
			isLoading: false,
			isError: false,
		} as ReturnType<typeof usePasswordPolicy>);
		renderWithProviders(<PasswordPolicies />);
		expect(
			screen.getByLabelText('Minimum Password Length'),
		).toBeInTheDocument();
		expect(screen.getByText('Character Requirements')).toBeInTheDocument();
	});
});
