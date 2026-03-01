import type {
	AcknowledgeBreachRequest,
	ActivateLicenseRequest,
	ActivateLicenseResponse,
	ActiveMaintenanceResponse,
	ActivityCategoriesResponse,
	ActivityEvent,
	ActivityEventCountResponse,
	ActivityEventFilter,
	ActivityEventsResponse,
	AddAgentToGroupRequest,
	AdminCreateOrgRequest,
	AdminOrgSettings,
	AdminOrgUsageStats,
	AdminOrganization,
	AdminOrganizationsResponse,
	Agent,
	AgentBackupsResponse,
	AgentCommand,
	AgentCommandsResponse,
	AgentGroup,
	AgentGroupsResponse,
	AgentHealthHistoryResponse,
	AgentImportJobResult,
	AgentImportPreviewResponse,
	AgentImportResponse,
	AgentImportTemplateResponse,
	AgentLogFilter,
	AgentLogsResponse,
	AgentSchedulesResponse,
	AgentStatsResponse,
	AgentWithGroups,
	AgentsResponse,
	AgentsWithGroupsResponse,
	AirGapLicenseInfo,
	AirGapStatus,
	Alert,
	AlertCountResponse,
	AlertRule,
	AlertRulesResponse,
	AlertsResponse,
	Announcement,
	AnnouncementsResponse,
	ApplyBackupHookTemplateRequest,
	ApplyBackupHookTemplateResponse,
	ApplyPolicyRequest,
	ApplyPolicyResponse,
	AssignSLARequest,
	AssignTagsRequest,
	AuditLog,
	AuditLogFilter,
	AuditLogsResponse,
	Backup,
	BackupCalendarResponse,
	BackupDurationTrend,
	BackupDurationTrendResponse,
	BackupHookTemplate,
	BackupHookTemplatesResponse,
	BackupQueueResponse,
	BackupQueueSummary,
	BackupScript,
	BackupScriptsResponse,
	BackupSuccessRate,
	BackupSuccessRatesResponse,
	BackupsResponse,
	BlockedRequestsResponse,
	BrandingSettings,
	BuiltInPattern,
	BuiltInPatternsResponse,
	BulkCloneResponse,
	BulkCloneScheduleRequest,
	CategoriesResponse,
	CategoryInfo,
	ChangePasswordRequest,
	ChangelogEntry,
	ChangelogResponse,
	ClassificationLevelsResponse,
	ClassificationRulesResponse,
	ClassificationSummary,
	CloneRepositoryRequest,
	CloneRepositoryResponse,
	CloneScheduleRequest,
	CloudRestoreProgress,
	ComplianceReport,
	ConcurrencyResponse,
	ConfigTemplate,
	ConfigTemplatesResponse,
	ContainerBackupHook,
	ContainerBackupHooksResponse,
	ContainerHookExecution,
	ContainerHookExecutionsResponse,
	ContainerHookTemplateInfo,
	ContainerHookTemplatesResponse,
	ConvertTrialRequest,
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
	CreateBackupHookTemplateRequest,
	CreateBackupScriptRequest,
	CreateCloudRestoreRequest,
	CreateContainerBackupHookRequest,
	CreateCostAlertRequest,
	CreateDRRunbookRequest,
	CreateDRTestScheduleRequest,
	CreateDockerRegistryRequest,
	CreateDockerRestoreRequest,
	CreateDockerStackRequest,
	CreateDowntimeAlertRequest,
	CreateDowntimeEventRequest,
	CreateExcludePatternRequest,
	CreateFavoriteRequest,
	CreateFirstOrgRequest,
	CreateIPAllowlistRequest,
	CreateIPBanRequest,
	CreateImmutabilityLockRequest,
	CreateKomodoIntegrationRequest,
	CreateLegalHoldRequest,
	CreateLicenseKeyRequest,
	CreateLifecyclePolicyRequest,
	CreateMaintenanceWindowRequest,
	CreateMetadataSchemaRequest,
	CreateNotificationChannelRequest,
	CreateNotificationPreferenceRequest,
	CreateNotificationRuleRequest,
	CreateOrgRequest,
	CreatePathClassificationRuleRequest,
	CreatePolicyRequest,
	CreateRateLimitConfigRequest,
	CreateRegistrationCodeRequest,
	CreateRegistrationCodeResponse,
	CreateReportScheduleRequest,
	CreateRepositoryRequest,
	CreateRepositoryResponse,
	CreateRestoreRequest,
	CreateSLADefinitionRequest,
	CreateSLAPolicyRequest,
	CreateSSOGroupMappingRequest,
	CreateSavedFilterRequest,
	CreateScheduleRequest,
	CreateSnapshotCommentRequest,
	CreateStoragePricingRequest,
	CreateSuperuserRequest,
	CreateSuperuserResponse,
	CreateTagRequest,
	CreateTemplateRequest,
	CreateVerificationScheduleRequest,
	CreateWebhookEndpointRequest,
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
	DatabaseTestResponse,
	DefaultPricingResponse,
	DiscoverDockerStacksRequest,
	DiscoverDockerStacksResponse,
	DiscoveredDockerStack,
	DockerBackupRequest,
	DockerBackupResponse,
	DockerContainer,
	DockerContainersResponse,
	DockerDaemonStatus,
	DockerHealthCheckAllResponse,
	DockerHealthCheckResponse,
	DockerLogBackup,
	DockerLogBackupsResponse,
	DockerLogRetentionResult,
	DockerLogSettings,
	DockerLogSettingsUpdate,
	DockerLogStorageStats,
	DockerLogViewResponse,
	DockerLoginAllResponse,
	DockerLoginResult,
	DockerLoginResultResponse,
	DockerRegistriesResponse,
	DockerRegistry,
	DockerRegistryHealthCheck,
	DockerRegistryResponse,
	DockerRegistryTypeInfo,
	DockerRegistryTypesResponse,
	DockerRestore,
	DockerRestorePlan,
	DockerRestorePreviewRequest,
	DockerRestoreProgress,
	DockerRestoresResponse,
	DockerStack,
	DockerStackBackup,
	DockerStackBackupListResponse,
	DockerStackListResponse,
	DockerStackRestore,
	DockerVolume,
	DockerVolumesResponse,
	DowntimeAlert,
	DowntimeAlertsResponse,
	DowntimeEvent,
	DowntimeEventsResponse,
	DryRunResponse,
	ErrorResponse,
	ExcludePattern,
	ExcludePatternsResponse,
	ExpiringCredentialsResponse,
	ExportBundleRequest,
	ExportFormat,
	ExtendImmutabilityLockRequest,
	ExtendTrialRequest,
	Favorite,
	FavoriteEntityType,
	FavoritesResponse,
	FeatureCheckResponse,
	FeatureCheckResult,
	FeatureInfo,
	FeaturesResponse,
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
	GroupedSearchResponse,
	IPAllowlist,
	IPAllowlistSettings,
	IPAllowlistsResponse,
	IPBan,
	IPBansResponse,
	IPBlockedAttemptsResponse,
	ImmutabilityLock,
	ImmutabilityLocksResponse,
	ImmutabilityStatus,
	ImpersonateUserRequest,
	ImpersonateUserResponse,
	ImpersonationLogsResponse,
	ImportConfigRequest,
	ImportPreviewRequest,
	ImportPreviewResponse,
	ImportRepositoryRequest,
	ImportRepositoryResponse,
	ImportResult,
	InvitationsResponse,
	InviteMemberRequest,
	InviteResponse,
	InviteUserRequest,
	InviteUserResponse,
	KeyRecoveryResponse,
	KomodoConnectionTestResponse,
	KomodoContainer,
	KomodoContainersResponse,
	KomodoDiscoveryResult,
	KomodoIntegration,
	KomodoIntegrationResponse,
	KomodoIntegrationsResponse,
	KomodoStack,
	KomodoStacksResponse,
	KomodoSyncResponse,
	KomodoWebhookEvent,
	KomodoWebhookEventsResponse,
	LegalHold,
	LegalHoldsResponse,
	License,
	LicenseFeature,
	LicenseHistoryResponse,
	LicenseInfo,
	LicenseInfoResponse,
	LicenseResponse,
	LicenseValidateResponse,
	LicenseWarningsResponse,
	LicensesResponse,
	LifecycleDeletionEvent,
	LifecycleDeletionEventsResponse,
	LifecycleDryRunRequest,
	LifecycleDryRunResult,
	LifecyclePoliciesResponse,
	LifecyclePolicy,
	MaintenanceWindow,
	MaintenanceWindowsResponse,
	MembersResponse,
	MessageResponse,
	MetadataEntityType,
	MetadataEntityTypesResponse,
	MetadataFieldTypesResponse,
	MetadataSchema,
	MetadataSchemasResponse,
	MetadataSearchResponse,
	MonthlyUptimeReport,
	MountSnapshotRequest,
	NotificationChannel,
	NotificationChannelWithPreferencesResponse,
	NotificationChannelsResponse,
	NotificationLog,
	NotificationLogsResponse,
	NotificationPreference,
	NotificationPreferencesResponse,
	NotificationRule,
	NotificationRuleEvent,
	NotificationRuleEventsResponse,
	NotificationRuleExecution,
	NotificationRuleExecutionsResponse,
	NotificationRulesResponse,
	OIDCSettings,
	OnboardingStatus,
	OnboardingStep,
	OrgInvitation,
	OrgMember,
	OrgResponse,
	OrgSettingsResponse,
	OrganizationWithRole,
	OrganizationsResponse,
	PasswordExpirationInfo,
	PasswordPolicy,
	PasswordPolicyResponse,
	PasswordRequirements,
	PathClassificationRule,
	PendingRegistration,
	PendingRegistrationsResponse,
	PoliciesResponse,
	Policy,
	PricingPlan,
	PublicBrandingSettings,
	RateLimitConfig,
	RateLimitConfigsResponse,
	RateLimitDashboardStats,
	RateLimitStatsResponse,
	RecentItem,
	RecentItemsResponse,
	RecentSearchesResponse,
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
	RerunStatusResponse,
	ResetPasswordRequest,
	ResolveDowntimeEventRequest,
	Restore,
	RestoreDockerStackRequest,
	RestorePreview,
	RestorePreviewRequest,
	RestoresResponse,
	RotateAPIKeyResponse,
	RotateCredentialsRequest,
	RunDRTestRequest,
	RunScheduleResponse,
	SLAAssignment,
	SLAAssignmentsResponse,
	SLABreach,
	SLABreachesResponse,
	SLACompliance,
	SLAComplianceResponse,
	SLADashboardResponse,
	SLADashboardStats,
	SLADefinitionsResponse,
	SLAPoliciesResponse,
	SLAPolicy,
	SLAReport,
	SLAReportResponse,
	SLAStatus,
	SLAStatusHistoryResponse,
	SLAStatusSnapshot,
	SLAWithAssignments,
	SMTPSettings,
	SSOGroupMapping,
	SSOGroupMappingResponse,
	SSOGroupMappingsResponse,
	SSOSettings,
	SaveRecentSearchRequest,
	SavedFilter,
	SavedFiltersResponse,
	Schedule,
	SchedulesResponse,
	SearchFilter,
	SearchResponse,
	SearchSuggestionsResponse,
	SecuritySettings,
	ServerLogComponentsResponse,
	ServerLogFilter,
	ServerLogsResponse,
	ServerSetupStatus,
	ServerVersion,
	SetDebugModeRequest,
	SetDebugModeResponse,
	SetScheduleClassificationRequest,
	SettingsAuditLogsResponse,
	SetupCompleteResponse,
	SetupStartTrialRequest,
	SetupStartTrialResponse,
	Snapshot,
	SnapshotComment,
	SnapshotCommentsResponse,
	SnapshotCompareResponse,
	SnapshotFilesResponse,
	SnapshotMount,
	SnapshotMountsResponse,
	SnapshotsResponse,
	StartTrialRequest,
	StartTrialResponse,
	StorageDefaultSettings,
	StorageGrowthPoint,
	StorageGrowthResponse,
	StorageGrowthTrend,
	StorageGrowthTrendResponse,
	StoragePricing,
	StoragePricingResponse,
	StorageStatsSummary,
	SwitchOrgRequest,
	SystemHealthHistoryResponse,
	SystemHealthResponse,
	Tag,
	TagsResponse,
	TestConnectionRequest,
	TestNotificationRuleRequest,
	TestNotificationRuleResponse,
	TestOIDCResponse,
	TestRepositoryResponse,
	TestSMTPRequest,
	TestSMTPResponse,
	TestWebhookRequest,
	TestWebhookResponse,
	TierInfo,
	TiersResponse,
	TrackRecentItemRequest,
	TransferOwnershipRequest,
	TrialActivityResponse,
	TrialCheckResponse,
	TrialExtension,
	TrialExtensionsResponse,
	TrialFeaturesResponse,
	TrialInfo,
	TriggerDockerStackBackupRequest,
	TriggerVerificationRequest,
	UpdateAgentGroupRequest,
	UpdateAlertRuleRequest,
	UpdateAnnouncementRequest,
	UpdateBackupHookTemplateRequest,
	UpdateBackupScriptRequest,
	UpdateBrandingSettingsRequest,
	UpdateConcurrencyRequest,
	UpdateContainerBackupHookRequest,
	UpdateCostAlertRequest,
	UpdateDRRunbookRequest,
	UpdateDockerRegistryRequest,
	UpdateDockerStackRequest,
	UpdateDowntimeAlertRequest,
	UpdateDowntimeEventRequest,
	UpdateEntityMetadataRequest,
	UpdateExcludePatternRequest,
	UpdateIPAllowlistRequest,
	UpdateIPAllowlistSettingsRequest,
	UpdateKomodoContainerRequest,
	UpdateKomodoIntegrationRequest,
	UpdateLicenseRequest,
	UpdateLifecyclePolicyRequest,
	UpdateMaintenanceWindowRequest,
	UpdateMemberRequest,
	UpdateMetadataSchemaRequest,
	UpdateNotificationChannelRequest,
	UpdateNotificationPreferenceRequest,
	UpdateNotificationRuleRequest,
	UpdateOIDCSettingsRequest,
	UpdateOrgRequest,
	UpdatePasswordPolicyRequest,
	UpdatePathClassificationRuleRequest,
	UpdatePolicyRequest,
	UpdateRateLimitConfigRequest,
	UpdateReportScheduleRequest,
	UpdateRepositoryImmutabilitySettingsRequest,
	UpdateRepositoryRequest,
	UpdateSLADefinitionRequest,
	UpdateSLAPolicyRequest,
	UpdateSMTPSettingsRequest,
	UpdateSSOGroupMappingRequest,
	UpdateSSOSettingsRequest,
	UpdateSavedFilterRequest,
	UpdateScheduleRequest,
	UpdateSecuritySettingsRequest,
	UpdateStorageDefaultsRequest,
	UpdateStoragePricingRequest,
	UpdateTagRequest,
	UpdateTemplateRequest,
	UpdateUserPreferencesRequest,
	UpdateUserRequest,
	UpdateVerificationScheduleRequest,
	UpdateWebhookEndpointRequest,
	UptimeBadge,
	UptimeBadgesResponse,
	UptimeSummary,
	UseTemplateRequest,
	User,
	UserActivityLog,
	UserActivityLogsResponse,
	UserImpersonationLog,
	UserSSOGroups,
	UserSession,
	UserSessionsResponse,
	UserWithMembership,
	UsersResponse,
	ValidateImportRequest,
	ValidationResult,
	Verification,
	VerificationSchedule,
	VerificationSchedulesResponse,
	VerificationStatusResponse,
	VerificationsResponse,
	VerifyImportAccessRequest,
	VerifyImportAccessResponse,
	WebhookDeliveriesResponse,
	WebhookDelivery,
	WebhookEndpoint,
	WebhookEndpointsResponse,
	WebhookEventTypesResponse,
} from './types';

