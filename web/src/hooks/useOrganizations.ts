import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { organizationsApi } from '../lib/api';
import type {
	CreateOrgRequest,
	InviteMemberRequest,
	OrgRole,
	UpdateOrgRequest,
} from '../lib/types';

export function useOrganizations() {
	return useQuery({
		queryKey: ['organizations'],
		queryFn: organizationsApi.list,
	});
}

export function useOrganization(id: string) {
	return useQuery({
		queryKey: ['organizations', id],
		queryFn: () => organizationsApi.get(id),
		enabled: !!id,
	});
}

export function useCurrentOrganization() {
	return useQuery({
		queryKey: ['organizations', 'current'],
		queryFn: organizationsApi.getCurrent,
		retry: false,
	});
}

export function useCreateOrganization() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (data: CreateOrgRequest) => organizationsApi.create(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['organizations'] });
			queryClient.invalidateQueries({ queryKey: ['auth', 'me'] });
		},
	});
}

export function useUpdateOrganization() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ id, data }: { id: string; data: UpdateOrgRequest }) =>
			organizationsApi.update(id, data),
		onSuccess: (_, { id }) => {
			queryClient.invalidateQueries({ queryKey: ['organizations'] });
			queryClient.invalidateQueries({ queryKey: ['organizations', id] });
			queryClient.invalidateQueries({ queryKey: ['organizations', 'current'] });
		},
	});
}

export function useDeleteOrganization() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => organizationsApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['organizations'] });
			queryClient.invalidateQueries({ queryKey: ['auth', 'me'] });
		},
	});
}

export function useSwitchOrganization() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (orgId: string) => organizationsApi.switch({ org_id: orgId }),
		onSuccess: () => {
			// Clear all resource caches when switching orgs
			queryClient.invalidateQueries({ queryKey: ['organizations', 'current'] });
			queryClient.invalidateQueries({ queryKey: ['auth', 'me'] });
			queryClient.invalidateQueries({ queryKey: ['agents'] });
			queryClient.invalidateQueries({ queryKey: ['repositories'] });
			queryClient.invalidateQueries({ queryKey: ['schedules'] });
			queryClient.invalidateQueries({ queryKey: ['backups'] });
		},
	});
}

// Member hooks
export function useOrgMembers(orgId: string) {
	return useQuery({
		queryKey: ['organizations', orgId, 'members'],
		queryFn: () => organizationsApi.listMembers(orgId),
		enabled: !!orgId,
	});
}

export function useUpdateMember() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			orgId,
			userId,
			role,
		}: {
			orgId: string;
			userId: string;
			role: OrgRole;
		}) => organizationsApi.updateMember(orgId, userId, { role }),
		onSuccess: (_, { orgId }) => {
			queryClient.invalidateQueries({
				queryKey: ['organizations', orgId, 'members'],
			});
		},
	});
}

export function useRemoveMember() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ orgId, userId }: { orgId: string; userId: string }) =>
			organizationsApi.removeMember(orgId, userId),
		onSuccess: (_, { orgId }) => {
			queryClient.invalidateQueries({
				queryKey: ['organizations', orgId, 'members'],
			});
		},
	});
}

// Invitation hooks
export function useOrgInvitations(orgId: string) {
	return useQuery({
		queryKey: ['organizations', orgId, 'invitations'],
		queryFn: () => organizationsApi.listInvitations(orgId),
		enabled: !!orgId,
	});
}

export function useCreateInvitation() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			orgId,
			data,
		}: {
			orgId: string;
			data: InviteMemberRequest;
		}) => organizationsApi.createInvitation(orgId, data),
		}: { orgId: string; data: InviteMemberRequest }) =>
			organizationsApi.createInvitation(orgId, data),
		onSuccess: (_, { orgId }) => {
			queryClient.invalidateQueries({
				queryKey: ['organizations', orgId, 'invitations'],
			});
		},
	});
}

export function useDeleteInvitation() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			orgId,
			invitationId,
		}: {
			orgId: string;
			invitationId: string;
		}) => organizationsApi.deleteInvitation(orgId, invitationId),
		onSuccess: (_, { orgId }) => {
			queryClient.invalidateQueries({
				queryKey: ['organizations', orgId, 'invitations'],
			});
		},
	});
}

export function useAcceptInvitation() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (token: string) => organizationsApi.acceptInvitation(token),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['organizations'] });
			queryClient.invalidateQueries({ queryKey: ['auth', 'me'] });
		},
	});
}

export function useResendInvitation() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			orgId,
			invitationId,
		}: {
			orgId: string;
			invitationId: string;
		}) => organizationsApi.resendInvitation(orgId, invitationId),
		onSuccess: (_, { orgId }) => {
			queryClient.invalidateQueries({
				queryKey: ['organizations', orgId, 'invitations'],
			});
		},
	});
}

export function useBulkInvite() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({
			orgId,
			invites,
		}: {
			orgId: string;
			invites: { email: string; role: string }[];
		}) => organizationsApi.bulkInvite(orgId, invites),
		onSuccess: (_, { orgId }) => {
			queryClient.invalidateQueries({
				queryKey: ['organizations', orgId, 'invitations'],
			});
		},
	});
}

export function useBulkInviteCSV() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ orgId, file }: { orgId: string; file: File }) =>
			organizationsApi.bulkInviteCSV(orgId, file),
		onSuccess: (_, { orgId }) => {
			queryClient.invalidateQueries({
				queryKey: ['organizations', orgId, 'invitations'],
			});
		},
	});
}

export function useInvitationByToken(token: string) {
	return useQuery({
		queryKey: ['invitations', token],
		queryFn: () => organizationsApi.getInvitationByToken(token),
		enabled: !!token,
		retry: false,
	});
}
