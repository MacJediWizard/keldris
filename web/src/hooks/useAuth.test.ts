import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useAuthStatus,
	useLogout,
	useMe,
	usePasswordLogin,
	useUpdatePreferences,
} from './useAuth';

vi.mock('../lib/api', () => ({
	authApi: {
		me: vi.fn(),
		logout: vi.fn(),
		updatePreferences: vi.fn(),
		getLoginUrl: vi.fn().mockReturnValue('/auth/login'),
	},
}));

import { authApi } from '../lib/api';

const mockUser = {
	id: 'user-1',
	email: 'test@example.com',
	name: 'Test User',
	current_org_id: 'org-1',
	current_org_role: 'admin',
};

const mockSuperUser = {
	id: 'super-1',
	email: 'admin@example.com',
	name: 'Super Admin',
	current_org_id: 'org-1',
	current_org_role: 'owner',
	is_superuser: true,
};

describe('useMe', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches current user', async () => {
		vi.mocked(authApi.me).mockResolvedValue(mockUser);

		const { result } = renderHook(() => useMe(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockUser);
	});

	it('returns user fields including org and role', async () => {
		vi.mocked(authApi.me).mockResolvedValue(mockUser);

		const { result } = renderHook(() => useMe(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data?.email).toBe('test@example.com');
		expect(result.current.data?.name).toBe('Test User');
		expect(result.current.data?.current_org_id).toBe('org-1');
		expect(result.current.data?.current_org_role).toBe('admin');
	});

	it('returns superuser flag when present', async () => {
		vi.mocked(authApi.me).mockResolvedValue(mockSuperUser);

		const { result } = renderHook(() => useMe(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data?.is_superuser).toBe(true);
	});

	it('handles auth failure with retry disabled', async () => {
		vi.mocked(authApi.me).mockRejectedValue(new Error('Unauthorized'));

		const { result } = renderHook(() => useMe(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isError).toBe(true));
		// retry: false means it should only call once
		expect(authApi.me).toHaveBeenCalledTimes(1);
	});

	it('returns undefined data while loading', () => {
		// Make the promise never resolve
		vi.mocked(authApi.me).mockReturnValue(new Promise(() => {}));

		const { result } = renderHook(() => useMe(), {
			wrapper: createWrapper(),
		});

		expect(result.current.data).toBeUndefined();
		expect(result.current.isLoading).toBe(true);
	});

	it('returns impersonation fields when impersonating', async () => {
		const impersonatingUser = {
			...mockUser,
			is_impersonating: true,
			impersonating_user_id: 'target-user-1',
			impersonating_id: 'impersonation-session-1',
		};
		vi.mocked(authApi.me).mockResolvedValue(impersonatingUser);

		const { result } = renderHook(() => useMe(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data?.is_impersonating).toBe(true);
		expect(result.current.data?.impersonating_user_id).toBe('target-user-1');
	});

	it('returns SSO group info when available', async () => {
		const ssoUser = {
			...mockUser,
			sso_groups: ['engineering', 'admins'],
			sso_groups_synced_at: '2026-03-01T00:00:00Z',
		};
		vi.mocked(authApi.me).mockResolvedValue(ssoUser);

		const { result } = renderHook(() => useMe(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data?.sso_groups).toEqual(['engineering', 'admins']);
		expect(result.current.data?.sso_groups_synced_at).toBe(
			'2026-03-01T00:00:00Z',
		);
	});
});

describe('useLogout', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('calls logout API and redirects to login URL', async () => {
		vi.mocked(authApi.logout).mockResolvedValue({ message: 'Logged out' });

		const { result } = renderHook(() => useLogout(), {
			wrapper: createWrapper(),
		});

		result.current.mutate();

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(authApi.logout).toHaveBeenCalledOnce();
		expect(window.location.href).toBe('/auth/login');
	});

	it('calls getLoginUrl for the redirect destination', async () => {
		vi.mocked(authApi.logout).mockResolvedValue({ message: 'Logged out' });

		const { result } = renderHook(() => useLogout(), {
			wrapper: createWrapper(),
		});

		result.current.mutate();

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(authApi.getLoginUrl).toHaveBeenCalled();
	});

	it('handles logout failure', async () => {
		vi.mocked(authApi.logout).mockRejectedValue(new Error('Network error'));

		const { result } = renderHook(() => useLogout(), {
			wrapper: createWrapper(),
		});

		result.current.mutate();

		await waitFor(() => expect(result.current.isError).toBe(true));
		expect(result.current.error?.message).toBe('Network error');
	});
});

describe('useUpdatePreferences', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('updates user preferences and returns updated user', async () => {
		const updatedUser = { ...mockUser, language: 'de' };
		vi.mocked(authApi.updatePreferences).mockResolvedValue(updatedUser);

		const { result } = renderHook(() => useUpdatePreferences(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ language: 'de' as never });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(authApi.updatePreferences).toHaveBeenCalledWith({
			language: 'de',
		});
		expect(result.current.data).toEqual(updatedUser);
	});

	it('handles update failure', async () => {
		vi.mocked(authApi.updatePreferences).mockRejectedValue(
			new Error('Forbidden'),
		);

		const { result } = renderHook(() => useUpdatePreferences(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ language: 'fr' as never });

		await waitFor(() => expect(result.current.isError).toBe(true));
		expect(result.current.error?.message).toBe('Forbidden');
	});

	it('sets query data for auth/me on success', async () => {
		const updatedUser = { ...mockUser, language: 'es' };
		vi.mocked(authApi.updatePreferences).mockResolvedValue(updatedUser);

		const { result } = renderHook(
			() => ({
				preferences: useUpdatePreferences(),
				me: useMe(),
			}),
			{ wrapper: createWrapper() },
		);

		// Pre-seed the me query
		vi.mocked(authApi.me).mockResolvedValue(mockUser);
		await waitFor(() => expect(result.current.me.isSuccess).toBe(true));

		result.current.preferences.mutate({ language: 'es' as never });

		await waitFor(() =>
			expect(result.current.preferences.isSuccess).toBe(true),
		);

		// The onSuccess handler calls setQueryData(['auth', 'me'], user)
		// which updates the me query data directly without a refetch
		await waitFor(() => {
			expect(result.current.me.data).toEqual(updatedUser);
		});
	});
});

