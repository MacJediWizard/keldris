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
