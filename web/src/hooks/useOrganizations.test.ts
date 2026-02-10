import { renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
	useOrganizations,
	useOrganization,
	useCurrentOrganization,
	useCreateOrganization,
	useDeleteOrganization,
	useSwitchOrganization,
	useOrgMembers,
	useRemoveMember,
	useOrgInvitations,
	useCreateInvitation,
	useDeleteInvitation,
	useAcceptInvitation,
} from './useOrganizations';
import { createWrapper } from '../test/helpers';

vi.mock('../lib/api', () => ({
	organizationsApi: {
		list: vi.fn(),
		get: vi.fn(),
		getCurrent: vi.fn(),
		create: vi.fn(),
		update: vi.fn(),
		delete: vi.fn(),
		switch: vi.fn(),
		listMembers: vi.fn(),
		updateMember: vi.fn(),
		removeMember: vi.fn(),
		listInvitations: vi.fn(),
		createInvitation: vi.fn(),
		deleteInvitation: vi.fn(),
		acceptInvitation: vi.fn(),
	},
}));

import { organizationsApi } from '../lib/api';

describe('useOrganizations', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches organizations', async () => {
		vi.mocked(organizationsApi.list).mockResolvedValue([]);
		const { result } = renderHook(() => useOrganizations(), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useOrganization', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches an organization', async () => {
		vi.mocked(organizationsApi.get).mockResolvedValue({ organization: { id: 'o1' } });
		const { result } = renderHook(() => useOrganization('o1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useOrganization(''), { wrapper: createWrapper() });
		expect(organizationsApi.get).not.toHaveBeenCalled();
	});
});

describe('useCurrentOrganization', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches current organization', async () => {
		vi.mocked(organizationsApi.getCurrent).mockResolvedValue({ organization: { id: 'o1' } });
		const { result } = renderHook(() => useCurrentOrganization(), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useCreateOrganization', () => {
	beforeEach(() => vi.clearAllMocks());

	it('creates an organization', async () => {
		vi.mocked(organizationsApi.create).mockResolvedValue({ organization: { id: 'o1' } });
		const { result } = renderHook(() => useCreateOrganization(), { wrapper: createWrapper() });
		result.current.mutate({ name: 'New Org' });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteOrganization', () => {
	beforeEach(() => vi.clearAllMocks());

	it('deletes an organization', async () => {
		vi.mocked(organizationsApi.delete).mockResolvedValue({ message: 'Deleted' });
		const { result } = renderHook(() => useDeleteOrganization(), { wrapper: createWrapper() });
		result.current.mutate('o1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useSwitchOrganization', () => {
	beforeEach(() => vi.clearAllMocks());

	it('switches organization', async () => {
		vi.mocked(organizationsApi.switch).mockResolvedValue({ organization: { id: 'o2' } });
		const { result } = renderHook(() => useSwitchOrganization(), { wrapper: createWrapper() });
		result.current.mutate({ organization_id: 'o2' });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useOrgMembers', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches members', async () => {
		vi.mocked(organizationsApi.listMembers).mockResolvedValue([]);
		const { result } = renderHook(() => useOrgMembers('o1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when orgId is empty', () => {
		renderHook(() => useOrgMembers(''), { wrapper: createWrapper() });
		expect(organizationsApi.listMembers).not.toHaveBeenCalled();
	});
});

describe('useRemoveMember', () => {
	beforeEach(() => vi.clearAllMocks());

	it('removes a member', async () => {
		vi.mocked(organizationsApi.removeMember).mockResolvedValue({ message: 'Removed' });
		const { result } = renderHook(() => useRemoveMember(), { wrapper: createWrapper() });
		result.current.mutate({ orgId: 'o1', userId: 'u1' });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useOrgInvitations', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches invitations', async () => {
		vi.mocked(organizationsApi.listInvitations).mockResolvedValue([]);
		const { result } = renderHook(() => useOrgInvitations('o1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useCreateInvitation', () => {
	beforeEach(() => vi.clearAllMocks());

	it('creates an invitation', async () => {
		vi.mocked(organizationsApi.createInvitation).mockResolvedValue({ token: 'abc' });
		const { result } = renderHook(() => useCreateInvitation(), { wrapper: createWrapper() });
		result.current.mutate({ orgId: 'o1', data: { email: 'test@example.com', role: 'member' } });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteInvitation', () => {
	beforeEach(() => vi.clearAllMocks());

	it('deletes an invitation', async () => {
		vi.mocked(organizationsApi.deleteInvitation).mockResolvedValue({ message: 'Deleted' });
		const { result } = renderHook(() => useDeleteInvitation(), { wrapper: createWrapper() });
		result.current.mutate({ orgId: 'o1', invitationId: 'inv-1' });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useAcceptInvitation', () => {
	beforeEach(() => vi.clearAllMocks());

	it('accepts an invitation', async () => {
		vi.mocked(organizationsApi.acceptInvitation).mockResolvedValue({ organization: { id: 'o1' } });
		const { result } = renderHook(() => useAcceptInvitation(), { wrapper: createWrapper() });
		result.current.mutate('token-123');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
