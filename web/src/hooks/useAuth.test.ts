import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import { useLogout, useMe, useUpdatePreferences } from './useAuth';

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
	org_id: 'org-1',
	role: 'admin' as const,
	created_at: '2024-01-01T00:00:00Z',
	updated_at: '2024-01-01T00:00:00Z',
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

	it('handles auth failure', async () => {
		vi.mocked(authApi.me).mockRejectedValue(new Error('Unauthorized'));

		const { result } = renderHook(() => useMe(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isError).toBe(true));
	});
});

describe('useLogout', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('calls logout and redirects', async () => {
		vi.mocked(authApi.logout).mockResolvedValue({ message: 'Logged out' });

		const { result } = renderHook(() => useLogout(), {
			wrapper: createWrapper(),
		});

		result.current.mutate();

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(authApi.logout).toHaveBeenCalled();
	});
});

describe('useUpdatePreferences', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('updates user preferences', async () => {
		const updatedUser = { ...mockUser, name: 'Updated User' };
		vi.mocked(authApi.updatePreferences).mockResolvedValue(updatedUser);

		const { result } = renderHook(() => useUpdatePreferences(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ name: 'Updated User' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(authApi.updatePreferences).toHaveBeenCalledWith({
			name: 'Updated User',
		});
	});
});
