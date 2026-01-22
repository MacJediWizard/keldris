// Agent types
export type AgentStatus = 'pending' | 'active' | 'offline' | 'disabled';

export interface OSInfo {
	os: string;
	arch: string;
	hostname: string;
	version?: string;
}

export interface Agent {
	id: string;
	org_id: string;
	hostname: string;
	os_info?: OSInfo;
	last_seen?: string;
	status: AgentStatus;
	created_at: string;
	updated_at: string;
}

export interface CreateAgentRequest {
	hostname: string;
}

export interface CreateAgentResponse {
	id: string;
	hostname: string;
	api_key: string;
}

export interface RotateAPIKeyResponse {
	id: string;
	hostname: string;
	api_key: string;
}

// Repository types
export type RepositoryType =
	| 'local'
	| 's3'
	| 'b2'
	| 'sftp'
	| 'rest'
	| 'dropbox';

// Backend configuration interfaces
export interface LocalBackendConfig {
	path: string;
}

export interface S3BackendConfig {
	endpoint?: string;
	bucket: string;
	prefix?: string;
	region?: string;
	access_key_id: string;
	secret_access_key: string;
	use_ssl?: boolean;
}

export interface B2BackendConfig {
	bucket: string;
	prefix?: string;
	account_id: string;
	application_key: string;
}

export interface SFTPBackendConfig {
	host: string;
	port?: number;
	user: string;
	path: string;
	password?: string;
	private_key?: string;
}

export interface RestBackendConfig {
	url: string;
	username?: string;
	password?: string;
}

export interface DropboxBackendConfig {
	remote_name: string;
	path?: string;
	token?: string;
	app_key?: string;
	app_secret?: string;
}

export type BackendConfig =
	| LocalBackendConfig
	| S3BackendConfig
	| B2BackendConfig
	| SFTPBackendConfig
	| RestBackendConfig
	| DropboxBackendConfig;

export interface Repository {
	id: string;
	name: string;
	type: RepositoryType;
	escrow_enabled: boolean;
	created_at: string;
	updated_at: string;
}

export interface CreateRepositoryRequest {
	name: string;
	type: RepositoryType;
	config: BackendConfig;
	escrow_enabled?: boolean;
}

export interface CreateRepositoryResponse {
	repository: Repository;
	password: string;
}

export interface UpdateRepositoryRequest {
	name?: string;
	config?: Record<string, unknown>;
}

export interface TestRepositoryResponse {
	success: boolean;
	message: string;
}

export interface TestConnectionRequest {
	type: RepositoryType;
	config: BackendConfig;
}

export interface KeyRecoveryResponse {
	repository_id: string;
	repository_name: string;
	password: string;
}

// Schedule types
export interface RetentionPolicy {
	keep_last?: number;
	keep_hourly?: number;
	keep_daily?: number;
	keep_weekly?: number;
	keep_monthly?: number;
	keep_yearly?: number;
}

export interface BackupWindow {
	start?: string; // HH:MM format (e.g., "02:00")
	end?: string; // HH:MM format (e.g., "06:00")
}

export interface Schedule {
	id: string;
	agent_id: string;
	repository_id: string;
	name: string;
	cron_expression: string;
	paths: string[];
	excludes?: string[];
	retention_policy?: RetentionPolicy;
	bandwidth_limit_kb?: number; // Upload limit in KB/s
	backup_window?: BackupWindow; // Allowed backup time window
	excluded_hours?: number[]; // Hours (0-23) when backups should not run
	enabled: boolean;
	created_at: string;
	updated_at: string;
}

export interface CreateScheduleRequest {
	agent_id: string;
	repository_id: string;
	name: string;
	cron_expression: string;
	paths: string[];
	excludes?: string[];
	retention_policy?: RetentionPolicy;
	bandwidth_limit_kb?: number;
	backup_window?: BackupWindow;
	excluded_hours?: number[];
	enabled?: boolean;
}

export interface UpdateScheduleRequest {
	name?: string;
	cron_expression?: string;
	paths?: string[];
	excludes?: string[];
	retention_policy?: RetentionPolicy;
	bandwidth_limit_kb?: number;
	backup_window?: BackupWindow;
	excluded_hours?: number[];
	enabled?: boolean;
}

