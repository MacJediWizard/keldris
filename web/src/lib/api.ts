import type {
	Agent,
	AgentsResponse,
	Backup,
	BackupsResponse,
	CreateAgentRequest,
	CreateAgentResponse,
	CreateDRRunbookRequest,
	CreateDRTestScheduleRequest,
	CreateRepositoryRequest,
	CreateScheduleRequest,
	DRRunbook,
	DRRunbookRenderResponse,
	DRRunbooksResponse,
	DRStatus,
	DRTest,
	DRTestSchedule,
	DRTestSchedulesResponse,
	DRTestsResponse,
	ErrorResponse,
	MessageResponse,
	RepositoriesResponse,
	Repository,
	RunDRTestRequest,
	RunScheduleResponse,
	Schedule,
	SchedulesResponse,
	TestRepositoryResponse,
	UpdateDRRunbookRequest,
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
