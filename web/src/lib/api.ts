import type {
	Agent,
	AgentsResponse,
	Backup,
	BackupsResponse,
	CreateAgentRequest,
	CreateAgentResponse,
	CreateRepositoryRequest,
	CreateRestoreRequest,
	CreateScheduleRequest,
	ErrorResponse,
	MessageResponse,
	RepositoriesResponse,
	Repository,
	Restore,
	RestoresResponse,
	RunScheduleResponse,
	Schedule,
	SchedulesResponse,
	Snapshot,
	SnapshotFilesResponse,
	SnapshotsResponse,
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
