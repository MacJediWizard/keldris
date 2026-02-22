// Agent types
export type AgentStatus = 'pending' | 'active' | 'offline' | 'disabled';
export type HealthStatus = 'healthy' | 'warning' | 'critical' | 'unknown';

export interface OSInfo {
	os: string;
	arch: string;
	hostname: string;
	version?: string;
}

export interface HealthMetrics {
	cpu_usage: number;
	memory_usage: number;
	disk_usage: number;
	disk_free_bytes: number;
	disk_total_bytes: number;
	network_up: boolean;
	uptime_seconds: number;
	restic_version?: string;
	restic_available: boolean;
	issues?: HealthIssue[];
}

export interface HealthIssue {
	component: string;
	severity: HealthStatus;
	message: string;
	value?: number;
	threshold?: number;
}

// Network mount types
export type MountType = 'nfs' | 'smb' | 'cifs';
export type MountStatus = 'connected' | 'stale' | 'disconnected';
export type MountBehavior = 'skip' | 'fail';

export interface NetworkMount {
	path: string;
	type: MountType;
	remote: string;
	status: MountStatus;
	last_checked: string;
}

export interface Agent {
	id: string;
	org_id: string;
	hostname: string;
	os_info?: OSInfo;
	network_mounts?: NetworkMount[];
	last_seen?: string;
	status: AgentStatus;
	health_status: HealthStatus;
	health_metrics?: HealthMetrics;
	health_checked_at?: string;
	debug_mode: boolean;
	debug_mode_expires_at?: string;
	debug_mode_enabled_at?: string;
	debug_mode_enabled_by?: string;
	created_at: string;
	updated_at: string;
}

export interface AgentHealthHistory {
	id: string;
	agent_id: string;
	org_id: string;
	health_status: HealthStatus;
	cpu_usage?: number;
	memory_usage?: number;
	disk_usage?: number;
	disk_free_bytes?: number;
	disk_total_bytes?: number;
	network_up: boolean;
	restic_version?: string;
	restic_available: boolean;
	issues?: HealthIssue[];
	recorded_at: string;
	created_at: string;
}

export interface FleetHealthSummary {
	total_agents: number;
	healthy_count: number;
	warning_count: number;
	critical_count: number;
	unknown_count: number;
	active_count: number;
	offline_count: number;
	avg_cpu_usage: number;
	avg_memory_usage: number;
	avg_disk_usage: number;
}

// Agent Group types
export interface AgentGroup {
	id: string;
	org_id: string;
	name: string;
	description?: string;
	color?: string;
	agent_count: number;
	created_at: string;
	updated_at: string;
}

export interface AgentWithGroups extends Agent {
	groups?: AgentGroup[];
}

export interface CreateAgentGroupRequest {
	name: string;
	description?: string;
	color?: string;
}

export interface UpdateAgentGroupRequest {
	name?: string;
	description?: string;
	color?: string;
}

export interface AddAgentToGroupRequest {
	agent_id: string;
}

export interface AgentGroupsResponse {
	groups: AgentGroup[];
}

