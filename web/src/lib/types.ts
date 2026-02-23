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

// Docker container info for agent display
export interface DockerContainerInfo {
	id: string;
	name: string;
	image: string;
	status: string;
	state: string; // running, paused, exited, etc.
}

// Docker volume info for agent display
export interface DockerVolumeInfo {
	name: string;
	driver: string;
	mountpoint?: string;
}

// Docker info for an agent
export interface DockerInfo {
	available: boolean;
	version?: string;
	api_version?: string;
	container_count: number;
	running_count: number;
	volume_count: number;
	containers?: DockerContainerInfo[];
	volumes?: DockerVolumeInfo[];
	error?: string;
	detected_at?: string;
}

export interface Agent {
	id: string;
	org_id: string;
	hostname: string;
	os_info?: OSInfo;
	docker_info?: DockerInfo;
	proxmox_info?: ProxmoxInfo;
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
	last_seen?: string;
	status: AgentStatus;
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
	host_key?: string;
	known_hosts_file?: string;
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
	config: Record<string, unknown>;
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

export type SchedulePriority = 1 | 2 | 3; // 1=high, 2=medium, 3=low

// Backup type determines what kind of backup to perform
export type BackupType = 'file' | 'docker' | 'proxmox';

// Docker backup options
export interface DockerBackupOptions {
	volume_ids?: string[]; // Specific volumes to backup (empty = all)
	container_ids?: string[]; // Specific container configs to backup (empty = all)
	pause_containers: boolean; // Pause containers during volume backup
	include_container_configs: boolean; // Backup container configurations
}

// Docker Compose stack backup configuration
export interface DockerStackScheduleConfig {
	stack_id: string;
	export_images: boolean;
	include_env_files: boolean;
	stop_for_backup: boolean;
}

// Proxmox backup options
export interface ProxmoxBackupOptions {
	connection_id?: string; // Proxmox connection ID
	vm_ids?: number[]; // Specific VMs to backup (empty = all)
	container_ids?: number[]; // Specific LXC containers to backup (empty = all)
	mode: 'snapshot' | 'suspend' | 'stop'; // Backup mode
	compress: '0' | 'gzip' | 'lzo' | 'zstd'; // Compression algorithm
	storage?: string; // Proxmox storage for temp backup
	max_wait?: number; // Max wait time in minutes
	include_ram: boolean; // Include RAM state (VMs only, requires snapshot mode)
	remove_after: boolean; // Remove from Proxmox after Restic backup
}

// Proxmox VM/container info
export interface ProxmoxVMInfo {
	vmid: number;
	name: string;
	type: 'qemu' | 'lxc';
	status: string;
	node: string;
	cpus: number;
	maxmem: number;
	maxdisk: number;
}

// Proxmox connection info on agent
export interface ProxmoxInfo {
	available: boolean;
	host?: string;
	node?: string;
	version?: string;
	vm_count: number;
	lxc_count: number;
	vms?: ProxmoxVMInfo[];
	connection_id?: string;
	error?: string;
	detected_at?: string;
}

// Proxmox connection configuration
export interface ProxmoxConnection {
	id: string;
	org_id: string;
	name: string;
	host: string;
	port: number;
	node: string;
	username: string;
	token_id?: string;
	has_token: boolean;
	verify_ssl: boolean;
	enabled: boolean;
	last_connected_at?: string;
	created_at: string;
	updated_at: string;
}

export interface Schedule {
	id: string;
	agent_id: string;
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
	priority: SchedulePriority; // Backup priority: 1=high, 2=medium, 3=low
	preemptible: boolean; // Can be preempted by higher priority backups
	docker_options?: DockerBackupOptions; // Docker-specific backup options
	docker_stack_config?: DockerStackScheduleConfig; // Docker stack backup configuration
	proxmox_options?: ProxmoxBackupOptions; // Proxmox-specific backup options
	enabled: boolean;
	repositories?: ScheduleRepository[];
	created_at: string;
	updated_at: string;
}

export interface CreateScheduleRequest {
	agent_id: string;
	repositories: ScheduleRepositoryRequest[];
	name: string;
	backup_type?: BackupType; // Type of backup: file (default) or docker
	cron_expression: string;
	paths?: string[]; // Required for file backups, optional for docker
	excludes?: string[];
	retention_policy?: RetentionPolicy;
	bandwidth_limit_kb?: number;
	backup_window?: BackupWindow;
	excluded_hours?: number[];
	compression_level?: CompressionLevel;
	max_file_size_mb?: number;
	on_mount_unavailable?: MountBehavior;
	priority?: SchedulePriority;
	preemptible?: boolean;
	docker_options?: DockerBackupOptions; // Docker-specific backup options
	docker_stack_config?: DockerStackScheduleConfig;
	proxmox_options?: ProxmoxBackupOptions; // Proxmox-specific backup options
	enabled?: boolean;
}

export interface UpdateScheduleRequest {
	name?: string;
	backup_type?: BackupType;
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
	priority?: SchedulePriority;
	preemptible?: boolean;
	docker_options?: DockerBackupOptions;
	proxmox_options?: ProxmoxBackupOptions;
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
export interface ReplicationStatusResponse {
	replication_status: ReplicationStatus[];
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
	container_pre_hook_output?: string;
	container_pre_hook_error?: string;
	container_post_hook_output?: string;
	container_post_hook_error?: string;
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

// Backup Hook Template types
export type BackupHookTemplateVisibility =
	| 'built_in'
	| 'private'
	| 'organization';

export interface BackupHookTemplateVariable {
	name: string;
	description: string;
	default: string;
	required: boolean;
	sensitive?: boolean;
}

export interface BackupHookTemplateScript {
	script: string;
	timeout_seconds: number;
	fail_on_error: boolean;
}

export interface BackupHookTemplateScripts {
	pre_backup?: BackupHookTemplateScript;
	post_success?: BackupHookTemplateScript;
	post_failure?: BackupHookTemplateScript;
	post_always?: BackupHookTemplateScript;
}

export interface BackupHookTemplate {
	id: string;
	org_id?: string;
	created_by_id?: string;
	name: string;
	description?: string;
	service_type: string;
	icon?: string;
	tags?: string[];
	variables?: BackupHookTemplateVariable[];
	scripts: BackupHookTemplateScripts;
	visibility: BackupHookTemplateVisibility;
	usage_count: number;
	created_at: string;
	updated_at: string;
}

export interface CreateBackupHookTemplateRequest {
	name: string;
	description?: string;
	service_type: string;
	icon?: string;
	tags?: string[];
	variables?: BackupHookTemplateVariable[];
	scripts: BackupHookTemplateScripts;
	visibility?: BackupHookTemplateVisibility;
}

export interface UpdateBackupHookTemplateRequest {
	name?: string;
	description?: string;
	service_type?: string;
	icon?: string;
	tags?: string[];
	variables?: BackupHookTemplateVariable[];
	scripts?: BackupHookTemplateScripts;
	visibility?: BackupHookTemplateVisibility;
}

export interface ApplyBackupHookTemplateRequest {
	schedule_id: string;
	variable_values?: Record<string, string>;
}

export interface ApplyBackupHookTemplateResponse {
	scripts: BackupScript[];
	message: string;
}

export interface BackupHookTemplatesResponse {
	templates: BackupHookTemplate[];
}

// Container Backup Hook types
export type ContainerHookType = 'pre_backup' | 'post_backup';

export type ContainerHookTemplate =
	| 'none'
	| 'postgres'
	| 'mysql'
	| 'mongodb'
	| 'redis'
	| 'elasticsearch';

export interface ContainerBackupHook {
	id: string;
	schedule_id: string;
	container_name: string;
	type: ContainerHookType;
	template: ContainerHookTemplate;
	command: string;
	working_dir?: string;
	user?: string;
	timeout_seconds: number;
	fail_on_error: boolean;
	enabled: boolean;
	description?: string;
	template_vars?: Record<string, string>;
	created_at: string;
	updated_at: string;
}

export interface CreateContainerBackupHookRequest {
	container_name: string;
	type: ContainerHookType;
	template?: ContainerHookTemplate;
	command?: string;
	working_dir?: string;
	user?: string;
	timeout_seconds?: number;
	fail_on_error?: boolean;
	enabled?: boolean;
	description?: string;
	template_vars?: Record<string, string>;
}

export interface UpdateContainerBackupHookRequest {
	container_name?: string;
	command?: string;
	working_dir?: string;
	user?: string;
	timeout_seconds?: number;
	fail_on_error?: boolean;
	enabled?: boolean;
	description?: string;
	template_vars?: Record<string, string>;
}

export interface ContainerBackupHooksResponse {
	hooks: ContainerBackupHook[];
}

export interface ContainerHookTemplateInfo {
	name: string;
	template: ContainerHookTemplate;
	description: string;
	pre_backup_cmd: string;
	post_backup_cmd: string;
	required_vars: string[];
	optional_vars: string[];
	default_vars: Record<string, string>;
}

export interface ContainerHookTemplatesResponse {
	templates: ContainerHookTemplateInfo[];
}

export interface ContainerHookExecution {
	hook_id: string;
	backup_id: string;
	container: string;
	type: ContainerHookType;
	command: string;
	output: string;
	exit_code: number;
	error?: string;
	duration: number;
	started_at: string;
	completed_at: string;
}

export interface ContainerHookExecutionsResponse {
	executions: ContainerHookExecution[];
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
	is_superuser?: boolean;
	is_impersonating?: boolean;
	impersonating_user_id?: string;
	impersonating_id?: string;
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

// Admin Organization types (superuser only)
export interface AdminOrganization {
	id: string;
	name: string;
	slug: string;
	logo_url?: string;
	storage_quota_bytes?: number;
	storage_used_bytes?: number;
	agent_limit?: number;
	agent_count?: number;
	feature_flags: OrgFeatureFlags;
	member_count: number;
	owner_email?: string;
	owner_id?: string;
	created_at: string;
	updated_at: string;
}

export interface OrgFeatureFlags {
	sso_enabled?: boolean;
	api_access?: boolean;
	advanced_reporting?: boolean;
	custom_branding?: boolean;
	priority_support?: boolean;
}

// Branding Settings (Enterprise)
export interface BrandingSettings {
	enabled: boolean;
	product_name: string;
	company_name: string;
	logo_url: string;
	logo_dark_url: string;
	favicon_url: string;
	primary_color: string;
	secondary_color: string;
	accent_color: string;
	support_url: string;
	support_email: string;
	privacy_url: string;
	terms_url: string;
	footer_text: string;
	login_title: string;
	login_subtitle: string;
	login_bg_url: string;
	hide_powered_by: boolean;
	custom_css: string;
}

export interface PublicBrandingSettings {
	enabled: boolean;
	product_name: string;
	logo_url: string;
	logo_dark_url: string;
	favicon_url: string;
	primary_color: string;
	secondary_color: string;
	accent_color: string;
	support_url: string;
	privacy_url: string;
	terms_url: string;
	login_title: string;
	login_subtitle: string;
	login_bg_url: string;
	hide_powered_by: boolean;
}

export interface UpdateBrandingSettingsRequest {
	enabled?: boolean;
	product_name?: string;
	company_name?: string;
	logo_url?: string;
	logo_dark_url?: string;
	favicon_url?: string;
	primary_color?: string;
	secondary_color?: string;
	accent_color?: string;
	support_url?: string;
	support_email?: string;
	privacy_url?: string;
	terms_url?: string;
	footer_text?: string;
	login_title?: string;
	login_subtitle?: string;
	login_bg_url?: string;
	hide_powered_by?: boolean;
	custom_css?: string;
}

export interface AdminOrgSettings {
	name?: string;
	slug?: string;
	logo_url?: string;
	storage_quota_bytes?: number;
	agent_limit?: number;
	feature_flags?: Partial<OrgFeatureFlags>;
}

export interface AdminCreateOrgRequest {
	name: string;
	slug: string;
	owner_email: string;
	logo_url?: string;
	storage_quota_bytes?: number;
	agent_limit?: number;
	feature_flags?: Partial<OrgFeatureFlags>;
}

export interface AdminOrganizationsResponse {
	organizations: AdminOrganization[];
	total_count: number;
}

export interface AdminOrgUsageStats {
	org_id: string;
	storage_used_bytes: number;
	storage_quota_bytes?: number;
	agent_count: number;
	agent_limit?: number;
	backup_count: number;
	total_backup_size_bytes: number;
	member_count: number;
	last_backup_at?: string;
	created_at: string;
}

export interface TransferOwnershipRequest {
	new_owner_user_id: string;
}

export type PlanType = 'free' | 'starter' | 'professional' | 'enterprise';

export interface BillingSettings {
	plan_type: PlanType;
	billing_email?: string;
	billing_cycle?: 'monthly' | 'annual';
	next_billing_date?: string;
	payment_method_last4?: string;
}

export interface PlanLimits {
	agent_limit?: number;
	storage_quota_bytes?: number;
	backup_retention_days?: number;
	concurrent_backups?: number;
}

export interface PlanFeatures {
	sso_enabled: boolean;
	api_access: boolean;
	advanced_reporting: boolean;
	audit_logs: boolean;
	custom_branding: boolean;
	priority_support: boolean;
	geo_replication: boolean;
	lifecycle_policies: boolean;
	legal_holds: boolean;
}

export interface OrganizationPlanInfo {
	plan_type: PlanType;
	limits: PlanLimits;
	features: PlanFeatures;
	usage: {
		agent_count: number;
		storage_used_bytes: number;
	};
}

export type UpgradeFeature =
	| 'agents'
	| 'storage'
	| 'sso'
	| 'api_access'
	| 'advanced_reporting'
	| 'audit_logs'
	| 'custom_branding'
	| 'priority_support'
	| 'geo_replication'
	| 'lifecycle_policies'
	| 'legal_holds';

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
	target_path?: string;
	include_paths?: string[];
	exclude_paths?: string[];
	path_mappings?: PathMapping[];
	cloud_target?: CloudRestoreTarget;
	verify_upload?: boolean;
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

// Docker Restore types
export type DockerRestoreStatus =
	| 'pending'
	| 'preparing'
	| 'restoring_volumes'
	| 'creating_container'
	| 'starting'
	| 'verifying'
	| 'completed'
	| 'failed'
	| 'canceled';

export type DockerRestoreTargetType = 'local' | 'remote';

export interface DockerRestoreTarget {
	type: DockerRestoreTargetType;
	host?: string;
	cert_path?: string;
	tls_verify?: boolean;
}

export interface DockerRestoreProgress {
	status: string;
	current_step: string;
	total_steps: number;
	completed_steps: number;
	percent_complete: number;
	total_bytes: number;
	restored_bytes: number;
	current_volume?: string;
	error_message?: string;
}

export interface DockerContainer {
	id: string;
	name: string;
	image: string;
	status?: string;
	state?: string;
	volumes?: string[];
	ports?: string[];
	networks?: string[];
	created?: string;
	created_at?: string;
}

export interface DockerVolume {
	name: string;
	driver: string;
	mountpoint?: string;
	size_bytes: number;
	created_at: string;
}

export interface DockerRestoreConflict {
	type: 'container' | 'volume' | 'network';
	name: string;
	existing_id?: string;
	description: string;
}

export interface DockerRestorePlan {
	container?: DockerContainer;
	volumes: DockerVolume[];
	total_size_bytes: number;
	conflicts: DockerRestoreConflict[];
	dependencies: string[];
}

export interface DockerRestore {
	id: string;
	agent_id: string;
	repository_id: string;
	snapshot_id: string;
	container_name?: string;
	volume_name?: string;
	new_container_name?: string;
	new_volume_name?: string;
	target?: DockerRestoreTarget;
	overwrite_existing: boolean;
	start_after_restore: boolean;
	verify_start: boolean;
	status: DockerRestoreStatus;
	progress?: DockerRestoreProgress;
	restored_container_id?: string;
	restored_volumes?: string[];
	start_verified: boolean;
	warnings?: string[];
	started_at?: string;
	completed_at?: string;
	error_message?: string;
	created_at: string;
}

export interface CreateDockerRestoreRequest {
	snapshot_id: string;
	agent_id: string;
	repository_id: string;
	container_name?: string;
	volume_name?: string;
	new_container_name?: string;
	new_volume_name?: string;
	target?: DockerRestoreTarget;
	overwrite_existing?: boolean;
	start_after_restore?: boolean;
	verify_start?: boolean;
}

export interface DockerRestorePreviewRequest {
	snapshot_id: string;
	agent_id: string;
	repository_id: string;
	container_name?: string;
	volume_name?: string;
	target?: DockerRestoreTarget;
}

export interface DockerRestoresResponse {
	docker_restores: DockerRestore[];
}

export interface DockerContainersResponse {
	containers: DockerContainer[];
	message?: string;
}

export interface DockerVolumesResponse {
	volumes: DockerVolume[];
	message?: string;
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

// Notification Rule types
export type RuleTriggerType =
	| 'backup_failed'
	| 'backup_success'
	| 'agent_offline'
	| 'agent_health_warning'
	| 'agent_health_critical'
	| 'storage_usage_high'
	| 'replication_lag'
	| 'ransomware_suspected'
	| 'maintenance_scheduled';

export type RuleActionType =
	| 'notify_channel'
	| 'escalate'
	| 'suppress'
	| 'webhook';

export interface RuleConditions {
	count?: number;
	time_window_minutes?: number;
	severity?: string;
	agent_ids?: string[];
	schedule_ids?: string[];
	repository_ids?: string[];
}

export interface RuleAction {
	type: RuleActionType;
	channel_id?: string;
	escalate_to_channel_id?: string;
	webhook_url?: string;
	suppress_duration_minutes?: number;
	message?: string;
}

export interface NotificationRule {
	id: string;
	org_id: string;
	name: string;
	description?: string;
	trigger_type: RuleTriggerType;
	enabled: boolean;
	priority: number;
	conditions: RuleConditions;
	actions: RuleAction[];
	created_at: string;
	updated_at: string;
}

export interface NotificationRuleEvent {
	id: string;
	org_id: string;
	rule_id: string;
	trigger_type: RuleTriggerType;
	resource_type?: string;
	resource_id?: string;
	event_data?: Record<string, unknown>;
	occurred_at: string;
	created_at: string;
}

export interface NotificationRuleExecution {
	id: string;
	org_id: string;
	rule_id: string;
	triggered_by_event_id?: string;
	actions_taken: RuleAction[];
	success: boolean;
	error_message?: string;
	executed_at: string;
	created_at: string;
}

export interface CreateNotificationRuleRequest {
	name: string;
	description?: string;
	trigger_type: RuleTriggerType;
	enabled: boolean;
	priority: number;
	conditions: RuleConditions;
	actions: RuleAction[];
}

export interface UpdateNotificationRuleRequest {
	name?: string;
	description?: string;
	enabled?: boolean;
	priority?: number;
	conditions?: RuleConditions;
	actions?: RuleAction[];
}

export interface TestNotificationRuleRequest {
	event_data?: Record<string, unknown>;
}

export interface NotificationRulesResponse {
	rules: NotificationRule[];
}

export interface NotificationRuleEventsResponse {
	events: NotificationRuleEvent[];
}

export interface NotificationRuleExecutionsResponse {
	executions: NotificationRuleExecution[];
}

export interface TestNotificationRuleResponse {
	success: boolean;
	message?: string;
	execution?: NotificationRuleExecution;
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

// DR Runbook types
export type DRRunbookStatus = 'active' | 'draft' | 'archived';

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

export interface GroupedSearchResult {
	type: SearchResultType;
	id: string;
	name: string;
	description?: string;
	status?: string;
	tags?: string[];
	created_at: string;
	metadata?: Record<string, unknown>;
}

export interface GroupedSearchResponse {
	agents: GroupedSearchResult[];
	backups: GroupedSearchResult[];
	snapshots: GroupedSearchResult[];
	schedules: GroupedSearchResult[];
	repositories: GroupedSearchResult[];
	query: string;
	total: number;
}

export interface SearchSuggestion {
	text: string;
	type: SearchResultType;
	id: string;
	detail?: string;
}

export interface SearchSuggestionsResponse {
	suggestions: SearchSuggestion[];
}

export interface RecentSearch {
	id: string;
	user_id: string;
	org_id: string;
	query: string;
	types?: string[];
	created_at: string;
}

export interface RecentSearchesResponse {
	recent_searches: RecentSearch[];
}

export interface SaveRecentSearchRequest {
	query: string;
	types?: string[];
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
	| 'license'
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

// Branding types
export interface BrandingSettings {
	id: string;
	org_id: string;
	logo_url: string;
	favicon_url: string;
	product_name: string;
	primary_color: string;
	secondary_color: string;
	support_url: string;
	custom_css: string;
	created_at: string;
	updated_at: string;
}

export interface UpdateBrandingRequest {
	logo_url?: string;
	favicon_url?: string;
	product_name?: string;
	primary_color?: string;
	secondary_color?: string;
	support_url?: string;
	custom_css?: string;
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

// SLA Policy types
export interface SLAPolicy {
	id: string;
	org_id: string;
	name: string;
	description?: string;
	target_rpo_hours: number;
	target_rto_hours: number;
	target_success_rate: number;
	enabled: boolean;
	created_at: string;
	updated_at: string;
}

export interface SLAStatus {
	policy_id: string;
	current_rpo_hours: number;
	current_rto_hours: number;
	success_rate: number;
	compliant: boolean;
	calculated_at: string;
}

export interface SLAStatusSnapshot {
	id: string;
	policy_id: string;
	rpo_hours: number;
	rto_hours: number;
	success_rate: number;
	compliant: boolean;
	calculated_at: string;
}

export interface CreateSLAPolicyRequest {
	name: string;
	description?: string;
	target_rpo_hours: number;
	target_rto_hours: number;
	target_success_rate: number;
}

export interface UpdateSLAPolicyRequest {
	name?: string;
	description?: string;
	target_rpo_hours?: number;
	target_rto_hours?: number;
	target_success_rate?: number;
	enabled?: boolean;
}

export interface SLAPoliciesResponse {
	policies: SLAPolicy[];
}

export interface SLAStatusHistoryResponse {
	history: SLAStatusSnapshot[];
}

// Air-Gap types
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
	component?: string;
	message: string;
	fields?: Record<string, unknown>;
}

export interface ServerLogFilter {
	level?: string;
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
}

export interface ServerLogComponentsResponse {
	components: string[];
}

export interface AirGapDisabledFeature {
	name: string;
	reason: string;
}

export interface DockerDaemonStatus {
	available: boolean;
	version: string;
	container_count: number;
	volume_count: number;
	server_os: string;
	docker_root_dir: string;
	storage_driver: string;
}

export interface DockerBackupRequest {
	agent_id: string;
	repository_id: string;
	container_ids?: string[];
	volume_names?: string[];
}

export interface DockerBackupResponse {
	id: string;
	status: string;
	created_at: string;
}

// License types

export interface LicenseInfo {
	tier: LicenseTier;
	customer_id: string;
	customer_name?: string;
	expires_at: string;
	issued_at: string;
	features: string[];
	limits: LicenseLimits;
	license_key_source: 'env' | 'database' | 'none';
	is_trial: boolean;
	trial_days_left?: number;
}

export interface ActivateLicenseRequest {
	license_key: string;
}

export interface ActivateLicenseResponse {
	status?: string;
	tier?: LicenseTier;
	expires_at?: string;
	features?: string[];
	limits?: LicenseLimits;
	license_type?: string;
	message?: string;
}

export interface PricingPlan {
	id: string;
	product_id: string;
	tier: string;
	name: string;
	base_price_cents: number;
	agent_price_cents: number;
	included_agents: number;
	included_servers: number;
	features: string[] | null;
	is_active: boolean;
	stripe_price_id?: string;
	created_at: string;
	updated_at: string;
}

export interface LicenseLimits {
	max_agents: number;
	max_users: number;
	max_orgs: number;
}

export interface StartTrialRequest {
	email: string;
	tier: string;
}

export interface StartTrialResponse {
	status: string;
	tier: string;
	expires_at: string;
	trial_duration_days: number;
	features: string[];
	limits: LicenseLimits;
}

export interface TrialCheckResponse {
	has_trial: boolean;
	is_active?: boolean;
	expired?: boolean;
	tier?: string;
	expires_at?: string;
}

export interface ServerVersion {
	version: string;
	commit?: string;
	build_date?: string;
}

export interface LimitExceededError {
	error: 'limit_exceeded';
	resource: string;
	current: number;
	limit: number;
	tier: LicenseTier;
	message: string;
}

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

// Backup Queue and Concurrency types
export type BackupQueueStatus = 'queued' | 'started' | 'canceled';

export interface BackupQueueEntry {
	id: string;
	org_id: string;
	agent_id: string;
	schedule_id: string;
	priority: number;
	queued_at: string;
	started_at?: string;
	status: BackupQueueStatus;
	queue_position: number;
}

export interface BackupQueueEntryWithDetails extends BackupQueueEntry {
	schedule_name: string;
	agent_hostname: string;
}

export interface ConcurrencyStatus {
	org_id: string;
	org_limit?: number;
	org_running_count: number;
	org_queued_count: number;
	agent_id?: string;
	agent_limit?: number;
	agent_running_count: number;
	agent_queued_count: number;
	can_start_now: boolean;
	queue_position?: number;
	estimated_wait_minutes?: number;
}

export interface BackupQueueSummary {
	total_queued: number;
	total_running: number;
	avg_wait_minutes: number;
	oldest_queued_at?: string;
	queued_by_agent?: Record<string, number>;
}

export interface ConcurrencyResponse {
	max_concurrent_backups?: number;
	running_count: number;
	queued_count: number;
}

export interface UpdateConcurrencyRequest {
	max_concurrent_backups?: number;
}

export interface BackupQueueResponse {
	queue: BackupQueueEntryWithDetails[];
}

// Extended Organization with concurrency settings
export interface OrganizationWithConcurrency extends Organization {
	max_concurrent_backups?: number;
}

// Extended Agent with concurrency settings
export interface AgentWithConcurrency extends Agent {
	max_concurrent_backups?: number;
}

// Backup Queue types for priority management
export interface BackupQueueItem {
	id: string;
	schedule_id: string;
	agent_id: string;
	priority: SchedulePriority;
	status:
		| 'pending'
		| 'running'
		| 'completed'
		| 'failed'
		| 'preempted'
		| 'canceled';
	queued_at: string;
	started_at?: string;
	completed_at?: string;
	preempted_by?: string;
	created_at: string;
	updated_at: string;
}

export interface PriorityQueueSummary {
	total_pending: number;
	total_running: number;
	high_priority: number;
	medium_priority: number;
	low_priority: number;
}

export interface PriorityQueueResponse {
	queue: BackupQueueItem[];
	summary: PriorityQueueSummary;
}

// IP Allowlist types
export type IPAllowlistType = 'ui' | 'agent' | 'both';

export interface IPAllowlist {
	id: string;
	org_id: string;
	cidr: string;
	description?: string;
	type: IPAllowlistType;
	enabled: boolean;
	created_by?: string;
	updated_by?: string;
	created_at: string;
	updated_at: string;
}

// Rate Limit types
export interface RateLimitConfig {
	id: string;
	org_id: string;
	endpoint: string;
	requests_per_period: number;
	period_seconds: number;
	enabled: boolean;
	created_by?: string;
	created_at: string;
	updated_at: string;
}

export interface IPAllowlistSettings {
	id: string;
	org_id: string;
	enabled: boolean;
	enforce_for_ui: boolean;
	enforce_for_agent: boolean;
	allow_admin_bypass: boolean;
	created_at: string;
	updated_at: string;
}

export interface IPBlockedAttempt {
	id: string;
	org_id: string;
	ip_address: string;
	request_type: string;
	path?: string;
	user_id?: string;
	agent_id?: string;
	reason?: string;
	created_at: string;
}

export interface CreateIPAllowlistRequest {
	cidr: string;
	description?: string;
	type: IPAllowlistType;
	enabled?: boolean;
}

export interface UpdateIPAllowlistRequest {
	cidr?: string;
	description?: string;
	type?: IPAllowlistType;
	enabled?: boolean;
}

export interface UpdateIPAllowlistSettingsRequest {
	enabled?: boolean;
	enforce_for_ui?: boolean;
	enforce_for_agent?: boolean;
	allow_admin_bypass?: boolean;
}

export interface IPAllowlistsResponse {
	allowlists: IPAllowlist[];
}

export interface IPBlockedAttemptsResponse {
	attempts: IPBlockedAttempt[];
	total: number;
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

// User Sessions
export interface UserSession {
	id: string;
	user_id: string;
	ip_address?: string;
	user_agent?: string;
	created_at: string;
	last_active_at: string;
	expires_at?: string;
	revoked: boolean;
	revoked_at?: string;
	is_current?: boolean;
}

export interface UserSessionsResponse {
	sessions: UserSession[];
}

export interface RevokeSessionsResponse {
	message: string;
	revoked_count?: number;
}

// Password Policy types
export interface PasswordPolicy {
	id: string;
	org_id: string;
	min_length: number;
	require_uppercase: boolean;
	require_lowercase: boolean;
	require_number: boolean;
	require_special: boolean;
	max_age_days?: number;
	history_count: number;
	created_at: string;
	updated_at: string;
}

export interface PasswordRequirements {
	min_length: number;
	require_uppercase: boolean;
	require_lowercase: boolean;
	require_number: boolean;
	require_special: boolean;
	max_age_days?: number;
	description: string;
}

export interface PasswordPolicyResponse {
	policy: PasswordPolicy;
	requirements: PasswordRequirements;
}

export interface UpdatePasswordPolicyRequest {
	min_length?: number;
	require_uppercase?: boolean;
	require_lowercase?: boolean;
	require_number?: boolean;
	require_special?: boolean;
	max_age_days?: number;
	history_count?: number;
}

export interface PasswordValidationResult {
	valid: boolean;
	errors?: string[];
	warnings?: string[];
}

export interface ChangePasswordRequest {
	current_password: string;
	new_password: string;
}

export interface PasswordExpirationInfo {
	is_expired: boolean;
	expires_at?: string;
	days_until_expiry?: number;
	must_change_now: boolean;
	warn_days_remaining: number;
}

export interface PasswordLoginRequest {
	email: string;
	password: string;
}

export interface PasswordLoginResponse {
	id: string;
	email: string;
	name: string;
	current_org_id?: string;
	current_org_role?: string;
	password_expired?: boolean;
	must_change_password?: boolean;
	expires_at?: string;
}

export interface CreateRateLimitConfigRequest {
	endpoint: string;
	requests_per_period: number;
	period_seconds: number;
	enabled?: boolean;
}

export interface UpdateRateLimitConfigRequest {
	requests_per_period?: number;
	period_seconds?: number;
	enabled?: boolean;
}

export interface RateLimitConfigsResponse {
	configs: RateLimitConfig[];
}

export interface BlockedRequest {
	id: string;
	org_id?: string;
	ip_address: string;
	endpoint: string;
	user_agent?: string;
	blocked_at: string;
	reason: string;
}

export interface IPBlockCount {
	ip_address: string;
	count: number;
}

export interface RouteBlockCount {
	endpoint: string;
	count: number;
}

export interface RateLimitStats {
	blocked_today: number;
	top_blocked_ips: IPBlockCount[];
	top_blocked_endpoints: RouteBlockCount[];
}

export interface RateLimitStatsResponse {
	stats: RateLimitStats;
}

export interface BlockedRequestsResponse {
	blocked_requests: BlockedRequest[];
}

export interface IPBan {
	id: string;
	org_id?: string;
	ip_address: string;
	reason: string;
	ban_count: number;
	banned_by?: string;
	banned_at: string;
	expires_at?: string;
	created_at: string;
}

export interface CreateIPBanRequest {
	ip_address: string;
	reason: string;
	duration_minutes?: number;
}

export interface IPBansResponse {
	bans: IPBan[];
}

// Storage Tier types
export type StorageTierType = 'hot' | 'warm' | 'cold' | 'archive';

export interface StorageTierConfig {
	id: string;
	org_id: string;
	tier_type: StorageTierType;
	name: string;
	description?: string;
	cost_per_gb_month: number;
	retrieval_cost: number;
	retrieval_time: string;
	enabled: boolean;
	created_at: string;
	updated_at: string;
}

export interface TierRule {
	id: string;
	org_id: string;
	repository_id?: string;
	schedule_id?: string;
	name: string;
	description?: string;
	from_tier: StorageTierType;
	to_tier: StorageTierType;
	age_threshold_days: number;
	min_copies: number;
	priority: number;
	enabled: boolean;
	created_at: string;
	updated_at: string;
}

export interface CreateTierRuleRequest {
	name: string;
	description?: string;
	from_tier: StorageTierType;
	to_tier: StorageTierType;
	age_threshold_days: number;
	min_copies?: number;
	priority?: number;
	repository_id?: string;
	schedule_id?: string;
}

export interface UpdateTierRuleRequest {
	name?: string;
	description?: string;
	age_threshold_days?: number;
	min_copies?: number;
	priority?: number;
	enabled?: boolean;
}

export interface SnapshotTier {
	id: string;
	snapshot_id: string;
	repository_id: string;
	org_id: string;
	current_tier: StorageTierType;
	size_bytes: number;
	snapshot_time: string;
	tiered_at: string;
	created_at: string;
	updated_at: string;
}

export interface TierTransition {
	id: string;
	snapshot_tier_id: string;
	snapshot_id: string;
	repository_id: string;
	org_id: string;
	from_tier: StorageTierType;
	to_tier: StorageTierType;
	trigger_rule_id?: string;
	trigger_reason: string;
	size_bytes: number;
	estimated_saving: number;
	status: string;
	error_message?: string;
	started_at?: string;
	completed_at?: string;
	created_at: string;
}

export interface ColdRestoreRequest {
	id: string;
	org_id: string;
	snapshot_id: string;
	repository_id: string;
	requested_by: string;
	from_tier: StorageTierType;
	target_path?: string;
	priority: 'standard' | 'expedited' | 'bulk';
	status:
		| 'pending'
		| 'warming'
		| 'ready'
		| 'restoring'
		| 'completed'
		| 'failed'
		| 'expired';
	estimated_ready_at?: string;
	ready_at?: string;
	expires_at?: string;
	error_message?: string;
	retrieval_cost: number;
	created_at: string;
	updated_at: string;
}

export interface TierBreakdownItem {
	tier_type: StorageTierType;
	snapshot_count: number;
	total_size_bytes: number;
	monthly_cost: number;
	percentage: number;
}

export interface TierOptSuggestion {
	snapshot_id: string;
	repository_id: string;
	current_tier: StorageTierType;
	suggested_tier: StorageTierType;
	age_days: number;
	size_bytes: number;
	monthly_savings: number;
	reason: string;
}

export interface TierCostReport {
	id: string;
	org_id: string;
	report_date: string;
	total_size_bytes: number;
	current_monthly_cost: number;
	optimized_monthly_cost: number;
	potential_monthly_savings: number;
	tier_breakdown: TierBreakdownItem[];
	suggestions: TierOptSuggestion[];
	created_at: string;
}

export interface TierStats {
	snapshot_count: number;
	total_size_bytes: number;
	monthly_cost: number;
	oldest_snapshot_days: number;
	newest_snapshot_days: number;
}

export interface TierStatsSummary {
	total_snapshots: number;
	total_size_bytes: number;
	estimated_monthly_cost: number;
	by_tier: Record<StorageTierType, TierStats>;
	potential_savings: number;
}

export interface StorageTierConfigsResponse {
	configs: StorageTierConfig[];
}

export interface TierRulesResponse {
	rules: TierRule[];
}

export interface SnapshotTiersResponse {
	tiers: SnapshotTier[];
}

export interface TierTransitionsResponse {
	history: TierTransition[];
}

export interface ColdRestoreRequestsResponse {
	requests: ColdRestoreRequest[];
}

export interface TierCostReportsResponse {
	reports: TierCostReport[];
}
// Lifecycle Policy types
export type LifecyclePolicyStatus = 'active' | 'draft' | 'disabled';

export interface RetentionDuration {
	min_days: number;
	max_days: number;
}

export interface DataTypeOverride {
	data_type: DataType;
	retention: RetentionDuration;
}

export interface ClassificationRetention {
	level: ClassificationLevel;
	retention: RetentionDuration;
	data_type_overrides?: DataTypeOverride[];
}

export interface LifecyclePolicy {
	id: string;
	name: string;
	description?: string;
	status: LifecyclePolicyStatus;
	rules: ClassificationRetention[];
	repository_ids?: string[];
	schedule_ids?: string[];
	last_evaluated_at?: string;
	last_deletion_at?: string;
	deletion_count: number;
	bytes_reclaimed: number;
	created_by: string;
	created_at: string;
	updated_at: string;
}

export interface CreateLifecyclePolicyRequest {
	name: string;
	description?: string;
	status?: LifecyclePolicyStatus;
	rules: ClassificationRetention[];
	repository_ids?: string[];
	schedule_ids?: string[];
}

export interface UpdateLifecyclePolicyRequest {
	name?: string;
	description?: string;
	status?: LifecyclePolicyStatus;
	rules?: ClassificationRetention[];
	repository_ids?: string[];
	schedule_ids?: string[];
}

export interface LifecycleDryRunRequest {
	policy_id?: string;
	rules?: ClassificationRetention[];
	repository_ids?: string[];
	schedule_ids?: string[];
}

export type LifecycleAction = 'keep' | 'can_delete' | 'must_delete' | 'hold';

export interface LifecycleSnapshotEvaluation {
	snapshot_id: string;
	action: LifecycleAction;
	reason: string;
	snapshot_age_days: number;
	min_retention_days: number;
	max_retention_days: number;
	days_until_deletable: number;
	days_until_auto_delete: number;
	classification_level: ClassificationLevel;
	is_on_legal_hold: boolean;
	size_bytes?: number;
	snapshot_time: string;
	repository_id: string;
	schedule_name?: string;
}

export interface LifecycleDryRunResult {
	evaluated_at: string;
	policy_id?: string;
	total_snapshots: number;
	keep_count: number;
	can_delete_count: number;
	must_delete_count: number;
	hold_count: number;
	total_size_to_delete: number;
	evaluations: LifecycleSnapshotEvaluation[];
}

export interface LifecycleDeletionEvent {
	id: string;
	org_id: string;
	policy_id: string;
	snapshot_id: string;
	repository_id: string;
	reason: string;
	size_bytes: number;
	deleted_by: string;
	deleted_at: string;
}

export interface LifecyclePoliciesResponse {
	policies: LifecyclePolicy[];
}

export interface LifecycleDeletionEventsResponse {
	events: LifecycleDeletionEvent[];
}

// Metadata types
export type MetadataEntityType = 'agent' | 'repository' | 'schedule';
export type MetadataFieldType =
	| 'text'
	| 'number'
	| 'date'
	| 'select'
	| 'boolean';

export interface MetadataValidationRules {
	min_length?: number;
	max_length?: number;
	pattern?: string;
	min?: number;
	max?: number;
	min_date?: string;
	max_date?: string;
}

export interface MetadataSelectOption {
	value: string;
	label: string;
}

export interface MetadataSchema {
	id: string;
	org_id: string;
	entity_type: MetadataEntityType;
	name: string;
	field_key: string;
	field_type: MetadataFieldType;
	description?: string;
	required: boolean;
	default_value?: unknown;
	options?: MetadataSelectOption[];
	validation?: MetadataValidationRules;
	display_order: number;
	created_at: string;
	updated_at: string;
}

export interface CreateMetadataSchemaRequest {
	entity_type: MetadataEntityType;
	name: string;
	field_key: string;
	field_type: MetadataFieldType;
	description?: string;
	required?: boolean;
	default_value?: unknown;
	options?: MetadataSelectOption[];
	validation?: MetadataValidationRules;
	display_order?: number;
}

export interface UpdateMetadataSchemaRequest {
	name?: string;
	description?: string;
	required?: boolean;
	default_value?: unknown;
	options?: MetadataSelectOption[];
	validation?: MetadataValidationRules;
	display_order?: number;
}

export interface MetadataSchemasResponse {
	schemas: MetadataSchema[];
}

export interface MetadataFieldTypesResponse {
	types: {
		type: MetadataFieldType;
		label: string;
		description: string;
	}[];
}

export interface MetadataEntityTypesResponse {
	types: {
		type: MetadataEntityType;
		label: string;
	}[];
}

export interface UpdateEntityMetadataRequest {
	metadata: Record<string, unknown>;
}

export interface MetadataSearchResult {
	entity_type: MetadataEntityType;
	entity_id: string;
	entity_name: string;
	metadata: Record<string, unknown>;
}

export interface MetadataSearchResponse {
	results: MetadataSearchResult[];
}

// Saved Filter types
export interface SavedFilter {
	id: string;
	user_id: string;
	org_id: string;
	name: string;
	entity_type: string;
	filters: Record<string, unknown>;
	shared: boolean;
	is_default: boolean;
	created_at: string;
	updated_at: string;
}

export interface CreateSavedFilterRequest {
	name: string;
	entity_type: string;
	filters: Record<string, unknown>;
	shared?: boolean;
	is_default?: boolean;
}

export interface UpdateSavedFilterRequest {
	name?: string;
	filters?: Record<string, unknown>;
	shared?: boolean;
	is_default?: boolean;
}

export interface SavedFiltersResponse {
	filters: SavedFilter[];
}

// Downtime types
export type ComponentType = 'agent' | 'server' | 'repository' | 'service';
export type DowntimeSeverity = 'info' | 'warning' | 'critical';
export type BadgeType = '7d' | '30d' | '90d' | '365d';

export interface DowntimeEvent {
	id: string;
	org_id: string;
	component_type: ComponentType;
	component_id?: string;
	component_name: string;
	started_at: string;
	ended_at?: string;
	duration_seconds?: number;
	severity: DowntimeSeverity;
	cause?: string;
	notes?: string;
	resolved_by?: string;
	auto_detected: boolean;
	alert_id?: string;
	created_at: string;
	updated_at: string;
}

export interface UptimeStats {
	id: string;
	org_id: string;
	component_type: ComponentType;
	component_id?: string;
	component_name: string;
	period_start: string;
	period_end: string;
	total_seconds: number;
	downtime_seconds: number;
	uptime_percent: number;
	incident_count: number;
	created_at: string;
}

export interface UptimeBadge {
	id: string;
	org_id: string;
	component_type?: ComponentType;
	component_id?: string;
	component_name?: string;
	badge_type: BadgeType;
	uptime_percent: number;
	last_updated: string;
	created_at: string;
}

export interface DowntimeAlert {
	id: string;
	org_id: string;
	name: string;
	enabled: boolean;
	uptime_threshold: number;
	evaluation_period: string;
	component_type?: ComponentType;
	notify_on_breach: boolean;
	notify_on_recovery: boolean;
	last_triggered_at?: string;
	created_at: string;
	updated_at: string;
}

export interface ComponentUptime {
	component_type: ComponentType;
	component_id?: string;
	component_name: string;
	status: string;
	uptime_percent_7d: number;
	uptime_percent_30d: number;
	incident_count_30d: number;
	last_incident_at?: string;
}

export interface UptimeSummary {
	total_components: number;
	components_up: number;
	components_down: number;
	active_incidents: number;
	overall_uptime_7d: number;
	overall_uptime_30d: number;
	overall_uptime_90d: number;
	badges?: UptimeBadge[];
	recent_incidents?: DowntimeEvent[];
	component_breakdown?: ComponentUptime[];
}

export interface DailyUptime {
	date: string;
	uptime_percent: number;
	downtime_seconds: number;
	incident_count: number;
}

export interface MonthlyUptimeReport {
	org_id: string;
	month: string;
	year: number;
	month_num: number;
	overall_uptime: number;
	total_downtime_seconds: number;
	incident_count: number;
	most_affected?: ComponentUptime[];
	daily_breakdown?: DailyUptime[];
	generated_at: string;
}

export interface CreateDowntimeEventRequest {
	component_type: ComponentType;
	component_id?: string;
	component_name: string;
	severity: DowntimeSeverity;
	cause?: string;
}

export interface UpdateDowntimeEventRequest {
	severity?: DowntimeSeverity;
	cause?: string;
	notes?: string;
}

export interface ResolveDowntimeEventRequest {
	notes?: string;
}

export interface CreateDowntimeAlertRequest {
	name: string;
	uptime_threshold: number;
	evaluation_period: string;
	component_type?: ComponentType;
	notify_on_breach?: boolean;
	notify_on_recovery?: boolean;
}

export interface UpdateDowntimeAlertRequest {
	name?: string;
	enabled?: boolean;
	uptime_threshold?: number;
	evaluation_period?: string;
	notify_on_breach?: boolean;
	notify_on_recovery?: boolean;
}

export interface DowntimeEventsResponse {
	events: DowntimeEvent[];
}

export interface UptimeBadgesResponse {
	badges: UptimeBadge[];
}

export interface DowntimeAlertsResponse {
	alerts: DowntimeAlert[];
}

// SLA types
export type SLAScope = 'agent' | 'repository' | 'organization';
export type BreachType = 'rpo' | 'rto' | 'uptime';

export interface SLADefinition {
	id: string;
	org_id: string;
	name: string;
	description?: string;
	rpo_minutes?: number;
	rto_minutes?: number;
	uptime_percentage?: number;
	scope: SLAScope;
	active: boolean;
	created_by?: string;
	created_at: string;
	updated_at: string;
}

export interface SLAWithAssignments extends SLADefinition {
	agent_count: number;
	repository_count: number;
}

export interface SLAAssignment {
	id: string;
	org_id: string;
	sla_id: string;
	agent_id?: string;
	repository_id?: string;
	assigned_by?: string;
	assigned_at: string;
}

export interface SLACompliance {
	id: string;
	org_id: string;
	sla_id: string;
	agent_id?: string;
	repository_id?: string;
	period_start: string;
	period_end: string;
	rpo_compliant?: boolean;
	rpo_actual_minutes?: number;
	rpo_breaches: number;
	rto_compliant?: boolean;
	rto_actual_minutes?: number;
	rto_breaches: number;
	uptime_compliant?: boolean;
	uptime_actual_percentage?: number;
	uptime_downtime_minutes: number;
	is_compliant: boolean;
	notes?: string;
	calculated_at: string;
}

export interface SLABreach {
	id: string;
	org_id: string;
	sla_id: string;
	agent_id?: string;
	repository_id?: string;
	breach_type: BreachType;
	expected_value?: number;
	actual_value?: number;
	breach_start: string;
	breach_end?: string;
	duration_minutes?: number;
	acknowledged: boolean;
	acknowledged_by?: string;
	acknowledged_at?: string;
	resolved: boolean;
	resolved_at?: string;
	description?: string;
	created_at: string;
}

export interface SLAComplianceSummary {
	sla_id: string;
	sla_name: string;
	total_targets: number;
	compliant_targets: number;
	compliance_rate: number;
	active_breaches: number;
	total_breaches: number;
	period_start: string;
	period_end: string;
}

export interface SLADashboardStats {
	total_slas: number;
	active_slas: number;
	overall_compliance: number;
	active_breaches: number;
	unacknowledged_count: number;
	compliance_trend?: SLAComplianceSummary[];
}

export interface SLAReport {
	org_id: string;
	report_month: string;
	generated_at: string;
	sla_summaries: SLAComplianceSummary[];
	total_breaches: number;
	resolved_breaches: number;
	mean_time_to_resolve_minutes?: number;
}

export interface CreateSLADefinitionRequest {
	name: string;
	description?: string;
	rpo_minutes?: number;
	rto_minutes?: number;
	uptime_percentage?: number;
	scope: SLAScope;
	active?: boolean;
}

export interface UpdateSLADefinitionRequest {
	name?: string;
	description?: string;
	rpo_minutes?: number;
	rto_minutes?: number;
	uptime_percentage?: number;
	scope?: SLAScope;
	active?: boolean;
}

export interface AssignSLARequest {
	agent_id?: string;
	repository_id?: string;
}

export interface AcknowledgeBreachRequest {
	notes?: string;
}

export interface SLADefinitionsResponse {
	slas: SLAWithAssignments[];
}

export interface SLAAssignmentsResponse {
	assignments: SLAAssignment[];
}

export interface SLAComplianceResponse {
	compliance: SLACompliance[];
}

export interface SLABreachesResponse {
	breaches: SLABreach[];
}

export interface SLADashboardResponse {
	stats: SLADashboardStats;
}

export interface SLAReportResponse {
	report: SLAReport;
}

// User Management types
export type UserStatus = 'active' | 'disabled' | 'pending' | 'locked';

export interface UserWithMembership {
	id: string;
	org_id: string;
	oidc_subject?: string;
	email: string;
	name: string;
	role: string;
	status: UserStatus;
	last_login_at?: string;
	last_login_ip?: string;
	failed_login_attempts?: number;
	locked_until?: string;
	invited_by?: string;
	invited_at?: string;
	is_superuser?: boolean;
	created_at: string;
	updated_at: string;
	org_role: OrgRole;
}

export interface UsersResponse {
	users: UserWithMembership[];
}

export interface InviteUserRequest {
	email: string;
	name?: string;
	role: OrgRole;
}

export interface InviteUserResponse {
	message: string;
	token: string;
}

export interface UpdateUserRequest {
	name?: string;
	role?: OrgRole;
	status?: UserStatus;
}

export interface ResetPasswordRequest {
	new_password: string;
	require_change_on_use?: boolean;
}

export interface ImpersonateUserRequest {
	reason: string;
}

export interface ImpersonateUserResponse {
	message: string;
	impersonating: string;
	impersonated_id: string;
}

export interface UserActivityLog {
	id: string;
	user_id: string;
	org_id: string;
	action: string;
	resource_type?: string;
	resource_id?: string;
	ip_address?: string;
	user_agent?: string;
	details?: Record<string, unknown>;
	created_at: string;
	user_email: string;
	user_name: string;
}

export interface UserActivityLogsResponse {
	activity_logs: UserActivityLog[];
}

export interface UserImpersonationLog {
	id: string;
	admin_user_id: string;
	target_user_id: string;
	org_id: string;
	reason?: string;
	started_at: string;
	ended_at?: string;
	ip_address?: string;
	user_agent?: string;
	admin_email: string;
	admin_name: string;
	target_email: string;
	target_name: string;
}

export interface ImpersonationLogsResponse {
	impersonation_logs: UserImpersonationLog[];
}

// Organization System Settings types (SMTP, OIDC, Storage, Security)
export interface SMTPSettings {
	host: string;
	port: number;
	username?: string;
	password?: string;
	from_email: string;
	from_name?: string;
	encryption: 'none' | 'tls' | 'starttls';
	enabled: boolean;
	skip_tls_verify: boolean;
	connection_timeout_seconds: number;
}

export interface OIDCSettings {
	enabled: boolean;
	issuer: string;
	client_id: string;
	client_secret?: string;
	redirect_url: string;
	scopes: string[];
	auto_create_users: boolean;
	default_role: 'member' | 'readonly';
	allowed_domains?: string[];
	require_email_verification: boolean;
}

export interface StorageDefaultSettings {
	default_retention_days: number;
	max_retention_days: number;
	default_storage_backend: 'local' | 's3' | 'b2' | 'sftp' | 'rest' | 'dropbox';
	max_backup_size_gb: number;
	enable_compression: boolean;
	compression_level: number;
	default_encryption_method: 'aes256' | 'none';
	prune_schedule: string;
	auto_prune_enabled: boolean;
}

export interface SecuritySettings {
	session_timeout_minutes: number;
	max_concurrent_sessions: number;
	require_mfa: boolean;
	mfa_grace_period_days: number;
	allowed_ip_ranges?: string[];
	blocked_ip_ranges?: string[];
	failed_login_lockout_attempts: number;
	failed_login_lockout_minutes: number;
	api_key_expiration_days: number;
	enable_audit_logging: boolean;
	audit_log_retention_days: number;
	force_https: boolean;
	allow_password_login: boolean;
}

export interface OrgSettingsResponse {
	smtp: SMTPSettings;
	oidc: OIDCSettings;
	storage_defaults: StorageDefaultSettings;
	security: SecuritySettings;
	updated_at: string;
}

export interface UpdateSMTPSettingsRequest {
	host?: string;
	port?: number;
	username?: string;
	password?: string;
	from_email?: string;
	from_name?: string;
	encryption?: 'none' | 'tls' | 'starttls';
	enabled?: boolean;
	skip_tls_verify?: boolean;
	connection_timeout_seconds?: number;
}

export interface UpdateOIDCSettingsRequest {
	enabled?: boolean;
	issuer?: string;
	client_id?: string;
	client_secret?: string;
	redirect_url?: string;
	scopes?: string[];
	auto_create_users?: boolean;
	default_role?: 'member' | 'readonly';
	allowed_domains?: string[];
	require_email_verification?: boolean;
}

export interface UpdateStorageDefaultsRequest {
	default_retention_days?: number;
	max_retention_days?: number;
	default_storage_backend?: 'local' | 's3' | 'b2' | 'sftp' | 'rest' | 'dropbox';
	max_backup_size_gb?: number;
	enable_compression?: boolean;
	compression_level?: number;
	default_encryption_method?: 'aes256' | 'none';
	prune_schedule?: string;
	auto_prune_enabled?: boolean;
}

export interface UpdateSecuritySettingsRequest {
	session_timeout_minutes?: number;
	max_concurrent_sessions?: number;
	require_mfa?: boolean;
	mfa_grace_period_days?: number;
	allowed_ip_ranges?: string[];
	blocked_ip_ranges?: string[];
	failed_login_lockout_attempts?: number;
	failed_login_lockout_minutes?: number;
	api_key_expiration_days?: number;
	enable_audit_logging?: boolean;
	audit_log_retention_days?: number;
	force_https?: boolean;
	allow_password_login?: boolean;
}

export interface TestSMTPRequest {
	recipient_email: string;
}

export interface TestSMTPResponse {
	success: boolean;
	message: string;
}

export interface TestOIDCResponse {
	success: boolean;
	message: string;
	provider_name?: string;
	auth_url?: string;
	supported_flow?: string;
}

export type SettingKey = 'smtp' | 'oidc' | 'storage_defaults' | 'security';

export interface SettingsAuditLog {
	id: string;
	org_id: string;
	setting_key: SettingKey;
	old_value?: object;
	new_value: object;
	changed_by: string;
	changed_by_email?: string;
	changed_at: string;
	ip_address?: string;
}

export interface SettingsAuditLogsResponse {
	logs: SettingsAuditLog[];
	limit: number;
	offset: number;
}

// Superuser types
export type SuperuserAction =
	| 'view_organizations'
	| 'view_organization'
	| 'impersonate_user'
	| 'end_impersonation'
	| 'update_system_settings'
	| 'view_system_settings'
	| 'grant_superuser'
	| 'revoke_superuser'
	| 'view_users'
	| 'view_superuser_audit_logs';

export interface SuperuserAuditLog {
	id: string;
	superuser_id: string;
	action: SuperuserAction;
	target_type: string;
	target_id?: string;
	target_org_id?: string;
	impersonated_user_id?: string;
	ip_address?: string;
	user_agent?: string;
	details?: Record<string, unknown>;
	created_at: string;
	superuser_email?: string;
	superuser_name?: string;
}

export interface SystemSetting {
	key: string;
	value: unknown;
	description?: string;
	updated_by?: string;
	updated_at: string;
}

export interface SuperuserOrganizationsResponse {
	organizations: Organization[];
}

export interface SuperuserUsersResponse {
	users: User[];
}

export interface SuperusersResponse {
	superusers: User[];
}

export interface SuperuserAuditLogsResponse {
	audit_logs: SuperuserAuditLog[];
	total: number;
	limit: number;
	offset: number;
}

export interface SystemSettingsResponse {
	settings: SystemSetting[];
}

export interface SystemSettingResponse {
	setting: SystemSetting;
}

export interface UpdateSystemSettingRequest {
	value: unknown;
}

export interface GrantSuperuserRequest {
	reason?: string;
}

export interface ImpersonationResponse {
	message: string;
	impersonating?: {
		id: string;
		email: string;
		name: string;
	};
}

// Recent Items types
export type RecentItemType =
	| 'agent'
	| 'repository'
	| 'schedule'
	| 'backup'
	| 'policy'
	| 'snapshot';

export interface RecentItem {
	id: string;
	org_id: string;
	user_id: string;
	item_type: RecentItemType;
	item_id: string;
	item_name: string;
	page_path: string;
	viewed_at: string;
	created_at: string;
}

export interface TrackRecentItemRequest {
	item_type: RecentItemType;
	item_id: string;
	item_name: string;
	page_path: string;
}

export interface RecentItemsResponse {
	items: RecentItem[];
}

// Activity Event types
export type ActivityEventType =
	| 'backup_started'
	| 'backup_completed'
	| 'backup_failed'
	| 'restore_started'
	| 'restore_completed'
	| 'restore_failed'
	| 'agent_connected'
	| 'agent_disconnected'
	| 'agent_created'
	| 'agent_deleted'
	| 'user_login'
	| 'user_logout'
	| 'schedule_created'
	| 'schedule_updated'
	| 'schedule_deleted'
	| 'schedule_enabled'
	| 'schedule_disabled'
	| 'repository_created'
	| 'repository_deleted'
	| 'alert_triggered'
	| 'alert_acknowledged'
	| 'alert_resolved'
	| 'policy_applied'
	| 'maintenance_started'
	| 'maintenance_ended'
	| 'system_startup'
	| 'system_shutdown';

export type ActivityEventCategory =
	| 'backup'
	| 'restore'
	| 'agent'
	| 'user'
	| 'schedule'
	| 'repository'
	| 'alert'
	| 'policy'
	| 'maintenance'
	| 'system';

export interface ActivityEvent {
	id: string;
	org_id: string;
	type: ActivityEventType;
	category: ActivityEventCategory;
	title: string;
	description: string;
	user_id?: string;
	user_name?: string;
	agent_id?: string;
	agent_name?: string;
	resource_type?: string;
	resource_id?: string;
	resource_name?: string;
	metadata?: Record<string, unknown>;
	created_at: string;
}

export interface ActivityEventsResponse {
	events: ActivityEvent[];
}

export interface ActivityEventCountResponse {
	count: number;
}

export interface ActivityCategoriesResponse {
	categories: Record<string, number>;
}

export interface ActivityEventFilter {
	category?: ActivityEventCategory;
	type?: ActivityEventType;
	user_id?: string;
	agent_id?: string;
	start_time?: string;
	end_time?: string;
	limit?: number;
	offset?: number;
}

// Favorite types
export type FavoriteEntityType = 'agent' | 'schedule' | 'repository';

export interface Favorite {
	id: string;
	user_id: string;
	org_id: string;
	entity_type: FavoriteEntityType;
	entity_id: string;
	created_at: string;
}

export interface CreateFavoriteRequest {
	entity_type: FavoriteEntityType;
	entity_id: string;
}

export interface FavoritesResponse {
	favorites: Favorite[];
}

// Docker Stack Backup types
export type DockerStackBackupStatus =
	| 'pending'
	| 'running'
	| 'completed'
	| 'failed'
	| 'canceled';

export type DockerStackRestoreStatus =
	| 'pending'
	| 'running'
	| 'completed'
	| 'failed';

export interface DockerContainerState {
	service_name: string;
	container_id: string;
	status: string;
	health?: string;
	image: string;
	image_id: string;
	created: string;
	started?: string;
}

export interface DockerStack {
	id: string;
	org_id: string;
	agent_id: string;
	name: string;
	compose_path: string;
	description?: string;
	service_count: number;
	is_running: boolean;
	last_backup_at?: string;
	last_backup_id?: string;
	backup_schedule_id?: string;
	export_images: boolean;
	include_env_files: boolean;
	stop_for_backup: boolean;
	exclude_paths?: string[];
	created_at: string;
	updated_at: string;
}

export interface DockerStackBackup {
	id: string;
	org_id: string;
	stack_id: string;
	agent_id: string;
	schedule_id?: string;
	backup_id?: string;
	status: DockerStackBackupStatus;
	backup_path: string;
	manifest_path?: string;
	volume_count: number;
	bind_mount_count: number;
	image_count?: number;
	total_size_bytes: number;
	container_states?: DockerContainerState[];
	dependency_order?: string[];
	includes_images: boolean;
	error_message?: string;
	started_at?: string;
	completed_at?: string;
	created_at: string;
	updated_at: string;
}

export interface DockerStackRestore {
	id: string;
	org_id: string;
	stack_backup_id: string;
	agent_id: string;
	status: DockerStackRestoreStatus;
	target_path: string;
	restore_volumes: boolean;
	restore_images: boolean;
	start_containers: boolean;
	path_mappings?: Record<string, string>;
	volumes_restored: number;
	images_restored: number;
	error_message?: string;
	started_at?: string;
	completed_at?: string;
	created_at: string;
	updated_at: string;
}

export interface DiscoveredDockerStack {
	name: string;
	compose_path: string;
	service_count: number;
	is_running: boolean;
	is_registered: boolean;
}

export interface CreateDockerStackRequest {
	name: string;
	agent_id: string;
	compose_path: string;
	description?: string;
	export_images?: boolean;
	include_env_files?: boolean;
	stop_for_backup?: boolean;
	exclude_paths?: string[];
}

export interface UpdateDockerStackRequest {
	name?: string;
	description?: string;
	export_images?: boolean;
	include_env_files?: boolean;
	stop_for_backup?: boolean;
	exclude_paths?: string[];
}

export interface TriggerDockerStackBackupRequest {
	export_images?: boolean;
	stop_for_backup?: boolean;
}

export interface RestoreDockerStackRequest {
	backup_id: string;
	target_agent_id: string;
	target_path: string;
	restore_volumes: boolean;
	restore_images: boolean;
	start_containers: boolean;
	path_mappings?: Record<string, string>;
}

export interface DiscoverDockerStacksRequest {
	agent_id: string;
	search_paths: string[];
}

export interface DockerStackListResponse {
	stacks: DockerStack[];
}

export interface DockerStackBackupListResponse {
	backups: DockerStackBackup[];
}

export interface DiscoverDockerStacksResponse {
	stacks: DiscoveredDockerStack[];
}

// Docker Container Logs types
export type DockerLogBackupStatus =
	| 'pending'
	| 'running'
	| 'completed'
	| 'failed';

export interface DockerLogRetentionPolicy {
	max_age_days: number;
	max_size_bytes: number;
	max_files_per_day: number;
	compress_enabled: boolean;
	compress_level: number;
}

export interface DockerLogBackup {
	id: string;
	agent_id: string;
	container_id: string;
	container_name: string;
	image_name?: string;
	log_path: string;
	original_size: number;
	compressed_size: number;
	compressed: boolean;
	start_time: string;
	end_time: string;
	line_count: number;
	status: DockerLogBackupStatus;
	error_message?: string;
	backup_schedule_id?: string;
	created_at: string;
	updated_at: string;
}

export interface DockerLogSettings {
	id: string;
	agent_id: string;
	enabled: boolean;
	cron_expression: string;
	retention_policy: DockerLogRetentionPolicy;
	include_containers?: string[];
	exclude_containers?: string[];
	include_labels?: Record<string, string>;
	exclude_labels?: Record<string, string>;
	timestamps: boolean;
	tail: number;
	since?: string;
	until?: string;
	created_at: string;
	updated_at: string;
}

export interface DockerLogEntry {
	timestamp: string;
	stream: string;
	message: string;
	line_num: number;
}

export interface DockerLogViewResponse {
	backup_id: string;
	container_id: string;
	container_name: string;
	entries: DockerLogEntry[];
	total_lines: number;
	offset: number;
	limit: number;
	start_time: string;
	end_time: string;
}

export interface DockerLogBackupsResponse {
	backups: DockerLogBackup[];
	total_count: number;
}

export interface DockerLogStorageStats {
	total_size: number;
	total_files: number;
	container_stats: Record<string, { size: number; files: number }>;
}

export interface DockerLogRetentionResult {
	removed_count: number;
	removed_bytes: number;
}

export interface DockerLogSettingsUpdate {
	enabled?: boolean;
	cron_expression?: string;
	retention_policy?: DockerLogRetentionPolicy;
	include_containers?: string[];
	exclude_containers?: string[];
	include_labels?: Record<string, string>;
	exclude_labels?: Record<string, string>;
	timestamps?: boolean;
	tail?: number;
	since?: string;
	until?: string;
}

// Docker Registry types
export type DockerRegistryType =
	| 'dockerhub'
	| 'gcr'
	| 'ecr'
	| 'acr'
	| 'ghcr'
	| 'private';

export type DockerRegistryHealthStatus = 'healthy' | 'unhealthy' | 'unknown';

export interface DockerRegistry {
	id: string;
	org_id: string;
	name: string;
	type: DockerRegistryType;
	url: string;
	is_default: boolean;
	enabled: boolean;
	health_status: DockerRegistryHealthStatus;
	last_health_check?: string;
	last_health_error?: string;
	credentials_rotated_at?: string;
	credentials_expires_at?: string;
	metadata?: Record<string, unknown>;
	created_by?: string;
	created_at: string;
	updated_at: string;
}

export interface DockerRegistryCredentials {
	username?: string;
	password?: string;
	access_token?: string;
	aws_access_key_id?: string;
	aws_secret_access_key?: string;
	aws_region?: string;
	azure_tenant_id?: string;
	azure_client_id?: string;
	azure_client_secret?: string;
	gcr_key_json?: string;
}

export interface DockerLoginResult {
	success: boolean;
	registry_id: string;
	registry_url: string;
	error_message?: string;
	logged_in_at: string;
}

export interface DockerRegistryHealthCheck {
	registry_id: string;
	status: DockerRegistryHealthStatus;
	response_time_ms: number;
	error_message?: string;
	checked_at: string;
}

export interface DockerRegistryTypeInfo {
	type: DockerRegistryType;
	name: string;
	description: string;
	default_url: string;
	fields: string[];
}

export interface CreateDockerRegistryRequest {
	name: string;
	type: DockerRegistryType;
	url?: string;
	credentials: DockerRegistryCredentials;
	is_default?: boolean;
}

export interface UpdateDockerRegistryRequest {
	name?: string;
	url?: string;
	enabled?: boolean;
	is_default?: boolean;
}

export interface RotateCredentialsRequest {
	credentials: DockerRegistryCredentials;
	expires_at?: string;
}

export interface DockerRegistriesResponse {
	registries: DockerRegistry[];
}

export interface DockerRegistryResponse {
	registry: DockerRegistry;
}

export interface DockerRegistryTypesResponse {
	types: DockerRegistryTypeInfo[];
}

export interface DockerLoginResultResponse {
	result: DockerLoginResult;
}

export interface DockerLoginAllResponse {
	results: DockerLoginResult[];
}

export interface DockerHealthCheckResponse {
	result: DockerRegistryHealthCheck;
}

export interface DockerHealthCheckAllResponse {
	results: DockerRegistryHealthCheck[];
}

export interface ExpiringCredentialsResponse {
	registries: DockerRegistry[];
	warning_days: number;
}

// =============================================================================
// Komodo Integration Types
// =============================================================================

export type KomodoIntegrationStatus = 'active' | 'disconnected' | 'error';
export type KomodoContainerStatus =
	| 'running'
	| 'stopped'
	| 'restarting'
	| 'unknown';
export type KomodoWebhookEventType =
	| 'container.start'
	| 'container.stop'
	| 'container.restart'
	| 'stack.deploy'
	| 'stack.update'
	| 'backup.trigger'
	| 'unknown';
export type KomodoWebhookEventStatus =
	| 'received'
	| 'processing'
	| 'processed'
	| 'failed';

export interface KomodoIntegration {
	id: string;
	org_id: string;
	name: string;
	url: string;
	status: KomodoIntegrationStatus;
	last_sync_at?: string;
	last_error?: string;
	enabled: boolean;
	created_at: string;
	updated_at: string;
}

export interface KomodoIntegrationConfig {
	api_key: string;
	username?: string;
	password?: string;
}

export interface KomodoContainer {
	id: string;
	org_id: string;
	integration_id: string;
	komodo_id: string;
	name: string;
	image?: string;
	stack_name?: string;
	stack_id?: string;
	status: KomodoContainerStatus;
	agent_id?: string;
	volumes?: string[];
	labels?: Record<string, string>;
	backup_enabled: boolean;
	last_discovered_at: string;
	created_at: string;
	updated_at: string;
}

export interface KomodoStack {
	id: string;
	org_id: string;
	integration_id: string;
	komodo_id: string;
	name: string;
	server_id?: string;
	server_name?: string;
	container_count: number;
	running_count: number;
	last_discovered_at: string;
	created_at: string;
	updated_at: string;
}

export interface KomodoWebhookEvent {
	id: string;
	org_id: string;
	integration_id: string;
	event_type: KomodoWebhookEventType;
	status: KomodoWebhookEventStatus;
	error_message?: string;
	processed_at?: string;
	created_at: string;
}

export interface KomodoDiscoveryResult {
	stacks: KomodoStack[];
	containers: KomodoContainer[];
	new_stacks: number;
	updated_stacks: number;
	new_containers: number;
	updated_containers: number;
	discovered_at: string;
}

export interface CreateKomodoIntegrationRequest {
	name: string;
	url: string;
	config: KomodoIntegrationConfig;
}

export interface UpdateKomodoIntegrationRequest {
	name?: string;
	url?: string;
	config?: KomodoIntegrationConfig;
	enabled?: boolean;
}

export interface UpdateKomodoContainerRequest {
	agent_id?: string;
	backup_enabled?: boolean;
}

export interface KomodoIntegrationsResponse {
	integrations: KomodoIntegration[];
}

export interface KomodoIntegrationResponse {
	integration: KomodoIntegration;
	containers?: KomodoContainer[];
	stacks?: KomodoStack[];
}

export interface KomodoContainersResponse {
	containers: KomodoContainer[];
}

export interface KomodoStacksResponse {
	stacks: KomodoStack[];
}

export interface KomodoWebhookEventsResponse {
	events: KomodoWebhookEvent[];
}

export interface KomodoSyncResponse {
	message: string;
	stacks: number;
	containers: number;
	result: KomodoDiscoveryResult;
}

export interface KomodoConnectionTestResponse {
	message: string;
	status: KomodoIntegrationStatus;
}

// System Health types (Admin)
export type SystemHealthStatus = 'healthy' | 'warning' | 'critical';
// Server Setup types
export type ServerSetupStep =
	| 'database'
	| 'superuser'
	| 'smtp'
	| 'oidc'
	| 'license'
	| 'organization'
	| 'complete';

export interface ServerSetupStatus {
	needs_setup: boolean;
	setup_completed: boolean;
	current_step: ServerSetupStep;
	completed_steps: ServerSetupStep[];
	database_ok: boolean;
	has_superuser: boolean;
}

export interface CreateSuperuserRequest {
	email: string;
	password: string;
	name: string;
}

export interface CreateSuperuserResponse {
	user_id: string;
	org_id: string;
	message: string;
}

export interface DatabaseTestResponse {
	ok: boolean;
	message: string;
}

export interface SetupStartTrialRequest {
	company_name?: string;
	contact_email: string;
}

export interface SetupStartTrialResponse {
	license_type: string;
	expires_at: string;
	message: string;
}

export interface CreateFirstOrgRequest {
	name: string;
}

export interface SetupCompleteResponse {
	message: string;
	redirect: string;
}

export interface SetupLicenseInfo {
	license_type: 'trial' | 'standard' | 'professional' | 'enterprise';
	status: string;
	max_agents?: number;
	max_repositories?: number;
	max_storage_gb?: number;
	expires_at?: string;
	company_name?: string;
}

export interface RerunStatusResponse {
	setup_completed: boolean;
	can_configure: string[];
	license?: SetupLicenseInfo;
}

// License types
export type LicenseTier = 'free' | 'pro' | 'professional' | 'enterprise';
export type LicenseStatus =
	| 'active'
	| 'expiring_soon'
	| 'expired'
	| 'grace_period';

export interface ServerStatus {
	status: SystemHealthStatus;
	cpu_usage: number;
	memory_usage: number;
	memory_alloc_mb: number;
	memory_total_alloc_mb: number;
	memory_sys_mb: number;
	goroutine_count: number;
	num_cpu: number;
	go_version: string;
	uptime_seconds: number;
}

export interface DatabaseStatus {
	status: SystemHealthStatus;
	connected: boolean;
	latency: string;
	active_connections: number;
	max_connections: number;
	size_bytes: number;
	size_formatted: string;
}

export interface QueueStatus {
	status: SystemHealthStatus;
	pending_backups: number;
	running_backups: number;
	total_queued: number;
}

export interface BackgroundJobStatus {
	status: SystemHealthStatus;
	goroutine_count: number;
	active_jobs: number;
}

export interface ServerError {
	id: string;
	level: string;
	message: string;
	component?: string;
	timestamp: string;
}

export interface SystemHealthResponse {
	status: SystemHealthStatus;
	timestamp: string;
	server: ServerStatus;
	database: DatabaseStatus;
	queue: QueueStatus;
	background_jobs: BackgroundJobStatus;
	recent_errors: ServerError[];
	issues?: string[];
}

export interface HealthHistoryRecord {
	id: string;
	timestamp: string;
	status: string;
	cpu_usage: number;
	memory_usage: number;
	memory_alloc_mb: number;
	memory_total_alloc_mb: number;
	goroutine_count: number;
	database_connections: number;
	database_size_bytes: number;
	pending_backups: number;
	running_backups: number;
	error_count: number;
}

export interface SystemHealthHistoryResponse {
	records: HealthHistoryRecord[];
	since: string;
	until: string;
}

export interface LicenseFeatures {
	max_agents: number;
	max_repositories: number;
	max_storage_bytes: number;
	sso_enabled: boolean;
	api_access: boolean;
	advanced_reporting: boolean;
	custom_branding: boolean;
	priority_support: boolean;
	backup_hooks: boolean;
	multi_destination: boolean;
}

export interface LicenseUsage {
	agents_used: number;
	agents_limit: number;
	repositories_used: number;
	repositories_limit: number;
	storage_used_bytes: number;
	storage_limit_bytes: number;
}

export interface License {
	id: string;
	org_id: string;
	license_key: string;
	tier: LicenseTier;
	status: LicenseStatus;
	valid_from: string;
	valid_until: string;
	grace_period_days: number;
	features: LicenseFeatures;
	usage: LicenseUsage;
	created_at: string;
	updated_at: string;
}

// Webhook types
export type WebhookEventType =
	| 'backup.started'
	| 'backup.completed'
	| 'backup.failed'
	| 'agent.online'
	| 'agent.offline'
	| 'restore.started'
	| 'restore.completed'
	| 'restore.failed'
	| 'alert.triggered'
	| 'alert.resolved';

export type WebhookDeliveryStatus =
	| 'pending'
	| 'delivered'
	| 'failed'
	| 'retrying';

export interface WebhookEndpoint {
	id: string;
	org_id: string;
	name: string;
	url: string;
	enabled: boolean;
	event_types: WebhookEventType[];
	headers?: Record<string, string>;
	retry_count: number;
	timeout_seconds: number;
	created_at: string;
	updated_at: string;
}

export interface LicenseHistory {
	id: string;
	license_id: string;
	org_id: string;
	action:
		| 'created'
		| 'renewed'
		| 'upgraded'
		| 'downgraded'
		| 'expired'
		| 'activated';
	previous_tier?: LicenseTier;
	new_tier?: LicenseTier;
	previous_expiry?: string;
	new_expiry?: string;
	notes?: string;
	created_at: string;
}

export interface LicenseExpirationInfo {
	license_id: string;
	is_expired: boolean;
	is_in_grace_period: boolean;
	days_until_expiry: number;
	grace_period_ends_at?: string;
}

export interface LicenseLimitsWarning {
	type: 'agents' | 'repositories' | 'storage';
	current: number;
	limit: number;
	percentage: number;
}

export interface LicenseWarnings {
	expiration?: LicenseExpirationInfo;
	limits: LicenseLimitsWarning[];
}

export interface CreateLicenseKeyRequest {
	license_key: string;
}

export interface UpdateLicenseRequest {
	tier?: LicenseTier;
	valid_until?: string;
	features?: Partial<LicenseFeatures>;
}

export interface LicenseResponse {
	license: License;
}

export interface LicensesResponse {
	licenses: License[];
	total_count: number;
}

export interface LicenseHistoryResponse {
	history: LicenseHistory[];
	total_count: number;
}

export interface LicenseValidateResponse {
	valid: boolean;
	tier?: LicenseTier;
	valid_until?: string;
	features?: LicenseFeatures;
	error?: string;
}

export interface LicenseWarningsResponse {
	warnings: LicenseWarnings;
}

export type ProFeature =
	| 'sso'
	| 'api_access'
	| 'advanced_reporting'
	| 'custom_branding'
	| 'priority_support'
	| 'backup_hooks'
	| 'multi_destination'
	| 'unlimited_agents'
	| 'unlimited_repositories'
	| 'unlimited_storage';

// License Feature Flag types
export type LicenseFeature =
	| 'oidc'
	| 'audit_logs'
	| 'multi_org'
	| 'sla_tracking'
	| 'white_label';

export interface FeatureInfo {
	name: LicenseFeature;
	display_name: string;
	description: string;
	required_tier: LicenseTier;
}

export interface TierInfo {
	name: LicenseTier;
	display_name: string;
	description: string;
	features: LicenseFeature[];
}

export interface UpgradeInfo {
	required_tier: LicenseTier;
	display_name: string;
	message: string;
}

export interface FeatureCheckResult {
	feature: LicenseFeature;
	enabled: boolean;
	current_tier: LicenseTier;
	required_tier: LicenseTier;
	upgrade_info?: UpgradeInfo;
}

export interface LicenseInfoSummary {
	org_id: string;
	tier: LicenseTier;
	features: LicenseFeature[];
}

export interface FeaturesResponse {
	features: FeatureInfo[];
}

export interface TiersResponse {
	tiers: TierInfo[];
}

export interface LicenseInfoResponse {
	license: LicenseInfoSummary;
}

export interface FeatureCheckResponse {
	result: FeatureCheckResult;
}

// Trial types
export type PlanTier = 'free' | 'pro' | 'enterprise';
export type TrialStatus = 'none' | 'active' | 'expired' | 'converted';

export interface TrialInfo {
	org_id: string;
	plan_tier: PlanTier;
	trial_status: TrialStatus;
	trial_started_at?: string;
	trial_ends_at?: string;
	trial_email?: string;
	trial_converted_at?: string;
	days_remaining: number;
	is_trial_active: boolean;
	has_pro_features: boolean;
}

export interface TrialExtension {
	id: string;
	org_id: string;
	extended_by: string;
	extended_by_name?: string;
	extension_days: number;
	reason?: string;
	previous_ends_at: string;
	new_ends_at: string;
	created_at: string;
}

export interface TrialActivity {
	id: string;
	org_id: string;
	user_id?: string;
	feature_name: string;
	action: string;
	details?: Record<string, unknown>;
	created_at: string;
}

export interface TrialProFeature {
	name: string;
	description: string;
	available: boolean;
	limit?: number;
}

export interface ExtendTrialRequest {
	extension_days: number;
	reason: string;
}

export interface ConvertTrialRequest {
	plan_tier: PlanTier;
}

export interface TrialFeaturesResponse {
	features: TrialProFeature[];
}

export interface TrialActivityResponse {
	activities: TrialActivity[];
}

export interface TrialExtensionsResponse {
	extensions: TrialExtension[];
}

// Migration Export/Import Types
export interface MigrationExportRequest {
	include_secrets?: boolean;
	include_system_config?: boolean;
	encryption_key?: string;
	description?: string;
}

export interface MigrationImportRequest {
	data: string;
	decryption_key?: string;
	conflict_resolution?: 'skip' | 'replace' | 'rename' | 'fail';
	dry_run?: boolean;
	target_org_slug?: string;
}

export interface MigrationValidationResult {
	valid: boolean;
	encrypted: boolean;
	requires_key: boolean;
	metadata?: MigrationMetadata;
	warnings?: string[];
	errors?: string[];
	entity_counts?: MigrationEntityCounts;
}

export interface MigrationMetadata {
	version: string;
	exported_at: string;
	exported_by: string;
	description?: string;
	source_instance?: string;
	includes_secrets: boolean;
	includes_system_config: boolean;
}

export interface MigrationEntityCounts {
	organizations?: number;
	users?: number;
	agents?: number;
	repositories?: number;
	schedules?: number;
	policies?: number;
}

export interface MigrationImportResult {
	success: boolean;
	dry_run: boolean;
	message?: string;
	imported: MigrationImportedCounts;
	skipped: MigrationImportedCounts;
	errors?: string[];
	warnings?: string[];
	id_mappings?: MigrationIDMappings;
}

export interface MigrationImportedCounts {
	organizations?: number;
	users?: number;
	agents?: number;
	repositories?: number;
	schedules?: number;
	policies?: number;
}

export interface MigrationIDMappings {
	organizations?: Record<string, string>;
	users?: Record<string, string>;
	agents?: Record<string, string>;
	repositories?: Record<string, string>;
	schedules?: Record<string, string>;
	policies?: Record<string, string>;
}

export interface AirGapLicenseInfo {
	customer_id: string;
	tier: string;
	expires_at: string;
	issued_at: string;
	valid: boolean;
}

export interface AirGapStatus {
	enabled: boolean;
	disabled_features: AirGapDisabledFeature[];
	license: AirGapLicenseInfo | null;
}

export interface WebhookDelivery {
	id: string;
	org_id: string;
	endpoint_id: string;
	event_type: WebhookEventType;
	event_id?: string;
	payload: Record<string, unknown>;
	request_headers?: Record<string, string>;
	response_status?: number;
	response_body?: string;
	response_headers?: Record<string, string>;
	attempt_number: number;
	max_attempts: number;
	status: WebhookDeliveryStatus;
	error_message?: string;
	delivered_at?: string;
	next_retry_at?: string;
	created_at: string;
}

export interface CreateWebhookEndpointRequest {
	name: string;
	url: string;
	secret: string;
	event_types: WebhookEventType[];
	headers?: Record<string, string>;
	retry_count?: number;
	timeout_seconds?: number;
}

export interface UpdateWebhookEndpointRequest {
	name?: string;
	url?: string;
	secret?: string;
	enabled?: boolean;
	event_types?: WebhookEventType[];
	headers?: Record<string, string>;
	retry_count?: number;
	timeout_seconds?: number;
}

export interface WebhookEndpointsResponse {
	endpoints: WebhookEndpoint[];
}

export interface WebhookDeliveriesResponse {
	deliveries: WebhookDelivery[];
	total: number;
}

export interface WebhookEventTypesResponse {
	event_types: WebhookEventType[];
}

export interface TestWebhookRequest {
	event_type?: WebhookEventType;
}

export interface TestWebhookResponse {
	success: boolean;
	response_status?: number;
	response_body?: string;
	error_message?: string;
	duration_ms: number;
}