describe('useAuthStatus', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		vi.restoreAllMocks();
	});

	it('fetches auth status with oidc and password flags', async () => {
		const mockStatus = {
			oidc_enabled: true,
			password_enabled: true,
		};
		const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue({
			ok: true,
			json: () => Promise.resolve(mockStatus),
		} as Response);

		const { result } = renderHook(() => useAuthStatus(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockStatus);
		expect(fetchSpy).toHaveBeenCalledWith(
			'/auth/status',
			expect.objectContaining({
				credentials: 'include',
				headers: { 'Content-Type': 'application/json' },
			}),
		);
	});

	it('returns oidc_enabled=false when OIDC is not configured', async () => {
		const mockStatus = {
			oidc_enabled: false,
			password_enabled: true,
		};
		vi.spyOn(globalThis, 'fetch').mockResolvedValue({
			ok: true,
			json: () => Promise.resolve(mockStatus),
		} as Response);

		const { result } = renderHook(() => useAuthStatus(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data?.oidc_enabled).toBe(false);
		expect(result.current.data?.password_enabled).toBe(true);
	});

	it('handles auth status fetch failure', async () => {
		vi.spyOn(globalThis, 'fetch').mockResolvedValue({
			ok: false,
			status: 500,
			json: () => Promise.resolve({ error: 'Internal error' }),
		} as Response);

		const { result } = renderHook(() => useAuthStatus(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isError).toBe(true));
		expect(result.current.error?.message).toBe('Failed to fetch auth status');
	});
});

describe('usePasswordLogin', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		vi.restoreAllMocks();
	});

	it('sends email and password to login endpoint', async () => {
		const mockResponse = { token: 'session-token' };
		const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue({
			ok: true,
			json: () => Promise.resolve(mockResponse),
		} as Response);

		const { result } = renderHook(() => usePasswordLogin(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			email: 'user@example.com',
			password: 'secret123',
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchSpy).toHaveBeenCalledWith(
			'/auth/login/password',
			expect.objectContaining({
				method: 'POST',
				credentials: 'include',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					email: 'user@example.com',
					password: 'secret123',
				}),
			}),
		);
	});

	it('returns error message from server on failure', async () => {
		vi.spyOn(globalThis, 'fetch').mockResolvedValue({
			ok: false,
			status: 401,
			json: () => Promise.resolve({ error: 'Invalid credentials' }),
		} as Response);

		const { result } = renderHook(() => usePasswordLogin(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			email: 'user@example.com',
			password: 'wrong',
		});

		await waitFor(() => expect(result.current.isError).toBe(true));
		expect(result.current.error?.message).toBe('Invalid credentials');
	});

	it('returns fallback message when server returns non-JSON error', async () => {
		vi.spyOn(globalThis, 'fetch').mockResolvedValue({
			ok: false,
			status: 500,
			json: () => Promise.reject(new Error('not json')),
		} as Response);

		const { result } = renderHook(() => usePasswordLogin(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			email: 'user@example.com',
			password: 'test',
		});

		await waitFor(() => expect(result.current.isError).toBe(true));
		expect(result.current.error?.message).toBe('Login failed');
	});

	it('uses message field when error field is absent', async () => {
		vi.spyOn(globalThis, 'fetch').mockResolvedValue({
			ok: false,
			status: 403,
			json: () => Promise.resolve({ message: 'Account locked' }),
		} as Response);

		const { result } = renderHook(() => usePasswordLogin(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			email: 'user@example.com',
			password: 'test',
		});

		await waitFor(() => expect(result.current.isError).toBe(true));
		expect(result.current.error?.message).toBe('Account locked');
	});

	it('handles network failure', async () => {
		vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('Network error'));

		const { result } = renderHook(() => usePasswordLogin(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			email: 'user@example.com',
			password: 'test',
		});

		await waitFor(() => expect(result.current.isError).toBe(true));
		expect(result.current.error?.message).toBe('Network error');
	});
});
