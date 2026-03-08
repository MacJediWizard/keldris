import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { PasswordReset } from './PasswordReset';

function renderPage(initialEntries: string[] = ['/reset-password']) {
	return render(
		<MemoryRouter initialEntries={initialEntries}>
			<PasswordReset />
		</MemoryRouter>,
	);
}

function mockFetchResponses(
	...responses: Array<{ ok: boolean; status?: number; json?: unknown }>
) {
	const mocked = vi.fn();
	for (const resp of responses) {
		mocked.mockResolvedValueOnce({
			ok: resp.ok,
			status: resp.status ?? (resp.ok ? 200 : 500),
			json: () => Promise.resolve(resp.json ?? {}),
		});
	}
	global.fetch = mocked;
	return mocked;
}

describe('PasswordReset - Request Form', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('renders reset request form with email input', () => {
		renderPage();

		expect(screen.getByText('Reset Your Password')).toBeInTheDocument();
		expect(screen.getByLabelText('Email address')).toBeInTheDocument();
		expect(
			screen.getByRole('button', { name: 'Send Reset Link' }),
		).toBeInTheDocument();
	});

	it('renders description text', () => {
		renderPage();

		expect(
			screen.getByText(
				/Enter your email address and we'll send you a link to reset your/,
			),
		).toBeInTheDocument();
	});

	it('shows "Return to Login" link', () => {
		renderPage();

		const link = screen.getByText('Return to Login');
		expect(link.closest('a')).toHaveAttribute('href', '/');
	});

	it('shows success message after successful submission', async () => {
		const user = userEvent.setup();
		mockFetchResponses({ ok: true, json: { message: 'sent' } });

		renderPage();

		await user.type(screen.getByLabelText('Email address'), 'user@test.com');
		await user.click(screen.getByRole('button', { name: 'Send Reset Link' }));

		await waitFor(() => {
			expect(screen.getByText('Check Your Email')).toBeInTheDocument();
		});

		expect(
			screen.getByText(
				/If an account exists with that email, you will receive a password reset link shortly/,
			),
		).toBeInTheDocument();
	});

	it('shows "Sending..." text while loading', async () => {
		const user = userEvent.setup();
		// Never resolve so loading state persists
		global.fetch = vi.fn(() => new Promise(() => {}));

		renderPage();

		await user.type(screen.getByLabelText('Email address'), 'user@test.com');
		await user.click(screen.getByRole('button', { name: 'Send Reset Link' }));

		await waitFor(() => {
			const button = screen.getByRole('button', { name: 'Sending...' });
			expect(button).toBeDisabled();
		});
	});

	it('shows error on rate limit (429)', async () => {
		const user = userEvent.setup();
		mockFetchResponses({
			ok: false,
			status: 429,
			json: { error: 'Rate limited' },
		});

		renderPage();

		await user.type(screen.getByLabelText('Email address'), 'user@test.com');
		await user.click(screen.getByRole('button', { name: 'Send Reset Link' }));

		await waitFor(() => {
			expect(
				screen.getByText('Too many requests. Please try again later.'),
			).toBeInTheDocument();
		});
	});

	it('shows error on server failure', async () => {
		const user = userEvent.setup();
		mockFetchResponses({
			ok: false,
			status: 500,
			json: { error: 'Server error' },
		});

		renderPage();

		await user.type(screen.getByLabelText('Email address'), 'user@test.com');
		await user.click(screen.getByRole('button', { name: 'Send Reset Link' }));

		await waitFor(() => {
			expect(screen.getByText('Server error')).toBeInTheDocument();
		});
	});

	it('shows error on network failure', async () => {
		const user = userEvent.setup();
		global.fetch = vi.fn().mockRejectedValueOnce(new Error('Network error'));

		renderPage();

		await user.type(screen.getByLabelText('Email address'), 'user@test.com');
		await user.click(screen.getByRole('button', { name: 'Send Reset Link' }));

		await waitFor(() => {
			expect(
				screen.getByText('Failed to send request. Please try again.'),
			).toBeInTheDocument();
		});
	});

	it('shows "Return to Login" link in success state', async () => {
		const user = userEvent.setup();
		mockFetchResponses({ ok: true, json: {} });

		renderPage();

		await user.type(screen.getByLabelText('Email address'), 'user@test.com');
		await user.click(screen.getByRole('button', { name: 'Send Reset Link' }));

		await waitFor(() => {
			expect(screen.getByText('Check Your Email')).toBeInTheDocument();
		});

		const returnLink = screen.getByText('Return to Login');
		expect(returnLink.closest('a')).toHaveAttribute('href', '/');
	});
});

