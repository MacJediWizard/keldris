import type {
	Agent,
	AgentsResponse,
	Alert,
	AlertCountResponse,
	AlertRule,
	AlertRulesResponse,
	AlertsResponse,
	Backup,
	BackupsResponse,
	CreateAgentRequest,
	CreateAgentResponse,
	CreateAlertRuleRequest,
	CreateNotificationChannelRequest,
	CreateNotificationPreferenceRequest,
	CreateOrgRequest,
	CreateRepositoryRequest,
	CreateRepositoryResponse,
	CreateScheduleRequest,
	ErrorResponse,
	InvitationsResponse,
	InviteMemberRequest,
	InviteResponse,
	KeyRecoveryResponse,
	MembersResponse,
	MessageResponse,
	NotificationChannel,
	NotificationChannelWithPreferencesResponse,
	NotificationChannelsResponse,
	NotificationLog,
	NotificationLogsResponse,
	NotificationPreference,
	NotificationPreferencesResponse,
	OrgInvitation,
	OrgMember,
	OrgResponse,
	OrganizationWithRole,
	OrganizationsResponse,
	RepositoriesResponse,
	Repository,
	RotateAPIKeyResponse,
	RunScheduleResponse,
	Schedule,
	SchedulesResponse,
	SwitchOrgRequest,
	TestConnectionRequest,
	TestRepositoryResponse,
	UpdateAlertRuleRequest,
	UpdateMemberRequest,
	UpdateNotificationChannelRequest,
	UpdateNotificationPreferenceRequest,
	UpdateOrgRequest,
	UpdateRepositoryRequest,
	UpdateScheduleRequest,
	User,
} from './types';

const API_BASE = '/api/v1';

export class ApiError extends Error {
	constructor(
		public status: number,
		message: string,
	) {
		super(message);
		this.name = 'ApiError';
	}
}

async function handleResponse<T>(response: Response): Promise<T> {
	if (response.status === 401) {
		window.location.href = '/auth/login';
		throw new ApiError(401, 'Unauthorized');
	}

	if (!response.ok) {
		const errorData = (await response.json().catch(() => ({
			error: 'Unknown error',
		}))) as ErrorResponse;
		throw new ApiError(response.status, errorData.error);
	}

	return response.json() as Promise<T>;
}

async function fetchApi<T>(
	endpoint: string,
	options: RequestInit = {},
): Promise<T> {
	const response = await fetch(`${API_BASE}${endpoint}`, {
		...options,
		credentials: 'include',
		headers: {
			'Content-Type': 'application/json',
			...options.headers,
		},
	});

	return handleResponse<T>(response);
}

async function fetchAuth<T>(
	endpoint: string,
	options: RequestInit = {},
): Promise<T> {
	const response = await fetch(endpoint, {
		...options,
		credentials: 'include',
		headers: {
			'Content-Type': 'application/json',
			...options.headers,
		},
	});

	return handleResponse<T>(response);
}

// Auth API
export const authApi = {
	me: async (): Promise<User> => fetchAuth<User>('/auth/me'),

	logout: async (): Promise<MessageResponse> =>
		fetchAuth<MessageResponse>('/auth/logout', { method: 'POST' }),

	getLoginUrl: () => '/auth/login',
};

// Agents API
export const agentsApi = {
	list: async (): Promise<Agent[]> => {
		const response = await fetchApi<AgentsResponse>('/agents');
		return response.agents ?? [];
	},

	get: async (id: string): Promise<Agent> => fetchApi<Agent>(`/agents/${id}`),

	create: async (data: CreateAgentRequest): Promise<CreateAgentResponse> =>
		fetchApi<CreateAgentResponse>('/agents', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/agents/${id}`, {
			method: 'DELETE',
		}),

	rotateApiKey: async (id: string): Promise<RotateAPIKeyResponse> =>
		fetchApi<RotateAPIKeyResponse>(`/agents/${id}/apikey/rotate`, {
			method: 'POST',
		}),

	revokeApiKey: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/agents/${id}/apikey`, {
			method: 'DELETE',
		}),
};

