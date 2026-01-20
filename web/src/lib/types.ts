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

// Repository types
export type RepositoryType = 'local' | 's3' | 'b2' | 'sftp' | 'rest';

export interface Repository {
	id: string;
	name: string;
	type: RepositoryType;
	created_at: string;
	updated_at: string;
}

export interface CreateRepositoryRequest {
	name: string;
	type: RepositoryType;
	config: Record<string, unknown>;
}

export interface UpdateRepositoryRequest {
	name?: string;
	config?: Record<string, unknown>;
}

export interface TestRepositoryResponse {
	success: boolean;
	message: string;
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

export interface Schedule {
	id: string;
	agent_id: string;
	repository_id: string;
	name: string;
	cron_expression: string;
	paths: string[];
	excludes?: string[];
	retention_policy?: RetentionPolicy;
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
	enabled?: boolean;
}

export interface UpdateScheduleRequest {
	name?: string;
	cron_expression?: string;
	paths?: string[];
	excludes?: string[];
	retention_policy?: RetentionPolicy;
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
	created_at: string;
}

// Auth types
export interface User {
	id: string;
	email: string;
	name: string;
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

// DR Runbook types
export type DRRunbookStatus = 'draft' | 'active' | 'archived';
export type DRRunbookStepType = 'manual' | 'restore' | 'verify' | 'notify';

export interface DRRunbookStep {
	order: number;
	title: string;
	description: string;
	type: DRRunbookStepType;
	command?: string;
	expected?: string;
}

export interface DRRunbookContact {
	name: string;
	role: string;
	email?: string;
	phone?: string;
	notify: boolean;
}

export interface DRRunbook {
	id: string;
	org_id: string;
	schedule_id?: string;
	name: string;
	description?: string;
	steps: DRRunbookStep[];
	contacts: DRRunbookContact[];
	credentials_location?: string;
	recovery_time_objective_minutes?: number;
	recovery_point_objective_minutes?: number;
	status: DRRunbookStatus;
	created_at: string;
	updated_at: string;
}

export interface CreateDRRunbookRequest {
	schedule_id?: string;
	name: string;
	description?: string;
	steps?: DRRunbookStep[];
	contacts?: DRRunbookContact[];
	credentials_location?: string;
	recovery_time_objective_minutes?: number;
	recovery_point_objective_minutes?: number;
}

export interface UpdateDRRunbookRequest {
	name?: string;
	description?: string;
	steps?: DRRunbookStep[];
	contacts?: DRRunbookContact[];
	credentials_location?: string;
	recovery_time_objective_minutes?: number;
	recovery_point_objective_minutes?: number;
	schedule_id?: string;
}

export interface DRRunbooksResponse {
	runbooks: DRRunbook[];
}

export interface DRRunbookRenderResponse {
	content: string;
	format: string;
}

// DR Test types
export type DRTestStatus =
	| 'scheduled'
	| 'running'
	| 'completed'
	| 'failed'
	| 'canceled';

export interface DRTest {
	id: string;
	runbook_id: string;
	schedule_id?: string;
	agent_id?: string;
	snapshot_id?: string;
	status: DRTestStatus;
	started_at?: string;
	completed_at?: string;
	restore_size_bytes?: number;
	restore_duration_seconds?: number;
	verification_passed?: boolean;
	notes?: string;
	error_message?: string;
	created_at: string;
}

export interface RunDRTestRequest {
	runbook_id: string;
	notes?: string;
}

export interface DRTestsResponse {
	tests: DRTest[];
}

// DR Test Schedule types
export interface DRTestSchedule {
	id: string;
	runbook_id: string;
	cron_expression: string;
	enabled: boolean;
	last_run_at?: string;
	next_run_at?: string;
	created_at: string;
	updated_at: string;
}

export interface CreateDRTestScheduleRequest {
	cron_expression: string;
	enabled?: boolean;
}

export interface DRTestSchedulesResponse {
	schedules: DRTestSchedule[];
}

// DR Status types
export interface DRStatus {
	total_runbooks: number;
	active_runbooks: number;
	last_test_at?: string;
	next_test_at?: string;
	tests_last_30_days: number;
	pass_rate: number;
}
