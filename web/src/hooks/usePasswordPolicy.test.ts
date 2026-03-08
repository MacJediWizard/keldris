import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useChangePassword,
	usePasswordExpiration,
	usePasswordPolicy,
	usePasswordRequirements,
	useUpdatePasswordPolicy,
	useValidatePassword,
} from './usePasswordPolicy';

vi.mock('../lib/api', () => ({
	passwordApi: {
		changePassword: vi.fn(),
		getExpiration: vi.fn(),
	},
	passwordPoliciesApi: {
		get: vi.fn(),
		getRequirements: vi.fn(),
		update: vi.fn(),
		validatePassword: vi.fn(),
	},
}));

import { passwordApi, passwordPoliciesApi } from '../lib/api';

const mockPolicy = {
	id: 'policy-1',
	org_id: 'org-1',
	min_length: 12,
	require_uppercase: true,
	require_lowercase: true,
	require_number: true,
	require_special: true,
	max_age_days: 90,
	history_count: 5,
	created_at: '2026-01-01T00:00:00Z',
	updated_at: '2026-01-01T00:00:00Z',
};

const mockRequirements = {
	min_length: 12,
	require_uppercase: true,
	require_lowercase: true,
	require_number: true,
	require_special: true,
	max_age_days: 90,
	description:
		'At least 12 characters with uppercase, lowercase, number, and special character',
};

const mockPolicyResponse = {
	policy: mockPolicy,
	requirements: mockRequirements,
};

describe('usePasswordPolicy', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('returns the full password policy with requirements', async () => {
		vi.mocked(passwordPoliciesApi.get).mockResolvedValue(mockPolicyResponse);

		const { result } = renderHook(() => usePasswordPolicy(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockPolicyResponse);
		expect(result.current.data?.policy.min_length).toBe(12);
		expect(result.current.data?.requirements.require_special).toBe(true);
	});

	it('handles error fetching policy', async () => {
		vi.mocked(passwordPoliciesApi.get).mockRejectedValue(
			new Error('Forbidden'),
		);

		const { result } = renderHook(() => usePasswordPolicy(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isError).toBe(true));
	});
});

