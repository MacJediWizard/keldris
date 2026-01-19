import type {
	Agent,
	AgentsResponse,
	AuditLog,
	AuditLogFilter,
	AuditLogsResponse,
	Backup,
	BackupsResponse,
	CreateAgentRequest,
	CreateAgentResponse,
	CreateRepositoryRequest,
	CreateScheduleRequest,
	ErrorResponse,
	MessageResponse,
	RepositoriesResponse,
	Repository,
	RunScheduleResponse,
	Schedule,
	SchedulesResponse,
	TestRepositoryResponse,
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
};

// Repositories API
export const repositoriesApi = {
	list: async (): Promise<Repository[]> => {
		const response = await fetchApi<RepositoriesResponse>('/repositories');
		return response.repositories ?? [];
	},

	get: async (id: string): Promise<Repository> =>
		fetchApi<Repository>(`/repositories/${id}`),

	create: async (data: CreateRepositoryRequest): Promise<Repository> =>
		fetchApi<Repository>('/repositories', {
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