export interface RunScheduleResponse {
	backup_id: string;
	message: string;
}

// Backup types
export type BackupStatus = 'running' | 'completed' | 'failed' | 'canceled';

export interface Backup {
	id: string;
	schedule_id: string;
	agent_id: string;
	snapshot_id?: string;
	started_at: string;
	completed_at?: string;
	status: BackupStatus;
	size_bytes?: number;
	files_new?: number;
	files_changed?: number;
	error_message?: string;
	retention_applied: boolean;
	snapshots_removed?: number;
	snapshots_kept?: number;
	retention_error?: string;
	created_at: string;
}

// Auth types
export interface User {
	id: string;
	email: string;
	name: string;
	current_org_id?: string;
	current_org_role?: string;
}

// Organization types
export type OrgRole = 'owner' | 'admin' | 'member' | 'readonly';

export interface Organization {
	id: string;
	name: string;
	slug: string;
	created_at: string;
	updated_at: string;
}

export interface OrganizationWithRole {
	id: string;
	name: string;
	slug: string;
	role: OrgRole;
	created_at: string;
	updated_at: string;
}

export interface OrgMember {
	id: string;
	user_id: string;
	org_id: string;
	role: OrgRole;
	email: string;
	name: string;
	created_at: string;
	updated_at: string;
}

export interface OrgInvitation {
	id: string;
	org_id: string;
	org_name: string;
	email: string;
	role: OrgRole;
	invited_by: string;
	inviter_name: string;
	expires_at: string;
	accepted_at?: string;
	created_at: string;
}

export interface CreateOrgRequest {
	name: string;
	slug: string;
}

export interface UpdateOrgRequest {
	name?: string;
	slug?: string;
}

export interface SwitchOrgRequest {
	org_id: string;
}

export interface InviteMemberRequest {
	email: string;
	role: OrgRole;
}

export interface UpdateMemberRequest {
	role: OrgRole;
}

export interface OrgResponse {
	organization: Organization;
	role: string;
}

export interface OrganizationsResponse {
	organizations: OrganizationWithRole[];
}

export interface MembersResponse {
	members: OrgMember[];
}

export interface InvitationsResponse {
	invitations: OrgInvitation[];
}

export interface InviteResponse {
	message: string;
	token: string;
}

// API response wrappers
export interface AgentsResponse {
	agents: Agent[];
}

export interface RepositoriesResponse {
	repositories: Repository[];
}

export interface SchedulesResponse {
	schedules: Schedule[];
}

export interface BackupsResponse {
	backups: Backup[];
}

export interface ErrorResponse {
	error: string;
}

export interface MessageResponse {
	message: string;
}

// Snapshot types
export interface Snapshot {
	id: string;
	short_id: string;
	time: string;
	hostname: string;
	paths: string[];
	agent_id: string;
	repository_id: string;
	backup_id?: string;
	size_bytes?: number;
}

export interface SnapshotFile {
	name: string;
	path: string;
	type: 'file' | 'dir';
	size: number;
	mod_time: string;
}

export interface SnapshotsResponse {
	snapshots: Snapshot[];
}

export interface SnapshotFilesResponse {
	files: SnapshotFile[];
	snapshot_id: string;
	path: string;
	message?: string;
}

// Snapshot comparison types
export type SnapshotDiffChangeType = 'added' | 'removed' | 'modified';

export interface SnapshotDiffEntry {
	path: string;
	change_type: SnapshotDiffChangeType;
	type: 'file' | 'dir';
	old_size?: number;
	new_size?: number;
	size_change?: number;
}

export interface SnapshotDiffStats {
	files_added: number;
	files_removed: number;
	files_modified: number;
	dirs_added: number;
	dirs_removed: number;
	total_size_added: number;
	total_size_removed: number;
}

export interface SnapshotCompareResponse {
	snapshot_id_1: string;
	snapshot_id_2: string;
	snapshot_1?: Snapshot;
	snapshot_2?: Snapshot;
	stats: SnapshotDiffStats;
	changes: SnapshotDiffEntry[];
}

