import type {
	Agent,
	AgentsResponse,
	Alert,
	AlertCountResponse,
	AlertRule,
	AlertRulesResponse,
	AlertsResponse,
	AuditLog,
	AuditLogFilter,
	AuditLogsResponse,
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
	CreateRestoreRequest,
	CreateScheduleRequest,
	CreateVerificationScheduleRequest,
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
	RepositoryGrowthResponse,
	RepositoryHistoryResponse,
	RepositoryStatsListItem,
	RepositoryStatsListResponse,
	RepositoryStatsResponse,
	Restore,
	RestoresResponse,
	RotateAPIKeyResponse,
	RunScheduleResponse,
	Schedule,
	SchedulesResponse,
	Snapshot,
	SnapshotFilesResponse,
	SnapshotsResponse,
	StorageGrowthPoint,
	StorageGrowthResponse,
	StorageStatsSummary,
	SwitchOrgRequest,
	TestConnectionRequest,
	TestRepositoryResponse,
	TriggerVerificationRequest,
	UpdateAlertRuleRequest,
	UpdateMemberRequest,
	UpdateNotificationChannelRequest,
	UpdateNotificationPreferenceRequest,
	UpdateOrgRequest,
	UpdateRepositoryRequest,
	UpdateScheduleRequest,
	UpdateVerificationScheduleRequest,
	User,
	Verification,
	VerificationSchedule,
	VerificationSchedulesResponse,
	VerificationStatusResponse,
	VerificationsResponse,
	CreateDRRunbookRequest,
	CreateDRTestScheduleRequest,
	DRRunbook,
	DRRunbookRenderResponse,
	DRRunbooksResponse,
	DRStatus,
	DRTest,
	DRTestSchedule,
	DRTestSchedulesResponse,
	DRTestsResponse,
	RunDRTestRequest,
	UpdateDRRunbookRequest,
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

// Snapshots API
export const snapshotsApi = {
	list: async (params?: {
		agent_id?: string;
		repository_id?: string;
	}): Promise<Snapshot[]> => {
		const searchParams = new URLSearchParams();
		if (params?.agent_id) searchParams.set('agent_id', params.agent_id);
		if (params?.repository_id)
			searchParams.set('repository_id', params.repository_id);

		const query = searchParams.toString();
		const endpoint = query ? `/snapshots?${query}` : '/snapshots';
		const response = await fetchApi<SnapshotsResponse>(endpoint);
		return response.snapshots ?? [];
	},

	get: async (id: string): Promise<Snapshot> =>
		fetchApi<Snapshot>(`/snapshots/${id}`),

	listFiles: async (
		id: string,
		path?: string,
	): Promise<SnapshotFilesResponse> => {
		const endpoint = path
			? `/snapshots/${id}/files?path=${encodeURIComponent(path)}`
			: `/snapshots/${id}/files`;
		return fetchApi<SnapshotFilesResponse>(endpoint);
	},
};

// Restores API
export const restoresApi = {
	list: async (params?: {
		agent_id?: string;
		status?: string;
	}): Promise<Restore[]> => {
		const searchParams = new URLSearchParams();
		if (params?.agent_id) searchParams.set('agent_id', params.agent_id);
		if (params?.status) searchParams.set('status', params.status);

		const query = searchParams.toString();
		const endpoint = query ? `/restores?${query}` : '/restores';
		const response = await fetchApi<RestoresResponse>(endpoint);
		return response.restores ?? [];
	},

	get: async (id: string): Promise<Restore> =>
		fetchApi<Restore>(`/restores/${id}`),

	create: async (data: CreateRestoreRequest): Promise<Restore> =>
		fetchApi<Restore>('/restores', {
			method: 'POST',
			body: JSON.stringify(data),
		}),
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

// Audit Logs API
export const auditLogsApi = {
	list: async (filter?: AuditLogFilter): Promise<AuditLogsResponse> => {
		const searchParams = new URLSearchParams();
		if (filter?.action) searchParams.set('action', filter.action);
		if (filter?.resource_type)
			searchParams.set('resource_type', filter.resource_type);
		if (filter?.result) searchParams.set('result', filter.result);
		if (filter?.start_date) searchParams.set('start_date', filter.start_date);
		if (filter?.end_date) searchParams.set('end_date', filter.end_date);
		if (filter?.search) searchParams.set('search', filter.search);
		if (filter?.limit) searchParams.set('limit', filter.limit.toString());
		if (filter?.offset) searchParams.set('offset', filter.offset.toString());

		const query = searchParams.toString();
		const endpoint = query ? `/audit-logs?${query}` : '/audit-logs';
		return fetchApi<AuditLogsResponse>(endpoint);
	},

	get: async (id: string): Promise<AuditLog> =>
		fetchApi<AuditLog>(`/audit-logs/${id}`),

	exportCsv: async (filter?: AuditLogFilter): Promise<Blob> => {
		const searchParams = new URLSearchParams();
		if (filter?.action) searchParams.set('action', filter.action);
		if (filter?.resource_type)
			searchParams.set('resource_type', filter.resource_type);
		if (filter?.result) searchParams.set('result', filter.result);
		if (filter?.start_date) searchParams.set('start_date', filter.start_date);
		if (filter?.end_date) searchParams.set('end_date', filter.end_date);
		if (filter?.search) searchParams.set('search', filter.search);

		const query = searchParams.toString();
		const endpoint = query
			? `/audit-logs/export/csv?${query}`
			: '/audit-logs/export/csv';
		const response = await fetch(`${API_BASE}${endpoint}`, {
			credentials: 'include',
		});
		if (!response.ok) {
			throw new ApiError(response.status, 'Failed to export audit logs');
		}
		return response.blob();
	},

	exportJson: async (filter?: AuditLogFilter): Promise<Blob> => {
		const searchParams = new URLSearchParams();
		if (filter?.action) searchParams.set('action', filter.action);
		if (filter?.resource_type)
			searchParams.set('resource_type', filter.resource_type);
		if (filter?.result) searchParams.set('result', filter.result);
		if (filter?.start_date) searchParams.set('start_date', filter.start_date);
		if (filter?.end_date) searchParams.set('end_date', filter.end_date);
		if (filter?.search) searchParams.set('search', filter.search);

		const query = searchParams.toString();
		const endpoint = query
			? `/audit-logs/export/json?${query}`
			: '/audit-logs/export/json';
		const response = await fetch(`${API_BASE}${endpoint}`, {
			credentials: 'include',
		});
		if (!response.ok) {
			throw new ApiError(response.status, 'Failed to export audit logs');
		}
		return response.blob();
	},
};

// Storage Stats API
export const statsApi = {
	getSummary: async (): Promise<StorageStatsSummary> =>
		fetchApi<StorageStatsSummary>('/stats/summary'),

	getGrowth: async (days = 30): Promise<StorageGrowthPoint[]> => {
		const response = await fetchApi<StorageGrowthResponse>(
			`/stats/growth?days=${days}`,
		);
		return response.growth ?? [];
	},

	listRepositoryStats: async (): Promise<RepositoryStatsListItem[]> => {
		const response = await fetchApi<RepositoryStatsListResponse>(
			'/stats/repositories',
		);
		return response.stats ?? [];
	},

	getRepositoryStats: async (id: string): Promise<RepositoryStatsResponse> =>
		fetchApi<RepositoryStatsResponse>(`/stats/repositories/${id}`),

	getRepositoryGrowth: async (
		id: string,
		days = 30,
	): Promise<RepositoryGrowthResponse> =>
		fetchApi<RepositoryGrowthResponse>(
			`/stats/repositories/${id}/growth?days=${days}`,
		),

	getRepositoryHistory: async (
		id: string,
		limit = 30,
	): Promise<RepositoryHistoryResponse> =>
		fetchApi<RepositoryHistoryResponse>(
			`/stats/repositories/${id}/history?limit=${limit}`,
		),
};

// Verifications API
export const verificationsApi = {
	listByRepository: async (repoId: string): Promise<Verification[]> => {
		const response = await fetchApi<VerificationsResponse>(
			`/repositories/${repoId}/verifications`,
		);
		return response.verifications ?? [];
	},

	get: async (id: string): Promise<Verification> =>
		fetchApi<Verification>(`/verifications/${id}`),

	getStatus: async (repoId: string): Promise<VerificationStatusResponse> =>
		fetchApi<VerificationStatusResponse>(
			`/repositories/${repoId}/verification-status`,
		),

	trigger: async (
		repoId: string,
		data: TriggerVerificationRequest,
	): Promise<Verification> =>
		fetchApi<Verification>(`/repositories/${repoId}/verifications`, {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	listSchedules: async (repoId: string): Promise<VerificationSchedule[]> => {
		const response = await fetchApi<VerificationSchedulesResponse>(
			`/repositories/${repoId}/verification-schedules`,
		);
		return response.schedules ?? [];
	},

	createSchedule: async (
		repoId: string,
		data: CreateVerificationScheduleRequest,
	): Promise<VerificationSchedule> =>
		fetchApi<VerificationSchedule>(
			`/repositories/${repoId}/verification-schedules`,
			{
				method: 'POST',
				body: JSON.stringify(data),
			},
		),

	getSchedule: async (id: string): Promise<VerificationSchedule> =>
		fetchApi<VerificationSchedule>(`/verification-schedules/${id}`),

	updateSchedule: async (
		id: string,
		data: UpdateVerificationScheduleRequest,
	): Promise<VerificationSchedule> =>
		fetchApi<VerificationSchedule>(`/verification-schedules/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	deleteSchedule: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/verification-schedules/${id}`, {
			method: 'DELETE',
		}),
};

// DR Runbooks API
export const drRunbooksApi = {
	list: async (): Promise<DRRunbook[]> => {
		const response = await fetchApi<DRRunbooksResponse>('/dr-runbooks');
		return response.runbooks ?? [];
	},

	get: async (id: string): Promise<DRRunbook> =>
		fetchApi<DRRunbook>(`/dr-runbooks/${id}`),

	create: async (data: CreateDRRunbookRequest): Promise<DRRunbook> =>
		fetchApi<DRRunbook>('/dr-runbooks', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateDRRunbookRequest,
	): Promise<DRRunbook> =>
		fetchApi<DRRunbook>(`/dr-runbooks/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/dr-runbooks/${id}`, {
			method: 'DELETE',
		}),

	activate: async (id: string): Promise<DRRunbook> =>
		fetchApi<DRRunbook>(`/dr-runbooks/${id}/activate`, {
			method: 'POST',
		}),

	archive: async (id: string): Promise<DRRunbook> =>
		fetchApi<DRRunbook>(`/dr-runbooks/${id}/archive`, {
			method: 'POST',
		}),

	render: async (id: string): Promise<DRRunbookRenderResponse> =>
		fetchApi<DRRunbookRenderResponse>(`/dr-runbooks/${id}/render`),

	generateFromSchedule: async (scheduleId: string): Promise<DRRunbook> =>
		fetchApi<DRRunbook>(`/dr-runbooks/${scheduleId}/generate`, {
			method: 'POST',
		}),

	getStatus: async (): Promise<DRStatus> =>
		fetchApi<DRStatus>('/dr-runbooks/status'),

	listTestSchedules: async (runbookId: string): Promise<DRTestSchedule[]> => {
		const response = await fetchApi<DRTestSchedulesResponse>(
			`/dr-runbooks/${runbookId}/test-schedules`,
		);
		return response.schedules ?? [];
	},

	createTestSchedule: async (
		runbookId: string,
		data: CreateDRTestScheduleRequest,
	): Promise<DRTestSchedule> =>
		fetchApi<DRTestSchedule>(`/dr-runbooks/${runbookId}/test-schedules`, {
			method: 'POST',
			body: JSON.stringify(data),
		}),
};

// DR Tests API
export const drTestsApi = {
	list: async (params?: {
		runbook_id?: string;
		status?: string;
	}): Promise<DRTest[]> => {
		const searchParams = new URLSearchParams();
		if (params?.runbook_id) searchParams.set('runbook_id', params.runbook_id);
		if (params?.status) searchParams.set('status', params.status);

		const query = searchParams.toString();
		const endpoint = query ? `/dr-tests?${query}` : '/dr-tests';
		const response = await fetchApi<DRTestsResponse>(endpoint);
		return response.tests ?? [];
	},

	get: async (id: string): Promise<DRTest> =>
		fetchApi<DRTest>(`/dr-tests/${id}`),

	run: async (data: RunDRTestRequest): Promise<DRTest> =>
		fetchApi<DRTest>('/dr-tests', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	cancel: async (id: string, notes?: string): Promise<DRTest> =>
		fetchApi<DRTest>(`/dr-tests/${id}/cancel`, {
			method: 'POST',
			body: JSON.stringify({ notes }),
		}),
};