export interface AgentsWithGroupsResponse {
	agents: AgentWithGroups[];
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

// Agent Registration Code types
export interface CreateRegistrationCodeRequest {
	hostname?: string;
}

export interface CreateRegistrationCodeResponse {
	id: string;
	code: string;
	hostname?: string;
	expires_at: string;
}

export interface PendingRegistration {
	id: string;
	hostname?: string;
	code: string;
	expires_at: string;
	created_at: string;
	created_by: string;
}

export interface PendingRegistrationsResponse {
	registrations: PendingRegistration[];
}

export interface AgentStats {
	agent_id: string;
	total_backups: number;
	successful_backups: number;
	failed_backups: number;
	success_rate: number;
	total_size_bytes: number;
	last_backup_at?: string;
	next_scheduled_at?: string;
	schedule_count: number;
	uptime?: string;
}

export interface AgentStatsResponse {
	agent: Agent;
	stats: AgentStats;
}

export interface AgentBackupsResponse {
	backups: Backup[];
}

export interface AgentSchedulesResponse {
	schedules: Schedule[];
}

export interface AgentHealthHistoryResponse {
	history: AgentHealthHistory[];
}

// Debug mode types
export interface SetDebugModeRequest {
	enabled: boolean;
	duration_hours?: number; // 0 means no auto-disable, default is 4
}

export interface SetDebugModeResponse {
	debug_mode: boolean;
	debug_mode_expires_at?: string;
	message: string;
}

export interface DebugConfig {
	enabled: boolean;
	log_level: string;
	include_restic_output: boolean;
	log_file_operations: boolean;
}

// Agent Log types
export type LogLevel = 'debug' | 'info' | 'warn' | 'error';

export interface AgentLog {
	id: string;
	agent_id: string;
	org_id: string;
	level: LogLevel;
	message: string;
	component?: string;
	metadata?: Record<string, unknown>;
	timestamp: string;
	created_at: string;
}

export interface AgentLogsResponse {
	logs: AgentLog[];
	total_count: number;
	has_more: boolean;
}

export interface AgentLogFilter {
	level?: LogLevel;
	component?: string;
	search?: string;
	limit?: number;
	offset?: number;
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

export interface CloneRepositoryRequest {
	name: string;
	credentials: Record<string, unknown>;
	target_org_id?: string;
}

export interface CloneRepositoryResponse {
	repository: Repository;
	password: string;
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

export interface ScheduleRepository {
	id: string;
	schedule_id: string;
	repository_id: string;
	priority: number; // 0 = primary, 1+ = secondary by order
	enabled: boolean;
	created_at: string;
}

export interface ScheduleRepositoryRequest {
	repository_id: string;
	priority: number;
	enabled: boolean;
}

export type ReplicationStatusType = 'pending' | 'syncing' | 'synced' | 'failed';

export interface ReplicationStatus {
	id: string;
	schedule_id: string;
	source_repository_id: string;
	target_repository_id: string;
	last_snapshot_id?: string;
	last_sync_at?: string;
	status: ReplicationStatusType;
	error_message?: string;
	created_at: string;
	updated_at: string;
}

export interface BackupWindow {
	start?: string; // HH:MM format (e.g., "02:00")
	end?: string; // HH:MM format (e.g., "06:00")
}

export type CompressionLevel = 'off' | 'auto' | 'max';

export interface Schedule {
	id: string;
	agent_id: string;
	policy_id?: string; // Policy this schedule was created from
	name: string;
	cron_expression: string;
	paths: string[];
	excludes?: string[];
	retention_policy?: RetentionPolicy;
	bandwidth_limit_kb?: number; // Upload limit in KB/s
	backup_window?: BackupWindow; // Allowed backup time window
	excluded_hours?: number[]; // Hours (0-23) when backups should not run
	compression_level?: CompressionLevel; // Compression level: off, auto, max
	max_file_size_mb?: number; // Max file size in MB (0 = disabled)
	on_mount_unavailable?: MountBehavior; // Behavior when network mount unavailable
	classification_level?: string; // Data classification level
	classification_data_types?: string[]; // Data types: pii, phi, pci, proprietary, general
	enabled: boolean;
	repositories?: ScheduleRepository[];
	created_at: string;
	updated_at: string;
}

export interface CreateScheduleRequest {
	agent_id: string;
	repositories: ScheduleRepositoryRequest[];
	name: string;
	cron_expression: string;
	paths: string[];
	excludes?: string[];
	retention_policy?: RetentionPolicy;
	bandwidth_limit_kb?: number;
	backup_window?: BackupWindow;
	excluded_hours?: number[];
	compression_level?: CompressionLevel;
	max_file_size_mb?: number;
	on_mount_unavailable?: MountBehavior;
	enabled?: boolean;
}

export interface UpdateScheduleRequest {
	name?: string;
	cron_expression?: string;
	paths?: string[];
	excludes?: string[];
	retention_policy?: RetentionPolicy;
	repositories?: ScheduleRepositoryRequest[];
	bandwidth_limit_kb?: number;
	backup_window?: BackupWindow;
	excluded_hours?: number[];
	compression_level?: CompressionLevel;
	max_file_size_mb?: number;
	on_mount_unavailable?: MountBehavior;
	enabled?: boolean;
}

export interface RunScheduleResponse {
	backup_id: string;
	message: string;
}

// Clone schedule types
export interface CloneScheduleRequest {
	name?: string;
	target_agent_id?: string;
	target_repo_ids?: string[];
}

export interface BulkCloneScheduleRequest {
	schedule_id: string;
	target_agent_ids: string[];
	name_prefix?: string;
}

export interface BulkCloneResponse {
	schedules: Schedule[];
	errors?: string[];
}

// Dry run types
export interface DryRunFile {
	path: string;
	type: 'file' | 'dir';
	size: number;
	action: 'new' | 'changed' | 'unchanged';
}

export interface DryRunExcluded {
	path: string;
	reason: string;
}

export interface DryRunResponse {
	schedule_id: string;
	total_files: number;
	total_size: number;
	new_files: number;
	changed_files: number;
	unchanged_files: number;
	files_to_backup: DryRunFile[];
	excluded_files: DryRunExcluded[];
	message: string;
}

export interface ReplicationStatusResponse {
	replication_status: ReplicationStatus[];
}

// Policy types
export interface Policy {
	id: string;
	org_id: string;
	name: string;
	description?: string;
	paths?: string[];
	excludes?: string[];
	retention_policy?: RetentionPolicy;
	bandwidth_limit_kb?: number;
	backup_window?: BackupWindow;
	excluded_hours?: number[];
	cron_expression?: string;
	created_at: string;
	updated_at: string;
}

export interface CreatePolicyRequest {
	name: string;
	description?: string;
	paths?: string[];
	excludes?: string[];
	retention_policy?: RetentionPolicy;
	bandwidth_limit_kb?: number;
	backup_window?: BackupWindow;
	excluded_hours?: number[];
	cron_expression?: string;
}

export interface UpdatePolicyRequest {
	name?: string;
	description?: string;
	paths?: string[];
	excludes?: string[];
	retention_policy?: RetentionPolicy;
	bandwidth_limit_kb?: number;
	backup_window?: BackupWindow;
	excluded_hours?: number[];
	cron_expression?: string;
}

export interface ApplyPolicyRequest {
	agent_ids: string[];
	repository_id: string;
	schedule_name?: string;
}

export interface ApplyPolicyResponse {
	schedules_created: number;
	schedules: Schedule[];
}

export interface PoliciesResponse {
	policies: Policy[];
}

// Backup types
export type BackupStatus = 'running' | 'completed' | 'failed' | 'canceled';

export interface ExcludedLargeFile {
	path: string;
	size_bytes: number;
	size_mb: number;
}

export interface Backup {
	id: string;
	schedule_id: string;
	agent_id: string;
	repository_id?: string;
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
	pre_script_output?: string;
	pre_script_error?: string;
	post_script_output?: string;
	post_script_error?: string;
	excluded_large_files?: ExcludedLargeFile[]; // Files excluded due to size limit
	resumed: boolean;
	checkpoint_id?: string;
	original_backup_id?: string;
	classification_level?: string;
	classification_data_types?: string[];
	created_at: string;
}

// Backup Checkpoint types for resumable backups
export type CheckpointStatus = 'active' | 'completed' | 'canceled' | 'expired';

export interface BackupCheckpoint {
	id: string;
	schedule_id: string;
	agent_id: string;
	repository_id: string;
	backup_id?: string;
	status: CheckpointStatus;
	files_processed: number;
	bytes_processed: number;
	total_files?: number;
	total_bytes?: number;
	last_processed_path?: string;
	error_message?: string;
	resume_count: number;
	expires_at?: string;
	started_at: string;
	last_updated_at: string;
	created_at: string;
}

export interface ResumeInfo {
	checkpoint: BackupCheckpoint;
	progress_percent?: number;
	files_processed: number;
	bytes_processed: number;
	total_files?: number;
	total_bytes?: number;
	interrupted_at: string;
	interrupted_error?: string;
	resume_count: number;
	can_resume: boolean;
}

export interface IncompleteBackupsResponse {
	checkpoints: BackupCheckpoint[];
}

export interface ResumeBackupRequest {
	checkpoint_id: string;
}

export interface CancelCheckpointRequest {
	checkpoint_id: string;
}

// Backup Calendar types
export interface BackupCalendarDay {
	date: string;
	completed: number;
	failed: number;
	running: number;
	scheduled: number;
	backups?: Backup[];
}

export interface ScheduledBackup {
	schedule_id: string;
	schedule_name: string;
	agent_id: string;
	agent_name: string;
	scheduled_at: string;
}

export interface BackupCalendarResponse {
	days: BackupCalendarDay[];
	scheduled: ScheduledBackup[];
}

// Backup Script types
export type BackupScriptType =
	| 'pre_backup'
	| 'post_success'
	| 'post_failure'
	| 'post_always';

export interface BackupScript {
	id: string;
	schedule_id: string;
	type: BackupScriptType;
	script: string;
	timeout_seconds: number;
	fail_on_error: boolean;
	enabled: boolean;
	created_at: string;
	updated_at: string;
}

export interface CreateBackupScriptRequest {
	type: BackupScriptType;
	script: string;
	timeout_seconds?: number;
	fail_on_error?: boolean;
	enabled?: boolean;
}

export interface UpdateBackupScriptRequest {
	script?: string;
	timeout_seconds?: number;
	fail_on_error?: boolean;
	enabled?: boolean;
}

export interface BackupScriptsResponse {
	scripts: BackupScript[];
}

// Auth types
export type SupportedLanguage = 'en' | 'es' | 'pt';

export interface User {
	id: string;
	email: string;
	name: string;
	current_org_id?: string;
	current_org_role?: string;
	sso_groups?: string[];
	sso_groups_synced_at?: string;
	language?: SupportedLanguage;
}

export interface UpdateUserPreferencesRequest {
	language?: SupportedLanguage;
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
	is_locked?: boolean;
	locked_until?: string;
	remaining_days?: number;
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

// File diff types for comparing file content between snapshots
export interface FileDiffResponse {
	path: string;
	is_binary: boolean;
	change_type: 'added' | 'removed' | 'modified';
	old_size?: number;
	new_size?: number;
	old_hash?: string;
	new_hash?: string;
	unified_diff?: string;
	old_content?: string;
	new_content?: string;
	snapshot_id_1: string;
	snapshot_id_2: string;
}

// Snapshot comment types
export interface SnapshotComment {
	id: string;
	snapshot_id: string;
	user_id: string;
	user_name: string;
	user_email: string;
	content: string;
	created_at: string;
	updated_at: string;
}

export interface CreateSnapshotCommentRequest {
	content: string;
}

export interface SnapshotCommentsResponse {
	comments: SnapshotComment[];
}

// Snapshot Mount types
export type SnapshotMountStatus =
	| 'pending'
	| 'mounting'
	| 'mounted'
	| 'unmounting'
	| 'unmounted'
	| 'failed';

export interface SnapshotMount {
	id: string;
	agent_id: string;
	repository_id: string;
	snapshot_id: string;
	mount_path: string;
	status: SnapshotMountStatus;
	mounted_at?: string;
	expires_at?: string;
	unmounted_at?: string;
	error_message?: string;
	created_at: string;
}

export interface MountSnapshotRequest {
	agent_id: string;
	repository_id: string;
	timeout_minutes?: number;
}

export interface SnapshotMountsResponse {
	mounts: SnapshotMount[];
}

// Restore types
export type RestoreStatus =
	| 'pending'
	| 'running'
	| 'completed'
	| 'failed'
	| 'canceled'
	| 'uploading'
	| 'verifying';

// Cloud restore target types
export type CloudRestoreTargetType = 's3' | 'b2' | 'restic';

export interface CloudRestoreTarget {
	type: CloudRestoreTargetType;
	// S3/B2 configuration
	bucket?: string;
	prefix?: string;
	region?: string;
	endpoint?: string;
	access_key_id?: string;
	secret_access_key?: string;
	use_ssl?: boolean;
	// B2 specific
	account_id?: string;
	application_key?: string;
	// Restic repository configuration
	repository?: string;
	repository_password?: string;
}

export interface CloudRestoreProgress {
	total_files: number;
	total_bytes: number;
	uploaded_files: number;
	uploaded_bytes: number;
	current_file?: string;
	percent_complete: number;
	verified_checksum: boolean;
}

export interface PathMapping {
	source_path: string;
	target_path: string;
}

export interface RestoreProgress {
	files_restored: number;
	bytes_restored: number;
	total_files?: number;
	total_bytes?: number;
	current_file?: string;
}

export interface Restore {
	id: string;
	agent_id: string; // Target agent (where restore executes)
	source_agent_id?: string; // Source agent for cross-agent restores
	repository_id: string;
	snapshot_id: string;
	target_path: string;
	include_paths?: string[];
	exclude_paths?: string[];
	path_mappings?: PathMapping[]; // Path remapping for cross-agent restores
	status: RestoreStatus;
	progress?: RestoreProgress; // Real-time progress tracking
	is_cross_agent: boolean;
	started_at?: string;
	completed_at?: string;
	error_message?: string;
	created_at: string;
	// Cloud restore fields
	is_cloud_restore?: boolean;
	cloud_target?: CloudRestoreTarget;
	cloud_progress?: CloudRestoreProgress;
	cloud_target_location?: string;
	verify_upload?: boolean;
}

export interface CreateRestoreRequest {
	snapshot_id: string;
	agent_id: string; // Target agent (where restore executes)
	source_agent_id?: string; // Source agent for cross-agent restores
	repository_id: string;
	target_path: string;
	include_paths?: string[];
	exclude_paths?: string[];
	path_mappings?: PathMapping[]; // Path remapping for cross-agent restores
}

export interface CreateCloudRestoreRequest {
	snapshot_id: string;
	agent_id: string;
	repository_id: string;
	include_paths?: string[];
	exclude_paths?: string[];
	cloud_target: CloudRestoreTarget;
	verify_upload?: boolean;
}

export interface RestorePreviewRequest {
	snapshot_id: string;
	agent_id: string; // Target agent
	source_agent_id?: string; // Source agent for cross-agent restores
	repository_id: string;
	include_paths?: string[];
	exclude_paths?: string[];
	cloud_target: CloudRestoreTarget;
	verify_upload?: boolean;
}

export interface RestorePreviewRequest {
	snapshot_id: string;
	agent_id: string; // Target agent
	source_agent_id?: string; // Source agent for cross-agent restores
	repository_id: string;
	target_path: string;
	include_paths?: string[];
	exclude_paths?: string[];
	path_mappings?: PathMapping[];
}

export interface RestorePreviewFile {
	path: string;
	type: 'file' | 'dir';
	size: number;
	mod_time: string;
	has_conflict: boolean;
}

export interface RestorePreview {
	snapshot_id: string;
	target_path: string;
	total_files: number;
	total_dirs: number;
	total_size: number;
	conflict_count: number;
	files: RestorePreviewFile[];
	disk_space_needed: number;
	selected_paths?: string[];
	selected_size?: number;
}

export interface RestoresResponse {
	restores: Restore[];
}

// Alert types
export type AlertType =
	| 'agent_offline'
	| 'backup_sla'
	| 'storage_usage'
	| 'agent_health_warning'
	| 'agent_health_critical';
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

// SSO Group Mapping types
export interface SSOGroupMapping {
	id: string;
	org_id: string;
	oidc_group_name: string;
	role: OrgRole;
	auto_create_org: boolean;
	created_at: string;
	updated_at: string;
}

// Maintenance Window types
export interface MaintenanceWindow {
	id: string;
	org_id: string;
	title: string;
	message?: string;
	starts_at: string;
	ends_at: string;
	notify_before_minutes: number;
	notification_sent: boolean;
	read_only: boolean;
	countdown_start_minutes: number;
	emergency_override: boolean;
	overridden_by?: string;
	overridden_at?: string;
	created_by?: string;
	created_at: string;
	updated_at: string;
}

export interface CreateSSOGroupMappingRequest {
	oidc_group_name: string;
	role: OrgRole;
	auto_create_org?: boolean;
}

export interface UpdateSSOGroupMappingRequest {
	role?: OrgRole;
	auto_create_org?: boolean;
}

export interface SSOGroupMappingResponse {
	mapping: SSOGroupMapping;
}

export interface SSOGroupMappingsResponse {
	mappings: SSOGroupMapping[];
}

export interface SSOSettings {
	default_role: OrgRole | null;
	auto_create_orgs: boolean;
}

export interface UpdateSSOSettingsRequest {
	default_role?: OrgRole | null;
	auto_create_orgs?: boolean;
}

export interface UserSSOGroups {
	user_id: string;
	oidc_groups: string[];
	synced_at: string | null;
}

// Exclude Pattern types
export type ExcludePatternCategory =
	| 'os'
	| 'ide'
	| 'language'
	| 'build'
	| 'cache'
	| 'temp'
	| 'logs'
	| 'security'
	| 'database'
	| 'container';

export interface ExcludePattern {
	id: string;
	org_id?: string;
	name: string;
	description?: string;
	patterns: string[];
	category: ExcludePatternCategory;
	is_builtin: boolean;
	created_at: string;
	updated_at: string;
}

export interface CreateMaintenanceWindowRequest {
	title: string;
	message?: string;
	starts_at: string;
	ends_at: string;
	notify_before_minutes?: number;
	read_only?: boolean;
	countdown_start_minutes?: number;
}

export interface UpdateMaintenanceWindowRequest {
	title?: string;
	message?: string;
	starts_at?: string;
	ends_at?: string;
	notify_before_minutes?: number;
	read_only?: boolean;
	countdown_start_minutes?: number;
}

export interface EmergencyOverrideRequest {
	override: boolean;
}

export interface MaintenanceWindowsResponse {
	maintenance_windows: MaintenanceWindow[];
}

export interface ActiveMaintenanceResponse {
	active: MaintenanceWindow | null;
	upcoming: MaintenanceWindow | null;
	read_only_mode: boolean;
	show_countdown: boolean;
	countdown_to?: string;
}

// DR Runbook types
export type DRRunbookStatus = 'active' | 'draft' | 'archived';

export interface DRRunbookStep {
	order: number;
	title: string;
	description: string;
	estimated_minutes?: number;
	requires_confirmation?: boolean;
}

export interface DRRunbookContact {
	name: string;
	role: string;
	email?: string;
	phone?: string;
}

export interface DRRunbook {
	id: string;
	org_id: string;
	name: string;
	description?: string;
	scenario: string;
	steps: DRRunbookStep[];
	contacts?: DRRunbookContact[];
	status: DRRunbookStatus;
	estimated_rto_minutes?: number;
	estimated_rpo_minutes?: number;
	last_tested_at?: string;
	last_test_result?: string;
	created_at: string;
	updated_at: string;
}

export interface BuiltInPattern {
	name: string;
	description: string;
	patterns: string[];
	category: ExcludePatternCategory;
}

export interface CategoryInfo {
	id: ExcludePatternCategory;
	name: string;
	description: string;
	icon: string;
}

export interface CreateExcludePatternRequest {
	name: string;
	description?: string;
	patterns: string[];
	category: ExcludePatternCategory;
}

export interface UpdateExcludePatternRequest {
	name?: string;
	description?: string;
	patterns?: string[];
	category?: ExcludePatternCategory;
}

export interface ExcludePatternsResponse {
	patterns: ExcludePattern[];
}

export interface BuiltInPatternsResponse {
	patterns: BuiltInPattern[];
}

export interface CategoriesResponse {
	categories: CategoryInfo[];
}

export interface CreateDRRunbookRequest {
	name: string;
	description?: string;
	scenario?: string;
	steps?: DRRunbookStep[];
	contacts?: DRRunbookContact[];
	schedule_id?: string;
	estimated_rto_minutes?: number;
	estimated_rpo_minutes?: number;
}

export interface UpdateDRRunbookRequest {
	name?: string;
	description?: string;
	scenario?: string;
	steps?: DRRunbookStep[];
	contacts?: DRRunbookContact[];
	estimated_rto_minutes?: number;
	estimated_rpo_minutes?: number;
}

export interface DRRunbooksResponse {
	runbooks: DRRunbook[];
}

export interface DRRunbookRenderResponse {
	runbook: DRRunbook;
	content: string;
}

// DR Test types
export type DRTestStatus =
	| 'pending'
	| 'running'
	| 'completed'
	| 'passed'
	| 'failed'
	| 'skipped';

export interface DRTest {
	id: string;
	org_id: string;
	runbook_id: string;
	runbook_name?: string;
	started_at?: string;
	completed_at?: string;
	status: DRTestStatus;
	actual_rto_minutes?: number;
	actual_rpo_minutes?: number;
	notes?: string;
	tested_by?: string;
	created_at: string;
}

export interface DRTestSchedule {
	id: string;
	org_id: string;
	runbook_id: string;
	runbook_name?: string;
	cron_expression: string;
	enabled: boolean;
	last_run_at?: string;
	next_run_at?: string;
	created_at: string;
	updated_at: string;
}

export interface RunDRTestRequest {
	runbook_id: string;
	notes?: string;
}

export interface UpdateDRTestRequest {
	status?: DRTestStatus;
	actual_rto_minutes?: number;
	actual_rpo_minutes?: number;
	notes?: string;
}

export interface CreateDRTestScheduleRequest {
	runbook_id: string;
	cron_expression: string;
	enabled?: boolean;
}

export interface UpdateDRTestScheduleRequest {
	cron_expression?: string;
	enabled?: boolean;
}

export interface DRTestsResponse {
	tests: DRTest[];
}

export interface DRTestSchedulesResponse {
	schedules: DRTestSchedule[];
}

// DR Status for dashboard
export interface DRStatus {
	total_runbooks: number;
	active_runbooks: number;
	tested_runbooks: number;
	untested_runbooks: number;
	overdue_runbooks: number;
	tests_last_30_days: number;
	pass_rate: number;
	last_test?: DRTest;
	last_test_at?: string;
	next_test_at?: string;
	upcoming_tests: DRTestSchedule[];
}

// Tag types
export interface Tag {
	id: string;
	org_id: string;
	name: string;
	color: string;
	created_at: string;
	updated_at: string;
}

export interface CreateTagRequest {
	name: string;
	color?: string;
}

export interface UpdateTagRequest {
	name?: string;
	color?: string;
}

export interface AssignTagsRequest {
	tag_ids: string[];
}

export interface TagsResponse {
	tags: Tag[];
}

// Search types
export type SearchResultType =
	| 'agent'
	| 'backup'
	| 'snapshot'
	| 'schedule'
	| 'repository';

export interface SearchResult {
	type: SearchResultType;
	id: string;
	name: string;
	description?: string;
	status?: string;
	created_at: string;
}

export interface SearchFilter {
	q: string;
	types?: string[];
	status?: string;
	tag_ids?: string[];
	date_from?: string;
	date_to?: string;
	size_min?: number;
	size_max?: number;
	limit?: number;
}

export interface SearchResponse {
	results: SearchResult[];
	query: string;
	total: number;
}

// Dashboard Metrics types
export interface DashboardStats {
	agent_total: number;
	agent_online: number;
	agent_offline: number;
	backup_total: number;
	backup_running: number;
	backup_failed_24h: number;
	repository_count: number;
	schedule_count: number;
	schedule_enabled: number;
	total_backup_size: number;
	total_raw_size: number;
	total_space_saved: number;
	avg_dedup_ratio: number;
	success_rate_7d: number;
	success_rate_30d: number;
}

export interface BackupSuccessRate {
	period: string;
	total: number;
	successful: number;
	failed: number;
	success_percent: number;
}

export interface BackupSuccessRatesResponse {
	rate_7d: BackupSuccessRate;
	rate_30d: BackupSuccessRate;
}

export interface StorageGrowthTrend {
	date: string;
	total_size: number;
	raw_size: number;
	snapshot_count: number;
}

export interface StorageGrowthTrendResponse {
	trend: StorageGrowthTrend[];
}

export interface BackupDurationTrend {
	date: string;
	avg_duration_ms: number;
	max_duration_ms: number;
	min_duration_ms: number;
	backup_count: number;
}

export interface BackupDurationTrendResponse {
	trend: BackupDurationTrend[];
}

export interface DailyBackupStats {
	date: string;
	total: number;
	successful: number;
	failed: number;
	total_size: number;
}

export interface DailyBackupStatsResponse {
	stats: DailyBackupStats[];
}

// Report types
export type ReportFrequency = 'daily' | 'weekly' | 'monthly';
export type ReportStatus = 'sent' | 'failed' | 'preview';

export interface ReportSchedule {
	id: string;
	org_id: string;
	name: string;
	frequency: ReportFrequency;
	recipients: string[];
	channel_id?: string;
	timezone: string;
	enabled: boolean;
	last_sent_at?: string;
	created_at: string;
	updated_at: string;
}

export interface ReportBackupSummary {
	total_backups: number;
	successful_backups: number;
	failed_backups: number;
	success_rate: number;
	total_data_backed: number;
	schedules_active: number;
}

export interface ReportStorageSummary {
	total_raw_size: number;
	total_restore_size: number;
	space_saved: number;
	space_saved_pct: number;
	repository_count: number;
	total_snapshots: number;
}

export interface ReportAgentSummary {
	total_agents: number;
	active_agents: number;
	offline_agents: number;
	pending_agents: number;
}

export interface ReportAlertSummary {
	total_alerts: number;
	critical_alerts: number;
	warning_alerts: number;
	acknowledged_alerts: number;
	resolved_alerts: number;
}

export interface ReportIssue {
	type: string;
	severity: string;
	title: string;
	description: string;
	occurred_at: string;
}

export interface ReportData {
	backup_summary: ReportBackupSummary;
	storage_summary: ReportStorageSummary;
	agent_summary: ReportAgentSummary;
	alert_summary: ReportAlertSummary;
	top_issues?: ReportIssue[];
}

export interface ReportHistory {
	id: string;
	org_id: string;
	schedule_id?: string;
	report_type: string;
	period_start: string;
	period_end: string;
	recipients: string[];
	status: ReportStatus;
	error_message?: string;
	report_data?: ReportData;
	sent_at?: string;
	created_at: string;
}

export interface CreateReportScheduleRequest {
	name: string;
	frequency: ReportFrequency;
	recipients: string[];
	channel_id?: string;
	timezone?: string;
	enabled?: boolean;
}

export interface UpdateReportScheduleRequest {
	name?: string;
	frequency?: ReportFrequency;
	recipients?: string[];
	channel_id?: string;
	timezone?: string;
	enabled?: boolean;
}

export interface ReportSchedulesResponse {
	schedules: ReportSchedule[];
}

export interface ReportHistoryResponse {
	history: ReportHistory[];
}

export interface ReportPreviewResponse {
	data: ReportData;
	period: {
		start: string;
		end: string;
	};
}

// Onboarding types
export type OnboardingStep =
	| 'welcome'
	| 'organization'
	| 'smtp'
	| 'repository'
	| 'agent'
	| 'schedule'
	| 'verify'
	| 'complete';

export interface OnboardingStatus {
	needs_onboarding: boolean;
	current_step: OnboardingStep;
	completed_steps: OnboardingStep[];
	skipped: boolean;
	is_complete: boolean;
}

// File History types
export interface FileVersion {
	snapshot_id: string;
	short_id: string;
	snapshot_time: string;
	size: number;
	mod_time: string;
}

export interface FileHistoryResponse {
	file_path: string;
	agent_id: string;
	repository_id: string;
	agent_name: string;
	repo_name: string;
	versions: FileVersion[];
	message?: string;
}

export interface FileHistoryParams {
	path: string;
	agent_id: string;
	repository_id: string;
}

// File Search types
export interface FileSearchResult {
	snapshot_id: string;
	snapshot_time: string;
	hostname: string;
	file_name: string;
	file_path: string;
	file_size: number;
	file_type: string;
	mod_time: string;
}

export interface SnapshotFileGroup {
	snapshot_id: string;
	snapshot_time: string;
	hostname: string;
	file_count: number;
	files: FileSearchResult[];
}

export interface FileSearchResponse {
	query: string;
	agent_id: string;
	repository_id: string;
	total_count: number;
	snapshot_count: number;
	snapshots: SnapshotFileGroup[];
	message?: string;
}

export interface FileSearchParams {
	q: string;
	agent_id: string;
	repository_id: string;
	path?: string;
	snapshot_ids?: string;
	date_from?: string;
	date_to?: string;
	size_min?: number;
	size_max?: number;
	limit?: number;
}

// Cost Estimation types
export interface StoragePricing {
	id: string;
	org_id: string;
	repository_type: RepositoryType;
	storage_per_gb_month: number;
	egress_per_gb: number;
	operations_per_k: number;
	provider_name?: string;
	provider_description?: string;
	created_at: string;
	updated_at: string;
}

export interface CostEstimate {
	repository_id: string;
	repository_name: string;
	repository_type: RepositoryType;
	storage_size_bytes: number;
	storage_size_gb: number;
	monthly_cost: number;
	yearly_cost: number;
	cost_per_gb: number;
	pricing: {
		storage_per_gb_month: number;
		egress_per_gb: number;
		operations_per_k: number;
		provider_name: string;
		provider_description: string;
	};
	estimated_at: string;
}

export interface CostForecast {
	period: string;
	months: number;
	projected_size_gb: number;
	projected_cost: number;
	growth_rate: number;
}

export interface CostSummary {
	total_monthly_cost: number;
	total_yearly_cost: number;
	total_storage_size_gb: number;
	repository_count: number;
	by_type: Record<string, number>;
	repositories: CostEstimate[];
	forecasts: CostForecast[];
	estimated_at: string;
}

export interface CostAlert {
	id: string;
	org_id: string;
	name: string;
	monthly_threshold: number;
	enabled: boolean;
	notify_on_exceed: boolean;
	notify_on_forecast: boolean;
	forecast_months: number;
	last_triggered_at?: string;
	created_at: string;
	updated_at: string;
}

export interface CostEstimateRecord {
	id: string;
	org_id: string;
	repository_id: string;
	storage_size_bytes: number;
	monthly_cost: number;
	yearly_cost: number;
	cost_per_gb: number;
	estimated_at: string;
	created_at: string;
}

export interface DefaultPricing {
	storage_per_gb_month: number;
	egress_per_gb: number;
	operations_per_k: number;
	provider_name: string;
	provider_description: string;
}

export interface CreateStoragePricingRequest {
	repository_type: RepositoryType;
	storage_per_gb_month: number;
	egress_per_gb?: number;
	operations_per_k?: number;
	provider_name?: string;
	provider_description?: string;
}

export interface UpdateStoragePricingRequest {
	storage_per_gb_month?: number;
	egress_per_gb?: number;
	operations_per_k?: number;
	provider_name?: string;
	provider_description?: string;
}

export interface CreateCostAlertRequest {
	name: string;
	monthly_threshold: number;
	enabled?: boolean;
	notify_on_exceed?: boolean;
	notify_on_forecast?: boolean;
	forecast_months?: number;
}

export interface UpdateCostAlertRequest {
	name?: string;
	monthly_threshold?: number;
	enabled?: boolean;
	notify_on_exceed?: boolean;
	notify_on_forecast?: boolean;
	forecast_months?: number;
}

export interface StoragePricingResponse {
	pricing: StoragePricing[];
}

export interface DefaultPricingResponse {
	defaults: Record<string, DefaultPricing>;
}

export interface RepositoryCostsResponse {
	repositories: CostEstimate[];
}

export interface RepositoryCostResponse {
	estimate: CostEstimate;
	forecasts: CostForecast[];
}

export interface CostForecastResponse {
	forecasts: CostForecast[];
	current_storage_gb: number;
	current_monthly_cost: number;
	monthly_growth_rate: number;
}

export interface CostHistoryResponse {
	estimates: CostEstimateRecord[];
	storage_growth: StorageGrowthPoint[];
}

export interface CostAlertsResponse {
	alerts: CostAlert[];
}

// Repository Import types
export interface VerifyImportAccessRequest {
	type: RepositoryType;
	config: BackendConfig;
	password: string;
}

export interface VerifyImportAccessResponse {
	success: boolean;
	message: string;
}

export interface ImportPreviewRequest {
	type: RepositoryType;
	config: BackendConfig;
	password: string;
}

export interface SnapshotPreview {
	id: string;
	short_id: string;
	time: string;
	hostname: string;
	username: string;
	paths: string[];
	tags?: string[];
}

export interface ImportPreviewResponse {
	snapshot_count: number;
	snapshots: SnapshotPreview[];
	hostnames: string[];
	total_size: number;
	total_file_count: number;
}

export interface ImportRepositoryRequest {
	name: string;
	type: RepositoryType;
	config: BackendConfig;
	password: string;
	escrow_enabled?: boolean;
	snapshot_ids?: string[];
	hostnames?: string[];
	agent_id?: string;
}

export interface ImportRepositoryResponse {
	repository: Repository;
	snapshots_imported: number;
}

// Changelog types
export interface ChangelogEntry {
	version: string;
	date: string;
	added?: string[];
	changed?: string[];
	deprecated?: string[];
	removed?: string[];
	fixed?: string[];
	security?: string[];
	is_unreleased?: boolean;
}

export interface ChangelogResponse {
	entries: ChangelogEntry[];
	current_version: string;
}

// Server Log types
export type ServerLogLevel = 'debug' | 'info' | 'warn' | 'error' | 'fatal';

export interface ServerLogEntry {
	timestamp: string;
	level: ServerLogLevel;
	message: string;
	component?: string;
	fields?: Record<string, unknown>;
}

export interface ServerLogFilter {
	level?: ServerLogLevel;
	component?: string;
	search?: string;
	start_time?: string;
	end_time?: string;
	limit?: number;
	offset?: number;
}

export interface ServerLogsResponse {
	logs: ServerLogEntry[];
	total_count: number;
	limit: number;
	offset: number;
}

export interface ServerLogComponentsResponse {
	components: string[];
}

// Classification types
export type ClassificationLevel =
	| 'public'
	| 'internal'
	| 'confidential'
	| 'restricted';
export type DataType = 'pii' | 'phi' | 'pci' | 'proprietary' | 'general';

export interface ClassificationLevelInfo {
	value: ClassificationLevel;
	label: string;
	description: string;
	priority: number;
}

export interface DataTypeInfo {
	value: DataType;
	label: string;
	description: string;
}

export interface PathClassificationRule {
	id: string;
	org_id: string;
	pattern: string;
	level: ClassificationLevel;
	data_types: DataType[];
	description?: string;
	is_builtin: boolean;
	priority: number;
	enabled: boolean;
	created_at: string;
	updated_at: string;
}

// Snapshot Immutability types
export interface ImmutabilityLock {
	id: string;
	repository_id: string;
	snapshot_id: string;
	short_id: string;
	locked_at: string;
	locked_until: string;
	locked_by?: string;
	reason?: string;
	remaining_days: number;
	s3_object_lock_enabled: boolean;
	created_at: string;
}

export interface ImmutabilityStatus {
	is_locked: boolean;
	locked_until?: string;
	remaining_days?: number;
	reason?: string;
	locked_at?: string;
}

export interface RepositoryImmutabilitySettings {
	enabled: boolean;
	default_days?: number;
}

export interface CreateImmutabilityLockRequest {
	repository_id: string;
	snapshot_id: string;
	days: number;
	reason?: string;
	enable_s3_lock?: boolean;
}

export interface ExtendImmutabilityLockRequest {
	additional_days: number;
	reason?: string;
}

export interface UpdateRepositoryImmutabilitySettingsRequest {
	enabled: boolean;
	default_days?: number;
}

export interface ImmutabilityLocksResponse {
	locks: ImmutabilityLock[];
}

// Legal Hold types
export interface LegalHold {
	id: string;
	snapshot_id: string;
	reason: string;
	placed_by: string;
	placed_by_name: string;
	created_at: string;
	updated_at: string;
}

export interface CreatePathClassificationRuleRequest {
	pattern: string;
	level: ClassificationLevel;
	data_types?: DataType[];
	description?: string;
	priority?: number;
}

export interface UpdatePathClassificationRuleRequest {
	pattern?: string;
	level?: ClassificationLevel;
	data_types?: DataType[];
	description?: string;
	priority?: number;
	enabled?: boolean;
}

export interface SetScheduleClassificationRequest {
	level: ClassificationLevel;
	data_types?: DataType[];
}

export interface ClassificationSummary {
	total_schedules: number;
	total_backups: number;
	by_level: Record<string, number>;
	by_data_type: Record<string, number>;
	restricted_count: number;
	confidential_count: number;
	internal_count: number;
	public_count: number;
}

export interface ClassificationRulesResponse {
	rules: PathClassificationRule[];
}

export interface ClassificationLevelsResponse {
	levels: ClassificationLevelInfo[];
}

export interface DataTypesResponse {
	data_types: DataTypeInfo[];
}

export interface ScheduleClassificationSummary {
	id: string;
	name: string;
	level: ClassificationLevel;
	data_types: DataType[];
	paths: string[];
	agent_id: string;
}

export interface ComplianceReport {
	generated_at: string;
	org_id: string;
	summary: ClassificationSummary;
	schedules_by_level: Record<string, ScheduleClassificationSummary[]>;
}

export interface CreateLegalHoldRequest {
	reason: string;
}

export interface LegalHoldsResponse {
	legal_holds: LegalHold[];
}

// Geo-Replication types
export interface GeoRegion {
	code: string;
	name: string;
	display_name: string;
	latitude: number;
	longitude: number;
}

export interface GeoRegionPair {
	primary: GeoRegion;
	secondary: GeoRegion;
}

export type GeoReplicationStatusType =
	| 'pending'
	| 'syncing'
	| 'synced'
	| 'failed'
	| 'disabled';

export interface ReplicationLag {
	snapshots_behind: number;
	time_behind_hours: number;
	is_healthy: boolean;
	last_sync_at?: string;
}

export interface GeoReplicationConfig {
	id: string;
	source_repository_id: string;
	target_repository_id: string;
	source_region: GeoRegion;
	target_region: GeoRegion;
	enabled: boolean;
	status: GeoReplicationStatusType;
	last_snapshot_id?: string;
	last_sync_at?: string;
	last_error?: string;
	max_lag_snapshots: number;
	max_lag_duration_hours: number;
	alert_on_lag: boolean;
	replication_lag?: ReplicationLag;
	created_at: string;
	updated_at: string;
}

export interface GeoReplicationCreateRequest {
	source_repository_id: string;
	target_repository_id: string;
	source_region: string;
	target_region: string;
	max_lag_snapshots?: number;
	max_lag_duration_hours?: number;
	alert_on_lag?: boolean;
}

export interface GeoReplicationUpdateRequest {
	enabled?: boolean;
	max_lag_snapshots?: number;
	max_lag_duration_hours?: number;
	alert_on_lag?: boolean;
}

export interface GeoReplicationEvent {
	id: string;
	config_id: string;
	snapshot_id: string;
	status: GeoReplicationStatusType;
	started_at: string;
	completed_at?: string;
	duration_ms: number;
	bytes_copied?: number;
	error_message?: string;
	created_at: string;
}

export interface GeoReplicationSummary {
	total_configs: number;
	enabled_configs: number;
	synced_count: number;
	syncing_count: number;
	pending_count: number;
	failed_count: number;
}

export interface GeoReplicationRegionsResponse {
	regions: GeoRegion[];
	pairs: GeoRegionPair[];
}

export interface GeoReplicationConfigsResponse {
	configs: GeoReplicationConfig[];
}

export interface GeoReplicationEventsResponse {
	events: GeoReplicationEvent[];
}

export interface GeoReplicationSummaryResponse {
	summary: GeoReplicationSummary;
	regions: GeoRegion[];
}

export interface RepositoryReplicationStatusResponse {
	configured: boolean;
	config?: GeoReplicationConfig;
	message?: string;
}

// Agent Command types
export type CommandType = 'backup_now' | 'update' | 'restart' | 'diagnostics';
export type CommandStatus =
	| 'pending'
	| 'acknowledged'
	| 'running'
	| 'completed'
	| 'failed'
	| 'timed_out'
	| 'canceled';

export interface CommandPayload {
	schedule_id?: string;
	target_version?: string;
	diagnostic_types?: string[];
}

export interface CommandResult {
	output?: string;
	error?: string;
	diagnostics?: Record<string, unknown>;
	backup_id?: string;
}

export interface AgentCommand {
	id: string;
	agent_id: string;
	org_id: string;
	type: CommandType;
	status: CommandStatus;
	payload?: CommandPayload;
	result?: CommandResult;
	created_by?: string;
	created_by_name?: string;
	acknowledged_at?: string;
	started_at?: string;
	completed_at?: string;
	timeout_at: string;
	created_at: string;
	updated_at: string;
}

export interface CreateAgentCommandRequest {
	type: CommandType;
	payload?: CommandPayload;
}

export interface AgentCommandsResponse {
	commands: AgentCommand[];
}

// Config Export/Import types
export type ConfigType = 'agent' | 'schedule' | 'repository' | 'bundle';
export type ExportFormat = 'json' | 'yaml';
export type ConflictResolution = 'skip' | 'replace' | 'rename' | 'fail';
export type TemplateType = 'schedule' | 'agent' | 'repository' | 'bundle';
export type TemplateVisibility = 'private' | 'organization' | 'public';

export interface ExportMetadata {
	version: string;
	type: ConfigType;
	exported_at: string;
	exported_by?: string;
	description?: string;
}

export interface ExportBundleRequest {
	agent_ids?: string[];
	schedule_ids?: string[];
	repository_ids?: string[];
	format?: ExportFormat;
	description?: string;
}

export interface ImportConfigRequest {
	config: string;
	format?: ExportFormat;
	target_agent_id?: string;
	repository_mappings?: Record<string, string>;
	conflict_resolution?: ConflictResolution;
}

export interface ValidateImportRequest {
	config: string;
	format?: ExportFormat;
}

export interface ImportedItems {
	agent_count: number;
	agent_ids?: string[];
	schedule_count: number;
	schedule_ids?: string[];
	repository_count: number;
	repository_ids?: string[];
}

export interface SkippedItem {
	type: ConfigType;
	name: string;
	reason: string;
}

export interface ImportError {
	type: ConfigType;
	name: string;
	message: string;
}

export interface ImportResult {
	success: boolean;
	message: string;
	imported: ImportedItems;
	skipped?: SkippedItem[];
	errors?: ImportError[];
	warnings?: string[];
}

export interface ValidationError {
	field: string;
	message: string;
}

export interface Conflict {
	type: ConfigType;
	name: string;
	existing_id: string;
	existing_name: string;
	message: string;
}

export interface ValidationResult {
	valid: boolean;
	errors?: ValidationError[];
	warnings?: string[];
	conflicts?: Conflict[];
	suggestions?: string[];
}

// Config Template types
export interface ConfigTemplate {
	id: string;
	org_id: string;
	created_by_id: string;
	name: string;
	description?: string;
	type: TemplateType;
	visibility: TemplateVisibility;
	tags?: string[];
	config: Record<string, unknown>;
	usage_count: number;
	created_at: string;
	updated_at: string;
}

export interface CreateTemplateRequest {
	name: string;
	description?: string;
	config: string;
	visibility?: TemplateVisibility;
	tags?: string[];
}

export interface UpdateTemplateRequest {
	name?: string;
	description?: string;
	visibility?: TemplateVisibility;
	tags?: string[];
}

export interface UseTemplateRequest {
	target_agent_id?: string;
	repository_mappings?: Record<string, string>;
	conflict_resolution?: ConflictResolution;
}

export interface ConfigTemplatesResponse {
	templates: ConfigTemplate[];
}

// Rate Limit types
export interface RateLimitClientStats {
	client_ip: string;
	total_requests: number;
	rejected_count: number;
	last_request: string;
}

export interface EndpointRateLimitInfo {
	pattern: string;
	limit: number;
	period: string;
}

export interface RateLimitDashboardStats {
	default_limit: number;
	default_period: string;
	endpoint_configs: EndpointRateLimitInfo[];
	client_stats: RateLimitClientStats[];
	total_requests: number;
	total_rejected: number;
}

// Announcement types
export type AnnouncementType = 'info' | 'warning' | 'critical';

export interface Announcement {
	id: string;
	org_id: string;
	title: string;
	message?: string;
	type: AnnouncementType;
	dismissible: boolean;
	starts_at?: string;
	ends_at?: string;
	active: boolean;
	created_by?: string;
	created_at: string;
	updated_at: string;
}

export interface CreateAnnouncementRequest {
	title: string;
	message?: string;
	type: AnnouncementType;
	dismissible?: boolean;
	starts_at?: string;
	ends_at?: string;
	active?: boolean;
}

export interface UpdateAnnouncementRequest {
	title?: string;
	message?: string;
	type?: AnnouncementType;
	dismissible?: boolean;
	starts_at?: string;
	ends_at?: string;
	active?: boolean;
}

export interface AnnouncementsResponse {
	announcements: Announcement[];
}

// Agent Import types
export interface AgentImportPreviewEntry {
	row_number: number;
	hostname: string;
	group_name?: string;
	tags?: string[];
	config?: Record<string, string>;
	is_valid: boolean;
	errors?: string[];
}

export interface AgentImportPreviewResponse {
	total_rows: number;
	valid_rows: number;
	invalid_rows: number;
	entries: AgentImportPreviewEntry[];
	detected_groups: string[];
	detected_tags: string[];
}

export interface AgentImportJobResult {
	row_number: number;
	hostname: string;
	agent_id?: string;
	group_id?: string;
	group_name?: string;
	registration_code?: string;
	expires_at?: string;
	success: boolean;
	error_message?: string;
}

export interface AgentImportResponse {
	job_id: string;
	total_agents: number;
	imported_count: number;
	failed_count: number;
	results: AgentImportJobResult[];
	groups_created?: string[];
}

export interface AgentImportTemplateResponse {
	headers: string[];
	examples: string[][];
}

export interface AgentRegistrationScriptRequest {
	hostname: string;
	registration_code: string;
}

export interface AgentRegistrationScriptResponse {
	script: string;
	hostname: string;
	registration_code: string;
	expires_at: string;
}