// Restore types
export type RestoreStatus =
	| 'pending'
	| 'running'
	| 'completed'
	| 'failed'
	| 'canceled';

export interface Restore {
	id: string;
	agent_id: string;
	repository_id: string;
	snapshot_id: string;
	target_path: string;
	include_paths?: string[];
	exclude_paths?: string[];
	status: RestoreStatus;
	started_at?: string;
	completed_at?: string;
	error_message?: string;
	created_at: string;
}

export interface CreateRestoreRequest {
	snapshot_id: string;
	agent_id: string;
	repository_id: string;
	target_path: string;
	include_paths?: string[];
	exclude_paths?: string[];
}

export interface RestoresResponse {
	restores: Restore[];
}

// Alert types
export type AlertType = 'agent_offline' | 'backup_sla' | 'storage_usage';
export type AlertSeverity = 'info' | 'warning' | 'critical';
export type AlertStatus = 'active' | 'acknowledged' | 'resolved';
export type ResourceType = 'agent' | 'schedule' | 'repository';

export interface Alert {
	id: string;
	org_id: string;
	rule_id?: string;
	type: AlertType;
	severity: AlertSeverity;
	status: AlertStatus;
	title: string;
	message: string;
	resource_type?: ResourceType;
	resource_id?: string;
	acknowledged_by?: string;
	acknowledged_at?: string;
	resolved_at?: string;
	metadata?: Record<string, unknown>;
	created_at: string;
	updated_at: string;
}

// Notification types
export type NotificationChannelType =
	| 'email'
	| 'slack'
	| 'webhook'
	| 'pagerduty';
export type NotificationEventType =
	| 'backup_success'
	| 'backup_failed'
	| 'agent_offline';
export type NotificationStatus = 'queued' | 'sent' | 'failed';

export interface NotificationChannel {
	id: string;
	org_id: string;
	name: string;
	type: NotificationChannelType;
	enabled: boolean;
	created_at: string;
	updated_at: string;
}

export interface AlertRuleConfig {
	offline_threshold_minutes?: number;
	max_hours_since_backup?: number;
	storage_usage_percent?: number;
	agent_ids?: string[];
	schedule_ids?: string[];
	repository_id?: string;
}

export interface AlertRule {
	id: string;
	org_id: string;
	name: string;
	type: AlertType;
	enabled: boolean;
	config: AlertRuleConfig;
	created_at: string;
	updated_at: string;
}

export interface EmailChannelConfig {
	host: string;
	port: number;
	username: string;
	password: string;
	from: string;
	tls: boolean;
}

export interface NotificationPreference {
	id: string;
	org_id: string;
	channel_id: string;
	event_type: NotificationEventType;
	enabled: boolean;
	created_at: string;
	updated_at: string;
}

export interface CreateAlertRuleRequest {
	name: string;
	type: AlertType;
	enabled: boolean;
	config: AlertRuleConfig;
}

export interface UpdateAlertRuleRequest {
	name?: string;
	enabled?: boolean;
	config?: AlertRuleConfig;
}

export interface AlertsResponse {
	alerts: Alert[];
}

export interface AlertRulesResponse {
	rules: AlertRule[];
}

export interface AlertCountResponse {
	count: number;
}

export interface NotificationLog {
	id: string;
	org_id: string;
	channel_id?: string;
	event_type: string;
	recipient: string;
	subject?: string;
	status: NotificationStatus;
	error_message?: string;
	sent_at?: string;
	created_at: string;
}

export interface CreateNotificationChannelRequest {
	name: string;
	type: NotificationChannelType;
	config: EmailChannelConfig | Record<string, unknown>;
}

export interface UpdateNotificationChannelRequest {
	name?: string;
	config?: EmailChannelConfig | Record<string, unknown>;
	enabled?: boolean;
}

export interface CreateNotificationPreferenceRequest {
	channel_id: string;
	event_type: NotificationEventType;
	enabled: boolean;
}

export interface UpdateNotificationPreferenceRequest {
	enabled: boolean;
}