// Repositories API
export const repositoriesApi = {
	list: async (): Promise<Repository[]> => {
		const response = await fetchApi<RepositoriesResponse>('/repositories');
		return response.repositories ?? [];
	},

	get: async (id: string): Promise<Repository> =>
		fetchApi<Repository>(`/repositories/${id}`),

	create: async (
		data: CreateRepositoryRequest,
	): Promise<CreateRepositoryResponse> =>
		fetchApi<CreateRepositoryResponse>('/repositories', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateRepositoryRequest,
	): Promise<Repository> =>
		fetchApi<Repository>(`/repositories/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/repositories/${id}`, {
			method: 'DELETE',
		}),

	test: async (id: string): Promise<TestRepositoryResponse> =>
		fetchApi<TestRepositoryResponse>(`/repositories/${id}/test`, {
			method: 'POST',
		}),

	testConnection: async (
		data: TestConnectionRequest,
	): Promise<TestRepositoryResponse> =>
		fetchApi<TestRepositoryResponse>('/repositories/test-connection', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	recoverKey: async (id: string): Promise<KeyRecoveryResponse> =>
		fetchApi<KeyRecoveryResponse>(`/repositories/${id}/key/recover`),
};

// Schedules API
export const schedulesApi = {
	list: async (agentId?: string): Promise<Schedule[]> => {
		const endpoint = agentId ? `/schedules?agent_id=${agentId}` : '/schedules';
		const response = await fetchApi<SchedulesResponse>(endpoint);
		return response.schedules ?? [];
	},

	get: async (id: string): Promise<Schedule> =>
		fetchApi<Schedule>(`/schedules/${id}`),

	create: async (data: CreateScheduleRequest): Promise<Schedule> =>
		fetchApi<Schedule>('/schedules', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (id: string, data: UpdateScheduleRequest): Promise<Schedule> =>
		fetchApi<Schedule>(`/schedules/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/schedules/${id}`, {
			method: 'DELETE',
		}),

	run: async (id: string): Promise<RunScheduleResponse> =>
		fetchApi<RunScheduleResponse>(`/schedules/${id}/run`, {
			method: 'POST',
		}),
};

// Backups API
export const backupsApi = {
	list: async (params?: {
		agent_id?: string;
		schedule_id?: string;
		status?: string;
	}): Promise<Backup[]> => {
		const searchParams = new URLSearchParams();
		if (params?.agent_id) searchParams.set('agent_id', params.agent_id);
		if (params?.schedule_id)
			searchParams.set('schedule_id', params.schedule_id);
		if (params?.status) searchParams.set('status', params.status);

		const query = searchParams.toString();
		const endpoint = query ? `/backups?${query}` : '/backups';
		const response = await fetchApi<BackupsResponse>(endpoint);
		return response.backups ?? [];
	},

	get: async (id: string): Promise<Backup> =>
		fetchApi<Backup>(`/backups/${id}`),
};

// Alerts API
export const alertsApi = {
	list: async (): Promise<Alert[]> => {
		const response = await fetchApi<AlertsResponse>('/alerts');
		return response.alerts ?? [];
	},

	listActive: async (): Promise<Alert[]> => {
		const response = await fetchApi<AlertsResponse>('/alerts/active');
		return response.alerts ?? [];
	},

	count: async (): Promise<number> => {
		const response = await fetchApi<AlertCountResponse>('/alerts/count');
		return response.count;
	},

	get: async (id: string): Promise<Alert> => fetchApi<Alert>(`/alerts/${id}`),

	acknowledge: async (id: string): Promise<Alert> =>
		fetchApi<Alert>(`/alerts/${id}/actions/acknowledge`, {
			method: 'POST',
		}),

	resolve: async (id: string): Promise<Alert> =>
		fetchApi<Alert>(`/alerts/${id}/actions/resolve`, {
			method: 'POST',
		}),
};

// Alert Rules API
export const alertRulesApi = {
	list: async (): Promise<AlertRule[]> => {
		const response = await fetchApi<AlertRulesResponse>('/alert-rules');
		return response.rules ?? [];
	},

	get: async (id: string): Promise<AlertRule> =>
		fetchApi<AlertRule>(`/alert-rules/${id}`),

	create: async (data: CreateAlertRuleRequest): Promise<AlertRule> =>
		fetchApi<AlertRule>('/alert-rules', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateAlertRuleRequest,
	): Promise<AlertRule> =>
		fetchApi<AlertRule>(`/alert-rules/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/alert-rules/${id}`, {
			method: 'DELETE',
		}),
};

// Organizations API
export const organizationsApi = {
	list: async (): Promise<OrganizationWithRole[]> => {
		const response = await fetchApi<OrganizationsResponse>('/organizations');
		return response.organizations ?? [];
	},

	get: async (id: string): Promise<OrgResponse> =>
		fetchApi<OrgResponse>(`/organizations/${id}`),

	getCurrent: async (): Promise<OrgResponse> =>
		fetchApi<OrgResponse>('/organizations/current'),

	create: async (data: CreateOrgRequest): Promise<OrgResponse> =>
		fetchApi<OrgResponse>('/organizations', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (id: string, data: UpdateOrgRequest): Promise<OrgResponse> =>
		fetchApi<OrgResponse>(`/organizations/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/organizations/${id}`, {
			method: 'DELETE',
		}),

	switch: async (data: SwitchOrgRequest): Promise<OrgResponse> =>
		fetchApi<OrgResponse>('/organizations/switch', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	// Members
	listMembers: async (orgId: string): Promise<OrgMember[]> => {
		const response = await fetchApi<MembersResponse>(
			`/organizations/${orgId}/members`,
		);
		return response.members ?? [];
	},

	updateMember: async (
		orgId: string,
		userId: string,
		data: UpdateMemberRequest,
	): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/organizations/${orgId}/members/${userId}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	removeMember: async (
		orgId: string,
		userId: string,
	): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/organizations/${orgId}/members/${userId}`, {
			method: 'DELETE',
		}),

	// Invitations
	listInvitations: async (orgId: string): Promise<OrgInvitation[]> => {
		const response = await fetchApi<InvitationsResponse>(
			`/organizations/${orgId}/invitations`,
		);
		return response.invitations ?? [];
	},

	createInvitation: async (
		orgId: string,
		data: InviteMemberRequest,
	): Promise<InviteResponse> =>
		fetchApi<InviteResponse>(`/organizations/${orgId}/invitations`, {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	deleteInvitation: async (
		orgId: string,
		invitationId: string,
	): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(
			`/organizations/${orgId}/invitations/${invitationId}`,
			{
				method: 'DELETE',
			},
		),

	acceptInvitation: async (token: string): Promise<OrgResponse> =>
		fetchApi<OrgResponse>('/invitations/accept', {
			method: 'POST',
			body: JSON.stringify({ token }),
		}),
};

// Notifications API
export const notificationsApi = {
	// Channels
	listChannels: async (): Promise<NotificationChannel[]> => {
		const response = await fetchApi<NotificationChannelsResponse>(
			'/notifications/channels',
		);
		return response.channels ?? [];
	},

	getChannel: async (
		id: string,
	): Promise<NotificationChannelWithPreferencesResponse> =>
		fetchApi<NotificationChannelWithPreferencesResponse>(
			`/notifications/channels/${id}`,
		),

	createChannel: async (
		data: CreateNotificationChannelRequest,
	): Promise<NotificationChannel> =>
		fetchApi<NotificationChannel>('/notifications/channels', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	updateChannel: async (
		id: string,
		data: UpdateNotificationChannelRequest,
	): Promise<NotificationChannel> =>
		fetchApi<NotificationChannel>(`/notifications/channels/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	deleteChannel: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/notifications/channels/${id}`, {
			method: 'DELETE',
		}),

	// Preferences
	listPreferences: async (): Promise<NotificationPreference[]> => {
		const response = await fetchApi<NotificationPreferencesResponse>(
			'/notifications/preferences',
		);
		return response.preferences ?? [];
	},

	createPreference: async (
		data: CreateNotificationPreferenceRequest,
	): Promise<NotificationPreference> =>
		fetchApi<NotificationPreference>('/notifications/preferences', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	updatePreference: async (
		id: string,
		data: UpdateNotificationPreferenceRequest,
	): Promise<NotificationPreference> =>
		fetchApi<NotificationPreference>(`/notifications/preferences/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	deletePreference: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/notifications/preferences/${id}`, {
			method: 'DELETE',
		}),

	// Logs
	listLogs: async (): Promise<NotificationLog[]> => {
		const response = await fetchApi<NotificationLogsResponse>(
			'/notifications/logs',
		);
		return response.logs ?? [];
	},
};
