import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCreateNotificationChannel,
	useCreateNotificationPreference,
	useDeleteNotificationChannel,
	useDeleteNotificationPreference,
	useNotificationChannel,
	useNotificationChannels,
	useNotificationLogs,
	useNotificationPreferences,
} from './useNotifications';

vi.mock('../lib/api', () => ({
	notificationsApi: {
		listChannels: vi.fn(),
		getChannel: vi.fn(),
		createChannel: vi.fn(),
		updateChannel: vi.fn(),
		deleteChannel: vi.fn(),
		listPreferences: vi.fn(),
		createPreference: vi.fn(),
		updatePreference: vi.fn(),
		deletePreference: vi.fn(),
		listLogs: vi.fn(),
	},
}));

import { notificationsApi } from '../lib/api';

describe('useNotificationChannels', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches channels', async () => {
		vi.mocked(notificationsApi.listChannels).mockResolvedValue([]);
		const { result } = renderHook(() => useNotificationChannels(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useNotificationChannel', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches a channel', async () => {
		vi.mocked(notificationsApi.getChannel).mockResolvedValue({
			channel: { id: 'ch1' },
		});
		const { result } = renderHook(() => useNotificationChannel('ch1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useNotificationChannel(''), { wrapper: createWrapper() });
		expect(notificationsApi.getChannel).not.toHaveBeenCalled();
	});
});

describe('useCreateNotificationChannel', () => {
	beforeEach(() => vi.clearAllMocks());

	it('creates a channel', async () => {
		vi.mocked(notificationsApi.createChannel).mockResolvedValue({ id: 'ch1' });
		const { result } = renderHook(() => useCreateNotificationChannel(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({
			type: 'email',
			name: 'Test',
			config: {},
		} as Parameters<typeof notificationsApi.createChannel>[0]);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteNotificationChannel', () => {
	beforeEach(() => vi.clearAllMocks());

	it('deletes a channel', async () => {
		vi.mocked(notificationsApi.deleteChannel).mockResolvedValue({
			message: 'Deleted',
		});
		const { result } = renderHook(() => useDeleteNotificationChannel(), {
			wrapper: createWrapper(),
		});
		result.current.mutate('ch1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useNotificationPreferences', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches preferences', async () => {
		vi.mocked(notificationsApi.listPreferences).mockResolvedValue([]);
		const { result } = renderHook(() => useNotificationPreferences(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useCreateNotificationPreference', () => {
	beforeEach(() => vi.clearAllMocks());

	it('creates a preference', async () => {
		vi.mocked(notificationsApi.createPreference).mockResolvedValue({
			id: 'pref1',
		});
		const { result } = renderHook(() => useCreateNotificationPreference(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({
			event_type: 'backup_failed',
			channel_id: 'ch1',
		} as Parameters<typeof notificationsApi.createPreference>[0]);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteNotificationPreference', () => {
	beforeEach(() => vi.clearAllMocks());

	it('deletes a preference', async () => {
		vi.mocked(notificationsApi.deletePreference).mockResolvedValue({
			message: 'Deleted',
		});
		const { result } = renderHook(() => useDeleteNotificationPreference(), {
			wrapper: createWrapper(),
		});
		result.current.mutate('pref1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useNotificationLogs', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches logs', async () => {
		vi.mocked(notificationsApi.listLogs).mockResolvedValue([]);
		const { result } = renderHook(() => useNotificationLogs(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
