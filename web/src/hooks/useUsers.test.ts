import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useDeleteUser,
	useDisableUser,
	useEnableUser,
	useEndImpersonation,
	useImpersonationLogs,
	useInviteUser,
	useOrgActivityLogs,
	useResetUserPassword,
	useStartImpersonation,
	useUpdateUser,
	useUser,
	useUserActivity,
	useUsers,
} from './useUsers';

vi.mock('../lib/api', () => ({
	usersApi: {
		list: vi.fn(),
		get: vi.fn(),
		invite: vi.fn(),
		update: vi.fn(),
		delete: vi.fn(),
		resetPassword: vi.fn(),
		disable: vi.fn(),
		enable: vi.fn(),
		getActivity: vi.fn(),
		getOrgActivityLogs: vi.fn(),
		startImpersonation: vi.fn(),
		endImpersonation: vi.fn(),
		getImpersonationLogs: vi.fn(),
	},
}));

import { usersApi } from '../lib/api';

const originalLocation = window.location;

beforeEach(() => {
	Object.defineProperty(window, 'location', {
		configurable: true,
		value: { reload: vi.fn() },
	});
});

afterEach(() => {
	Object.defineProperty(window, 'location', {
		configurable: true,
		value: originalLocation,
	});
});

describe('useUsers', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('lists users', async () => {
		vi.mocked(usersApi.list).mockResolvedValue([]);

		const { result } = renderHook(() => useUsers(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(usersApi.list).toHaveBeenCalledOnce();
	});
});

describe('useUser', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches a user', async () => {
		vi.mocked(usersApi.get).mockResolvedValue({ id: 'u-1' });

		const { result } = renderHook(() => useUser('u-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(usersApi.get).toHaveBeenCalledWith('u-1');
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useUser(''), { wrapper: createWrapper() });
		expect(usersApi.get).not.toHaveBeenCalled();
	});
});

describe('useInviteUser', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('invites a user', async () => {
		vi.mocked(usersApi.invite).mockResolvedValue({ id: 'u-2' });

		const { result } = renderHook(() => useInviteUser(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ email: 'a@b.com' } as never);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(usersApi.invite).toHaveBeenCalled();
	});
});

describe('useUpdateUser', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('updates a user', async () => {
		vi.mocked(usersApi.update).mockResolvedValue({ id: 'u-1' });

		const { result } = renderHook(() => useUpdateUser(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 'u-1', data: { name: 'x' } as never });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(usersApi.update).toHaveBeenCalledWith('u-1', { name: 'x' });
	});
});

describe('useDeleteUser', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('deletes a user', async () => {
		vi.mocked(usersApi.delete).mockResolvedValue({ message: 'Deleted' });

		const { result } = renderHook(() => useDeleteUser(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('u-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(usersApi.delete).toHaveBeenCalledWith('u-1');
	});
});

describe('useResetUserPassword', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('resets a password', async () => {
		vi.mocked(usersApi.resetPassword).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useResetUserPassword(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 'u-1', data: { password: 'x' } as never });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(usersApi.resetPassword).toHaveBeenCalledWith('u-1', {
			password: 'x',
		});
	});
});

describe('useDisableUser', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('disables a user', async () => {
		vi.mocked(usersApi.disable).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useDisableUser(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('u-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(usersApi.disable).toHaveBeenCalledWith('u-1');
	});
});

describe('useEnableUser', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('enables a user', async () => {
		vi.mocked(usersApi.enable).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useEnableUser(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('u-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(usersApi.enable).toHaveBeenCalledWith('u-1');
	});
});

describe('useUserActivity', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches user activity', async () => {
		vi.mocked(usersApi.getActivity).mockResolvedValue([]);

		const { result } = renderHook(() => useUserActivity('u-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(usersApi.getActivity).toHaveBeenCalledWith('u-1', 50, 0);
	});
});

describe('useOrgActivityLogs', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches org activity logs', async () => {
		vi.mocked(usersApi.getOrgActivityLogs).mockResolvedValue([]);

		const { result } = renderHook(() => useOrgActivityLogs(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(usersApi.getOrgActivityLogs).toHaveBeenCalledWith(50, 0);
	});
});

describe('useStartImpersonation', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('starts impersonation', async () => {
		vi.mocked(usersApi.startImpersonation).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useStartImpersonation(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 'u-1', data: { reason: 'test' } as never });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(usersApi.startImpersonation).toHaveBeenCalledWith('u-1', {
			reason: 'test',
		});
	});
});

describe('useEndImpersonation', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('ends impersonation', async () => {
		vi.mocked(usersApi.endImpersonation).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useEndImpersonation(), {
			wrapper: createWrapper(),
		});

		result.current.mutate();

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(usersApi.endImpersonation).toHaveBeenCalledOnce();
	});
});

describe('useImpersonationLogs', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches impersonation logs', async () => {
		vi.mocked(usersApi.getImpersonationLogs).mockResolvedValue([]);

		const { result } = renderHook(() => useImpersonationLogs(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(usersApi.getImpersonationLogs).toHaveBeenCalledWith(50, 0);
	});
});
