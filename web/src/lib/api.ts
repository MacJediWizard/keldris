import type {
	ActiveMaintenanceResponse,
	AddAgentToGroupRequest,
	Agent,
	AgentBackupsResponse,
	AgentGroup,
	AgentGroupsResponse,
	AgentHealthHistoryResponse,
	AgentSchedulesResponse,
	AgentStatsResponse,
	AgentWithGroups,
	AgentsResponse,
	AgentsWithGroupsResponse,
	Alert,
	AlertCountResponse,
	AlertRule,
	AlertRulesResponse,
	AlertsResponse,
	ApplyPolicyRequest,
	ApplyPolicyResponse,
	AssignTagsRequest,
	AuditLog,
	AuditLogFilter,
	AuditLogsResponse,
	Backup,
	BackupDurationTrend,
	BackupDurationTrendResponse,
	BackupScript,
	BackupScriptsResponse,
	BackupSuccessRate,
	BackupSuccessRatesResponse,
	BackupsResponse,
	BuiltInPattern,
	BuiltInPatternsResponse,
	CategoriesResponse,
	CategoryInfo,
	CostAlert,
	CostAlertsResponse,
	CostForecastResponse,
	CostHistoryResponse,
	CostSummary,
	CreateAgentGroupRequest,
	CreateAgentRequest,
	CreateAgentResponse,
	CreateAlertRuleRequest,
	CreateBackupScriptRequest,
	CreateCostAlertRequest,
	CreateDRRunbookRequest,
	CreateDRTestScheduleRequest,
	CreateExcludePatternRequest,
	CreateMaintenanceWindowRequest,
	CreateNotificationChannelRequest,
	CreateNotificationPreferenceRequest,
	CreateOrgRequest,
	CreatePolicyRequest,
	CreateRegistrationCodeRequest,
	CreateRegistrationCodeResponse,
	CreateReportScheduleRequest,
	CreateRepositoryRequest,
	CreateRepositoryResponse,
	CreateRestoreRequest,
	CreateSSOGroupMappingRequest,
	CreateScheduleRequest,
	CreateSnapshotCommentRequest,
	CreateStoragePricingRequest,
	CreateTagRequest,
	CreateVerificationScheduleRequest,
	DRRunbook,
	DRRunbookRenderResponse,
	DRRunbooksResponse,
	DRStatus,
	DRTest,
	DRTestSchedule,
	DRTestSchedulesResponse,
	DRTestsResponse,
	DailyBackupStats,
	DailyBackupStatsResponse,
	DashboardStats,
	DefaultPricingResponse,
	ErrorResponse,
	ExcludePattern,
	ExcludePatternsResponse,
	FileHistoryParams,
	FileHistoryResponse,
	FleetHealthSummary,
	ImportPreviewRequest,
	ImportPreviewResponse,
	ImportRepositoryRequest,
	ImportRepositoryResponse,
	InvitationsResponse,
	InviteMemberRequest,
	InviteResponse,
	KeyRecoveryResponse,
	LegalHold,
	LegalHoldsResponse,
	CreateLegalHoldRequest,
	MaintenanceWindow,
	MaintenanceWindowsResponse,
	MembersResponse,
	MessageResponse,
	NotificationChannel,
	NotificationChannelWithPreferencesResponse,
	NotificationChannelsResponse,
	NotificationLog,
	NotificationLogsResponse,
	NotificationPreference,
	NotificationPreferencesResponse,
	OnboardingStatus,
	OnboardingStep,
	OrgInvitation,
	OrgMember,
	OrgResponse,
	OrganizationWithRole,
	OrganizationsResponse,
	PendingRegistration,
	PendingRegistrationsResponse,
	PoliciesResponse,
	Policy,
	ReplicationStatus,
	ReplicationStatusResponse,
	ReportFrequency,
	ReportHistory,
	ReportHistoryResponse,
	ReportPreviewResponse,
	ReportSchedule,
	ReportSchedulesResponse,
	RepositoriesResponse,
	Repository,
	RepositoryCostResponse,
	RepositoryCostsResponse,
	RepositoryGrowthResponse,
	RepositoryHistoryResponse,
	RepositoryStatsListItem,
	RepositoryStatsListResponse,
	RepositoryStatsResponse,
	Restore,
	RestoresResponse,
	RotateAPIKeyResponse,
	RunDRTestRequest,
	RunScheduleResponse,
	SSOGroupMapping,
	SSOGroupMappingResponse,
	SSOGroupMappingsResponse,
	SSOSettings,
	Schedule,
	SchedulesResponse,
	SearchFilter,
	SearchResponse,
	Snapshot,
	SnapshotComment,
	SnapshotCommentsResponse,
	SnapshotCompareResponse,
	SnapshotFilesResponse,
	SnapshotsResponse,
	StorageGrowthPoint,
	StorageGrowthResponse,
	StorageGrowthTrend,
	StorageGrowthTrendResponse,
	StoragePricing,
	StoragePricingResponse,
	StorageStatsSummary,
	SwitchOrgRequest,
	Tag,
	TagsResponse,
	TestConnectionRequest,
	TestRepositoryResponse,
	TriggerVerificationRequest,
	UpdateAgentGroupRequest,
	UpdateAlertRuleRequest,
	UpdateBackupScriptRequest,
	UpdateCostAlertRequest,
	UpdateDRRunbookRequest,
	UpdateExcludePatternRequest,
	UpdateMaintenanceWindowRequest,
	UpdateMemberRequest,
	UpdateNotificationChannelRequest,
	UpdateNotificationPreferenceRequest,
	UpdateOrgRequest,
	UpdatePolicyRequest,
	UpdateReportScheduleRequest,
	UpdateRepositoryRequest,
	UpdateSSOGroupMappingRequest,
	UpdateSSOSettingsRequest,
	UpdateScheduleRequest,
	UpdateStoragePricingRequest,
	UpdateTagRequest,
	UpdateUserPreferencesRequest,
	UpdateVerificationScheduleRequest,
	User,
	UserSSOGroups,
	Verification,
	VerificationSchedule,
	VerificationSchedulesResponse,
	VerificationStatusResponse,
	VerificationsResponse,
	VerifyImportAccessRequest,
	VerifyImportAccessResponse,
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

	updatePreferences: async (
		data: UpdateUserPreferencesRequest,
	): Promise<User> =>
		fetchAuth<User>('/auth/preferences', {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

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

	getStats: async (id: string): Promise<AgentStatsResponse> =>
		fetchApi<AgentStatsResponse>(`/agents/${id}/stats`),

	getBackups: async (id: string): Promise<AgentBackupsResponse> =>
		fetchApi<AgentBackupsResponse>(`/agents/${id}/backups`),

	getSchedules: async (id: string): Promise<AgentSchedulesResponse> =>
		fetchApi<AgentSchedulesResponse>(`/agents/${id}/schedules`),

	getHealthHistory: async (
		id: string,
		limit = 100,
	): Promise<AgentHealthHistoryResponse> =>
		fetchApi<AgentHealthHistoryResponse>(
			`/agents/${id}/health-history?limit=${limit}`,
		),

	getFleetHealth: async (): Promise<FleetHealthSummary> =>
		fetchApi<FleetHealthSummary>('/agents/fleet-health'),

	listWithGroups: async (): Promise<AgentWithGroups[]> => {
		const response = await fetchApi<AgentsWithGroupsResponse>(
			'/agents/with-groups',
		);
		return response.agents ?? [];
	},
};

// Agent Groups API
export const agentGroupsApi = {
	list: async (): Promise<AgentGroup[]> => {
		const response = await fetchApi<AgentGroupsResponse>('/agent-groups');
		return response.groups ?? [];
	},

	get: async (id: string): Promise<AgentGroup> =>
		fetchApi<AgentGroup>(`/agent-groups/${id}`),

	create: async (data: CreateAgentGroupRequest): Promise<AgentGroup> =>
		fetchApi<AgentGroup>('/agent-groups', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateAgentGroupRequest,
	): Promise<AgentGroup> =>
		fetchApi<AgentGroup>(`/agent-groups/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/agent-groups/${id}`, {
			method: 'DELETE',
		}),

	listMembers: async (groupId: string): Promise<Agent[]> => {
		const response = await fetchApi<AgentsResponse>(
			`/agent-groups/${groupId}/agents`,
		);
		return response.agents ?? [];
	},

	addAgent: async (
		groupId: string,
		data: AddAgentToGroupRequest,
	): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/agent-groups/${groupId}/agents`, {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	removeAgent: async (
		groupId: string,
		agentId: string,
	): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/agent-groups/${groupId}/agents/${agentId}`, {
			method: 'DELETE',
		}),
};

// Agent Registration Codes API
export const agentRegistrationApi = {
	createCode: async (
		data: CreateRegistrationCodeRequest,
	): Promise<CreateRegistrationCodeResponse> =>
		fetchApi<CreateRegistrationCodeResponse>('/agent-registration-codes', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	listPending: async (): Promise<PendingRegistration[]> => {
		const response = await fetchApi<PendingRegistrationsResponse>(
			'/agent-registration-codes',
		);
		return response.registrations ?? [];
	},

	deleteCode: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/agent-registration-codes/${id}`, {
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

// Repository Import API
export const repositoryImportApi = {
	verifyAccess: async (
		data: VerifyImportAccessRequest,
	): Promise<VerifyImportAccessResponse> =>
		fetchApi<VerifyImportAccessResponse>('/repositories/import/verify', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	preview: async (data: ImportPreviewRequest): Promise<ImportPreviewResponse> =>
		fetchApi<ImportPreviewResponse>('/repositories/import/preview', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	import: async (
		data: ImportRepositoryRequest,
	): Promise<ImportRepositoryResponse> =>
		fetchApi<ImportRepositoryResponse>('/repositories/import', {
			method: 'POST',
			body: JSON.stringify(data),
		}),
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

	getReplicationStatus: async (id: string): Promise<ReplicationStatus[]> => {
		const response = await fetchApi<ReplicationStatusResponse>(
			`/schedules/${id}/replication`,
		);
		return response.replication_status ?? [];
	},
};

// Policies API
export const policiesApi = {
	list: async (): Promise<Policy[]> => {
		const response = await fetchApi<PoliciesResponse>('/policies');
		return response.policies ?? [];
	},

	get: async (id: string): Promise<Policy> =>
		fetchApi<Policy>(`/policies/${id}`),

	create: async (data: CreatePolicyRequest): Promise<Policy> =>
		fetchApi<Policy>('/policies', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (id: string, data: UpdatePolicyRequest): Promise<Policy> =>
		fetchApi<Policy>(`/policies/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/policies/${id}`, {
			method: 'DELETE',
		}),

	listSchedules: async (id: string): Promise<Schedule[]> => {
		const response = await fetchApi<SchedulesResponse>(
			`/policies/${id}/schedules`,
		);
		return response.schedules ?? [];
	},

	apply: async (
		id: string,
		data: ApplyPolicyRequest,
	): Promise<ApplyPolicyResponse> =>
		fetchApi<ApplyPolicyResponse>(`/policies/${id}/apply`, {
			method: 'POST',
			body: JSON.stringify(data),
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

// Backup Scripts API
export const backupScriptsApi = {
	list: async (scheduleId: string): Promise<BackupScript[]> => {
		const response = await fetchApi<BackupScriptsResponse>(
			`/schedules/${scheduleId}/scripts`,
		);
		return response.scripts ?? [];
	},

	get: async (scheduleId: string, id: string): Promise<BackupScript> =>
		fetchApi<BackupScript>(`/schedules/${scheduleId}/scripts/${id}`),

	create: async (
		scheduleId: string,
		data: CreateBackupScriptRequest,
	): Promise<BackupScript> =>
		fetchApi<BackupScript>(`/schedules/${scheduleId}/scripts`, {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		scheduleId: string,
		id: string,
		data: UpdateBackupScriptRequest,
	): Promise<BackupScript> =>
		fetchApi<BackupScript>(`/schedules/${scheduleId}/scripts/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (scheduleId: string, id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/schedules/${scheduleId}/scripts/${id}`, {
			method: 'DELETE',
		}),
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

	compare: async (id1: string, id2: string): Promise<SnapshotCompareResponse> =>
		fetchApi<SnapshotCompareResponse>(`/snapshots/${id1}/compare/${id2}`),
};

// Snapshot Comments API
export const snapshotCommentsApi = {
	list: async (snapshotId: string): Promise<SnapshotComment[]> => {
		const response = await fetchApi<SnapshotCommentsResponse>(
			`/snapshots/${snapshotId}/comments`,
		);
		return response.comments ?? [];
	},

	create: async (
		snapshotId: string,
		data: CreateSnapshotCommentRequest,
	): Promise<SnapshotComment> =>
		fetchApi<SnapshotComment>(`/snapshots/${snapshotId}/comments`, {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	delete: async (commentId: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/comments/${commentId}`, {
			method: 'DELETE',
		}),
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

// File History API
export const fileHistoryApi = {
	getHistory: async (
		params: FileHistoryParams,
	): Promise<FileHistoryResponse> => {
		const searchParams = new URLSearchParams();
		searchParams.set('path', params.path);
		searchParams.set('agent_id', params.agent_id);
		searchParams.set('repository_id', params.repository_id);
		return fetchApi<FileHistoryResponse>(
			`/files/history?${searchParams.toString()}`,
		);
	},
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

// SSO Group Mappings API
export const ssoGroupMappingsApi = {
	list: async (orgId: string): Promise<SSOGroupMapping[]> => {
		const response = await fetchApi<SSOGroupMappingsResponse>(
			`/organizations/${orgId}/sso-group-mappings`,
		);
		return response.mappings ?? [];
	},

	get: async (orgId: string, id: string): Promise<SSOGroupMapping> => {
		const response = await fetchApi<SSOGroupMappingResponse>(
			`/organizations/${orgId}/sso-group-mappings/${id}`,
		);
		return response.mapping;
	},

	create: async (
		orgId: string,
		data: CreateSSOGroupMappingRequest,
	): Promise<SSOGroupMapping> => {
		const response = await fetchApi<SSOGroupMappingResponse>(
			`/organizations/${orgId}/sso-group-mappings`,
			{
				method: 'POST',
				body: JSON.stringify(data),
			},
		);
		return response.mapping;
	},

	update: async (
		orgId: string,
		id: string,
		data: UpdateSSOGroupMappingRequest,
	): Promise<SSOGroupMapping> => {
		const response = await fetchApi<SSOGroupMappingResponse>(
			`/organizations/${orgId}/sso-group-mappings/${id}`,
			{
				method: 'PUT',
				body: JSON.stringify(data),
			},
		);
		return response.mapping;
	},

	delete: async (orgId: string, id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(
			`/organizations/${orgId}/sso-group-mappings/${id}`,
			{
				method: 'DELETE',
			},
		),

	// SSO Settings
	getSettings: async (orgId: string): Promise<SSOSettings> =>
		fetchApi<SSOSettings>(`/organizations/${orgId}/sso-settings`),

	updateSettings: async (
		orgId: string,
		data: UpdateSSOSettingsRequest,
	): Promise<SSOSettings> =>
		fetchApi<SSOSettings>(`/organizations/${orgId}/sso-settings`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	// User SSO Groups
	getUserSSOGroups: async (userId: string): Promise<UserSSOGroups> =>
		fetchApi<UserSSOGroups>(`/users/${userId}/sso-groups`),
};

// Maintenance Windows API
export const maintenanceApi = {
	list: async (): Promise<MaintenanceWindow[]> => {
		const response = await fetchApi<MaintenanceWindowsResponse>(
			'/maintenance-windows',
		);
		return response.maintenance_windows ?? [];
	},

	get: async (id: string): Promise<MaintenanceWindow> =>
		fetchApi<MaintenanceWindow>(`/maintenance-windows/${id}`),

	create: async (
		data: CreateMaintenanceWindowRequest,
	): Promise<MaintenanceWindow> =>
		fetchApi<MaintenanceWindow>('/maintenance-windows', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateMaintenanceWindowRequest,
	): Promise<MaintenanceWindow> =>
		fetchApi<MaintenanceWindow>(`/maintenance-windows/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),
	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/maintenance-windows/${id}`, {
			method: 'DELETE',
		}),

	getActive: async (): Promise<ActiveMaintenanceResponse> =>
		fetchApi<ActiveMaintenanceResponse>('/maintenance/active'),
};

// Exclude Patterns API
export const excludePatternsApi = {
	list: async (category?: string): Promise<ExcludePattern[]> => {
		const endpoint = category
			? `/exclude-patterns?category=${category}`
			: '/exclude-patterns';
		const response = await fetchApi<ExcludePatternsResponse>(endpoint);
		return response.patterns ?? [];
	},

	get: async (id: string): Promise<ExcludePattern> =>
		fetchApi<ExcludePattern>(`/exclude-patterns/${id}`),

	getLibrary: async (): Promise<BuiltInPattern[]> => {
		const response = await fetchApi<BuiltInPatternsResponse>(
			'/exclude-patterns/library',
		);
		return response.patterns ?? [];
	},

	getCategories: async (): Promise<CategoryInfo[]> => {
		const response = await fetchApi<CategoriesResponse>(
			'/exclude-patterns/categories',
		);
		return response.categories ?? [];
	},

	create: async (data: CreateExcludePatternRequest): Promise<ExcludePattern> =>
		fetchApi<ExcludePattern>('/exclude-patterns', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateExcludePatternRequest,
	): Promise<ExcludePattern> =>
		fetchApi<ExcludePattern>(`/exclude-patterns/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/exclude-patterns/${id}`, {
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

// Tags API
export const tagsApi = {
	list: async (): Promise<Tag[]> => {
		const response = await fetchApi<TagsResponse>('/tags');
		return response.tags ?? [];
	},

	get: async (id: string): Promise<Tag> => fetchApi<Tag>(`/tags/${id}`),

	create: async (data: CreateTagRequest): Promise<Tag> =>
		fetchApi<Tag>('/tags', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (id: string, data: UpdateTagRequest): Promise<Tag> =>
		fetchApi<Tag>(`/tags/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/tags/${id}`, {
			method: 'DELETE',
		}),

	// Backup tags
	getBackupTags: async (backupId: string): Promise<Tag[]> => {
		const response = await fetchApi<TagsResponse>(`/backups/${backupId}/tags`);
		return response.tags ?? [];
	},

	setBackupTags: async (
		backupId: string,
		data: AssignTagsRequest,
	): Promise<Tag[]> => {
		const response = await fetchApi<TagsResponse>(`/backups/${backupId}/tags`, {
			method: 'POST',
			body: JSON.stringify(data),
		});
		return response.tags ?? [];
	},
};

// Search API
export const searchApi = {
	search: async (filter: SearchFilter): Promise<SearchResponse> => {
		const searchParams = new URLSearchParams();
		searchParams.set('q', filter.q);

		if (filter.types?.length) {
			searchParams.set('types', filter.types.join(','));
		}
		if (filter.status) {
			searchParams.set('status', filter.status);
		}
		if (filter.tag_ids?.length) {
			searchParams.set('tag_ids', filter.tag_ids.join(','));
		}
		if (filter.date_from) {
			searchParams.set('date_from', filter.date_from);
		}
		if (filter.date_to) {
			searchParams.set('date_to', filter.date_to);
		}
		if (filter.size_min !== undefined) {
			searchParams.set('size_min', filter.size_min.toString());
		}
		if (filter.size_max !== undefined) {
			searchParams.set('size_max', filter.size_max.toString());
		}
		if (filter.limit) {
			searchParams.set('limit', filter.limit.toString());
		}

		return fetchApi<SearchResponse>(`/search?${searchParams.toString()}`);
	},
};

// Dashboard Metrics API
export const metricsApi = {
	getDashboardStats: async (): Promise<DashboardStats> =>
		fetchApi<DashboardStats>('/dashboard-metrics/stats'),

	getBackupSuccessRates: async (): Promise<{
		rate_7d: BackupSuccessRate;
		rate_30d: BackupSuccessRate;
	}> =>
		fetchApi<BackupSuccessRatesResponse>('/dashboard-metrics/success-rates'),

	getStorageGrowthTrend: async (days = 30): Promise<StorageGrowthTrend[]> => {
		const response = await fetchApi<StorageGrowthTrendResponse>(
			`/dashboard-metrics/storage-growth?days=${days}`,
		);
		return response.trend ?? [];
	},

	getBackupDurationTrend: async (days = 30): Promise<BackupDurationTrend[]> => {
		const response = await fetchApi<BackupDurationTrendResponse>(
			`/dashboard-metrics/backup-duration?days=${days}`,
		);
		return response.trend ?? [];
	},

	getDailyBackupStats: async (days = 30): Promise<DailyBackupStats[]> => {
		const response = await fetchApi<DailyBackupStatsResponse>(
			`/dashboard-metrics/daily-backups?days=${days}`,
		);
		return response.stats ?? [];
	},
};

export const reportsApi = {
	// Schedules
	listSchedules: async (): Promise<ReportSchedule[]> => {
		const response =
			await fetchApi<ReportSchedulesResponse>('/reports/schedules');
		return response.schedules ?? [];
	},

	getSchedule: async (id: string): Promise<ReportSchedule> =>
		fetchApi<ReportSchedule>(`/reports/schedules/${id}`),

	createSchedule: async (
		data: CreateReportScheduleRequest,
	): Promise<ReportSchedule> =>
		fetchApi<ReportSchedule>('/reports/schedules', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	updateSchedule: async (
		id: string,
		data: UpdateReportScheduleRequest,
	): Promise<ReportSchedule> =>
		fetchApi<ReportSchedule>(`/reports/schedules/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	deleteSchedule: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/reports/schedules/${id}`, {
			method: 'DELETE',
		}),

	// Actions
	sendReport: async (
		id: string,
		preview = false,
	): Promise<ReportPreviewResponse | MessageResponse> =>
		fetchApi(`/reports/schedules/${id}/send`, {
			method: 'POST',
			body: JSON.stringify({ preview }),
		}),

	previewReport: async (
		frequency: ReportFrequency,
		timezone = 'UTC',
	): Promise<ReportPreviewResponse> =>
		fetchApi<ReportPreviewResponse>('/reports/preview', {
			method: 'POST',
			body: JSON.stringify({ frequency, timezone }),
		}),

	// History
	listHistory: async (): Promise<ReportHistory[]> => {
		const response = await fetchApi<ReportHistoryResponse>('/reports/history');
		return response.history ?? [];
	},

	getHistory: async (id: string): Promise<ReportHistory> =>
		fetchApi<ReportHistory>(`/reports/history/${id}`),
};
// Onboarding API
export const onboardingApi = {
	getStatus: async (): Promise<OnboardingStatus> =>
		fetchApi<OnboardingStatus>('/onboarding/status'),

	completeStep: async (step: OnboardingStep): Promise<OnboardingStatus> =>
		fetchApi<OnboardingStatus>(`/onboarding/step/${step}`, {
			method: 'POST',
		}),

	skip: async (): Promise<OnboardingStatus> =>
		fetchApi<OnboardingStatus>('/onboarding/skip', {
			method: 'POST',
		}),
};

// Cost Estimation API
export const costsApi = {
	getSummary: async (): Promise<CostSummary> =>
		fetchApi<CostSummary>('/costs/summary'),

	listRepositoryCosts: async (): Promise<RepositoryCostsResponse> =>
		fetchApi<RepositoryCostsResponse>('/costs/repositories'),

	getRepositoryCost: async (id: string): Promise<RepositoryCostResponse> =>
		fetchApi<RepositoryCostResponse>(`/costs/repositories/${id}`),

	getForecast: async (days = 30): Promise<CostForecastResponse> =>
		fetchApi<CostForecastResponse>(`/costs/forecast?days=${days}`),

	getHistory: async (days = 30): Promise<CostHistoryResponse> =>
		fetchApi<CostHistoryResponse>(`/costs/history?days=${days}`),
};

// Pricing API
export const pricingApi = {
	list: async (): Promise<StoragePricing[]> => {
		const response = await fetchApi<StoragePricingResponse>('/pricing');
		return response.pricing ?? [];
	},

	getDefaults: async (): Promise<DefaultPricingResponse> =>
		fetchApi<DefaultPricingResponse>('/pricing/defaults'),

	create: async (data: CreateStoragePricingRequest): Promise<StoragePricing> =>
		fetchApi<StoragePricing>('/pricing', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateStoragePricingRequest,
	): Promise<StoragePricing> =>
		fetchApi<StoragePricing>(`/pricing/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/pricing/${id}`, {
			method: 'DELETE',
		}),
};

// Cost Alerts API
export const costAlertsApi = {
	list: async (): Promise<CostAlert[]> => {
		const response = await fetchApi<CostAlertsResponse>('/cost-alerts');
		return response.alerts ?? [];
	},

	get: async (id: string): Promise<CostAlert> =>
		fetchApi<CostAlert>(`/cost-alerts/${id}`),

	create: async (data: CreateCostAlertRequest): Promise<CostAlert> =>
		fetchApi<CostAlert>('/cost-alerts', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateCostAlertRequest,
	): Promise<CostAlert> =>
		fetchApi<CostAlert>(`/cost-alerts/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/cost-alerts/${id}`, {
			method: 'DELETE',
		}),
};

// Legal Holds API
export const legalHoldsApi = {
	list: async (): Promise<LegalHold[]> => {
		const response = await fetchApi<LegalHoldsResponse>('/legal-holds');
		return response.legal_holds ?? [];
	},

	get: async (snapshotId: string): Promise<LegalHold> =>
		fetchApi<LegalHold>(`/snapshots/${snapshotId}/hold`),

	create: async (
		snapshotId: string,
		data: CreateLegalHoldRequest,
	): Promise<LegalHold> =>
		fetchApi<LegalHold>(`/snapshots/${snapshotId}/hold`, {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	delete: async (snapshotId: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/snapshots/${snapshotId}/hold`, {
			method: 'DELETE',
		}),
};
