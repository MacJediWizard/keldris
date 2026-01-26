import type {
	ActiveMaintenanceResponse,
	AddAgentToGroupRequest,
	Agent,
	AgentBackupsResponse,
	AgentCommand,
	AgentCommandsResponse,
	AgentGroup,
	AgentGroupsResponse,
	AgentHealthHistoryResponse,
	AgentLogFilter,
	AgentLogsResponse,
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
	Announcement,
	AnnouncementsResponse,
	ApplyPolicyRequest,
	ApplyPolicyResponse,
	AssignTagsRequest,
	AuditLog,
	AuditLogFilter,
	AuditLogsResponse,
	Backup,
	BackupCalendarResponse,
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
	ChangelogEntry,
	ChangelogResponse,
	ClassificationLevelsResponse,
	ClassificationRulesResponse,
	ClassificationSummary,
	ComplianceReport,
	ConfigTemplate,
	ConfigTemplatesResponse,
	CostAlert,
	CostAlertsResponse,
	CostForecastResponse,
	CostHistoryResponse,
	CostSummary,
	CreateAgentCommandRequest,
	CreateAgentGroupRequest,
	CreateAgentRequest,
	CreateAgentResponse,
	CreateAlertRuleRequest,
	CreateAnnouncementRequest,
	CreateBackupScriptRequest,
	CreateCostAlertRequest,
	CreateDRRunbookRequest,
	CreateDRTestScheduleRequest,
	CreateExcludePatternRequest,
	CreateImmutabilityLockRequest,
	CreateLegalHoldRequest,
	CreateMaintenanceWindowRequest,
	CreateNotificationChannelRequest,
	CreateNotificationPreferenceRequest,
	CreateOrgRequest,
	CreatePathClassificationRuleRequest,
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
	CreateTemplateRequest,
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
	DataTypesResponse,
	DefaultPricingResponse,
	DryRunResponse,
	ErrorResponse,
	ExcludePattern,
	ExcludePatternsResponse,
	ExportBundleRequest,
	ExportFormat,
	ExtendImmutabilityLockRequest,
	FileDiffResponse,
	FileHistoryParams,
	FileHistoryResponse,
	FileSearchParams,
	FileSearchResponse,
	FleetHealthSummary,
	GeoRegion,
	GeoReplicationConfig,
	GeoReplicationConfigsResponse,
	GeoReplicationCreateRequest,
	GeoReplicationEvent,
	GeoReplicationEventsResponse,
	GeoReplicationRegionsResponse,
	GeoReplicationSummary,
	GeoReplicationSummaryResponse,
	GeoReplicationUpdateRequest,
	ImmutabilityLock,
	ImmutabilityLocksResponse,
	ImmutabilityStatus,
	ImportConfigRequest,
	ImportPreviewRequest,
	ImportPreviewResponse,
	ImportRepositoryRequest,
	ImportRepositoryResponse,
	ImportResult,
	InvitationsResponse,
	InviteMemberRequest,
	InviteResponse,
	KeyRecoveryResponse,
	LegalHold,
	LegalHoldsResponse,
	MaintenanceWindow,
	MaintenanceWindowsResponse,
	MembersResponse,
	MessageResponse,
	MountSnapshotRequest,
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
	PathClassificationRule,
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
	RepositoryImmutabilitySettings,
	RepositoryReplicationStatusResponse,
	RepositoryStatsListItem,
	RepositoryStatsListResponse,
	RepositoryStatsResponse,
	Restore,
	RestorePreview,
	RestorePreviewRequest,
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
	ServerLogComponentsResponse,
	ServerLogFilter,
	ServerLogsResponse,
	SetDebugModeRequest,
	SetDebugModeResponse,
	SetScheduleClassificationRequest,
	Snapshot,
	SnapshotComment,
	SnapshotCommentsResponse,
	SnapshotCompareResponse,
	SnapshotFilesResponse,
	SnapshotMount,
	SnapshotMountsResponse,
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
	UpdateAnnouncementRequest,
	UpdateBackupScriptRequest,
	UpdateCostAlertRequest,
	UpdateDRRunbookRequest,
	UpdateExcludePatternRequest,
	UpdateMaintenanceWindowRequest,
	UpdateMemberRequest,
	UpdateNotificationChannelRequest,
	UpdateNotificationPreferenceRequest,
	UpdateOrgRequest,
	UpdatePathClassificationRuleRequest,
	UpdatePolicyRequest,
	UpdateReportScheduleRequest,
	UpdateRepositoryImmutabilitySettingsRequest,
	UpdateRepositoryRequest,
	UpdateSSOGroupMappingRequest,
	UpdateSSOSettingsRequest,
	UpdateScheduleRequest,
	UpdateStoragePricingRequest,
	UpdateTagRequest,
	UpdateTemplateRequest,
	UpdateUserPreferencesRequest,
	UpdateVerificationScheduleRequest,
	UseTemplateRequest,
	User,
	UserSSOGroups,
	ValidateImportRequest,
	ValidationResult,
	Verification,
	VerificationSchedule,
	VerificationSchedulesResponse,
	VerificationStatusResponse,
	VerificationsResponse,
	VerifyImportAccessRequest,
	VerifyImportAccessResponse,
	SLAWithAssignments,
	SLADefinitionsResponse,
	SLAAssignment,
	SLAAssignmentsResponse,
	SLACompliance,
	SLAComplianceResponse,
	SLABreach,
	SLABreachesResponse,
	SLADashboardStats,
	SLADashboardResponse,
	SLAReport,
	SLAReportResponse,
	CreateSLADefinitionRequest,
	UpdateSLADefinitionRequest,
	AssignSLARequest,
	AcknowledgeBreachRequest,
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

	setDebugMode: async (
		id: string,
		data: SetDebugModeRequest,
	): Promise<SetDebugModeResponse> =>
		fetchApi<SetDebugModeResponse>(`/agents/${id}/debug`, {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	listWithGroups: async (): Promise<AgentWithGroups[]> => {
		const response = await fetchApi<AgentsWithGroupsResponse>(
			'/agents/with-groups',
		);
		return response.agents ?? [];
	},

	getLogs: async (
		id: string,
		filter?: AgentLogFilter,
	): Promise<AgentLogsResponse> => {
		const params = new URLSearchParams();
		if (filter?.level) params.set('level', filter.level);
		if (filter?.component) params.set('component', filter.component);
		if (filter?.search) params.set('search', filter.search);
		if (filter?.limit) params.set('limit', filter.limit.toString());
		if (filter?.offset) params.set('offset', filter.offset.toString());
		const queryString = params.toString();
		return fetchApi<AgentLogsResponse>(
			`/agents/${id}/logs${queryString ? `?${queryString}` : ''}`,
		);
	},

	// Command methods
	getCommands: async (
		agentId: string,
		limit = 50,
	): Promise<AgentCommandsResponse> =>
		fetchApi<AgentCommandsResponse>(
			`/agents/${agentId}/commands?limit=${limit}`,
		),

	createCommand: async (
		agentId: string,
		data: CreateAgentCommandRequest,
	): Promise<AgentCommand> =>
		fetchApi<AgentCommand>(`/agents/${agentId}/commands`, {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	getCommand: async (
		agentId: string,
		commandId: string,
	): Promise<AgentCommand> =>
		fetchApi<AgentCommand>(`/agents/${agentId}/commands/${commandId}`),

	cancelCommand: async (
		agentId: string,
		commandId: string,
	): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/agents/${agentId}/commands/${commandId}`, {
			method: 'DELETE',
		}),
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

	dryRun: async (id: string): Promise<DryRunResponse> =>
		fetchApi<DryRunResponse>(`/schedules/${id}/dry-run`, {
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

	getCalendar: async (month: string): Promise<BackupCalendarResponse> =>
		fetchApi<BackupCalendarResponse>(`/backups/calendar?month=${month}`),
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

	diffFile: async (
		id1: string,
		id2: string,
		path: string,
	): Promise<FileDiffResponse> =>
		fetchApi<FileDiffResponse>(
			`/snapshots/${id1}/files/diff/${id2}?path=${encodeURIComponent(path)}`,
		),

	mount: async (
		snapshotId: string,
		data: MountSnapshotRequest,
	): Promise<SnapshotMount> =>
		fetchApi<SnapshotMount>(`/snapshots/${snapshotId}/mount`, {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	unmount: async (
		snapshotId: string,
		agentId: string,
	): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(
			`/snapshots/${snapshotId}/mount?agent_id=${agentId}`,
			{
				method: 'DELETE',
			},
		),

	getMount: async (
		snapshotId: string,
		agentId: string,
	): Promise<SnapshotMount> =>
		fetchApi<SnapshotMount>(
			`/snapshots/${snapshotId}/mount?agent_id=${agentId}`,
		),
};

// Snapshot Mounts API
export const snapshotMountsApi = {
	list: async (agentId?: string): Promise<SnapshotMount[]> => {
		const endpoint = agentId ? `/mounts?agent_id=${agentId}` : '/mounts';
		const response = await fetchApi<SnapshotMountsResponse>(endpoint);
		return response.mounts ?? [];
	},
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

	preview: async (data: RestorePreviewRequest): Promise<RestorePreview> =>
		fetchApi<RestorePreview>('/restores/preview', {
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

// File Search API
export const fileSearchApi = {
	search: async (params: FileSearchParams): Promise<FileSearchResponse> => {
		const searchParams = new URLSearchParams();
		searchParams.set('q', params.q);
		searchParams.set('agent_id', params.agent_id);
		searchParams.set('repository_id', params.repository_id);

		if (params.path) {
			searchParams.set('path', params.path);
		}
		if (params.snapshot_ids) {
			searchParams.set('snapshot_ids', params.snapshot_ids);
		}
		if (params.date_from) {
			searchParams.set('date_from', params.date_from);
		}
		if (params.date_to) {
			searchParams.set('date_to', params.date_to);
		}
		if (params.size_min !== undefined) {
			searchParams.set('size_min', params.size_min.toString());
		}
		if (params.size_max !== undefined) {
			searchParams.set('size_max', params.size_max.toString());
		}
		if (params.limit !== undefined) {
			searchParams.set('limit', params.limit.toString());
		}

		return fetchApi<FileSearchResponse>(
			`/search/files?${searchParams.toString()}`,
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

// Announcements API
export const announcementsApi = {
	list: async (): Promise<Announcement[]> => {
		const response = await fetchApi<AnnouncementsResponse>('/announcements');
		return response.announcements ?? [];
	},

	getActive: async (): Promise<Announcement[]> => {
		const response = await fetchApi<AnnouncementsResponse>(
			'/announcements/active',
		);
		return response.announcements ?? [];
	},

	get: async (id: string): Promise<Announcement> =>
		fetchApi<Announcement>(`/announcements/${id}`),

	create: async (data: CreateAnnouncementRequest): Promise<Announcement> =>
		fetchApi<Announcement>('/announcements', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateAnnouncementRequest,
	): Promise<Announcement> =>
		fetchApi<Announcement>(`/announcements/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/announcements/${id}`, {
			method: 'DELETE',
		}),

	dismiss: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/announcements/${id}/dismiss`, {
			method: 'POST',
		}),
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

	emergencyOverride: async (
		id: string,
		override: boolean,
	): Promise<MaintenanceWindow> =>
		fetchApi<MaintenanceWindow>(
			`/maintenance-windows/${id}/emergency-override`,
			{
				method: 'POST',
				body: JSON.stringify({ override }),
			},
		),
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

// Support API
export const supportApi = {
	generateBundle: async (): Promise<Blob> => {
		const response = await fetch(`${API_BASE}/support/bundle`, {
			method: 'POST',
			credentials: 'include',
		});
		if (!response.ok) {
			throw new ApiError(response.status, 'Failed to generate support bundle');
		}
		return response.blob();
	},
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

// Changelog API
export const changelogApi = {
	list: async (): Promise<ChangelogResponse> =>
		fetchApi<ChangelogResponse>('/changelog'),

	get: async (version: string): Promise<ChangelogEntry> =>
		fetchApi<ChangelogEntry>(`/changelog/${version}`),
};

// Server Logs API (Admin only)
export const serverLogsApi = {
	list: async (filter?: ServerLogFilter): Promise<ServerLogsResponse> => {
		const searchParams = new URLSearchParams();
		if (filter?.level) searchParams.set('level', filter.level);
		if (filter?.component) searchParams.set('component', filter.component);
		if (filter?.search) searchParams.set('search', filter.search);
		if (filter?.start_time) searchParams.set('start_time', filter.start_time);
		if (filter?.end_time) searchParams.set('end_time', filter.end_time);
		if (filter?.limit) searchParams.set('limit', filter.limit.toString());
		if (filter?.offset) searchParams.set('offset', filter.offset.toString());

		const query = searchParams.toString();
		const endpoint = query ? `/admin/logs?${query}` : '/admin/logs';
		return fetchApi<ServerLogsResponse>(endpoint);
	},

	getComponents: async (): Promise<string[]> => {
		const response = await fetchApi<ServerLogComponentsResponse>(
			'/admin/logs/components',
		);
		return response.components ?? [];
	},

	exportCsv: async (filter?: ServerLogFilter): Promise<Blob> => {
		const searchParams = new URLSearchParams();
		if (filter?.level) searchParams.set('level', filter.level);
		if (filter?.component) searchParams.set('component', filter.component);
		if (filter?.search) searchParams.set('search', filter.search);
		if (filter?.start_time) searchParams.set('start_time', filter.start_time);
		if (filter?.end_time) searchParams.set('end_time', filter.end_time);

		const query = searchParams.toString();
		const endpoint = query
			? `/admin/logs/export/csv?${query}`
			: '/admin/logs/export/csv';
		const response = await fetch(`${API_BASE}${endpoint}`, {
			credentials: 'include',
		});
		if (!response.ok) {
			throw new ApiError(response.status, 'Failed to export server logs');
		}
		return response.blob();
	},

	exportJson: async (filter?: ServerLogFilter): Promise<Blob> => {
		const searchParams = new URLSearchParams();
		if (filter?.level) searchParams.set('level', filter.level);
		if (filter?.component) searchParams.set('component', filter.component);
		if (filter?.search) searchParams.set('search', filter.search);
		if (filter?.start_time) searchParams.set('start_time', filter.start_time);
		if (filter?.end_time) searchParams.set('end_time', filter.end_time);

		const query = searchParams.toString();
		const endpoint = query
			? `/admin/logs/export/json?${query}`
			: '/admin/logs/export/json';
		const response = await fetch(`${API_BASE}${endpoint}`, {
			credentials: 'include',
		});
		if (!response.ok) {
			throw new ApiError(response.status, 'Failed to export server logs');
		}
		return response.blob();
	},

	clear: async (): Promise<MessageResponse> =>
		fetchApi<MessageResponse>('/admin/logs', {
			method: 'DELETE',
		}),
};

// Classification API
export const classificationsApi = {
	// Reference data
	getLevels: async (): Promise<ClassificationLevelsResponse> =>
		fetchApi<ClassificationLevelsResponse>('/classifications/levels'),

	getDataTypes: async (): Promise<DataTypesResponse> =>
		fetchApi<DataTypesResponse>('/classifications/data-types'),

	getDefaultRules: async (): Promise<PathClassificationRule[]> => {
		const response = await fetchApi<{ rules: PathClassificationRule[] }>(
			'/classifications/default-rules',
		);
		return response.rules ?? [];
	},

	// Rules
	listRules: async (): Promise<PathClassificationRule[]> => {
		const response = await fetchApi<ClassificationRulesResponse>(
			'/classifications/rules',
		);
		return response.rules ?? [];
	},

	getRule: async (id: string): Promise<PathClassificationRule> =>
		fetchApi<PathClassificationRule>(`/classifications/rules/${id}`),

	createRule: async (
		data: CreatePathClassificationRuleRequest,
	): Promise<PathClassificationRule> =>
		fetchApi<PathClassificationRule>('/classifications/rules', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	updateRule: async (
		id: string,
		data: UpdatePathClassificationRuleRequest,
	): Promise<PathClassificationRule> =>
		fetchApi<PathClassificationRule>(`/classifications/rules/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	deleteRule: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/classifications/rules/${id}`, {
			method: 'DELETE',
		}),

	// Schedule classifications
	listScheduleClassifications: async (level?: string): Promise<Schedule[]> => {
		const url = level
			? `/classifications/schedules?level=${level}`
			: '/classifications/schedules';
		const response = await fetchApi<{ schedules: Schedule[] }>(url);
		return response.schedules ?? [];
	},

	getScheduleClassification: async (
		scheduleId: string,
	): Promise<{ schedule_id: string; level: string; data_types: string[] }> =>
		fetchApi<{ schedule_id: string; level: string; data_types: string[] }>(
			`/classifications/schedules/${scheduleId}`,
		),

	setScheduleClassification: async (
		scheduleId: string,
		data: SetScheduleClassificationRequest,
	): Promise<{ schedule_id: string; level: string; data_types: string[] }> =>
		fetchApi<{ schedule_id: string; level: string; data_types: string[] }>(
			`/classifications/schedules/${scheduleId}`,
			{
				method: 'PUT',
				body: JSON.stringify(data),
			},
		),

	autoClassifySchedule: async (
		scheduleId: string,
	): Promise<{
		schedule_id: string;
		level: string;
		data_types: string[];
		auto_classified: boolean;
	}> =>
		fetchApi<{
			schedule_id: string;
			level: string;
			data_types: string[];
			auto_classified: boolean;
		}>(`/classifications/schedules/${scheduleId}/auto-classify`, {
			method: 'POST',
		}),

	// Backup classifications
	listBackupsByClassification: async (level: string): Promise<Backup[]> => {
		const response = await fetchApi<{ backups: Backup[]; level: string }>(
			`/classifications/backups?level=${level}`,
		);
		return response.backups ?? [];
	},

	// Summary and reports
	getSummary: async (): Promise<ClassificationSummary> =>
		fetchApi<ClassificationSummary>('/classifications/summary'),

	getComplianceReport: async (): Promise<ComplianceReport> =>
		fetchApi<ComplianceReport>('/classifications/compliance-report'),
};

// Immutability API
export const immutabilityApi = {
	listLocks: async (): Promise<ImmutabilityLock[]> => {
		const response = await fetchApi<ImmutabilityLocksResponse>('/immutability');
		return response.locks ?? [];
	},

	getLock: async (id: string): Promise<ImmutabilityLock> =>
		fetchApi<ImmutabilityLock>(`/immutability/${id}`),

	createLock: async (
		data: CreateImmutabilityLockRequest,
	): Promise<ImmutabilityLock> =>
		fetchApi<ImmutabilityLock>('/immutability', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	extendLock: async (
		id: string,
		data: ExtendImmutabilityLockRequest,
	): Promise<ImmutabilityLock> =>
		fetchApi<ImmutabilityLock>(`/immutability/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	getSnapshotStatus: async (
		snapshotId: string,
		repositoryId: string,
	): Promise<ImmutabilityStatus> =>
		fetchApi<ImmutabilityStatus>(
			`/snapshots/${snapshotId}/immutability?repository_id=${repositoryId}`,
		),

	getRepositorySettings: async (
		repositoryId: string,
	): Promise<RepositoryImmutabilitySettings> =>
		fetchApi<RepositoryImmutabilitySettings>(
			`/repositories/${repositoryId}/immutability`,
		),

	updateRepositorySettings: async (
		repositoryId: string,
		data: UpdateRepositoryImmutabilitySettingsRequest,
	): Promise<RepositoryImmutabilitySettings> =>
		fetchApi<RepositoryImmutabilitySettings>(
			`/repositories/${repositoryId}/immutability`,
			{
				method: 'PUT',
				body: JSON.stringify(data),
			},
		),

	listRepositoryLocks: async (
		repositoryId: string,
	): Promise<ImmutabilityLock[]> => {
		const response = await fetchApi<ImmutabilityLocksResponse>(
			`/repositories/${repositoryId}/immutability/locks`,
		);
		return response.locks ?? [];
	},
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

// Geo-Replication API
export const geoReplicationApi = {
	listRegions: async (): Promise<{
		regions: GeoRegion[];
		pairs: { primary: GeoRegion; secondary: GeoRegion }[];
	}> => fetchApi<GeoReplicationRegionsResponse>('/geo-replication/regions'),

	listConfigs: async (): Promise<GeoReplicationConfig[]> => {
		const response = await fetchApi<GeoReplicationConfigsResponse>(
			'/geo-replication/configs',
		);
		return response.configs ?? [];
	},

	getConfig: async (id: string): Promise<GeoReplicationConfig> =>
		fetchApi<GeoReplicationConfig>(`/geo-replication/configs/${id}`),

	createConfig: async (
		data: GeoReplicationCreateRequest,
	): Promise<GeoReplicationConfig> =>
		fetchApi<GeoReplicationConfig>('/geo-replication/configs', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	updateConfig: async (
		id: string,
		data: GeoReplicationUpdateRequest,
	): Promise<GeoReplicationConfig> =>
		fetchApi<GeoReplicationConfig>(`/geo-replication/configs/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	deleteConfig: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/geo-replication/configs/${id}`, {
			method: 'DELETE',
		}),

	triggerReplication: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/geo-replication/configs/${id}/trigger`, {
			method: 'POST',
		}),

	getEvents: async (id: string): Promise<GeoReplicationEvent[]> => {
		const response = await fetchApi<GeoReplicationEventsResponse>(
			`/geo-replication/configs/${id}/events`,
		);
		return response.events ?? [];
	},

	getRepositoryStatus: async (
		repoId: string,
	): Promise<RepositoryReplicationStatusResponse> =>
		fetchApi<RepositoryReplicationStatusResponse>(
			`/geo-replication/repositories/${repoId}/status`,
		),

	setRepositoryRegion: async (
		repoId: string,
		region: string,
	): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(
			`/geo-replication/repositories/${repoId}/region`,
			{
				method: 'PUT',
				body: JSON.stringify({ region }),
			},
		),

	getSummary: async (): Promise<{
		summary: GeoReplicationSummary;
		regions: GeoRegion[];
	}> => fetchApi<GeoReplicationSummaryResponse>('/geo-replication/summary'),
};

// Config Export/Import API
export const configExportApi = {
	// Export endpoints
	exportAgent: async (
		id: string,
		format: ExportFormat = 'json',
	): Promise<string> => {
		const response = await fetch(
			`${API_BASE}/export/agents/${id}?format=${format}`,
			{
				credentials: 'include',
			},
		);
		if (!response.ok) {
			const errorData = await response
				.json()
				.catch(() => ({ error: 'Export failed' }));
			throw new ApiError(response.status, errorData.error);
		}
		return response.text();
	},

	exportSchedule: async (
		id: string,
		format: ExportFormat = 'json',
	): Promise<string> => {
		const response = await fetch(
			`${API_BASE}/export/schedules/${id}?format=${format}`,
			{
				credentials: 'include',
			},
		);
		if (!response.ok) {
			const errorData = await response
				.json()
				.catch(() => ({ error: 'Export failed' }));
			throw new ApiError(response.status, errorData.error);
		}
		return response.text();
	},

	exportRepository: async (
		id: string,
		format: ExportFormat = 'json',
	): Promise<string> => {
		const response = await fetch(
			`${API_BASE}/export/repositories/${id}?format=${format}`,
			{
				credentials: 'include',
			},
		);
		if (!response.ok) {
			const errorData = await response
				.json()
				.catch(() => ({ error: 'Export failed' }));
			throw new ApiError(response.status, errorData.error);
		}
		return response.text();
	},

	exportBundle: async (data: ExportBundleRequest): Promise<string> => {
		const response = await fetch(`${API_BASE}/export/bundle`, {
			method: 'POST',
			credentials: 'include',
			headers: {
				'Content-Type': 'application/json',
			},
			body: JSON.stringify(data),
		});
		if (!response.ok) {
			const errorData = await response
				.json()
				.catch(() => ({ error: 'Export failed' }));
			throw new ApiError(response.status, errorData.error);
		}
		return response.text();
	},

	// Import endpoints
	importConfig: async (data: ImportConfigRequest): Promise<ImportResult> =>
		fetchApi<ImportResult>('/import', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	validateImport: async (
		data: ValidateImportRequest,
	): Promise<ValidationResult> =>
		fetchApi<ValidationResult>('/import/validate', {
			method: 'POST',
			body: JSON.stringify(data),
		}),
};

// Config Templates API
export const templatesApi = {
	list: async (): Promise<ConfigTemplate[]> => {
		const response = await fetchApi<ConfigTemplatesResponse>('/templates');
		return response.templates ?? [];
	},

	get: async (id: string): Promise<ConfigTemplate> =>
		fetchApi<ConfigTemplate>(`/templates/${id}`),

	create: async (data: CreateTemplateRequest): Promise<ConfigTemplate> =>
		fetchApi<ConfigTemplate>('/templates', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateTemplateRequest,
	): Promise<ConfigTemplate> =>
		fetchApi<ConfigTemplate>(`/templates/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/templates/${id}`, {
			method: 'DELETE',
		}),

	use: async (
		id: string,
		data: UseTemplateRequest = {},
	): Promise<ImportResult> =>
		fetchApi<ImportResult>(`/templates/${id}/use`, {
			method: 'POST',
			body: JSON.stringify(data),
		}),
};

// SLA API
export const slaApi = {
	// SLA Definitions
	list: async (): Promise<SLAWithAssignments[]> => {
		const response = await fetchApi<SLADefinitionsResponse>('/slas');
		return response.slas ?? [];
	},

	get: async (id: string): Promise<SLAWithAssignments> =>
		fetchApi<SLAWithAssignments>(`/slas/${id}`),

	create: async (data: CreateSLADefinitionRequest): Promise<SLAWithAssignments> =>
		fetchApi<SLAWithAssignments>('/slas', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateSLADefinitionRequest,
	): Promise<SLAWithAssignments> =>
		fetchApi<SLAWithAssignments>(`/slas/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/slas/${id}`, {
			method: 'DELETE',
		}),

	// SLA Assignments
	listAssignments: async (slaId: string): Promise<SLAAssignment[]> => {
		const response = await fetchApi<SLAAssignmentsResponse>(
			`/slas/${slaId}/assignments`,
		);
		return response.assignments ?? [];
	},

	createAssignment: async (
		slaId: string,
		data: AssignSLARequest,
	): Promise<SLAAssignment> =>
		fetchApi<SLAAssignment>(`/slas/${slaId}/assignments`, {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	deleteAssignment: async (
		slaId: string,
		assignmentId: string,
	): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/slas/${slaId}/assignments/${assignmentId}`, {
			method: 'DELETE',
		}),

	// SLA Compliance
	getCompliance: async (slaId: string): Promise<SLACompliance[]> => {
		const response = await fetchApi<SLAComplianceResponse>(
			`/slas/${slaId}/compliance`,
		);
		return response.compliance ?? [];
	},

	listOrgCompliance: async (): Promise<SLACompliance[]> => {
		const response = await fetchApi<SLAComplianceResponse>('/sla-compliance');
		return response.compliance ?? [];
	},

	// SLA Breaches
	listBreaches: async (): Promise<SLABreach[]> => {
		const response = await fetchApi<SLABreachesResponse>('/sla-breaches');
		return response.breaches ?? [];
	},

	listActiveBreaches: async (): Promise<SLABreach[]> => {
		const response = await fetchApi<SLABreachesResponse>('/sla-breaches/active');
		return response.breaches ?? [];
	},

	listBreachesBySLA: async (slaId: string): Promise<SLABreach[]> => {
		const response = await fetchApi<SLABreachesResponse>(
			`/slas/${slaId}/breaches`,
		);
		return response.breaches ?? [];
	},

	getBreach: async (id: string): Promise<SLABreach> =>
		fetchApi<SLABreach>(`/sla-breaches/${id}`),

	acknowledgeBreach: async (
		id: string,
		data?: AcknowledgeBreachRequest,
	): Promise<SLABreach> =>
		fetchApi<SLABreach>(`/sla-breaches/${id}/acknowledge`, {
			method: 'POST',
			body: JSON.stringify(data ?? {}),
		}),

	resolveBreach: async (id: string): Promise<SLABreach> =>
		fetchApi<SLABreach>(`/sla-breaches/${id}/resolve`, {
			method: 'POST',
		}),

	// SLA Dashboard
	getDashboard: async (): Promise<SLADashboardStats> => {
		const response = await fetchApi<SLADashboardResponse>('/sla-dashboard');
		return response.stats;
	},

	// SLA Report
	getReport: async (month?: string): Promise<SLAReport> => {
		const url = month ? `/sla-report?month=${month}` : '/sla-report';
		const response = await fetchApi<SLAReportResponse>(url);
		return response.report;
	},
};