export interface NotificationChannelsResponse {
	channels: NotificationChannel[];
}

export interface NotificationChannelWithPreferencesResponse {
	channel: NotificationChannel;
	preferences: NotificationPreference[];
}

export interface NotificationPreferencesResponse {
	preferences: NotificationPreference[];
}

export interface NotificationLogsResponse {
	logs: NotificationLog[];
}

// Audit log types
export type AuditAction =
	| 'login'
	| 'logout'
	| 'create'
	| 'read'
	| 'update'
	| 'delete'
	| 'backup'
	| 'restore';

export type AuditResult = 'success' | 'failure' | 'denied';

export interface AuditLog {
	id: string;
	org_id: string;
	user_id?: string;
	agent_id?: string;
	action: AuditAction;
	resource_type: string;
	resource_id?: string;
	result: AuditResult;
	ip_address?: string;
	user_agent?: string;
	details?: string;
	created_at: string;
}

export interface AuditLogFilter {
	action?: string;
	resource_type?: string;
	result?: string;
	start_date?: string;
	end_date?: string;
	search?: string;
	limit?: number;
	offset?: number;
}

export interface AuditLogsResponse {
	audit_logs: AuditLog[];
	total_count: number;
	limit: number;
	offset: number;
}

// Storage Stats types
export interface StorageStats {
	id: string;
	repository_id: string;
	total_size: number;
	total_file_count: number;
	raw_data_size: number;
	restore_size: number;
	dedup_ratio: number;
	space_saved: number;
	space_saved_pct: number;
	snapshot_count: number;
	collected_at: string;
	created_at: string;
}

export interface StorageStatsSummary {
	total_raw_size: number;
	total_restore_size: number;
	total_space_saved: number;
	avg_dedup_ratio: number;
	repository_count: number;
	total_snapshots: number;
}

export interface StorageGrowthPoint {
	date: string;
	raw_data_size: number;
	restore_size: number;
}

export interface RepositoryStatsResponse {
	stats: StorageStats;
	repository_name: string;
}

export interface RepositoryStatsListItem extends StorageStats {
	repository_name: string;
}

export interface RepositoryStatsListResponse {
	stats: RepositoryStatsListItem[];
}

export interface StorageGrowthResponse {
	growth: StorageGrowthPoint[];
}

export interface RepositoryGrowthResponse {
	repository_id: string;
	repository_name: string;
	growth: StorageGrowthPoint[];
}

export interface RepositoryHistoryResponse {
	repository_id: string;
	repository_name: string;
	history: StorageStats[];
}

// Verification types
export type VerificationStatus = 'pending' | 'running' | 'passed' | 'failed';
export type VerificationType = 'check' | 'check_read_data' | 'test_restore';

export interface VerificationDetails {
	errors_found?: string[];
	files_restored?: number;
	bytes_restored?: number;
	read_data_subset?: string;
}

export interface Verification {
	id: string;
	repository_id: string;
	type: VerificationType;
	snapshot_id?: string;
	started_at: string;
	completed_at?: string;
	status: VerificationStatus;
	duration_ms?: number;
	error_message?: string;
	details?: VerificationDetails;
	created_at: string;
}

export interface VerificationSchedule {
	id: string;
	repository_id: string;
	type: VerificationType;
	cron_expression: string;
	enabled: boolean;
	read_data_subset?: string;
	created_at: string;
	updated_at: string;
}

export interface VerificationStatusResponse {
	repository_id: string;
	last_verification?: Verification;
	next_scheduled_at?: string;
	consecutive_fails: number;
}

export interface TriggerVerificationRequest {
	type: VerificationType;
}

export interface CreateVerificationScheduleRequest {
	type: VerificationType;
	cron_expression: string;
	enabled?: boolean;
	read_data_subset?: string;
}

export interface UpdateVerificationScheduleRequest {
	cron_expression?: string;
	enabled?: boolean;
	read_data_subset?: string;
}

export interface VerificationsResponse {
	verifications: Verification[];
}

export interface VerificationSchedulesResponse {
	schedules: VerificationSchedule[];
}