describe('usePasswordRequirements', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('returns policy rules (min length, complexity)', async () => {
		vi.mocked(passwordPoliciesApi.getRequirements).mockResolvedValue(
			mockRequirements,
		);

		const { result } = renderHook(() => usePasswordRequirements(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data?.min_length).toBe(12);
		expect(result.current.data?.require_uppercase).toBe(true);
		expect(result.current.data?.require_lowercase).toBe(true);
		expect(result.current.data?.require_number).toBe(true);
		expect(result.current.data?.require_special).toBe(true);
		expect(result.current.data?.max_age_days).toBe(90);
	});

	it('returns requirements without optional fields', async () => {
		const minimalRequirements = {
			min_length: 8,
			require_uppercase: false,
			require_lowercase: false,
			require_number: false,
			require_special: false,
			description: 'At least 8 characters',
		};
		vi.mocked(passwordPoliciesApi.getRequirements).mockResolvedValue(
			minimalRequirements,
		);

		const { result } = renderHook(() => usePasswordRequirements(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data?.min_length).toBe(8);
		expect(result.current.data?.require_uppercase).toBe(false);
		expect(result.current.data?.max_age_days).toBeUndefined();
	});
});

describe('useValidatePassword', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('validates a strong password successfully', async () => {
		vi.mocked(passwordPoliciesApi.validatePassword).mockResolvedValue({
			valid: true,
		});

		const { result } = renderHook(() => useValidatePassword(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('Str0ng!P@ssw0rd');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data?.valid).toBe(true);
		expect(result.current.data?.errors).toBeUndefined();
		expect(passwordPoliciesApi.validatePassword).toHaveBeenCalledWith(
			'Str0ng!P@ssw0rd',
		);
	});

	it('returns validation errors for weak password', async () => {
		vi.mocked(passwordPoliciesApi.validatePassword).mockResolvedValue({
			valid: false,
			errors: [
				'Password must be at least 12 characters',
				'Password must contain a special character',
			],
		});

		const { result } = renderHook(() => useValidatePassword(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('short');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data?.valid).toBe(false);
		expect(result.current.data?.errors).toHaveLength(2);
		expect(result.current.data?.errors?.[0]).toBe(
			'Password must be at least 12 characters',
		);
		expect(result.current.data?.errors?.[1]).toBe(
			'Password must contain a special character',
		);
	});

	it('returns warnings alongside valid status', async () => {
		vi.mocked(passwordPoliciesApi.validatePassword).mockResolvedValue({
			valid: true,
			warnings: ['This password has been seen in data breaches'],
		});

		const { result } = renderHook(() => useValidatePassword(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('CommonButLong123!');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data?.valid).toBe(true);
		expect(result.current.data?.warnings).toHaveLength(1);
		expect(result.current.data?.warnings?.[0]).toBe(
			'This password has been seen in data breaches',
		);
	});

	it('returns multiple errors per rule', async () => {
		vi.mocked(passwordPoliciesApi.validatePassword).mockResolvedValue({
			valid: false,
			errors: [
				'Password must be at least 12 characters',
				'Password must contain an uppercase letter',
				'Password must contain a number',
				'Password must contain a special character',
			],
		});

		const { result } = renderHook(() => useValidatePassword(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('abc');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data?.valid).toBe(false);
		expect(result.current.data?.errors).toHaveLength(4);
	});
});

describe('useUpdatePasswordPolicy', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('updates the password policy', async () => {
		vi.mocked(passwordPoliciesApi.update).mockResolvedValue(mockPolicy);

		const { result } = renderHook(() => useUpdatePasswordPolicy(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			min_length: 16,
			require_special: true,
			max_age_days: 60,
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(passwordPoliciesApi.update).toHaveBeenCalledWith({
			min_length: 16,
			require_special: true,
			max_age_days: 60,
		});
	});

	it('handles update failure', async () => {
		vi.mocked(passwordPoliciesApi.update).mockRejectedValue(
			new Error('Permission denied'),
		);

		const { result } = renderHook(() => useUpdatePasswordPolicy(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ min_length: 4 });

		await waitFor(() => expect(result.current.isError).toBe(true));
	});
});

describe('useChangePassword', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('changes password with valid credentials', async () => {
		vi.mocked(passwordApi.changePassword).mockResolvedValue({
			message: 'Password changed successfully',
		});

		const { result } = renderHook(() => useChangePassword(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			current_password: 'OldPassword123!',
			new_password: 'NewPassword456!',
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(passwordApi.changePassword).toHaveBeenCalledWith({
			current_password: 'OldPassword123!',
			new_password: 'NewPassword456!',
		});
	});

	it('rejects wrong current password', async () => {
		vi.mocked(passwordApi.changePassword).mockRejectedValue(
			new Error('Current password is incorrect'),
		);

		const { result } = renderHook(() => useChangePassword(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			current_password: 'WrongPassword',
			new_password: 'NewPassword456!',
		});

		await waitFor(() => expect(result.current.isError).toBe(true));
	});
});

describe('usePasswordExpiration', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches password expiration info', async () => {
		const mockExpiration = {
			expires_at: '2026-06-01T00:00:00Z',
			days_until_expiry: 85,
			expired: false,
			policy_max_age_days: 90,
		};
		vi.mocked(passwordApi.getExpiration).mockResolvedValue(mockExpiration);

		const { result } = renderHook(() => usePasswordExpiration(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data?.expired).toBe(false);
		expect(result.current.data?.days_until_expiry).toBe(85);
	});

	it('handles expired password', async () => {
		const mockExpiration = {
			expires_at: '2026-01-01T00:00:00Z',
			days_until_expiry: 0,
			expired: true,
			policy_max_age_days: 90,
		};
		vi.mocked(passwordApi.getExpiration).mockResolvedValue(mockExpiration);

		const { result } = renderHook(() => usePasswordExpiration(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data?.expired).toBe(true);
	});
});
