import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { usersApi } from '../lib/api';
import type {
	ImpersonateUserRequest,
	InviteUserRequest,
	ResetPasswordRequest,
	UpdateUserRequest,
} from '../lib/types';

// Query keys
const USERS_KEY = ['users'];
const USER_KEY = (id: string) => ['users', id];
const USER_ACTIVITY_KEY = (id: string) => ['users', id, 'activity'];
const ORG_ACTIVITY_KEY = ['org-activity-logs'];
const IMPERSONATION_LOGS_KEY = ['impersonation-logs'];

// Query: List all users in the organization
export function useUsers() {
	return useQuery({
		queryKey: USERS_KEY,
		queryFn: usersApi.list,
		staleTime: 30 * 1000,
	});
}

// Query: Get a specific user
export function useUser(id: string) {
	return useQuery({
		queryKey: USER_KEY(id),
		queryFn: () => usersApi.get(id),
		enabled: !!id,
	});
}

// Mutation: Invite a new user
export function useInviteUser() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (data: InviteUserRequest) => usersApi.invite(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: USERS_KEY });
		},
	});
}

// Mutation: Update a user
export function useUpdateUser() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data: UpdateUserRequest;
		}) => usersApi.update(id, data),
		onSuccess: (_, { id }) => {
			queryClient.invalidateQueries({ queryKey: USERS_KEY });
			queryClient.invalidateQueries({ queryKey: USER_KEY(id) });
		},
	});
}

// Mutation: Delete a user
export function useDeleteUser() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (id: string) => usersApi.delete(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: USERS_KEY });
		},
	});
}

// Mutation: Reset user password
export function useResetUserPassword() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data: ResetPasswordRequest;
		}) => usersApi.resetPassword(id, data),
		onSuccess: (_, { id }) => {
			queryClient.invalidateQueries({ queryKey: USER_KEY(id) });
		},
	});
}

// Mutation: Disable a user
export function useDisableUser() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (id: string) => usersApi.disable(id),
		onSuccess: (_, id) => {
			queryClient.invalidateQueries({ queryKey: USERS_KEY });
			queryClient.invalidateQueries({ queryKey: USER_KEY(id) });
		},
	});
}

// Mutation: Enable a user
export function useEnableUser() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (id: string) => usersApi.enable(id),
		onSuccess: (_, id) => {
			queryClient.invalidateQueries({ queryKey: USERS_KEY });
			queryClient.invalidateQueries({ queryKey: USER_KEY(id) });
		},
	});
}

// Query: Get user activity logs
export function useUserActivity(id: string, limit = 50, offset = 0) {
	return useQuery({
		queryKey: [...USER_ACTIVITY_KEY(id), limit, offset],
		queryFn: () => usersApi.getActivity(id, limit, offset),
		enabled: !!id,
	});
}

// Query: Get organization activity logs
export function useOrgActivityLogs(limit = 50, offset = 0) {
	return useQuery({
		queryKey: [...ORG_ACTIVITY_KEY, limit, offset],
		queryFn: () => usersApi.getOrgActivityLogs(limit, offset),
	});
}

// Mutation: Start impersonating a user
export function useStartImpersonation() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data: ImpersonateUserRequest;
		}) => usersApi.startImpersonation(id, data),
		onSuccess: () => {
			// Invalidate auth data as we're now acting as a different user
			queryClient.invalidateQueries({ queryKey: ['auth'] });
			// Reload the page to reflect the impersonation
			window.location.reload();
		},
	});
}

// Mutation: End impersonation
export function useEndImpersonation() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: () => usersApi.endImpersonation(),
		onSuccess: () => {
			// Invalidate auth data as we're returning to the original user
			queryClient.invalidateQueries({ queryKey: ['auth'] });
			// Reload the page to reflect the end of impersonation
			window.location.reload();
		},
	});
}

// Query: Get impersonation logs
export function useImpersonationLogs(limit = 50, offset = 0) {
	return useQuery({
		queryKey: [...IMPERSONATION_LOGS_KEY, limit, offset],
		queryFn: () => usersApi.getImpersonationLogs(limit, offset),
	});
}
