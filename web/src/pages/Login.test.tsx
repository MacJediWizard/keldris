import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const mockNavigate = vi.fn();

vi.mock('react-router-dom', async () => {
	const actual =
		await vi.importActual<typeof import('react-router-dom')>(
			'react-router-dom',
		);
	return {
		...actual,
		useNavigate: () => mockNavigate,
	};
});

import LoginPage from './Login';

function renderPage() {
	return render(
		<MemoryRouter>
			<LoginPage />
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

describe('LoginPage', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('renders login form with email and password inputs', async () => {
		mockFetchResponses({
			ok: true,
			json: { oidc_enabled: false, password_enabled: true },
		});

		renderPage();

		await waitFor(() => {
			expect(screen.getByLabelText('Email address')).toBeInTheDocument();
		});
		expect(screen.getByLabelText('Password')).toBeInTheDocument();
		expect(screen.getByRole('button', { name: 'Sign in' })).toBeInTheDocument();
	});

	it('shows loading spinner while fetching auth status', () => {
		// Never resolve the fetch so auth status stays loading
		global.fetch = vi.fn(() => new Promise(() => {}));

		renderPage();

		const spinner = document.querySelector('.animate-spin');
		expect(spinner).not.toBeNull();
	});

	it('shows "Forgot your password?" link that points to /reset-password', async () => {
		mockFetchResponses({
			ok: true,
			json: { oidc_enabled: false, password_enabled: true },
		});

		renderPage();

		await waitFor(() => {
			expect(screen.getByText('Forgot your password?')).toBeInTheDocument();
		});

		const link = screen.getByText('Forgot your password?');
		expect(link.closest('a')).toHaveAttribute('href', '/reset-password');
	});

	it('submits credentials and navigates to / on success', async () => {
		const user = userEvent.setup();
		const fetchMock = mockFetchResponses(
			// Auth status response
			{ ok: true, json: { oidc_enabled: false, password_enabled: true } },
			// Login response
			{ ok: true, json: {} },
		);

		renderPage();

		await waitFor(() => {
			expect(screen.getByLabelText('Email address')).toBeInTheDocument();
		});

		await user.type(screen.getByLabelText('Email address'), 'user@test.com');
		await user.type(screen.getByLabelText('Password'), 'password123');
		await user.click(screen.getByRole('button', { name: 'Sign in' }));

		await waitFor(() => {
			expect(fetchMock).toHaveBeenCalledWith('/auth/login/password', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				credentials: 'include',
				body: JSON.stringify({
					email: 'user@test.com',
					password: 'password123',
				}),
			});
		});

		await waitFor(() => {
			expect(mockNavigate).toHaveBeenCalledWith('/');
		});
	});

	it('shows error message on invalid credentials (401)', async () => {
		const user = userEvent.setup();
		mockFetchResponses(
			// Auth status
			{ ok: true, json: { oidc_enabled: false, password_enabled: true } },
			// Login fails with 401
			{
				ok: false,
				status: 401,
				json: { error: 'Invalid credentials' },
			},
		);

		renderPage();

		await waitFor(() => {
			expect(screen.getByLabelText('Email address')).toBeInTheDocument();
		});

		await user.type(screen.getByLabelText('Email address'), 'bad@test.com');
		await user.type(screen.getByLabelText('Password'), 'wrong');
		await user.click(screen.getByRole('button', { name: 'Sign in' }));

		await waitFor(() => {
			expect(
				screen.getByText('Invalid email or password.'),
			).toBeInTheDocument();
		});
	});

	it('shows error message on unverified email (403)', async () => {
		const user = userEvent.setup();
		mockFetchResponses(
			{ ok: true, json: { oidc_enabled: false, password_enabled: true } },
			{
				ok: false,
				status: 403,
				json: { error: 'Email not verified' },
			},
		);

		renderPage();

		await waitFor(() => {
			expect(screen.getByLabelText('Email address')).toBeInTheDocument();
		});

		await user.type(screen.getByLabelText('Email address'), 'user@test.com');
		await user.type(screen.getByLabelText('Password'), 'pass1234');
		await user.click(screen.getByRole('button', { name: 'Sign in' }));

		await waitFor(() => {
			expect(screen.getByText('Email not verified')).toBeInTheDocument();
		});
	});

	it('shows network error when fetch throws', async () => {
		const user = userEvent.setup();
		const fetchMock = vi.fn();
		// Auth status succeeds
		fetchMock.mockResolvedValueOnce({
			ok: true,
			status: 200,
			json: () =>
				Promise.resolve({ oidc_enabled: false, password_enabled: true }),
		});
		// Login throws network error
		fetchMock.mockRejectedValueOnce(new Error('Network error'));
		global.fetch = fetchMock;

		renderPage();

		await waitFor(() => {
			expect(screen.getByLabelText('Email address')).toBeInTheDocument();
		});

		await user.type(screen.getByLabelText('Email address'), 'user@test.com');
		await user.type(screen.getByLabelText('Password'), 'password123');
		await user.click(screen.getByRole('button', { name: 'Sign in' }));

		await waitFor(() => {
			expect(
				screen.getByText(
					'Unable to connect to the server. Please try again later.',
				),
			).toBeInTheDocument();
		});
	});

	it('shows SSO login button when oidc_enabled is true', async () => {
		mockFetchResponses({
			ok: true,
			json: { oidc_enabled: true, password_enabled: true },
		});

		renderPage();

		await waitFor(() => {
			expect(
				screen.getByRole('button', { name: /Sign in with SSO/ }),
			).toBeInTheDocument();
		});
		expect(screen.getByText('Or continue with')).toBeInTheDocument();
	});

	it('does not show SSO button when oidc_enabled is false', async () => {
		mockFetchResponses({
			ok: true,
			json: { oidc_enabled: false, password_enabled: true },
		});

		renderPage();

		await waitFor(() => {
			expect(screen.getByLabelText('Email address')).toBeInTheDocument();
		});

		expect(screen.queryByText('Sign in with SSO')).not.toBeInTheDocument();
	});

	it('disables submit button during loading and shows "Signing in..."', async () => {
		const user = userEvent.setup();

		// Auth status resolves immediately
		const fetchMock = vi.fn();
		fetchMock.mockResolvedValueOnce({
			ok: true,
			status: 200,
			json: () =>
				Promise.resolve({ oidc_enabled: false, password_enabled: true }),
		});
		// Login never resolves so loading stays true
		fetchMock.mockReturnValueOnce(new Promise(() => {}));
		global.fetch = fetchMock;

		renderPage();

		await waitFor(() => {
			expect(screen.getByLabelText('Email address')).toBeInTheDocument();
		});

		await user.type(screen.getByLabelText('Email address'), 'user@test.com');
		await user.type(screen.getByLabelText('Password'), 'password123');
		await user.click(screen.getByRole('button', { name: 'Sign in' }));

		await waitFor(() => {
			const button = screen.getByRole('button', { name: 'Signing in...' });
			expect(button).toBeDisabled();
		});
	});

	it('shows SSO-only login when password_enabled is false', async () => {
		mockFetchResponses({
			ok: true,
			json: { oidc_enabled: true, password_enabled: false },
		});

		renderPage();

		await waitFor(() => {
			expect(
				screen.getByRole('button', { name: /Sign in with SSO/ }),
			).toBeInTheDocument();
		});

		expect(screen.queryByLabelText('Email address')).not.toBeInTheDocument();
		expect(screen.queryByLabelText('Password')).not.toBeInTheDocument();
	});

	it('shows fallback message when no login methods are configured', async () => {
		mockFetchResponses({
			ok: true,
			json: { oidc_enabled: false, password_enabled: false },
		});

		renderPage();

		await waitFor(() => {
			expect(
				screen.getByText(
					'No login methods are currently configured. Please contact your administrator.',
				),
			).toBeInTheDocument();
		});
	});

	it('defaults to password login when auth status fetch fails', async () => {
		global.fetch = vi.fn().mockRejectedValueOnce(new Error('Network error'));

		renderPage();

		await waitFor(() => {
			expect(screen.getByLabelText('Email address')).toBeInTheDocument();
		});
		expect(screen.getByLabelText('Password')).toBeInTheDocument();
	});

	it('shows generic error for unexpected server error (500)', async () => {
		const user = userEvent.setup();
		mockFetchResponses(
			{ ok: true, json: { oidc_enabled: false, password_enabled: true } },
			{
				ok: false,
				status: 500,
				json: { error: 'Internal server error' },
			},
		);

		renderPage();

		await waitFor(() => {
			expect(screen.getByLabelText('Email address')).toBeInTheDocument();
		});

		await user.type(screen.getByLabelText('Email address'), 'user@test.com');
		await user.type(screen.getByLabelText('Password'), 'password123');
		await user.click(screen.getByRole('button', { name: 'Sign in' }));

		await waitFor(() => {
			expect(screen.getByText('Internal server error')).toBeInTheDocument();
		});
	});

	it('renders Keldris branding', async () => {
		mockFetchResponses({
			ok: true,
			json: { oidc_enabled: false, password_enabled: true },
		});

		renderPage();

		await waitFor(() => {
			expect(screen.getByText('Keldris')).toBeInTheDocument();
		});
		expect(screen.getByText('Sign in to your account')).toBeInTheDocument();
		expect(screen.getByAltText('Keldris')).toBeInTheDocument();
	});
});
