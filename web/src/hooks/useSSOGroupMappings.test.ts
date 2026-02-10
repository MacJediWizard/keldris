import { renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
	useSSOGroupMappings,
	useSSOGroupMapping,
	useCreateSSOGroupMapping,
	useDeleteSSOGroupMapping,
	useSSOSettings,
	useUpdateSSOSettings,
	useUserSSOGroups,
} from './useSSOGroupMappings';
import { createWrapper } from '../test/helpers';

vi.mock('../lib/api', () => ({
	ssoGroupMappingsApi: {
		list: vi.fn(),
		get: vi.fn(),
		create: vi.fn(),
		update: vi.fn(),
		delete: vi.fn(),
		getSettings: vi.fn(),
		updateSettings: vi.fn(),
		getUserSSOGroups: vi.fn(),
	},
}));

import { ssoGroupMappingsApi } from '../lib/api';

describe('useSSOGroupMappings', () => {
	beforeEach(() => vi.clearAllMocks());
	it('fetches mappings', async () => {
		vi.mocked(ssoGroupMappingsApi.list).mockResolvedValue([]);
		const { result } = renderHook(() => useSSOGroupMappings('o1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
	it('does not fetch when orgId is empty', () => {
		renderHook(() => useSSOGroupMappings(''), { wrapper: createWrapper() });
		expect(ssoGroupMappingsApi.list).not.toHaveBeenCalled();
	});
});

describe('useSSOGroupMapping', () => {
	beforeEach(() => vi.clearAllMocks());
	it('fetches a mapping', async () => {
		vi.mocked(ssoGroupMappingsApi.get).mockResolvedValue({ id: 'm1' });
		const { result } = renderHook(() => useSSOGroupMapping('o1', 'm1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useCreateSSOGroupMapping', () => {
	beforeEach(() => vi.clearAllMocks());
	it('creates a mapping', async () => {
		vi.mocked(ssoGroupMappingsApi.create).mockResolvedValue({ id: 'm1' });
		const { result } = renderHook(() => useCreateSSOGroupMapping(), { wrapper: createWrapper() });
		result.current.mutate({ orgId: 'o1', data: { sso_group: 'admins', role: 'admin' } as Parameters<typeof ssoGroupMappingsApi.create>[1] });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteSSOGroupMapping', () => {
	beforeEach(() => vi.clearAllMocks());
	it('deletes a mapping', async () => {
		vi.mocked(ssoGroupMappingsApi.delete).mockResolvedValue({ message: 'Deleted' });
		const { result } = renderHook(() => useDeleteSSOGroupMapping(), { wrapper: createWrapper() });
		result.current.mutate({ orgId: 'o1', id: 'm1' });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useSSOSettings', () => {
	beforeEach(() => vi.clearAllMocks());
	it('fetches SSO settings', async () => {
		vi.mocked(ssoGroupMappingsApi.getSettings).mockResolvedValue({ enabled: true });
		const { result } = renderHook(() => useSSOSettings('o1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useUpdateSSOSettings', () => {
	beforeEach(() => vi.clearAllMocks());
	it('updates SSO settings', async () => {
		vi.mocked(ssoGroupMappingsApi.updateSettings).mockResolvedValue({ enabled: false });
		const { result } = renderHook(() => useUpdateSSOSettings(), { wrapper: createWrapper() });
		result.current.mutate({ orgId: 'o1', data: { auto_provision: true } as Parameters<typeof ssoGroupMappingsApi.updateSettings>[1] });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useUserSSOGroups', () => {
	beforeEach(() => vi.clearAllMocks());
	it('fetches user SSO groups', async () => {
		vi.mocked(ssoGroupMappingsApi.getUserSSOGroups).mockResolvedValue({ groups: [] });
		const { result } = renderHook(() => useUserSSOGroups('u1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
	it('does not fetch when userId is empty', () => {
		renderHook(() => useUserSSOGroups(''), { wrapper: createWrapper() });
		expect(ssoGroupMappingsApi.getUserSSOGroups).not.toHaveBeenCalled();
	});
});