const API_BASE = '/api/v1';

export class ApiError extends Error {
	public resource?: string;
	public limit?: number;
	public tier?: string;
	public feature?: string;

	constructor(
		public status: number,
		message: string,
	) {
		super(message);
		this.name = 'ApiError';
	}
}

// Global upgrade event emitter for 402 Payment Required responses.
// The UpgradePromptProvider subscribes to this to show the upgrade modal.
export type UpgradeEvent = { feature: string; tier: string };
type UpgradeListener = (event: UpgradeEvent) => void;
const upgradeListeners = new Set<UpgradeListener>();

export function onUpgradeRequired(listener: UpgradeListener): () => void {
	upgradeListeners.add(listener);
	return () => upgradeListeners.delete(listener);
}

function emitUpgradeRequired(event: UpgradeEvent) {
	for (const listener of upgradeListeners) {
		listener(event);
	}
}

async function handleResponse<T>(response: Response): Promise<T> {
	if (response.status === 401) {
		// Don't redirect if already on a public page (prevents infinite loop)
		const currentPath = window.location.pathname;
		if (
			currentPath !== '/login' &&
			currentPath !== '/setup' &&
			!currentPath.startsWith('/reset-password')
		) {
			window.location.href = '/login';
		}
		throw new ApiError(401, 'Unauthorized');
	}

	if (response.status === 503) {
		const data = await response.json().catch(() => ({}));
		if (data.redirect === '/setup') {
			window.location.href = '/setup';
			throw new ApiError(503, 'Setup required');
		}
	}

	if (!response.ok) {
		const errorData = await response.json().catch(() => ({
			error: 'Unknown error',
		}));
		if (response.status === 402) {
			const err = new ApiError(
				402,
				errorData.message || errorData.error || 'Upgrade required',
			);
			err.resource = errorData.resource;
			err.limit = errorData.limit;
			err.tier = errorData.tier;
			err.feature = errorData.feature;
			emitUpgradeRequired({
				feature: errorData.feature || errorData.resource || 'This feature',
				tier: errorData.tier || 'free',
			});
			throw err;
		}
		throw new ApiError(response.status, (errorData as ErrorResponse).error);
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

async function fetchFormData<T>(
	endpoint: string,
	formData: FormData,
): Promise<T> {
	const response = await fetch(`${API_BASE}${endpoint}`, {
		method: 'POST',
		credentials: 'include',
		body: formData,
		// Don't set Content-Type header - browser will set it with boundary
	});

	return handleResponse<T>(response);
}

// Bulk invite types
export interface BulkInviteEntry {
	email: string;
	role: string;
}

export interface BulkInviteResult {
	email: string;
	role: string;
	token?: string;
}

export interface BulkInviteError {
	email: string;
	error: string;
}

export interface BulkInviteResponse {
	successful: BulkInviteResult[];
	failed: BulkInviteError[];
	total: number;
}

export interface InvitationDetails {
	id: string;
	org_id: string;
	org_name: string;
	email: string;
	role: string;
	inviter_name: string;
	expires_at: string;
	created_at: string;
}

export interface InvitationDetailsResponse {
	invitation: InvitationDetails;
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

	getLoginUrl: () => '/login',
};

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

	clone: async (
		id: string,
		data: CloneRepositoryRequest,
	): Promise<CloneRepositoryResponse> =>
		fetchApi<CloneRepositoryResponse>(`/repositories/${id}/clone`, {
			method: 'POST',
			body: JSON.stringify(data),
		}),
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

	clone: async (id: string, data: CloneScheduleRequest): Promise<Schedule> =>
		fetchApi<Schedule>(`/schedules/${id}/clone`, {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	bulkClone: async (
		data: BulkCloneScheduleRequest,
	): Promise<BulkCloneResponse> =>
		fetchApi<BulkCloneResponse>('/schedules/bulk-clone', {
			method: 'POST',
			body: JSON.stringify(data),
		}),
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
		fetchApi<SnapshotCompareResponse>(
			`/snapshots/compare?id1=${encodeURIComponent(id1)}&id2=${encodeURIComponent(id2)}`,
		),

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

	createCloud: async (data: CreateCloudRestoreRequest): Promise<Restore> =>
		fetchApi<Restore>('/restores/cloud', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	getProgress: async (id: string): Promise<CloudRestoreProgress> =>
		fetchApi<CloudRestoreProgress>(`/restores/${id}/progress`),
};

// Docker Restores API
export const dockerRestoresApi = {
	list: async (params?: {
		agent_id?: string;
		status?: string;
	}): Promise<DockerRestore[]> => {
		const searchParams = new URLSearchParams();
		if (params?.agent_id) searchParams.set('agent_id', params.agent_id);
		if (params?.status) searchParams.set('status', params.status);

		const query = searchParams.toString();
		const endpoint = query ? `/docker-restores?${query}` : '/docker-restores';
		const response = await fetchApi<DockerRestoresResponse>(endpoint);
		return response.docker_restores ?? [];
	},

	get: async (id: string): Promise<DockerRestore> =>
		fetchApi<DockerRestore>(`/docker-restores/${id}`),

	create: async (data: CreateDockerRestoreRequest): Promise<DockerRestore> =>
		fetchApi<DockerRestore>('/docker-restores', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	preview: async (
		data: DockerRestorePreviewRequest,
	): Promise<DockerRestorePlan> =>
		fetchApi<DockerRestorePlan>('/docker-restores/preview', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	getProgress: async (id: string): Promise<DockerRestoreProgress> =>
		fetchApi<DockerRestoreProgress>(`/docker-restores/${id}/progress`),

	listContainers: async (
		snapshotId: string,
		agentId: string,
	): Promise<DockerContainer[]> => {
		const searchParams = new URLSearchParams();
		searchParams.set('agent_id', agentId);
		const response = await fetchApi<DockerContainersResponse>(
			`/docker-restores/snapshot/${snapshotId}/containers?${searchParams.toString()}`,
		);
		return response.containers ?? [];
	},

	listVolumes: async (
		snapshotId: string,
		agentId: string,
	): Promise<DockerVolume[]> => {
		const searchParams = new URLSearchParams();
		searchParams.set('agent_id', agentId);
		const response = await fetchApi<DockerVolumesResponse>(
			`/docker-restores/snapshot/${snapshotId}/volumes?${searchParams.toString()}`,
		);
		return response.volumes ?? [];
	},

	cancel: async (id: string): Promise<DockerRestore> =>
		fetchApi<DockerRestore>(`/docker-restores/${id}/cancel`, {
			method: 'POST',
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

	resendInvitation: async (
		orgId: string,
		invitationId: string,
	): Promise<InviteResponse> =>
		fetchApi<InviteResponse>(
			`/organizations/${orgId}/invitations/${invitationId}/resend`,
			{
				method: 'POST',
			},
		),

	bulkInvite: async (
		orgId: string,
		invites: BulkInviteEntry[],
	): Promise<BulkInviteResponse> =>
		fetchApi<BulkInviteResponse>(`/organizations/${orgId}/invitations/bulk`, {
			method: 'POST',
			body: JSON.stringify({ invites }),
		}),

	bulkInviteCSV: async (
		orgId: string,
		file: File,
	): Promise<BulkInviteResponse> => {
		const formData = new FormData();
		formData.append('file', file);
		return fetchFormData<BulkInviteResponse>(
			`/organizations/${orgId}/invitations/bulk`,
			formData,
		);
	},

	getInvitationByToken: async (
		token: string,
	): Promise<InvitationDetailsResponse> =>
		fetchApi<InvitationDetailsResponse>(`/invitations/${token}`),

	acceptInvitation: async (token: string): Promise<OrgResponse> =>
		fetchApi<OrgResponse>('/invitations/accept', {
			method: 'POST',
			body: JSON.stringify({ token }),
		}),
};

// Admin Organizations API (superuser only)
export const adminOrganizationsApi = {
	list: async (params?: {
		search?: string;
		limit?: number;
		offset?: number;
	}): Promise<AdminOrganizationsResponse> => {
		const query = new URLSearchParams();
		if (params?.search) query.set('search', params.search);
		if (params?.limit) query.set('limit', params.limit.toString());
		if (params?.offset) query.set('offset', params.offset.toString());
		const queryStr = query.toString();
		return fetchApi<AdminOrganizationsResponse>(
			`/admin/organizations${queryStr ? `?${queryStr}` : ''}`,
		);
	},

	get: async (id: string): Promise<AdminOrganization> =>
		fetchApi<AdminOrganization>(`/admin/organizations/${id}`),

	create: async (data: AdminCreateOrgRequest): Promise<AdminOrganization> =>
		fetchApi<AdminOrganization>('/admin/organizations', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: AdminOrgSettings,
	): Promise<AdminOrganization> =>
		fetchApi<AdminOrganization>(`/admin/organizations/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/admin/organizations/${id}`, {
			method: 'DELETE',
		}),

	getUsageStats: async (id: string): Promise<AdminOrgUsageStats> =>
		fetchApi<AdminOrgUsageStats>(`/admin/organizations/${id}/usage`),

	transferOwnership: async (
		id: string,
		data: TransferOwnershipRequest,
	): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/admin/organizations/${id}/transfer-ownership`, {
			method: 'POST',
			body: JSON.stringify(data),
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

// Notification Rules API
export const notificationRulesApi = {
	list: async (): Promise<NotificationRule[]> => {
		const response = await fetchApi<NotificationRulesResponse>(
			'/notification-rules',
		);
		return response.rules ?? [];
	},

	get: async (id: string): Promise<NotificationRule> =>
		fetchApi<NotificationRule>(`/notification-rules/${id}`),

	create: async (
		data: CreateNotificationRuleRequest,
	): Promise<NotificationRule> =>
		fetchApi<NotificationRule>('/notification-rules', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateNotificationRuleRequest,
	): Promise<NotificationRule> =>
		fetchApi<NotificationRule>(`/notification-rules/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/notification-rules/${id}`, {
			method: 'DELETE',
		}),

	test: async (
		id: string,
		data?: TestNotificationRuleRequest,
	): Promise<TestNotificationRuleResponse> =>
		fetchApi<TestNotificationRuleResponse>(`/notification-rules/${id}/test`, {
			method: 'POST',
			body: JSON.stringify(data ?? {}),
		}),

	listEvents: async (id: string): Promise<NotificationRuleEvent[]> => {
		const response = await fetchApi<NotificationRuleEventsResponse>(
			`/notification-rules/${id}/events`,
		);
		return response.events ?? [];
	},

	listExecutions: async (id: string): Promise<NotificationRuleExecution[]> => {
		const response = await fetchApi<NotificationRuleExecutionsResponse>(
			`/notification-rules/${id}/executions`,
		);
		return response.executions ?? [];
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

	searchGrouped: async (
		filter: SearchFilter,
	): Promise<GroupedSearchResponse> => {
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

		return fetchApi<GroupedSearchResponse>(
			`/search/grouped?${searchParams.toString()}`,
		);
	},

	getSuggestions: async (
		query: string,
		limit = 10,
	): Promise<SearchSuggestionsResponse> => {
		const searchParams = new URLSearchParams();
		searchParams.set('q', query);
		searchParams.set('limit', limit.toString());
		return fetchApi<SearchSuggestionsResponse>(
			`/search/suggestions?${searchParams.toString()}`,
		);
	},

	getRecentSearches: async (limit = 10): Promise<RecentSearchesResponse> => {
		const searchParams = new URLSearchParams();
		searchParams.set('limit', limit.toString());
		return fetchApi<RecentSearchesResponse>(
			`/search/recent?${searchParams.toString()}`,
		);
	},

	saveRecentSearch: async (
		data: SaveRecentSearchRequest,
	): Promise<MessageResponse> =>
		fetchApi<MessageResponse>('/search/recent', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	deleteRecentSearch: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/search/recent/${id}`, {
			method: 'DELETE',
		}),

	clearRecentSearches: async (): Promise<MessageResponse> =>
		fetchApi<MessageResponse>('/search/recent', {
			method: 'DELETE',
		}),
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

	completeStepWithBody: async <T>(
		step: OnboardingStep,
		body: T,
	): Promise<OnboardingStatus> =>
		fetchApi<OnboardingStatus>(`/onboarding/step/${step}`, {
			method: 'POST',
			body: JSON.stringify(body),
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

// SLA Policies API
export const slaPoliciesApi = {
	list: async (): Promise<SLAPolicy[]> => {
		const response = await fetchApi<SLAPoliciesResponse>('/sla/policies');
		return response.policies ?? [];
	},

	get: async (id: string): Promise<SLAPolicy> =>
		fetchApi<SLAPolicy>(`/sla/policies/${id}`),

	create: async (data: CreateSLAPolicyRequest): Promise<SLAPolicy> =>
		fetchApi<SLAPolicy>('/sla/policies', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateSLAPolicyRequest,
	): Promise<SLAPolicy> =>
		fetchApi<SLAPolicy>(`/sla/policies/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/sla/policies/${id}`, {
			method: 'DELETE',
		}),

	getStatus: async (id: string): Promise<SLAStatus> =>
		fetchApi<SLAStatus>(`/sla/policies/${id}/status`),

	getHistory: async (id: string, limit = 100): Promise<SLAStatusSnapshot[]> => {
		const response = await fetchApi<SLAStatusHistoryResponse>(
			`/sla/policies/${id}/history?limit=${limit}`,
		);
		return response.history ?? [];
	},
};

// Changelog API
export const changelogApi = {
	list: async (): Promise<ChangelogResponse> =>
		fetchApi<ChangelogResponse>('/changelog'),

	get: async (version: string): Promise<ChangelogEntry> =>
		fetchApi<ChangelogEntry>(`/changelog/${version}`),
};

// Docker Backup API
export const dockerBackupApi = {
	listContainers: async (agentId: string): Promise<DockerContainer[]> => {
		const response = await fetchApi<DockerContainersResponse>(
			`/docker/containers?agent_id=${agentId}`,
		);
		return response.containers ?? [];
	},

	listVolumes: async (agentId: string): Promise<DockerVolume[]> => {
		const response = await fetchApi<DockerVolumesResponse>(
			`/docker/volumes?agent_id=${agentId}`,
		);
		return response.volumes ?? [];
	},

	triggerBackup: async (
		data: DockerBackupRequest,
	): Promise<DockerBackupResponse> =>
		fetchApi<DockerBackupResponse>('/docker/backup', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	getDaemonStatus: async (agentId: string): Promise<DockerDaemonStatus> =>
		fetchApi<DockerDaemonStatus>(`/docker/status?agent_id=${agentId}`),
};

// Air-Gap API
export const airGapApi = {
	getStatus: async (): Promise<AirGapStatus> =>
		fetchApi<AirGapStatus>('/system/airgap'),
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

export const versionApi = {
	get: async (): Promise<ServerVersion> => fetchApi<ServerVersion>('/version'),
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

// Metadata Schemas API
export const metadataApi = {
	listSchemas: async (
		entityType: MetadataEntityType,
	): Promise<MetadataSchema[]> => {
		const response = await fetchApi<MetadataSchemasResponse>(
			`/metadata/schemas?entity_type=${entityType}`,
		);
		return response.schemas ?? [];
	},

	getSchema: async (id: string): Promise<MetadataSchema> =>
		fetchApi<MetadataSchema>(`/metadata/schemas/${id}`),

	createSchema: async (
		data: CreateMetadataSchemaRequest,
	): Promise<MetadataSchema> =>
		fetchApi<MetadataSchema>('/metadata/schemas', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	updateSchema: async (
		id: string,
		data: UpdateMetadataSchemaRequest,
	): Promise<MetadataSchema> =>
		fetchApi<MetadataSchema>(`/metadata/schemas/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	deleteSchema: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/metadata/schemas/${id}`, {
			method: 'DELETE',
		}),

	getFieldTypes: async (): Promise<MetadataFieldTypesResponse> =>
		fetchApi<MetadataFieldTypesResponse>('/metadata/schemas/types'),

	getEntityTypes: async (): Promise<MetadataEntityTypesResponse> =>
		fetchApi<MetadataEntityTypesResponse>('/metadata/schemas/entities'),

	updateAgentMetadata: async (
		agentId: string,
		data: UpdateEntityMetadataRequest,
	): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/agents/${agentId}/metadata`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	updateRepositoryMetadata: async (
		repositoryId: string,
		data: UpdateEntityMetadataRequest,
	): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/repositories/${repositoryId}/metadata`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	updateScheduleMetadata: async (
		scheduleId: string,
		data: UpdateEntityMetadataRequest,
	): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/schedules/${scheduleId}/metadata`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	search: async (
		entityType: MetadataEntityType,
		key: string,
		value: string,
	): Promise<MetadataSearchResponse> =>
		fetchApi<MetadataSearchResponse>(
			`/metadata/search?entity_type=${entityType}&key=${encodeURIComponent(key)}&value=${encodeURIComponent(value)}`,
		),
};

// Agent Import API
export const agentImportApi = {
	preview: async (
		file: File,
		options?: {
			hasHeader?: boolean;
			hostnameCol?: number;
			groupCol?: number;
			tagsCol?: number;
			configCol?: number;
			createMissingGroups?: boolean;
			tokenExpiryHours?: number;
		},
	): Promise<AgentImportPreviewResponse> => {
		const formData = new FormData();
		formData.append('file', file);
		if (options) {
			formData.append('options', JSON.stringify(options));
		}
		const response = await fetch(`${API_BASE}/agents/import/preview`, {
			method: 'POST',
			credentials: 'include',
			body: formData,
		});
		return handleResponse<AgentImportPreviewResponse>(response);
	},

	import: async (
		file: File,
		options?: {
			hasHeader?: boolean;
			hostnameCol?: number;
			groupCol?: number;
			tagsCol?: number;
			configCol?: number;
			createMissingGroups?: boolean;
			tokenExpiryHours?: number;
		},
	): Promise<AgentImportResponse> => {
		const formData = new FormData();
		formData.append('file', file);
		if (options) {
			formData.append('options', JSON.stringify(options));
		}
		const response = await fetch(`${API_BASE}/agents/import`, {
			method: 'POST',
			credentials: 'include',
			body: formData,
		});
		return handleResponse<AgentImportResponse>(response);
	},

	getTemplate: async (
		format: 'json' | 'csv' = 'csv',
	): Promise<AgentImportTemplateResponse> =>
		fetchApi<AgentImportTemplateResponse>(
			`/agents/import/template?format=${format}`,
		),

	downloadTemplate: async (): Promise<Blob> => {
		const response = await fetch(
			`${API_BASE}/agents/import/template/download`,
			{
				credentials: 'include',
			},
		);
		if (!response.ok) {
			throw new ApiError(response.status, 'Failed to download template');
		}
		return response.blob();
	},

	generateScript: async (data: {
		hostname: string;
		registration_code: string;
	}): Promise<{ script: string }> =>
		fetchApi<{ script: string }>('/agents/import/script', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	exportTokens: async (results: AgentImportJobResult[]): Promise<Blob> => {
		const response = await fetch(`${API_BASE}/agents/import/export-tokens`, {
			method: 'POST',
			credentials: 'include',
			headers: {
				'Content-Type': 'application/json',
			},
			body: JSON.stringify({ results }),
		});
		if (!response.ok) {
			throw new ApiError(response.status, 'Failed to export tokens');
		}
		return response.blob();
	},
};

// Backup Queue API
export const backupQueueApi = {
	list: async (): Promise<BackupQueueResponse['queue']> => {
		const response = await fetchApi<BackupQueueResponse>('/backup-queue');
		return response.queue ?? [];
	},

	getSummary: async (): Promise<BackupQueueSummary> =>
		fetchApi<BackupQueueSummary>('/backup-queue/summary'),

	cancel: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/backup-queue/${id}`, {
			method: 'DELETE',
		}),
};

// Concurrency API
export const concurrencyApi = {
	getOrgConcurrency: async (orgId: string): Promise<ConcurrencyResponse> =>
		fetchApi<ConcurrencyResponse>(`/organizations/${orgId}/concurrency`),

	updateOrgConcurrency: async (
		orgId: string,
		data: UpdateConcurrencyRequest,
	): Promise<ConcurrencyResponse> =>
		fetchApi<ConcurrencyResponse>(`/organizations/${orgId}/concurrency`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	getAgentConcurrency: async (agentId: string): Promise<ConcurrencyResponse> =>
		fetchApi<ConcurrencyResponse>(`/agents/${agentId}/concurrency`),

	updateAgentConcurrency: async (
		agentId: string,
		data: UpdateConcurrencyRequest,
	): Promise<ConcurrencyResponse> =>
		fetchApi<ConcurrencyResponse>(`/agents/${agentId}/concurrency`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),
};

// IP Allowlists API
export const ipAllowlistsApi = {
	list: async (): Promise<IPAllowlist[]> => {
		const response = await fetchApi<IPAllowlistsResponse>('/ip-allowlists');
		return response.allowlists ?? [];
	},

	get: async (id: string): Promise<IPAllowlist> =>
		fetchApi<IPAllowlist>(`/ip-allowlists/${id}`),

	create: async (data: CreateIPAllowlistRequest): Promise<IPAllowlist> =>
		fetchApi<IPAllowlist>('/ip-allowlists', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateIPAllowlistRequest,
	): Promise<IPAllowlist> =>
		fetchApi<IPAllowlist>(`/ip-allowlists/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/ip-allowlists/${id}`, {
			method: 'DELETE',
		}),

	getSettings: async (): Promise<IPAllowlistSettings> =>
		fetchApi<IPAllowlistSettings>('/ip-allowlists/settings'),

	updateSettings: async (
		data: UpdateIPAllowlistSettingsRequest,
	): Promise<IPAllowlistSettings> =>
		fetchApi<IPAllowlistSettings>('/ip-allowlists/settings', {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	listBlockedAttempts: async (
		limit = 50,
		offset = 0,
	): Promise<IPBlockedAttemptsResponse> =>
		fetchApi<IPBlockedAttemptsResponse>(
			`/ip-allowlists/blocked?limit=${limit}&offset=${offset}`,
		),
};

// Lifecycle Policies API
export const lifecyclePoliciesApi = {
	list: async (): Promise<LifecyclePolicy[]> => {
		const response = await fetchApi<LifecyclePoliciesResponse>(
			'/lifecycle-policies',
		);
		return response.policies ?? [];
	},

	get: async (id: string): Promise<LifecyclePolicy> =>
		fetchApi<LifecyclePolicy>(`/lifecycle-policies/${id}`),

	create: async (
		data: CreateLifecyclePolicyRequest,
	): Promise<LifecyclePolicy> =>
		fetchApi<LifecyclePolicy>('/lifecycle-policies', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateLifecyclePolicyRequest,
	): Promise<LifecyclePolicy> =>
		fetchApi<LifecyclePolicy>(`/lifecycle-policies/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/lifecycle-policies/${id}`, {
			method: 'DELETE',
		}),

	dryRun: async (id: string): Promise<LifecycleDryRunResult> =>
		fetchApi<LifecycleDryRunResult>(`/lifecycle-policies/${id}/dry-run`, {
			method: 'POST',
		}),

	preview: async (
		data: LifecycleDryRunRequest,
	): Promise<LifecycleDryRunResult> =>
		fetchApi<LifecycleDryRunResult>('/lifecycle-policies/preview', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	listDeletions: async (
		policyId: string,
		limit?: number,
	): Promise<LifecycleDeletionEvent[]> => {
		const params = new URLSearchParams();
		if (limit) params.set('limit', limit.toString());
		const query = params.toString();
		const response = await fetchApi<LifecycleDeletionEventsResponse>(
			`/lifecycle-policies/${policyId}/deletions${query ? `?${query}` : ''}`,
		);
		return response.events ?? [];
	},

	listOrgDeletions: async (
		limit?: number,
	): Promise<LifecycleDeletionEvent[]> => {
		const params = new URLSearchParams();
		if (limit) params.set('limit', limit.toString());
		const query = params.toString();
		const response = await fetchApi<LifecycleDeletionEventsResponse>(
			`/lifecycle-policies/deletions${query ? `?${query}` : ''}`,
		);
		return response.events ?? [];
	},
};

// Password API
export const passwordApi = {
	changePassword: async (
		data: ChangePasswordRequest,
	): Promise<MessageResponse> =>
		fetchApi<MessageResponse>('/auth/password', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	getExpiration: async (): Promise<PasswordExpirationInfo> =>
		fetchApi<PasswordExpirationInfo>('/password/expiration'),
};

// Password Policies API
export const passwordPoliciesApi = {
	get: async (): Promise<PasswordPolicyResponse> =>
		fetchApi<PasswordPolicyResponse>('/password-policies'),

	getRequirements: async (): Promise<PasswordRequirements> =>
		fetchApi<PasswordRequirements>('/password-policies/requirements'),

	update: async (data: UpdatePasswordPolicyRequest): Promise<PasswordPolicy> =>
		fetchApi<PasswordPolicy>('/password-policies', {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	validatePassword: async (
		password: string,
	): Promise<{ valid: boolean; errors?: string[]; warnings?: string[] }> =>
		fetchApi<{ valid: boolean; errors?: string[]; warnings?: string[] }>(
			'/password-policies/validate',
			{
				method: 'POST',
				body: JSON.stringify({ password }),
			},
		),
};

// IP Bans API
export const ipBansApi = {
	list: async (): Promise<IPBan[]> => {
		const response = await fetchApi<IPBansResponse>('/ip-bans');
		return response.bans ?? [];
	},

	create: async (data: CreateIPBanRequest): Promise<IPBan> =>
		fetchApi<IPBan>('/ip-bans', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/ip-bans/${id}`, {
			method: 'DELETE',
		}),
};

// Rate Limit Configs API
export const rateLimitConfigsApi = {
	list: async (): Promise<RateLimitConfig[]> => {
		const response = await fetchApi<RateLimitConfigsResponse>(
			'/rate-limit-configs',
		);
		return response.configs ?? [];
	},

	get: async (id: string): Promise<RateLimitConfig> =>
		fetchApi<RateLimitConfig>(`/rate-limit-configs/${id}`),

	create: async (
		data: CreateRateLimitConfigRequest,
	): Promise<RateLimitConfig> =>
		fetchApi<RateLimitConfig>('/rate-limit-configs', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateRateLimitConfigRequest,
	): Promise<RateLimitConfig> =>
		fetchApi<RateLimitConfig>(`/rate-limit-configs/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/rate-limit-configs/${id}`, {
			method: 'DELETE',
		}),

	getStats: async (): Promise<RateLimitStatsResponse> =>
		fetchApi<RateLimitStatsResponse>('/rate-limit-configs/stats'),

	listBlocked: async (): Promise<BlockedRequestsResponse> =>
		fetchApi<BlockedRequestsResponse>('/rate-limit-configs/blocked'),
};

// Rate Limits API (Dashboard)
export const rateLimitsApi = {
	getDashboardStats: async (): Promise<RateLimitDashboardStats> =>
		fetchApi<RateLimitDashboardStats>('/admin/rate-limits'),
};

// User Sessions API
export const userSessionsApi = {
	list: async (): Promise<UserSession[]> => {
		const response = await fetchApi<UserSessionsResponse>('/user-sessions');
		return response.sessions ?? [];
	},

	revoke: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/user-sessions/${id}`, {
			method: 'DELETE',
		}),

	revokeAll: async (): Promise<MessageResponse> =>
		fetchApi<MessageResponse>('/user-sessions', {
			method: 'DELETE',
		}),
};

// Saved Filters API
export const savedFiltersApi = {
	list: async (entityType?: string): Promise<SavedFilter[]> => {
		const endpoint = entityType
			? `/filters?entity_type=${entityType}`
			: '/filters';
		const response = await fetchApi<SavedFiltersResponse>(endpoint);
		return response.filters ?? [];
	},

	get: async (id: string): Promise<SavedFilter> =>
		fetchApi<SavedFilter>(`/filters/${id}`),

	create: async (data: CreateSavedFilterRequest): Promise<SavedFilter> =>
		fetchApi<SavedFilter>('/filters', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateSavedFilterRequest,
	): Promise<SavedFilter> =>
		fetchApi<SavedFilter>(`/filters/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/filters/${id}`, {
			method: 'DELETE',
		}),
};

// Downtime API
export const downtimeApi = {
	list: async (limit = 100, offset = 0): Promise<DowntimeEvent[]> => {
		const response = await fetchApi<DowntimeEventsResponse>(
			`/downtime?limit=${limit}&offset=${offset}`,
		);
		return response.events ?? [];
	},

	listActive: async (): Promise<DowntimeEvent[]> => {
		const response = await fetchApi<DowntimeEventsResponse>('/downtime/active');
		return response.events ?? [];
	},

	getSummary: async (): Promise<UptimeSummary> =>
		fetchApi<UptimeSummary>('/downtime/summary'),

	get: async (id: string): Promise<DowntimeEvent> =>
		fetchApi<DowntimeEvent>(`/downtime/${id}`),

	create: async (data: CreateDowntimeEventRequest): Promise<DowntimeEvent> =>
		fetchApi<DowntimeEvent>('/downtime', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateDowntimeEventRequest,
	): Promise<DowntimeEvent> =>
		fetchApi<DowntimeEvent>(`/downtime/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	resolve: async (
		id: string,
		data: ResolveDowntimeEventRequest = {},
	): Promise<DowntimeEvent> =>
		fetchApi<DowntimeEvent>(`/downtime/${id}/resolve`, {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/downtime/${id}`, {
			method: 'DELETE',
		}),
};

// Uptime API
export const uptimeApi = {
	getBadges: async (): Promise<UptimeBadge[]> => {
		const response = await fetchApi<UptimeBadgesResponse>('/uptime/badges');
		return response.badges ?? [];
	},

	refreshBadges: async (): Promise<MessageResponse> =>
		fetchApi<MessageResponse>('/uptime/badges/refresh', {
			method: 'POST',
		}),

	getMonthlyReport: async (
		year: number,
		month: number,
	): Promise<MonthlyUptimeReport> =>
		fetchApi<MonthlyUptimeReport>(`/uptime/report/${year}/${month}`),
};

// Downtime Alerts API
export const downtimeAlertsApi = {
	list: async (): Promise<DowntimeAlert[]> => {
		const response = await fetchApi<DowntimeAlertsResponse>('/downtime-alerts');
		return response.alerts ?? [];
	},

	get: async (id: string): Promise<DowntimeAlert> =>
		fetchApi<DowntimeAlert>(`/downtime-alerts/${id}`),

	create: async (data: CreateDowntimeAlertRequest): Promise<DowntimeAlert> =>
		fetchApi<DowntimeAlert>('/downtime-alerts', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateDowntimeAlertRequest,
	): Promise<DowntimeAlert> =>
		fetchApi<DowntimeAlert>(`/downtime-alerts/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/downtime-alerts/${id}`, {
			method: 'DELETE',
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

	create: async (
		data: CreateSLADefinitionRequest,
	): Promise<SLAWithAssignments> =>
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
		const response = await fetchApi<SLABreachesResponse>(
			'/sla-breaches/active',
		);
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

// Organization System Settings API (SMTP, OIDC, Storage, Security)
export const orgSettingsApi = {
	// Get all settings
	getAll: async (): Promise<OrgSettingsResponse> =>
		fetchApi<OrgSettingsResponse>('/system-settings'),

	// SMTP settings
	getSMTP: async (): Promise<SMTPSettings> =>
		fetchApi<SMTPSettings>('/system-settings/smtp'),

	updateSMTP: async (data: UpdateSMTPSettingsRequest): Promise<SMTPSettings> =>
		fetchApi<SMTPSettings>('/system-settings/smtp', {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	testSMTP: async (data: TestSMTPRequest): Promise<TestSMTPResponse> =>
		fetchApi<TestSMTPResponse>('/system-settings/smtp/test', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	// OIDC settings
	getOIDC: async (): Promise<OIDCSettings> =>
		fetchApi<OIDCSettings>('/system-settings/oidc'),

	updateOIDC: async (data: UpdateOIDCSettingsRequest): Promise<OIDCSettings> =>
		fetchApi<OIDCSettings>('/system-settings/oidc', {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	testOIDC: async (): Promise<TestOIDCResponse> =>
		fetchApi<TestOIDCResponse>('/system-settings/oidc/test', {
			method: 'POST',
		}),

	// Storage settings
	getStorageDefaults: async (): Promise<StorageDefaultSettings> =>
		fetchApi<StorageDefaultSettings>('/system-settings/storage'),

	updateStorageDefaults: async (
		data: UpdateStorageDefaultsRequest,
	): Promise<StorageDefaultSettings> =>
		fetchApi<StorageDefaultSettings>('/system-settings/storage', {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	// Security settings
	getSecurity: async (): Promise<SecuritySettings> =>
		fetchApi<SecuritySettings>('/system-settings/security'),

	updateSecurity: async (
		data: UpdateSecuritySettingsRequest,
	): Promise<SecuritySettings> =>
		fetchApi<SecuritySettings>('/system-settings/security', {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	// Audit log
	getAuditLog: async (
		limit = 50,
		offset = 0,
	): Promise<SettingsAuditLogsResponse> =>
		fetchApi<SettingsAuditLogsResponse>(
			`/system-settings/audit-log?limit=${limit}&offset=${offset}`,
		),
};

// Trial API
export const trialApi = {
	// Get current trial status
	getStatus: async (): Promise<TrialInfo> =>
		fetchApi<TrialInfo>('/trial/status'),

	// Start a new trial
	startTrial: async (data: StartTrialRequest): Promise<TrialInfo> =>
		fetchApi<TrialInfo>('/trial/start', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	// Get available Pro features
	getFeatures: async (): Promise<TrialFeaturesResponse> =>
		fetchApi<TrialFeaturesResponse>('/trial/features'),

	// Get trial activity log
	getActivity: async (limit = 50, offset = 0): Promise<TrialActivityResponse> =>
		fetchApi<TrialActivityResponse>(
			`/trial/activity?limit=${limit}&offset=${offset}`,
		),

	// Extend trial (admin/superuser only)
	extendTrial: async (data: ExtendTrialRequest): Promise<TrialExtension> =>
		fetchApi<TrialExtension>('/trial/extend', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	// Convert trial to paid
	convertTrial: async (data: ConvertTrialRequest): Promise<TrialInfo> =>
		fetchApi<TrialInfo>('/trial/convert', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	// Get extension history
	getExtensions: async (): Promise<TrialExtensionsResponse> =>
		fetchApi<TrialExtensionsResponse>('/trial/extensions'),
};

// Docker Container Logs API
export const dockerLogsApi = {
	// List all backups
	list: async (status?: string): Promise<DockerLogBackupsResponse> => {
		const url = status ? `/docker-logs?status=${status}` : '/docker-logs';
		return fetchApi<DockerLogBackupsResponse>(url);
	},

	// Get a specific backup
	get: async (id: string): Promise<DockerLogBackup> =>
		fetchApi<DockerLogBackup>(`/docker-logs/${id}`),

	// View backup contents
	view: async (
		id: string,
		offset = 0,
		limit = 1000,
	): Promise<DockerLogViewResponse> =>
		fetchApi<DockerLogViewResponse>(
			`/docker-logs/${id}/view?offset=${offset}&limit=${limit}`,
		),

	// Download backup
	download: async (
		id: string,
		format: 'json' | 'csv' | 'raw' = 'json',
	): Promise<Blob> => {
		const response = await fetch(
			`${API_BASE}/docker-logs/${id}/download?format=${format}`,
			{
				credentials: 'include',
			},
		);
		if (!response.ok) {
			throw new ApiError(response.status, 'Failed to download docker logs');
		}
		return response.blob();
	},

	// Delete a backup
	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/docker-logs/${id}`, {
			method: 'DELETE',
		}),

	// Get settings for an agent
	getSettings: async (agentId: string): Promise<DockerLogSettings> =>
		fetchApi<DockerLogSettings>(`/docker-logs/settings/${agentId}`),

	// Update settings for an agent
	updateSettings: async (
		agentId: string,
		settings: DockerLogSettingsUpdate,
	): Promise<DockerLogSettings> =>
		fetchApi<DockerLogSettings>(`/docker-logs/settings/${agentId}`, {
			method: 'PUT',
			body: JSON.stringify(settings),
		}),

	// List backups by agent
	listByAgent: async (agentId: string): Promise<DockerLogBackupsResponse> =>
		fetchApi<DockerLogBackupsResponse>(`/docker-logs/agent/${agentId}`),

	// List backups by container
	listByContainer: async (
		agentId: string,
		containerId: string,
	): Promise<DockerLogBackupsResponse> =>
		fetchApi<DockerLogBackupsResponse>(
			`/docker-logs/agent/${agentId}/container/${containerId}`,
		),

	// Get storage stats for an agent
	getStorageStats: async (agentId: string): Promise<DockerLogStorageStats> =>
		fetchApi<DockerLogStorageStats>(`/docker-logs/stats/${agentId}`),

	// Apply retention policy
	applyRetention: async (
		agentId: string,
		containerId?: string,
	): Promise<DockerLogRetentionResult> => {
		const url = containerId
			? `/docker-logs/retention/${agentId}?container_id=${containerId}`
			: `/docker-logs/retention/${agentId}`;
		return fetchApi<DockerLogRetentionResult>(url, {
			method: 'POST',
		});
	},
};

// Recent Items API
export const recentItemsApi = {
	list: async (type?: string): Promise<RecentItem[]> => {
		const url = type ? `/recent-items?type=${type}` : '/recent-items';
		const response = await fetchApi<RecentItemsResponse>(url);
		return response.items ?? [];
	},

	track: async (data: TrackRecentItemRequest): Promise<RecentItem> =>
		fetchApi<RecentItem>('/recent-items', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/recent-items/${id}`, {
			method: 'DELETE',
		}),

	clearAll: async (): Promise<MessageResponse> =>
		fetchApi<MessageResponse>('/recent-items', {
			method: 'DELETE',
		}),
};

// Activity API
export const activityApi = {
	list: async (filter?: ActivityEventFilter): Promise<ActivityEvent[]> => {
		const params = new URLSearchParams();
		if (filter?.category) params.set('category', filter.category);
		if (filter?.type) params.set('type', filter.type);
		if (filter?.user_id) params.set('user_id', filter.user_id);
		if (filter?.agent_id) params.set('agent_id', filter.agent_id);
		if (filter?.start_time) params.set('start_time', filter.start_time);
		if (filter?.end_time) params.set('end_time', filter.end_time);
		if (filter?.limit) params.set('limit', String(filter.limit));
		if (filter?.offset) params.set('offset', String(filter.offset));

		const queryString = params.toString();
		const url = queryString ? `/activity?${queryString}` : '/activity';
		const response = await fetchApi<ActivityEventsResponse>(url);
		return response.events ?? [];
	},

	recent: async (limit?: number): Promise<ActivityEvent[]> => {
		const url = limit ? `/activity/recent?limit=${limit}` : '/activity/recent';
		const response = await fetchApi<ActivityEventsResponse>(url);
		return response.events ?? [];
	},

	count: async (category?: string, type?: string): Promise<number> => {
		const params = new URLSearchParams();
		if (category) params.set('category', category);
		if (type) params.set('type', type);

		const queryString = params.toString();
		const url = queryString
			? `/activity/count?${queryString}`
			: '/activity/count';
		const response = await fetchApi<ActivityEventCountResponse>(url);
		return response.count;
	},

	categories: async (): Promise<Record<string, number>> => {
		const response = await fetchApi<ActivityCategoriesResponse>(
			'/activity/categories',
		);
		return response.categories ?? {};
	},

	search: async (query: string, limit?: number): Promise<ActivityEvent[]> => {
		const params = new URLSearchParams({ q: query });
		if (limit) params.set('limit', String(limit));

		const response = await fetchApi<ActivityEventsResponse>(
			`/activity/search?${params.toString()}`,
		);
		return response.events ?? [];
	},
};

// Favorites API
export const favoritesApi = {
	list: async (entityType?: FavoriteEntityType): Promise<Favorite[]> => {
		const endpoint = entityType
			? `/favorites?entity_type=${entityType}`
			: '/favorites';
		const response = await fetchApi<FavoritesResponse>(endpoint);
		return response.favorites ?? [];
	},

	create: async (data: CreateFavoriteRequest): Promise<Favorite> =>
		fetchApi<Favorite>('/favorites', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	delete: async (
		entityType: FavoriteEntityType,
		entityId: string,
	): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/favorites/${entityType}/${entityId}`, {
			method: 'DELETE',
		}),
};

// Docker Stacks API
export const dockerStacksApi = {
	// Stack Management
	list: async (agentId?: string): Promise<DockerStack[]> => {
		const endpoint = agentId
			? `/docker-stacks?agent_id=${agentId}`
			: '/docker-stacks';
		const response = await fetchApi<DockerStackListResponse>(endpoint);
		return response.stacks ?? [];
	},

	get: async (id: string): Promise<DockerStack> =>
		fetchApi<DockerStack>(`/docker-stacks/${id}`),

	create: async (data: CreateDockerStackRequest): Promise<DockerStack> =>
		fetchApi<DockerStack>('/docker-stacks', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateDockerStackRequest,
	): Promise<DockerStack> =>
		fetchApi<DockerStack>(`/docker-stacks/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/docker-stacks/${id}`, {
			method: 'DELETE',
		}),

	// Backup Operations
	triggerBackup: async (
		id: string,
		data?: TriggerDockerStackBackupRequest,
	): Promise<DockerStackBackup> =>
		fetchApi<DockerStackBackup>(`/docker-stacks/${id}/backup`, {
			method: 'POST',
			body: JSON.stringify(data ?? {}),
		}),

	listBackups: async (stackId: string): Promise<DockerStackBackup[]> => {
		const response = await fetchApi<DockerStackBackupListResponse>(
			`/docker-stacks/${stackId}/backups`,
		);
		return response.backups ?? [];
	},

	getBackup: async (id: string): Promise<DockerStackBackup> =>
		fetchApi<DockerStackBackup>(`/docker-stack-backups/${id}`),

	deleteBackup: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/docker-stack-backups/${id}`, {
			method: 'DELETE',
		}),

	// Restore Operations
	restoreBackup: async (
		backupId: string,
		data: RestoreDockerStackRequest,
	): Promise<DockerStackRestore> =>
		fetchApi<DockerStackRestore>(`/docker-stack-backups/${backupId}/restore`, {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	getRestore: async (id: string): Promise<DockerStackRestore> =>
		fetchApi<DockerStackRestore>(`/docker-stack-restores/${id}`),

	// Discovery
	discoverStacks: async (
		data: DiscoverDockerStacksRequest,
	): Promise<DiscoveredDockerStack[]> => {
		const response = await fetchApi<DiscoverDockerStacksResponse>(
			'/docker-stacks/discover',
			{
				method: 'POST',
				body: JSON.stringify(data),
			},
		);
		return response.stacks ?? [];
	},
};

// Docker Registry API
export const dockerRegistriesApi = {
	list: async (): Promise<DockerRegistry[]> => {
		const response =
			await fetchApi<DockerRegistriesResponse>('/docker-registries');
		return response.registries ?? [];
	},

	get: async (id: string): Promise<DockerRegistry> => {
		const response = await fetchApi<DockerRegistryResponse>(
			`/docker-registries/${id}`,
		);
		return response.registry;
	},

	create: async (data: CreateDockerRegistryRequest): Promise<DockerRegistry> =>
		fetchApi<DockerRegistry>('/docker-registries', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateDockerRegistryRequest,
	): Promise<DockerRegistry> =>
		fetchApi<DockerRegistry>(`/docker-registries/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/docker-registries/${id}`, {
			method: 'DELETE',
		}),

	getTypes: async (): Promise<DockerRegistryTypeInfo[]> => {
		const response = await fetchApi<DockerRegistryTypesResponse>(
			'/docker-registries/types',
		);
		return response.types ?? [];
	},

	getExpiringCredentials: async (): Promise<{
		registries: DockerRegistry[];
		warning_days: number;
	}> => fetchApi<ExpiringCredentialsResponse>('/docker-registries/expiring'),

	login: async (id: string): Promise<DockerLoginResult> => {
		const response = await fetchApi<DockerLoginResultResponse>(
			`/docker-registries/${id}/login`,
			{ method: 'POST' },
		);
		return response.result;
	},

	loginAll: async (): Promise<DockerLoginResult[]> => {
		const response = await fetchApi<DockerLoginAllResponse>(
			'/docker-registries/login-all',
			{ method: 'POST' },
		);
		return response.results ?? [];
	},

	healthCheck: async (id: string): Promise<DockerRegistryHealthCheck> => {
		const response = await fetchApi<DockerHealthCheckResponse>(
			`/docker-registries/${id}/health-check`,
			{ method: 'POST' },
		);
		return response.result;
	},

	healthCheckAll: async (): Promise<DockerRegistryHealthCheck[]> => {
		const response = await fetchApi<DockerHealthCheckAllResponse>(
			'/docker-registries/health-check-all',
			{ method: 'POST' },
		);
		return response.results ?? [];
	},

	rotateCredentials: async (
		id: string,
		data: RotateCredentialsRequest,
	): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/docker-registries/${id}/rotate-credentials`, {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	setDefault: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/docker-registries/${id}/set-default`, {
			method: 'POST',
		}),
};

// =============================================================================
// Komodo Integration API
// =============================================================================

export const komodoApi = {
	// Integrations
	listIntegrations: async (): Promise<KomodoIntegration[]> => {
		const response = await fetchApi<KomodoIntegrationsResponse>(
			'/integrations/komodo',
		);
		return response.integrations ?? [];
	},

	getIntegration: async (id: string): Promise<KomodoIntegrationResponse> =>
		fetchApi<KomodoIntegrationResponse>(`/integrations/komodo/${id}`),

	createIntegration: async (
		data: CreateKomodoIntegrationRequest,
	): Promise<KomodoIntegration> =>
		fetchApi<KomodoIntegration>('/integrations/komodo', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	updateIntegration: async (
		id: string,
		data: UpdateKomodoIntegrationRequest,
	): Promise<KomodoIntegration> =>
		fetchApi<KomodoIntegration>(`/integrations/komodo/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	deleteIntegration: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/integrations/komodo/${id}`, {
			method: 'DELETE',
		}),

	testConnection: async (id: string): Promise<KomodoConnectionTestResponse> =>
		fetchApi<KomodoConnectionTestResponse>(`/integrations/komodo/${id}/test`, {
			method: 'POST',
		}),

	syncIntegration: async (id: string): Promise<KomodoSyncResponse> =>
		fetchApi<KomodoSyncResponse>(`/integrations/komodo/${id}/sync`, {
			method: 'POST',
		}),

	discoverContainers: async (id: string): Promise<KomodoDiscoveryResult> =>
		fetchApi<KomodoDiscoveryResult>(`/integrations/komodo/${id}/discover`),

	// Containers
	listContainers: async (): Promise<KomodoContainer[]> => {
		const response = await fetchApi<KomodoContainersResponse>(
			'/integrations/komodo/containers',
		);
		return response.containers ?? [];
	},

	getContainer: async (id: string): Promise<KomodoContainer> =>
		fetchApi<KomodoContainer>(`/integrations/komodo/containers/${id}`),

	updateContainer: async (
		id: string,
		data: UpdateKomodoContainerRequest,
	): Promise<KomodoContainer> =>
		fetchApi<KomodoContainer>(`/integrations/komodo/containers/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	// Stacks
	listStacks: async (): Promise<KomodoStack[]> => {
		const response = await fetchApi<KomodoStacksResponse>(
			'/integrations/komodo/stacks',
		);
		return response.stacks ?? [];
	},

	getStack: async (id: string): Promise<KomodoStack> =>
		fetchApi<KomodoStack>(`/integrations/komodo/stacks/${id}`),

	// Webhook Events
	listWebhookEvents: async (): Promise<KomodoWebhookEvent[]> => {
		const response = await fetchApi<KomodoWebhookEventsResponse>(
			'/integrations/komodo/events',
		);
		return response.events ?? [];
	},
};

// Users API (Admin user management)
export const usersApi = {
	list: async (): Promise<UserWithMembership[]> => {
		const response = await fetchApi<UsersResponse>('/users');
		return response.users ?? [];
	},

	get: async (id: string): Promise<UserWithMembership> =>
		fetchApi<UserWithMembership>(`/users/${id}`),

	invite: async (data: InviteUserRequest): Promise<InviteUserResponse> =>
		fetchApi<InviteUserResponse>('/users/invite', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateUserRequest,
	): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/users/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/users/${id}`, {
			method: 'DELETE',
		}),

	resetPassword: async (
		id: string,
		data: ResetPasswordRequest,
	): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/users/${id}/reset-password`, {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	disable: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/users/${id}/disable`, {
			method: 'POST',
		}),

	enable: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/users/${id}/enable`, {
			method: 'POST',
		}),

	getActivity: async (
		id: string,
		limit = 50,
		offset = 0,
	): Promise<UserActivityLog[]> => {
		const response = await fetchApi<UserActivityLogsResponse>(
			`/users/${id}/activity?limit=${limit}&offset=${offset}`,
		);
		return response.activity_logs ?? [];
	},

	getOrgActivityLogs: async (
		limit = 50,
		offset = 0,
	): Promise<UserActivityLog[]> => {
		const response = await fetchApi<UserActivityLogsResponse>(
			`/users/activity?limit=${limit}&offset=${offset}`,
		);
		return response.activity_logs ?? [];
	},

	startImpersonation: async (
		id: string,
		data: ImpersonateUserRequest,
	): Promise<ImpersonateUserResponse> =>
		fetchApi<ImpersonateUserResponse>(`/users/${id}/impersonate`, {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	endImpersonation: async (): Promise<MessageResponse> =>
		fetchApi<MessageResponse>('/users/end-impersonation', {
			method: 'POST',
		}),

	getImpersonationLogs: async (
		limit = 50,
		offset = 0,
	): Promise<UserImpersonationLog[]> => {
		const response = await fetchApi<ImpersonationLogsResponse>(
			`/users/impersonation-logs?limit=${limit}&offset=${offset}`,
		);
		return response.impersonation_logs ?? [];
	},
};

// System health API (admin only)
export const systemHealthApi = {
	getHealth: async (): Promise<SystemHealthResponse> =>
		fetchApi<SystemHealthResponse>('/admin/health'),

	getHistory: async (): Promise<SystemHealthHistoryResponse> =>
		fetchApi<SystemHealthHistoryResponse>('/admin/health/history'),
};
// Server Setup API
export const setupApi = {
	getStatus: async (): Promise<ServerSetupStatus> =>
		fetchApi<ServerSetupStatus>('/setup/status'),

	testDatabase: async (): Promise<DatabaseTestResponse> =>
		fetchApi<DatabaseTestResponse>('/setup/database/test', { method: 'POST' }),

	createSuperuser: async (
		data: CreateSuperuserRequest,
	): Promise<CreateSuperuserResponse> =>
		fetchApi<CreateSuperuserResponse>('/setup/superuser', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	configureSMTP: async (data: SMTPSettings): Promise<MessageResponse> =>
		fetchApi<MessageResponse>('/setup/smtp', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	skipSMTP: async (): Promise<MessageResponse> =>
		fetchApi<MessageResponse>('/setup/smtp/skip', { method: 'POST' }),

	configureOIDC: async (data: OIDCSettings): Promise<MessageResponse> =>
		fetchApi<MessageResponse>('/setup/oidc', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	skipOIDC: async (): Promise<MessageResponse> =>
		fetchApi<MessageResponse>('/setup/oidc/skip', { method: 'POST' }),

	activateLicense: async (
		data: ActivateLicenseRequest,
	): Promise<ActivateLicenseResponse> =>
		fetchApi<ActivateLicenseResponse>('/setup/license/activate', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	startTrial: async (
		data: SetupStartTrialRequest,
	): Promise<SetupStartTrialResponse> =>
		fetchApi<SetupStartTrialResponse>('/setup/license/trial', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	createOrganization: async (
		data: CreateFirstOrgRequest,
	): Promise<MessageResponse> =>
		fetchApi<MessageResponse>('/setup/organization', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	completeSetup: async (): Promise<SetupCompleteResponse> =>
		fetchApi<SetupCompleteResponse>('/setup/complete', { method: 'POST' }),

	// Superuser re-run endpoints
	getRerunStatus: async (): Promise<RerunStatusResponse> =>
		fetchApi<RerunStatusResponse>('/setup/rerun'),

	rerunConfigureSMTP: async (data: SMTPSettings): Promise<MessageResponse> =>
		fetchApi<MessageResponse>('/setup/rerun/smtp', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	rerunConfigureOIDC: async (data: OIDCSettings): Promise<MessageResponse> =>
		fetchApi<MessageResponse>('/setup/rerun/oidc', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	rerunUpdateLicense: async (
		data: ActivateLicenseRequest,
	): Promise<ActivateLicenseResponse> =>
		fetchApi<ActivateLicenseResponse>('/setup/rerun/license', {
			method: 'POST',
			body: JSON.stringify(data),
		}),
};

export const licensesApi = {
	getCurrent: async (): Promise<License> => {
		const response = await fetchApi<LicenseResponse>('/licenses/current');
		return response.license;
	},

	getWarnings: async (): Promise<LicenseWarningsResponse> =>
		fetchApi<LicenseWarningsResponse>('/licenses/warnings'),

	getHistory: async (limit = 50, offset = 0): Promise<LicenseHistoryResponse> =>
		fetchApi<LicenseHistoryResponse>(
			`/licenses/history?limit=${limit}&offset=${offset}`,
		),

	validate: async (key: string): Promise<LicenseValidateResponse> =>
		fetchApi<LicenseValidateResponse>('/licenses/validate', {
			method: 'POST',
			body: JSON.stringify({ license_key: key }),
		}),

	activate: async (data: CreateLicenseKeyRequest): Promise<License> => {
		const response = await fetchApi<LicenseResponse>('/licenses/activate', {
			method: 'POST',
			body: JSON.stringify(data),
		});
		return response.license;
	},

	// Admin endpoints
	adminList: async (params?: {
		org_id?: string;
		tier?: string;
		status?: string;
		limit?: number;
		offset?: number;
	}): Promise<LicensesResponse> => {
		const query = new URLSearchParams();
		if (params?.org_id) query.set('org_id', params.org_id);
		if (params?.tier) query.set('tier', params.tier);
		if (params?.status) query.set('status', params.status);
		if (params?.limit) query.set('limit', params.limit.toString());
		if (params?.offset) query.set('offset', params.offset.toString());
		const queryStr = query.toString();
		return fetchApi<LicensesResponse>(
			`/admin/licenses${queryStr ? `?${queryStr}` : ''}`,
		);
	},

	adminGet: async (id: string): Promise<License> => {
		const response = await fetchApi<LicenseResponse>(`/admin/licenses/${id}`);
		return response.license;
	},

	adminUpdate: async (
		id: string,
		data: UpdateLicenseRequest,
	): Promise<License> => {
		const response = await fetchApi<LicenseResponse>(`/admin/licenses/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		});
		return response.license;
	},

	adminRevoke: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/admin/licenses/${id}`, {
			method: 'DELETE',
		}),

	getPurchaseUrl: async (): Promise<{ url: string }> =>
		fetchApi<{ url: string }>('/licenses/purchase-url'),
};

// License and Feature Flags API
export const licenseApi = {
	// Get current organization's license info
	getLicense: async (): Promise<LicenseInfoResponse> =>
		fetchApi<LicenseInfoResponse>('/license'),

	// Alias for backward compatibility
	getInfo: async (): Promise<LicenseInfo> => fetchApi<LicenseInfo>('/license'),

	// Check if a specific feature is enabled
	checkFeature: async (
		feature: LicenseFeature,
	): Promise<FeatureCheckResult> => {
		const response = await fetchApi<FeatureCheckResponse>(
			`/license/features/${feature}/check`,
		);
		return response.result;
	},

	// Get all available features with their tier requirements
	getFeatures: async (): Promise<FeatureInfo[]> => {
		const response = await fetchApi<FeaturesResponse>('/license/features');
		return response.features ?? [];
	},

	// Get all tier information
	getTiers: async (): Promise<TierInfo[]> => {
		const response = await fetchApi<TiersResponse>('/license/tiers');
		return response.tiers ?? [];
	},

	uploadLicense: async (licenseData: ArrayBuffer): Promise<AirGapLicenseInfo> =>
		fetchApi<AirGapLicenseInfo>('/system/license', {
			method: 'POST',
			headers: { 'Content-Type': 'application/octet-stream' },
			body: licenseData,
		}),

	activate: async (licenseKey: string): Promise<ActivateLicenseResponse> =>
		fetchApi<ActivateLicenseResponse>('/system/license/activate', {
			method: 'POST',
			body: JSON.stringify({ license_key: licenseKey }),
		}),
	deactivate: async (): Promise<{ status: string; tier: string }> =>
		fetchApi<{ status: string; tier: string }>('/system/license/deactivate', {
			method: 'POST',
		}),
	getPlans: async (): Promise<PricingPlan[]> =>
		fetchApi<PricingPlan[]>('/system/license/plans'),
	startTrial: async (data: StartTrialRequest): Promise<StartTrialResponse> =>
		fetchApi<StartTrialResponse>('/system/license/trial/start', {
			method: 'POST',
			body: JSON.stringify(data),
		}),
	checkTrial: async (email: string): Promise<TrialCheckResponse> =>
		fetchApi<TrialCheckResponse>(
			`/system/license/trial/check?email=${encodeURIComponent(email)}`,
		),
};

// Branding Settings API
export const brandingApi = {
	get: async (): Promise<BrandingSettings> =>
		fetchApi<BrandingSettings>('/branding'),

	update: async (
		data: UpdateBrandingSettingsRequest,
	): Promise<BrandingSettings> =>
		fetchApi<BrandingSettings>('/branding', {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	reset: async (): Promise<MessageResponse> =>
		fetchApi<MessageResponse>('/branding', {
			method: 'DELETE',
		}),

	// Get public branding settings (no auth required, for login page)
	getPublic: async (orgSlug: string): Promise<PublicBrandingSettings> =>
		fetch(`/api/public/branding/${orgSlug}`).then((res) => {
			if (!res.ok) {
				// Return default branding on error
				return {
					enabled: false,
					product_name: 'Keldris',
					logo_url: '',
					logo_dark_url: '',
					favicon_url: '',
					primary_color: '#4f46e5',
					secondary_color: '#64748b',
					accent_color: '#06b6d4',
					support_url: '',
					privacy_url: '',
					terms_url: '',
					login_title: '',
					login_subtitle: '',
					login_bg_url: '',
					hide_powered_by: false,
				};
			}
			return res.json();
		}),
};

export const webhooksApi = {
	// Event types
	listEventTypes: async (): Promise<WebhookEventTypesResponse> =>
		fetchApi<WebhookEventTypesResponse>('/webhooks/event-types'),

	// Endpoints
	listEndpoints: async (): Promise<WebhookEndpoint[]> => {
		const response = await fetchApi<WebhookEndpointsResponse>(
			'/webhooks/endpoints',
		);
		return response.endpoints ?? [];
	},

	getEndpoint: async (id: string): Promise<WebhookEndpoint> =>
		fetchApi<WebhookEndpoint>(`/webhooks/endpoints/${id}`),

	createEndpoint: async (
		data: CreateWebhookEndpointRequest,
	): Promise<WebhookEndpoint> =>
		fetchApi<WebhookEndpoint>('/webhooks/endpoints', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	updateEndpoint: async (
		id: string,
		data: UpdateWebhookEndpointRequest,
	): Promise<WebhookEndpoint> =>
		fetchApi<WebhookEndpoint>(`/webhooks/endpoints/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	deleteEndpoint: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/webhooks/endpoints/${id}`, {
			method: 'DELETE',
		}),

	testEndpoint: async (
		id: string,
		data?: TestWebhookRequest,
	): Promise<TestWebhookResponse> =>
		fetchApi<TestWebhookResponse>(`/webhooks/endpoints/${id}/test`, {
			method: 'POST',
			body: JSON.stringify(data ?? {}),
		}),

	// Deliveries
	listDeliveries: async (
		limit = 50,
		offset = 0,
	): Promise<WebhookDeliveriesResponse> =>
		fetchApi<WebhookDeliveriesResponse>(
			`/webhooks/deliveries?limit=${limit}&offset=${offset}`,
		),

	listEndpointDeliveries: async (
		endpointId: string,
		limit = 50,
		offset = 0,
	): Promise<WebhookDeliveriesResponse> =>
		fetchApi<WebhookDeliveriesResponse>(
			`/webhooks/endpoints/${endpointId}/deliveries?limit=${limit}&offset=${offset}`,
		),

	getDelivery: async (id: string): Promise<WebhookDelivery> =>
		fetchApi<WebhookDelivery>(`/webhooks/deliveries/${id}`),

	retryDelivery: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/webhooks/deliveries/${id}/retry`, {
			method: 'POST',
		}),
};

// Backup Hook Templates API
export const backupHookTemplatesApi = {
	list: async (params?: {
		service_type?: string;
		visibility?: string;
		tag?: string;
	}): Promise<BackupHookTemplate[]> => {
		const searchParams = new URLSearchParams();
		if (params?.service_type)
			searchParams.set('service_type', params.service_type);
		if (params?.visibility) searchParams.set('visibility', params.visibility);
		if (params?.tag) searchParams.set('tag', params.tag);
		const query = searchParams.toString();
		const endpoint = query
			? `/backup-hook-templates?${query}`
			: '/backup-hook-templates';
		const response = await fetchApi<BackupHookTemplatesResponse>(endpoint);
		return response.templates ?? [];
	},

	listBuiltIn: async (): Promise<BackupHookTemplate[]> => {
		const response = await fetchApi<BackupHookTemplatesResponse>(
			'/backup-hook-templates/built-in',
		);
		return response.templates ?? [];
	},

	get: async (id: string): Promise<BackupHookTemplate> =>
		fetchApi<BackupHookTemplate>(`/backup-hook-templates/${id}`),

	create: async (
		data: CreateBackupHookTemplateRequest,
	): Promise<BackupHookTemplate> =>
		fetchApi<BackupHookTemplate>('/backup-hook-templates', {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		id: string,
		data: UpdateBackupHookTemplateRequest,
	): Promise<BackupHookTemplate> =>
		fetchApi<BackupHookTemplate>(`/backup-hook-templates/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/backup-hook-templates/${id}`, {
			method: 'DELETE',
		}),

	apply: async (
		templateId: string,
		data: ApplyBackupHookTemplateRequest,
	): Promise<ApplyBackupHookTemplateResponse> =>
		fetchApi<ApplyBackupHookTemplateResponse>(
			`/backup-hook-templates/${templateId}/apply`,
			{
				method: 'POST',
				body: JSON.stringify(data),
			},
		),
};

// Container Backup Hooks API
export const containerHooksApi = {
	list: async (scheduleId: string): Promise<ContainerBackupHook[]> => {
		const response = await fetchApi<ContainerBackupHooksResponse>(
			`/schedules/${scheduleId}/hooks`,
		);
		return response.hooks ?? [];
	},

	get: async (scheduleId: string, id: string): Promise<ContainerBackupHook> =>
		fetchApi<ContainerBackupHook>(`/schedules/${scheduleId}/hooks/${id}`),

	create: async (
		scheduleId: string,
		data: CreateContainerBackupHookRequest,
	): Promise<ContainerBackupHook> =>
		fetchApi<ContainerBackupHook>(`/schedules/${scheduleId}/hooks`, {
			method: 'POST',
			body: JSON.stringify(data),
		}),

	update: async (
		scheduleId: string,
		id: string,
		data: UpdateContainerBackupHookRequest,
	): Promise<ContainerBackupHook> =>
		fetchApi<ContainerBackupHook>(`/schedules/${scheduleId}/hooks/${id}`, {
			method: 'PUT',
			body: JSON.stringify(data),
		}),

	delete: async (scheduleId: string, id: string): Promise<MessageResponse> =>
		fetchApi<MessageResponse>(`/schedules/${scheduleId}/hooks/${id}`, {
			method: 'DELETE',
		}),

	listTemplates: async (): Promise<ContainerHookTemplateInfo[]> => {
		const response = await fetchApi<ContainerHookTemplatesResponse>(
			'/container-hook-templates',
		);
		return response.templates ?? [];
	},

	listExecutions: async (
		backupId: string,
	): Promise<ContainerHookExecution[]> => {
		const response = await fetchApi<ContainerHookExecutionsResponse>(
			`/backups/${backupId}/hook-executions`,
		);
		return response.executions ?? [];
	},
};