describe('PasswordReset - Token Reset Form', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('shows loading state while validating token', () => {
		// Never resolve so validating stays true
		global.fetch = vi.fn(() => new Promise(() => {}));

		renderPage(['/reset-password?token=valid-token-123']);

		expect(screen.getByText('Validating reset link...')).toBeInTheDocument();
	});

	it('renders new password form when token is valid', async () => {
		mockFetchResponses({
			ok: true,
			json: { valid: true, email: 'user@test.com' },
		});

		renderPage(['/reset-password?token=valid-token-123']);

		await waitFor(() => {
			expect(screen.getByText('Reset Your Password')).toBeInTheDocument();
		});

		expect(screen.getByText('for user@test.com')).toBeInTheDocument();
		expect(screen.getByLabelText('New Password')).toBeInTheDocument();
		expect(screen.getByLabelText('Confirm Password')).toBeInTheDocument();
		expect(
			screen.getByRole('button', { name: 'Reset Password' }),
		).toBeInTheDocument();
	});

	it('shows error message when token is invalid/expired', async () => {
		mockFetchResponses({
			ok: false,
			status: 400,
			json: { error: 'Token has expired' },
		});

		renderPage(['/reset-password?token=expired-token']);

		await waitFor(() => {
			expect(screen.getByText('Invalid Reset Link')).toBeInTheDocument();
		});

		expect(screen.getByText('Token has expired')).toBeInTheDocument();
	});

	it('shows "Request New Reset Link" button for invalid token', async () => {
		mockFetchResponses({
			ok: false,
			status: 400,
			json: { error: 'Invalid token' },
		});

		renderPage(['/reset-password?token=bad-token']);

		await waitFor(() => {
			expect(screen.getByText('Invalid Reset Link')).toBeInTheDocument();
		});

		const link = screen.getByText('Request New Reset Link');
		expect(link.closest('a')).toHaveAttribute('href', '/reset-password');
	});

	it('shows error when token validation network request fails', async () => {
		global.fetch = vi.fn().mockRejectedValueOnce(new Error('Network error'));

		renderPage(['/reset-password?token=some-token']);

		await waitFor(() => {
			expect(screen.getByText('Invalid Reset Link')).toBeInTheDocument();
		});

		expect(
			screen.getByText('Failed to validate reset link'),
		).toBeInTheDocument();
	});

	it('shows error for mismatched passwords', async () => {
		const user = userEvent.setup();
		mockFetchResponses({
			ok: true,
			json: { valid: true, email: 'user@test.com' },
		});

		renderPage(['/reset-password?token=valid-token']);

		await waitFor(() => {
			expect(screen.getByLabelText('New Password')).toBeInTheDocument();
		});

		await user.type(screen.getByLabelText('New Password'), 'password123');
		await user.type(
			screen.getByLabelText('Confirm Password'),
			'differentpassword',
		);
		await user.click(screen.getByRole('button', { name: 'Reset Password' }));

		await waitFor(() => {
			expect(screen.getByText('Passwords do not match')).toBeInTheDocument();
		});
	});

	it('shows error for password shorter than 8 characters', async () => {
		const user = userEvent.setup();
		mockFetchResponses({
			ok: true,
			json: { valid: true, email: 'user@test.com' },
		});

		renderPage(['/reset-password?token=valid-token']);

		await waitFor(() => {
			expect(screen.getByLabelText('New Password')).toBeInTheDocument();
		});

		await user.type(screen.getByLabelText('New Password'), 'short');
		await user.type(screen.getByLabelText('Confirm Password'), 'short');
		await user.click(screen.getByRole('button', { name: 'Reset Password' }));

		await waitFor(() => {
			expect(
				screen.getByText('Password must be at least 8 characters'),
			).toBeInTheDocument();
		});
	});

	it('submits new password successfully and shows completion message', async () => {
		const user = userEvent.setup();
		const fetchMock = mockFetchResponses(
			// Token validation
			{ ok: true, json: { valid: true, email: 'user@test.com' } },
			// Reset submission
			{ ok: true, json: { message: 'Password reset' } },
		);

		renderPage(['/reset-password?token=valid-token']);

		await waitFor(() => {
			expect(screen.getByLabelText('New Password')).toBeInTheDocument();
		});

		await user.type(screen.getByLabelText('New Password'), 'newpassword123');
		await user.type(
			screen.getByLabelText('Confirm Password'),
			'newpassword123',
		);
		await user.click(screen.getByRole('button', { name: 'Reset Password' }));

		await waitFor(() => {
			expect(screen.getByText('Password Reset Complete')).toBeInTheDocument();
		});

		expect(
			screen.getByText(/Your password has been reset successfully/),
		).toBeInTheDocument();

		// Verify the POST body
		expect(fetchMock).toHaveBeenCalledWith('/auth/reset-password/reset', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				token: 'valid-token',
				new_password: 'newpassword123',
			}),
		});
	});

	it('shows "Go to Login" link after successful reset', async () => {
		const user = userEvent.setup();
		mockFetchResponses(
			{ ok: true, json: { valid: true, email: 'user@test.com' } },
			{ ok: true, json: {} },
		);

		renderPage(['/reset-password?token=valid-token']);

		await waitFor(() => {
			expect(screen.getByLabelText('New Password')).toBeInTheDocument();
		});

		await user.type(screen.getByLabelText('New Password'), 'newpassword123');
		await user.type(
			screen.getByLabelText('Confirm Password'),
			'newpassword123',
		);
		await user.click(screen.getByRole('button', { name: 'Reset Password' }));

		await waitFor(() => {
			expect(screen.getByText('Password Reset Complete')).toBeInTheDocument();
		});

		const loginLink = screen.getByText('Go to Login');
		expect(loginLink.closest('a')).toHaveAttribute('href', '/');
	});

	it('shows error when reset API call fails', async () => {
		const user = userEvent.setup();
		mockFetchResponses(
			{ ok: true, json: { valid: true, email: 'user@test.com' } },
			{
				ok: false,
				status: 400,
				json: { error: 'Token expired during reset' },
			},
		);

		renderPage(['/reset-password?token=valid-token']);

		await waitFor(() => {
			expect(screen.getByLabelText('New Password')).toBeInTheDocument();
		});

		await user.type(screen.getByLabelText('New Password'), 'newpassword123');
		await user.type(
			screen.getByLabelText('Confirm Password'),
			'newpassword123',
		);
		await user.click(screen.getByRole('button', { name: 'Reset Password' }));

		await waitFor(() => {
			expect(
				screen.getByText('Token expired during reset'),
			).toBeInTheDocument();
		});
	});

	it('shows error when reset network request fails', async () => {
		const user = userEvent.setup();
		const fetchMock = vi.fn();
		// Token validation succeeds
		fetchMock.mockResolvedValueOnce({
			ok: true,
			status: 200,
			json: () => Promise.resolve({ valid: true, email: 'user@test.com' }),
		});
		// Reset request fails
		fetchMock.mockRejectedValueOnce(new Error('Network error'));
		global.fetch = fetchMock;

		renderPage(['/reset-password?token=valid-token']);

		await waitFor(() => {
			expect(screen.getByLabelText('New Password')).toBeInTheDocument();
		});

		await user.type(screen.getByLabelText('New Password'), 'newpassword123');
		await user.type(
			screen.getByLabelText('Confirm Password'),
			'newpassword123',
		);
		await user.click(screen.getByRole('button', { name: 'Reset Password' }));

		await waitFor(() => {
			expect(
				screen.getByText('Failed to reset password. Please try again.'),
			).toBeInTheDocument();
		});
	});

	it('disables submit button while resetting', async () => {
		const user = userEvent.setup();
		const fetchMock = vi.fn();
		// Token validation succeeds
		fetchMock.mockResolvedValueOnce({
			ok: true,
			status: 200,
			json: () => Promise.resolve({ valid: true, email: 'user@test.com' }),
		});
		// Reset never resolves
		fetchMock.mockReturnValueOnce(new Promise(() => {}));
		global.fetch = fetchMock;

		renderPage(['/reset-password?token=valid-token']);

		await waitFor(() => {
			expect(screen.getByLabelText('New Password')).toBeInTheDocument();
		});

		await user.type(screen.getByLabelText('New Password'), 'newpassword123');
		await user.type(
			screen.getByLabelText('Confirm Password'),
			'newpassword123',
		);
		await user.click(screen.getByRole('button', { name: 'Reset Password' }));

		await waitFor(() => {
			const button = screen.getByRole('button', { name: 'Resetting...' });
			expect(button).toBeDisabled();
		});
	});

	it('validates token on mount by calling correct endpoint', async () => {
		const fetchMock = mockFetchResponses({
			ok: true,
			json: { valid: true, email: 'user@test.com' },
		});

		renderPage(['/reset-password?token=my-reset-token']);

		await waitFor(() => {
			expect(fetchMock).toHaveBeenCalledWith(
				'/auth/reset-password/validate/my-reset-token',
			);
		});
	});
});
