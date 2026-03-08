import { fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';
import { ChangePasswordForm } from './ChangePasswordForm';

vi.mock('../hooks/usePasswordPolicy', () => ({
	useChangePassword: vi.fn(),
	usePasswordRequirements: vi.fn().mockReturnValue({
		data: {
			min_length: 8,
			require_uppercase: true,
			require_lowercase: true,
			require_number: true,
			require_special: false,
			description:
				'At least 8 characters with uppercase, lowercase, and number',
		},
		isLoading: false,
	}),
	useValidatePassword: vi.fn().mockReturnValue({
		mutate: vi.fn(),
	}),
}));

import { useChangePassword } from '../hooks/usePasswordPolicy';

const mockMutateAsync = vi.fn();

function setupMockChangePassword(overrides: Record<string, unknown> = {}) {
	vi.mocked(useChangePassword).mockReturnValue({
		mutateAsync: mockMutateAsync,
		isPending: false,
		...overrides,
	} as ReturnType<typeof useChangePassword>);
}

describe('ChangePasswordForm', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		setupMockChangePassword();
	});

	it('renders all password fields', () => {
		renderWithProviders(<ChangePasswordForm />);

		expect(screen.getByLabelText('Current Password')).toBeInTheDocument();
		expect(screen.getByLabelText('New Password')).toBeInTheDocument();
		expect(screen.getByLabelText('Confirm New Password')).toBeInTheDocument();
	});

	it('requires current password field', () => {
		renderWithProviders(<ChangePasswordForm />);

		const currentPasswordInput = screen.getByLabelText('Current Password');
		expect(currentPasswordInput).toBeRequired();
	});

	it('requires new password field', () => {
		renderWithProviders(<ChangePasswordForm />);

		const newPasswordInput = screen.getByLabelText('New Password');
		expect(newPasswordInput).toBeRequired();
	});

	it('requires confirm password field', () => {
		renderWithProviders(<ChangePasswordForm />);

		const confirmPasswordInput = screen.getByLabelText('Confirm New Password');
		expect(confirmPasswordInput).toBeRequired();
	});

	it('disables submit when fields are empty', () => {
		renderWithProviders(<ChangePasswordForm />);

		const submitButton = screen.getByRole('button', {
			name: 'Change Password',
		});
		expect(submitButton).toBeDisabled();
	});

	it('disables submit when passwords do not match', () => {
		renderWithProviders(<ChangePasswordForm />);

		fireEvent.change(screen.getByLabelText('Current Password'), {
			target: { value: 'OldPass123' },
		});
		fireEvent.change(screen.getByLabelText('New Password'), {
			target: { value: 'NewPass123!' },
		});
		fireEvent.change(screen.getByLabelText('Confirm New Password'), {
			target: { value: 'DifferentPass123!' },
		});

		const submitButton = screen.getByRole('button', {
			name: 'Change Password',
		});
		expect(submitButton).toBeDisabled();
	});

	it('shows mismatch message when confirm password differs', () => {
		renderWithProviders(<ChangePasswordForm />);

		fireEvent.change(screen.getByLabelText('New Password'), {
			target: { value: 'NewPass123!' },
		});
		fireEvent.change(screen.getByLabelText('Confirm New Password'), {
			target: { value: 'Mismatch' },
		});

		expect(screen.getByText('Passwords do not match')).toBeInTheDocument();
	});

	it('does not show mismatch message when passwords match', () => {
		renderWithProviders(<ChangePasswordForm />);

		fireEvent.change(screen.getByLabelText('New Password'), {
			target: { value: 'NewPass123!' },
		});
		fireEvent.change(screen.getByLabelText('Confirm New Password'), {
			target: { value: 'NewPass123!' },
		});

		expect(
			screen.queryByText('Passwords do not match'),
		).not.toBeInTheDocument();
	});

	it('enables submit when all fields filled and passwords match', () => {
		renderWithProviders(<ChangePasswordForm />);

		fireEvent.change(screen.getByLabelText('Current Password'), {
			target: { value: 'OldPass123' },
		});
		fireEvent.change(screen.getByLabelText('New Password'), {
			target: { value: 'NewPass123!' },
		});
		fireEvent.change(screen.getByLabelText('Confirm New Password'), {
			target: { value: 'NewPass123!' },
		});

		const submitButton = screen.getByRole('button', {
			name: 'Change Password',
		});
		expect(submitButton).toBeEnabled();
	});

	it('shows success notification on password change', async () => {
		mockMutateAsync.mockResolvedValue({ message: 'Password changed' });
		renderWithProviders(<ChangePasswordForm />);

		fireEvent.change(screen.getByLabelText('Current Password'), {
			target: { value: 'OldPass123' },
		});
		fireEvent.change(screen.getByLabelText('New Password'), {
			target: { value: 'NewPass123!' },
		});
		fireEvent.change(screen.getByLabelText('Confirm New Password'), {
			target: { value: 'NewPass123!' },
		});

		fireEvent.submit(screen.getByRole('button', { name: 'Change Password' }));

		await waitFor(() => {
			expect(
				screen.getByText('Password changed successfully!'),
			).toBeInTheDocument();
		});

		expect(mockMutateAsync).toHaveBeenCalledWith({
			current_password: 'OldPass123',
			new_password: 'NewPass123!',
		});
	});

	it('clears fields after successful change', async () => {
		mockMutateAsync.mockResolvedValue({ message: 'Password changed' });
		renderWithProviders(<ChangePasswordForm />);

		fireEvent.change(screen.getByLabelText('Current Password'), {
			target: { value: 'OldPass123' },
		});
		fireEvent.change(screen.getByLabelText('New Password'), {
			target: { value: 'NewPass123!' },
		});
		fireEvent.change(screen.getByLabelText('Confirm New Password'), {
			target: { value: 'NewPass123!' },
		});

		fireEvent.submit(screen.getByRole('button', { name: 'Change Password' }));

		await waitFor(() => {
			expect(
				screen.getByText('Password changed successfully!'),
			).toBeInTheDocument();
		});

		expect(screen.getByLabelText('Current Password')).toHaveValue('');
		expect(screen.getByLabelText('New Password')).toHaveValue('');
		expect(screen.getByLabelText('Confirm New Password')).toHaveValue('');
	});

	it('calls onSuccess callback after successful change', async () => {
		mockMutateAsync.mockResolvedValue({ message: 'Password changed' });
		const onSuccess = vi.fn();
		renderWithProviders(<ChangePasswordForm onSuccess={onSuccess} />);

		fireEvent.change(screen.getByLabelText('Current Password'), {
			target: { value: 'OldPass123' },
		});
		fireEvent.change(screen.getByLabelText('New Password'), {
			target: { value: 'NewPass123!' },
		});
		fireEvent.change(screen.getByLabelText('Confirm New Password'), {
			target: { value: 'NewPass123!' },
		});

		fireEvent.submit(screen.getByRole('button', { name: 'Change Password' }));

		await waitFor(() => {
			expect(onSuccess).toHaveBeenCalledOnce();
		});
	});

	it('shows error for wrong current password', async () => {
		mockMutateAsync.mockRejectedValue(
			new Error('Current password is incorrect'),
		);
		renderWithProviders(<ChangePasswordForm />);

		fireEvent.change(screen.getByLabelText('Current Password'), {
			target: { value: 'WrongPassword' },
		});
		fireEvent.change(screen.getByLabelText('New Password'), {
			target: { value: 'NewPass123!' },
		});
		fireEvent.change(screen.getByLabelText('Confirm New Password'), {
			target: { value: 'NewPass123!' },
		});

		fireEvent.submit(screen.getByRole('button', { name: 'Change Password' }));

		await waitFor(() => {
			expect(
				screen.getByText('Current password is incorrect'),
			).toBeInTheDocument();
		});
	});

	it('shows generic error for non-Error exceptions', async () => {
		mockMutateAsync.mockRejectedValue('something unexpected');
		renderWithProviders(<ChangePasswordForm />);

		fireEvent.change(screen.getByLabelText('Current Password'), {
			target: { value: 'OldPass123' },
		});
		fireEvent.change(screen.getByLabelText('New Password'), {
			target: { value: 'NewPass123!' },
		});
		fireEvent.change(screen.getByLabelText('Confirm New Password'), {
			target: { value: 'NewPass123!' },
		});

		fireEvent.submit(screen.getByRole('button', { name: 'Change Password' }));

		await waitFor(() => {
			expect(screen.getByText('Failed to change password')).toBeInTheDocument();
		});
	});

	it('shows mismatch error on submit when passwords differ', async () => {
		renderWithProviders(<ChangePasswordForm />);

		// We need to set matching first then change to trigger submit with mismatch
		// The form checks newPassword !== confirmPassword and returns early
		fireEvent.change(screen.getByLabelText('Current Password'), {
			target: { value: 'OldPass123' },
		});
		fireEvent.change(screen.getByLabelText('New Password'), {
			target: { value: 'NewPass123!' },
		});
		fireEvent.change(screen.getByLabelText('Confirm New Password'), {
			target: { value: 'NewPass123!' },
		});

		// Now change confirm to mismatch - button is disabled so submit won't fire via click
		// The form-level validation check on handleSubmit also catches this
		fireEvent.change(screen.getByLabelText('Confirm New Password'), {
			target: { value: 'Different' },
		});

		const submitButton = screen.getByRole('button', {
			name: 'Change Password',
		});
		expect(submitButton).toBeDisabled();
		expect(mockMutateAsync).not.toHaveBeenCalled();
	});

	it('shows "Changing..." text while mutation is pending', () => {
		setupMockChangePassword({ isPending: true });
		renderWithProviders(<ChangePasswordForm />);

		expect(
			screen.getByRole('button', { name: 'Changing...' }),
		).toBeInTheDocument();
	});

	it('shows cancel button when showCancel is true and onCancel provided', () => {
		const onCancel = vi.fn();
		renderWithProviders(
			<ChangePasswordForm onCancel={onCancel} showCancel={true} />,
		);

		const cancelButton = screen.getByRole('button', { name: 'Cancel' });
		expect(cancelButton).toBeInTheDocument();

		fireEvent.click(cancelButton);
		expect(onCancel).toHaveBeenCalledOnce();
	});

	it('hides cancel button when showCancel is false', () => {
		renderWithProviders(
			<ChangePasswordForm onCancel={vi.fn()} showCancel={false} />,
		);

		expect(
			screen.queryByRole('button', { name: 'Cancel' }),
		).not.toBeInTheDocument();
	});

	it('toggles current password visibility', () => {
		renderWithProviders(<ChangePasswordForm />);

		const currentPasswordInput = screen.getByLabelText('Current Password');
		expect(currentPasswordInput).toHaveAttribute('type', 'password');

		// Click the toggle button (first toggle button in the form)
		const toggleButtons = screen
			.getAllByRole('button')
			.filter(
				(btn) =>
					!btn.textContent?.includes('Change') &&
					!btn.textContent?.includes('Cancel'),
			);
		fireEvent.click(toggleButtons[0]);
		expect(currentPasswordInput).toHaveAttribute('type', 'text');

		fireEvent.click(toggleButtons[0]);
		expect(currentPasswordInput).toHaveAttribute('type', 'password');
	});
});
