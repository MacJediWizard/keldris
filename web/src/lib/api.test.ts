import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';

// We need to re-import fresh modules per test group because fetchApi captures fetch at module load.
// To ensure our mocked fetch is used, we use vi.stubGlobal before importing.

function mockFetch(data: unknown, status = 200, ok = true) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

function mockFetchBlob(status = 200, ok = true) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		blob: () => Promise.resolve(new Blob(['data'])),
		json: () => Promise.resolve({ error: 'Failed' }),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

beforeEach(() => {
	vi.resetModules();
});

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('ApiError', () => {
	it('creates an error with status and message', async () => {
		const { ApiError } = await import('./api');
		const error = new ApiError(404, 'Not found');
		expect(error.status).toBe(404);
		expect(error.message).toBe('Not found');
		expect(error.name).toBe('ApiError');
	});

	it('is an instance of Error', async () => {
		const { ApiError } = await import('./api');
		const error = new ApiError(500, 'Server error');
		expect(error).toBeInstanceOf(Error);
	});
});

describe('fetchApi core behavior', () => {
	it('sends requests with credentials included', async () => {
		const fetchFn = mockFetch({ agents: [] });
		const { agentsApi } = await import('./api');
		await agentsApi.list();
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/agents',
			expect.objectContaining({
				credentials: 'include',
				headers: expect.objectContaining({ 'Content-Type': 'application/json' }),
			}),
		);
	});

	it('redirects to login on 401', async () => {
		mockFetch({ error: 'Unauthorized' }, 401, false);
		const { agentsApi } = await import('./api');
		await expect(agentsApi.list()).rejects.toThrow('Unauthorized');
		expect(window.location.href).toBe('/auth/login');
	});

	it('throws ApiError on non-ok responses', async () => {
		mockFetch({ error: 'Agent not found' }, 404, false);
		const { agentsApi, ApiError } = await import('./api');
		try {
			await agentsApi.get('x');
		} catch (e) {
			expect(e).toBeInstanceOf(ApiError);
			expect((e as InstanceType<typeof ApiError>).status).toBe(404);
		}
	});

	it('handles JSON parse errors gracefully', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn().mockResolvedValue({
				ok: false,
				status: 500,
				json: () => Promise.reject(new Error('bad json')),
			}),
		);
		const { agentsApi, ApiError } = await import('./api');
		try {
			await agentsApi.list();
		} catch (e) {
			expect(e).toBeInstanceOf(ApiError);
			expect((e as InstanceType<typeof ApiError>).message).toBe('Unknown error');
		}
	});
});

describe('authApi', () => {
	it('me calls /auth/me', async () => {
		const fetchFn = mockFetch({ id: '1', email: 'a@b.com', name: 'A' });
		const { authApi } = await import('./api');
		await authApi.me();
		expect(fetchFn).toHaveBeenCalledWith('/auth/me', expect.objectContaining({ credentials: 'include' }));
	});

	it('logout calls POST /auth/logout', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { authApi } = await import('./api');
		await authApi.logout();
		expect(fetchFn).toHaveBeenCalledWith('/auth/logout', expect.objectContaining({ method: 'POST' }));
	});

	it('updatePreferences sends PUT to /auth/preferences', async () => {
		const fetchFn = mockFetch({ id: '1', email: 'a@b.com', name: 'A' });
		const { authApi } = await import('./api');
		await authApi.updatePreferences({ language: 'es' });
		expect(fetchFn).toHaveBeenCalledWith(
			'/auth/preferences',
			expect.objectContaining({ method: 'PUT', body: JSON.stringify({ language: 'es' }) }),
		);
	});

	it('getLoginUrl returns /auth/login', async () => {
		mockFetch({});
		const { authApi } = await import('./api');
		expect(authApi.getLoginUrl()).toBe('/auth/login');
	});
});

describe('agentsApi', () => {
	it('list returns agents array', async () => {
		mockFetch({ agents: [{ id: '1' }] });
		const { agentsApi } = await import('./api');
		const result = await agentsApi.list();
		expect(result).toEqual([{ id: '1' }]);
	});

	it('list returns empty array when null', async () => {
		mockFetch({ agents: null });
		const { agentsApi } = await import('./api');
		const result = await agentsApi.list();
		expect(result).toEqual([]);
	});

	it('get calls /agents/:id', async () => {
		const fetchFn = mockFetch({ id: 'a1' });
		const { agentsApi } = await import('./api');
		await agentsApi.get('a1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/agents/a1', expect.any(Object));
	});

	it('create sends POST', async () => {
		const fetchFn = mockFetch({ id: '1', api_key: 'k' });
		const { agentsApi } = await import('./api');
		await agentsApi.create({ hostname: 'h1' });
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/agents',
			expect.objectContaining({ method: 'POST', body: JSON.stringify({ hostname: 'h1' }) }),
		);
	});

	it('delete sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { agentsApi } = await import('./api');
		await agentsApi.delete('a1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/agents/a1', expect.objectContaining({ method: 'DELETE' }));
	});

	it('rotateApiKey sends POST', async () => {
		const fetchFn = mockFetch({ api_key: 'new' });
		const { agentsApi } = await import('./api');
		await agentsApi.rotateApiKey('a1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/agents/a1/apikey/rotate', expect.objectContaining({ method: 'POST' }));
	});

	it('revokeApiKey sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { agentsApi } = await import('./api');
		await agentsApi.revokeApiKey('a1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/agents/a1/apikey', expect.objectContaining({ method: 'DELETE' }));
	});

	it('getStats calls /agents/:id/stats', async () => {
		const fetchFn = mockFetch({ total_backups: 10 });
		const { agentsApi } = await import('./api');
		await agentsApi.getStats('a1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/agents/a1/stats', expect.any(Object));
	});

	it('getBackups calls /agents/:id/backups', async () => {
		const fetchFn = mockFetch({ backups: [] });
		const { agentsApi } = await import('./api');
		await agentsApi.getBackups('a1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/agents/a1/backups', expect.any(Object));
	});

	it('getSchedules calls /agents/:id/schedules', async () => {
		const fetchFn = mockFetch({ schedules: [] });
		const { agentsApi } = await import('./api');
		await agentsApi.getSchedules('a1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/agents/a1/schedules', expect.any(Object));
	});

	it('getHealthHistory passes limit param', async () => {
		const fetchFn = mockFetch({ history: [] });
		const { agentsApi } = await import('./api');
		await agentsApi.getHealthHistory('a1', 50);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/agents/a1/health-history?limit=50', expect.any(Object));
	});

	it('getFleetHealth calls /agents/fleet-health', async () => {
		const fetchFn = mockFetch({ agents: [] });
		const { agentsApi } = await import('./api');
		await agentsApi.getFleetHealth();
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/agents/fleet-health', expect.any(Object));
	});

	it('listWithGroups returns agents array', async () => {
		mockFetch({ agents: [{ id: '1', groups: [] }] });
		const { agentsApi } = await import('./api');
		const result = await agentsApi.listWithGroups();
		expect(result).toEqual([{ id: '1', groups: [] }]);
	});
});

describe('agentGroupsApi', () => {
	it('list returns groups', async () => {
		mockFetch({ groups: [{ id: 'g1' }] });
		const { agentGroupsApi } = await import('./api');
		const result = await agentGroupsApi.list();
		expect(result).toEqual([{ id: 'g1' }]);
	});

	it('get calls correct endpoint', async () => {
		const fetchFn = mockFetch({ id: 'g1' });
		const { agentGroupsApi } = await import('./api');
		await agentGroupsApi.get('g1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/agent-groups/g1', expect.any(Object));
	});

	it('create sends POST', async () => {
		const fetchFn = mockFetch({ id: 'g1' });
		const { agentGroupsApi } = await import('./api');
		await agentGroupsApi.create({ name: 'Group 1', description: 'desc' } as Parameters<typeof agentGroupsApi.create>[0]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/agent-groups', expect.objectContaining({ method: 'POST' }));
	});

	it('update sends PUT', async () => {
		const fetchFn = mockFetch({ id: 'g1' });
		const { agentGroupsApi } = await import('./api');
		await agentGroupsApi.update('g1', { name: 'Updated' } as Parameters<typeof agentGroupsApi.update>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/agent-groups/g1', expect.objectContaining({ method: 'PUT' }));
	});

	it('delete sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { agentGroupsApi } = await import('./api');
		await agentGroupsApi.delete('g1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/agent-groups/g1', expect.objectContaining({ method: 'DELETE' }));
	});

	it('listMembers returns agents', async () => {
		mockFetch({ agents: [{ id: 'a1' }] });
		const { agentGroupsApi } = await import('./api');
		const result = await agentGroupsApi.listMembers('g1');
		expect(result).toEqual([{ id: 'a1' }]);
	});

	it('addAgent sends POST', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { agentGroupsApi } = await import('./api');
		await agentGroupsApi.addAgent('g1', { agent_id: 'a1' } as Parameters<typeof agentGroupsApi.addAgent>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/agent-groups/g1/agents', expect.objectContaining({ method: 'POST' }));
	});

	it('removeAgent sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { agentGroupsApi } = await import('./api');
		await agentGroupsApi.removeAgent('g1', 'a1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/agent-groups/g1/agents/a1', expect.objectContaining({ method: 'DELETE' }));
	});
});

describe('repositoriesApi', () => {
	it('list returns repositories', async () => {
		mockFetch({ repositories: [{ id: 'r1' }] });
		const { repositoriesApi } = await import('./api');
		const result = await repositoriesApi.list();
		expect(result).toEqual([{ id: 'r1' }]);
	});

	it('get calls /repositories/:id', async () => {
		const fetchFn = mockFetch({ id: 'r1' });
		const { repositoriesApi } = await import('./api');
		await repositoriesApi.get('r1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/repositories/r1', expect.any(Object));
	});

	it('create sends POST', async () => {
		const fetchFn = mockFetch({ id: 'r1' });
		const { repositoriesApi } = await import('./api');
		await repositoriesApi.create({ name: 'repo' } as Parameters<typeof repositoriesApi.create>[0]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/repositories', expect.objectContaining({ method: 'POST' }));
	});

	it('update sends PUT', async () => {
		const fetchFn = mockFetch({ id: 'r1' });
		const { repositoriesApi } = await import('./api');
		await repositoriesApi.update('r1', { name: 'updated' } as Parameters<typeof repositoriesApi.update>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/repositories/r1', expect.objectContaining({ method: 'PUT' }));
	});

	it('delete sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { repositoriesApi } = await import('./api');
		await repositoriesApi.delete('r1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/repositories/r1', expect.objectContaining({ method: 'DELETE' }));
	});

	it('test sends POST', async () => {
		const fetchFn = mockFetch({ success: true });
		const { repositoriesApi } = await import('./api');
		await repositoriesApi.test('r1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/repositories/r1/test', expect.objectContaining({ method: 'POST' }));
	});

	it('testConnection sends POST', async () => {
		const fetchFn = mockFetch({ success: true });
		const { repositoriesApi } = await import('./api');
		await repositoriesApi.testConnection({ type: 's3' } as Parameters<typeof repositoriesApi.testConnection>[0]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/repositories/test-connection', expect.objectContaining({ method: 'POST' }));
	});

	it('recoverKey calls correct endpoint', async () => {
		const fetchFn = mockFetch({ key: 'abc' });
		const { repositoriesApi } = await import('./api');
		await repositoriesApi.recoverKey('r1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/repositories/r1/key/recover', expect.any(Object));
	});
});

describe('schedulesApi', () => {
	it('list returns schedules', async () => {
		mockFetch({ schedules: [{ id: 's1' }] });
		const { schedulesApi } = await import('./api');
		const result = await schedulesApi.list();
		expect(result).toEqual([{ id: 's1' }]);
	});

	it('list with agentId adds query param', async () => {
		const fetchFn = mockFetch({ schedules: [] });
		const { schedulesApi } = await import('./api');
		await schedulesApi.list('a1');
		expect(fetchFn).toHaveBeenCalledWith(expect.stringContaining('agent_id=a1'), expect.any(Object));
	});

	it('get calls /schedules/:id', async () => {
		const fetchFn = mockFetch({ id: 's1' });
		const { schedulesApi } = await import('./api');
		await schedulesApi.get('s1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/schedules/s1', expect.any(Object));
	});

	it('create sends POST', async () => {
		const fetchFn = mockFetch({ id: 's1' });
		const { schedulesApi } = await import('./api');
		await schedulesApi.create({ name: 'sched' } as Parameters<typeof schedulesApi.create>[0]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/schedules', expect.objectContaining({ method: 'POST' }));
	});

	it('update sends PUT', async () => {
		const fetchFn = mockFetch({ id: 's1' });
		const { schedulesApi } = await import('./api');
		await schedulesApi.update('s1', { name: 'upd' } as Parameters<typeof schedulesApi.update>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/schedules/s1', expect.objectContaining({ method: 'PUT' }));
	});

	it('delete sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { schedulesApi } = await import('./api');
		await schedulesApi.delete('s1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/schedules/s1', expect.objectContaining({ method: 'DELETE' }));
	});

	it('run sends POST', async () => {
		const fetchFn = mockFetch({ message: 'started' });
		const { schedulesApi } = await import('./api');
		await schedulesApi.run('s1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/schedules/s1/run', expect.objectContaining({ method: 'POST' }));
	});

	it('getReplicationStatus returns status array', async () => {
		mockFetch({ replication_status: [{ id: 'rs1' }] });
		const { schedulesApi } = await import('./api');
		const result = await schedulesApi.getReplicationStatus('s1');
		expect(result).toEqual([{ id: 'rs1' }]);
	});
});

describe('policiesApi', () => {
	it('list returns policies', async () => {
		mockFetch({ policies: [{ id: 'p1' }] });
		const { policiesApi } = await import('./api');
		const result = await policiesApi.list();
		expect(result).toEqual([{ id: 'p1' }]);
	});

	it('get calls /policies/:id', async () => {
		const fetchFn = mockFetch({ id: 'p1' });
		const { policiesApi } = await import('./api');
		await policiesApi.get('p1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/policies/p1', expect.any(Object));
	});

	it('create sends POST', async () => {
		const fetchFn = mockFetch({ id: 'p1' });
		const { policiesApi } = await import('./api');
		await policiesApi.create({ name: 'pol' } as Parameters<typeof policiesApi.create>[0]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/policies', expect.objectContaining({ method: 'POST' }));
	});

	it('update sends PUT', async () => {
		const fetchFn = mockFetch({ id: 'p1' });
		const { policiesApi } = await import('./api');
		await policiesApi.update('p1', { name: 'u' } as Parameters<typeof policiesApi.update>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/policies/p1', expect.objectContaining({ method: 'PUT' }));
	});

	it('delete sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { policiesApi } = await import('./api');
		await policiesApi.delete('p1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/policies/p1', expect.objectContaining({ method: 'DELETE' }));
	});

	it('listSchedules returns schedules for policy', async () => {
		mockFetch({ schedules: [{ id: 's1' }] });
		const { policiesApi } = await import('./api');
		const result = await policiesApi.listSchedules('p1');
		expect(result).toEqual([{ id: 's1' }]);
	});

	it('apply sends POST', async () => {
		const fetchFn = mockFetch({ applied: 3 });
		const { policiesApi } = await import('./api');
		await policiesApi.apply('p1', { agent_ids: ['a1'] } as Parameters<typeof policiesApi.apply>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/policies/p1/apply', expect.objectContaining({ method: 'POST' }));
	});
});

describe('backupsApi', () => {
	it('list returns backups', async () => {
		mockFetch({ backups: [{ id: 'b1' }] });
		const { backupsApi } = await import('./api');
		const result = await backupsApi.list();
		expect(result).toEqual([{ id: 'b1' }]);
	});

	it('list with params adds query string', async () => {
		const fetchFn = mockFetch({ backups: [] });
		const { backupsApi } = await import('./api');
		await backupsApi.list({ agent_id: 'a1', status: 'completed', schedule_id: 's1' });
		expect(fetchFn).toHaveBeenCalledWith(
			expect.stringContaining('agent_id=a1'),
			expect.any(Object),
		);
		expect(fetchFn).toHaveBeenCalledWith(
			expect.stringContaining('status=completed'),
			expect.any(Object),
		);
		expect(fetchFn).toHaveBeenCalledWith(
			expect.stringContaining('schedule_id=s1'),
			expect.any(Object),
		);
	});

	it('get calls /backups/:id', async () => {
		const fetchFn = mockFetch({ id: 'b1' });
		const { backupsApi } = await import('./api');
		await backupsApi.get('b1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/backups/b1', expect.any(Object));
	});
});

describe('backupScriptsApi', () => {
	it('list returns scripts', async () => {
		mockFetch({ scripts: [{ id: 'bs1' }] });
		const { backupScriptsApi } = await import('./api');
		const result = await backupScriptsApi.list('s1');
		expect(result).toEqual([{ id: 'bs1' }]);
	});

	it('get calls correct endpoint', async () => {
		const fetchFn = mockFetch({ id: 'bs1' });
		const { backupScriptsApi } = await import('./api');
		await backupScriptsApi.get('s1', 'bs1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/schedules/s1/scripts/bs1', expect.any(Object));
	});

	it('create sends POST', async () => {
		const fetchFn = mockFetch({ id: 'bs1' });
		const { backupScriptsApi } = await import('./api');
		await backupScriptsApi.create('s1', { name: 'script' } as Parameters<typeof backupScriptsApi.create>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/schedules/s1/scripts', expect.objectContaining({ method: 'POST' }));
	});

	it('update sends PUT', async () => {
		const fetchFn = mockFetch({ id: 'bs1' });
		const { backupScriptsApi } = await import('./api');
		await backupScriptsApi.update('s1', 'bs1', { name: 'u' } as Parameters<typeof backupScriptsApi.update>[2]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/schedules/s1/scripts/bs1', expect.objectContaining({ method: 'PUT' }));
	});

	it('delete sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { backupScriptsApi } = await import('./api');
		await backupScriptsApi.delete('s1', 'bs1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/schedules/s1/scripts/bs1', expect.objectContaining({ method: 'DELETE' }));
	});
});

describe('snapshotsApi', () => {
	it('list returns snapshots', async () => {
		mockFetch({ snapshots: [{ id: 'sn1' }] });
		const { snapshotsApi } = await import('./api');
		const result = await snapshotsApi.list();
		expect(result).toEqual([{ id: 'sn1' }]);
	});

	it('list with params adds query string', async () => {
		const fetchFn = mockFetch({ snapshots: [] });
		const { snapshotsApi } = await import('./api');
		await snapshotsApi.list({ agent_id: 'a1', repository_id: 'r1' });
		expect(fetchFn).toHaveBeenCalledWith(
			expect.stringContaining('agent_id=a1'),
			expect.any(Object),
		);
	});

	it('get calls /snapshots/:id', async () => {
		const fetchFn = mockFetch({ id: 'sn1' });
		const { snapshotsApi } = await import('./api');
		await snapshotsApi.get('sn1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/snapshots/sn1', expect.any(Object));
	});

	it('listFiles calls correct endpoint', async () => {
		const fetchFn = mockFetch({ files: [] });
		const { snapshotsApi } = await import('./api');
		await snapshotsApi.listFiles('sn1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/snapshots/sn1/files', expect.any(Object));
	});

	it('listFiles with path encodes path', async () => {
		const fetchFn = mockFetch({ files: [] });
		const { snapshotsApi } = await import('./api');
		await snapshotsApi.listFiles('sn1', '/home/user');
		expect(fetchFn).toHaveBeenCalledWith(
			expect.stringContaining('path=%2Fhome%2Fuser'),
			expect.any(Object),
		);
	});

	it('compare calls /snapshots/:id1/compare/:id2', async () => {
		const fetchFn = mockFetch({ added: [], removed: [] });
		const { snapshotsApi } = await import('./api');
		await snapshotsApi.compare('sn1', 'sn2');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/snapshots/sn1/compare/sn2', expect.any(Object));
	});
});

describe('snapshotCommentsApi', () => {
	it('list returns comments', async () => {
		mockFetch({ comments: [{ id: 'c1' }] });
		const { snapshotCommentsApi } = await import('./api');
		const result = await snapshotCommentsApi.list('sn1');
		expect(result).toEqual([{ id: 'c1' }]);
	});

	it('create sends POST', async () => {
		const fetchFn = mockFetch({ id: 'c1' });
		const { snapshotCommentsApi } = await import('./api');
		await snapshotCommentsApi.create('sn1', { content: 'hello' } as Parameters<typeof snapshotCommentsApi.create>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/snapshots/sn1/comments', expect.objectContaining({ method: 'POST' }));
	});

	it('delete sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { snapshotCommentsApi } = await import('./api');
		await snapshotCommentsApi.delete('c1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/comments/c1', expect.objectContaining({ method: 'DELETE' }));
	});
});

describe('restoresApi', () => {
	it('list returns restores', async () => {
		mockFetch({ restores: [{ id: 'rs1' }] });
		const { restoresApi } = await import('./api');
		const result = await restoresApi.list();
		expect(result).toEqual([{ id: 'rs1' }]);
	});

	it('list with params adds query string', async () => {
		const fetchFn = mockFetch({ restores: [] });
		const { restoresApi } = await import('./api');
		await restoresApi.list({ agent_id: 'a1', status: 'completed' });
		expect(fetchFn).toHaveBeenCalledWith(
			expect.stringContaining('agent_id=a1'),
			expect.any(Object),
		);
	});

	it('get calls /restores/:id', async () => {
		const fetchFn = mockFetch({ id: 'rs1' });
		const { restoresApi } = await import('./api');
		await restoresApi.get('rs1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/restores/rs1', expect.any(Object));
	});

	it('create sends POST', async () => {
		const fetchFn = mockFetch({ id: 'rs1' });
		const { restoresApi } = await import('./api');
		await restoresApi.create({ snapshot_id: 'sn1' } as Parameters<typeof restoresApi.create>[0]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/restores', expect.objectContaining({ method: 'POST' }));
	});
});

describe('fileHistoryApi', () => {
	it('getHistory builds correct query params', async () => {
		const fetchFn = mockFetch({ versions: [] });
		const { fileHistoryApi } = await import('./api');
		await fileHistoryApi.getHistory({ path: '/etc/config', agent_id: 'a1', repository_id: 'r1' });
		expect(fetchFn).toHaveBeenCalledWith(
			expect.stringContaining('path=%2Fetc%2Fconfig'),
			expect.any(Object),
		);
		expect(fetchFn).toHaveBeenCalledWith(
			expect.stringContaining('agent_id=a1'),
			expect.any(Object),
		);
	});
});

describe('alertsApi', () => {
	it('list returns alerts', async () => {
		mockFetch({ alerts: [{ id: 'al1' }] });
		const { alertsApi } = await import('./api');
		const result = await alertsApi.list();
		expect(result).toEqual([{ id: 'al1' }]);
	});

	it('listActive calls /alerts/active', async () => {
		const fetchFn = mockFetch({ alerts: [] });
		const { alertsApi } = await import('./api');
		await alertsApi.listActive();
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/alerts/active', expect.any(Object));
	});

	it('count returns number', async () => {
		mockFetch({ count: 5 });
		const { alertsApi } = await import('./api');
		const result = await alertsApi.count();
		expect(result).toBe(5);
	});

	it('get calls /alerts/:id', async () => {
		const fetchFn = mockFetch({ id: 'al1' });
		const { alertsApi } = await import('./api');
		await alertsApi.get('al1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/alerts/al1', expect.any(Object));
	});

	it('acknowledge sends POST', async () => {
		const fetchFn = mockFetch({ id: 'al1' });
		const { alertsApi } = await import('./api');
		await alertsApi.acknowledge('al1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/alerts/al1/actions/acknowledge', expect.objectContaining({ method: 'POST' }));
	});

	it('resolve sends POST', async () => {
		const fetchFn = mockFetch({ id: 'al1' });
		const { alertsApi } = await import('./api');
		await alertsApi.resolve('al1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/alerts/al1/actions/resolve', expect.objectContaining({ method: 'POST' }));
	});
});

describe('alertRulesApi', () => {
	it('list returns rules', async () => {
		mockFetch({ rules: [{ id: 'ar1' }] });
		const { alertRulesApi } = await import('./api');
		const result = await alertRulesApi.list();
		expect(result).toEqual([{ id: 'ar1' }]);
	});

	it('create sends POST', async () => {
		const fetchFn = mockFetch({ id: 'ar1' });
		const { alertRulesApi } = await import('./api');
		await alertRulesApi.create({ name: 'rule' } as Parameters<typeof alertRulesApi.create>[0]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/alert-rules', expect.objectContaining({ method: 'POST' }));
	});

	it('update sends PUT', async () => {
		const fetchFn = mockFetch({ id: 'ar1' });
		const { alertRulesApi } = await import('./api');
		await alertRulesApi.update('ar1', { name: 'u' } as Parameters<typeof alertRulesApi.update>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/alert-rules/ar1', expect.objectContaining({ method: 'PUT' }));
	});

	it('delete sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { alertRulesApi } = await import('./api');
		await alertRulesApi.delete('ar1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/alert-rules/ar1', expect.objectContaining({ method: 'DELETE' }));
	});
});

describe('organizationsApi', () => {
	it('list returns organizations', async () => {
		mockFetch({ organizations: [{ id: 'o1' }] });
		const { organizationsApi } = await import('./api');
		const result = await organizationsApi.list();
		expect(result).toEqual([{ id: 'o1' }]);
	});

	it('get calls /organizations/:id', async () => {
		const fetchFn = mockFetch({ organization: {}, role: 'admin' });
		const { organizationsApi } = await import('./api');
		await organizationsApi.get('o1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/organizations/o1', expect.any(Object));
	});

	it('getCurrent calls /organizations/current', async () => {
		const fetchFn = mockFetch({ organization: {}, role: 'admin' });
		const { organizationsApi } = await import('./api');
		await organizationsApi.getCurrent();
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/organizations/current', expect.any(Object));
	});

	it('create sends POST', async () => {
		const fetchFn = mockFetch({ organization: {} });
		const { organizationsApi } = await import('./api');
		await organizationsApi.create({ name: 'Org', slug: 'org' } as Parameters<typeof organizationsApi.create>[0]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/organizations', expect.objectContaining({ method: 'POST' }));
	});

	it('update sends PUT', async () => {
		const fetchFn = mockFetch({ organization: {} });
		const { organizationsApi } = await import('./api');
		await organizationsApi.update('o1', { name: 'U' } as Parameters<typeof organizationsApi.update>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/organizations/o1', expect.objectContaining({ method: 'PUT' }));
	});

	it('delete sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { organizationsApi } = await import('./api');
		await organizationsApi.delete('o1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/organizations/o1', expect.objectContaining({ method: 'DELETE' }));
	});

	it('switch sends POST', async () => {
		const fetchFn = mockFetch({ organization: {} });
		const { organizationsApi } = await import('./api');
		await organizationsApi.switch({ org_id: 'o2' } as Parameters<typeof organizationsApi.switch>[0]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/organizations/switch', expect.objectContaining({ method: 'POST' }));
	});

	it('listMembers returns members', async () => {
		mockFetch({ members: [{ id: 'm1' }] });
		const { organizationsApi } = await import('./api');
		const result = await organizationsApi.listMembers('o1');
		expect(result).toEqual([{ id: 'm1' }]);
	});

	it('updateMember sends PUT', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { organizationsApi } = await import('./api');
		await organizationsApi.updateMember('o1', 'u1', { role: 'admin' });
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/organizations/o1/members/u1', expect.objectContaining({ method: 'PUT' }));
	});

	it('removeMember sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { organizationsApi } = await import('./api');
		await organizationsApi.removeMember('o1', 'u1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/organizations/o1/members/u1', expect.objectContaining({ method: 'DELETE' }));
	});

	it('listInvitations returns invitations', async () => {
		mockFetch({ invitations: [{ id: 'i1' }] });
		const { organizationsApi } = await import('./api');
		const result = await organizationsApi.listInvitations('o1');
		expect(result).toEqual([{ id: 'i1' }]);
	});

	it('createInvitation sends POST', async () => {
		const fetchFn = mockFetch({ id: 'i1' });
		const { organizationsApi } = await import('./api');
		await organizationsApi.createInvitation('o1', { email: 'a@b.com', role: 'member' });
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/organizations/o1/invitations', expect.objectContaining({ method: 'POST' }));
	});

	it('deleteInvitation sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { organizationsApi } = await import('./api');
		await organizationsApi.deleteInvitation('o1', 'i1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/organizations/o1/invitations/i1', expect.objectContaining({ method: 'DELETE' }));
	});

	it('acceptInvitation sends POST', async () => {
		const fetchFn = mockFetch({ organization: {} });
		const { organizationsApi } = await import('./api');
		await organizationsApi.acceptInvitation('tok123');
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/invitations/accept',
			expect.objectContaining({ method: 'POST', body: JSON.stringify({ token: 'tok123' }) }),
		);
	});
});

describe('notificationsApi', () => {
	it('listChannels returns channels', async () => {
		mockFetch({ channels: [{ id: 'ch1' }] });
		const { notificationsApi } = await import('./api');
		const result = await notificationsApi.listChannels();
		expect(result).toEqual([{ id: 'ch1' }]);
	});

	it('getChannel calls correct endpoint', async () => {
		const fetchFn = mockFetch({ channel: {} });
		const { notificationsApi } = await import('./api');
		await notificationsApi.getChannel('ch1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/notifications/channels/ch1', expect.any(Object));
	});

	it('createChannel sends POST', async () => {
		const fetchFn = mockFetch({ id: 'ch1' });
		const { notificationsApi } = await import('./api');
		await notificationsApi.createChannel({ type: 'email' } as Parameters<typeof notificationsApi.createChannel>[0]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/notifications/channels', expect.objectContaining({ method: 'POST' }));
	});

	it('updateChannel sends PUT', async () => {
		const fetchFn = mockFetch({ id: 'ch1' });
		const { notificationsApi } = await import('./api');
		await notificationsApi.updateChannel('ch1', { enabled: false } as Parameters<typeof notificationsApi.updateChannel>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/notifications/channels/ch1', expect.objectContaining({ method: 'PUT' }));
	});

	it('deleteChannel sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { notificationsApi } = await import('./api');
		await notificationsApi.deleteChannel('ch1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/notifications/channels/ch1', expect.objectContaining({ method: 'DELETE' }));
	});

	it('listPreferences returns preferences', async () => {
		mockFetch({ preferences: [{ id: 'pr1' }] });
		const { notificationsApi } = await import('./api');
		const result = await notificationsApi.listPreferences();
		expect(result).toEqual([{ id: 'pr1' }]);
	});

	it('createPreference sends POST', async () => {
		const fetchFn = mockFetch({ id: 'pr1' });
		const { notificationsApi } = await import('./api');
		await notificationsApi.createPreference({ channel_id: 'ch1' } as Parameters<typeof notificationsApi.createPreference>[0]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/notifications/preferences', expect.objectContaining({ method: 'POST' }));
	});

	it('updatePreference sends PUT', async () => {
		const fetchFn = mockFetch({ id: 'pr1' });
		const { notificationsApi } = await import('./api');
		await notificationsApi.updatePreference('pr1', { enabled: true } as Parameters<typeof notificationsApi.updatePreference>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/notifications/preferences/pr1', expect.objectContaining({ method: 'PUT' }));
	});

	it('deletePreference sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { notificationsApi } = await import('./api');
		await notificationsApi.deletePreference('pr1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/notifications/preferences/pr1', expect.objectContaining({ method: 'DELETE' }));
	});

	it('listLogs returns logs', async () => {
		mockFetch({ logs: [{ id: 'l1' }] });
		const { notificationsApi } = await import('./api');
		const result = await notificationsApi.listLogs();
		expect(result).toEqual([{ id: 'l1' }]);
	});
});

describe('auditLogsApi', () => {
	it('list returns audit logs response', async () => {
		const fetchFn = mockFetch({ logs: [], total: 0 });
		const { auditLogsApi } = await import('./api');
		await auditLogsApi.list();
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/audit-logs', expect.any(Object));
	});

	it('list with filters adds query params', async () => {
		const fetchFn = mockFetch({ logs: [], total: 0 });
		const { auditLogsApi } = await import('./api');
		await auditLogsApi.list({ action: 'create', resource_type: 'agent', result: 'success', start_date: '2024-01-01', end_date: '2024-12-31', search: 'test', limit: 10, offset: 5 });
		const url = fetchFn.mock.calls[0][0] as string;
		expect(url).toContain('action=create');
		expect(url).toContain('resource_type=agent');
		expect(url).toContain('result=success');
		expect(url).toContain('limit=10');
		expect(url).toContain('offset=5');
	});

	it('get calls /audit-logs/:id', async () => {
		const fetchFn = mockFetch({ id: 'al1' });
		const { auditLogsApi } = await import('./api');
		await auditLogsApi.get('al1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/audit-logs/al1', expect.any(Object));
	});

	it('exportCsv returns blob', async () => {
		const fetchFn = mockFetchBlob();
		const { auditLogsApi } = await import('./api');
		const result = await auditLogsApi.exportCsv();
		expect(result).toBeInstanceOf(Blob);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/audit-logs/export/csv', expect.any(Object));
	});

	it('exportCsv with filters adds query params', async () => {
		const fetchFn = mockFetchBlob();
		const { auditLogsApi } = await import('./api');
		await auditLogsApi.exportCsv({ action: 'create' });
		expect(fetchFn).toHaveBeenCalledWith(expect.stringContaining('action=create'), expect.any(Object));
	});

	it('exportJson returns blob', async () => {
		const fetchFn = mockFetchBlob();
		const { auditLogsApi } = await import('./api');
		const result = await auditLogsApi.exportJson();
		expect(result).toBeInstanceOf(Blob);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/audit-logs/export/json', expect.any(Object));
	});

	it('exportCsv throws on error', async () => {
		mockFetchBlob(500, false);
		const { auditLogsApi } = await import('./api');
		await expect(auditLogsApi.exportCsv()).rejects.toThrow('Failed to export audit logs');
	});

	it('exportJson throws on error', async () => {
		mockFetchBlob(500, false);
		const { auditLogsApi } = await import('./api');
		await expect(auditLogsApi.exportJson()).rejects.toThrow('Failed to export audit logs');
	});
});

describe('statsApi', () => {
	it('getSummary calls /stats/summary', async () => {
		const fetchFn = mockFetch({ total_repos: 5 });
		const { statsApi } = await import('./api');
		await statsApi.getSummary();
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/stats/summary', expect.any(Object));
	});

	it('getGrowth returns growth array', async () => {
		mockFetch({ growth: [{ date: '2024-01-01', bytes: 100 }] });
		const { statsApi } = await import('./api');
		const result = await statsApi.getGrowth(7);
		expect(result).toEqual([{ date: '2024-01-01', bytes: 100 }]);
	});

	it('listRepositoryStats returns stats', async () => {
		mockFetch({ stats: [{ id: 'r1' }] });
		const { statsApi } = await import('./api');
		const result = await statsApi.listRepositoryStats();
		expect(result).toEqual([{ id: 'r1' }]);
	});

	it('getRepositoryStats calls correct endpoint', async () => {
		const fetchFn = mockFetch({ stats: {} });
		const { statsApi } = await import('./api');
		await statsApi.getRepositoryStats('r1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/stats/repositories/r1', expect.any(Object));
	});

	it('getRepositoryGrowth passes days param', async () => {
		const fetchFn = mockFetch({ growth: [] });
		const { statsApi } = await import('./api');
		await statsApi.getRepositoryGrowth('r1', 14);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/stats/repositories/r1/growth?days=14', expect.any(Object));
	});

	it('getRepositoryHistory passes limit param', async () => {
		const fetchFn = mockFetch({ history: [] });
		const { statsApi } = await import('./api');
		await statsApi.getRepositoryHistory('r1', 10);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/stats/repositories/r1/history?limit=10', expect.any(Object));
	});
});

describe('verificationsApi', () => {
	it('listByRepository returns verifications', async () => {
		mockFetch({ verifications: [{ id: 'v1' }] });
		const { verificationsApi } = await import('./api');
		const result = await verificationsApi.listByRepository('r1');
		expect(result).toEqual([{ id: 'v1' }]);
	});

	it('get calls correct endpoint', async () => {
		const fetchFn = mockFetch({ id: 'v1' });
		const { verificationsApi } = await import('./api');
		await verificationsApi.get('v1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/verifications/v1', expect.any(Object));
	});

	it('getStatus calls correct endpoint', async () => {
		const fetchFn = mockFetch({ status: 'ok' });
		const { verificationsApi } = await import('./api');
		await verificationsApi.getStatus('r1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/repositories/r1/verification-status', expect.any(Object));
	});

	it('trigger sends POST', async () => {
		const fetchFn = mockFetch({ id: 'v1' });
		const { verificationsApi } = await import('./api');
		await verificationsApi.trigger('r1', { type: 'full' } as Parameters<typeof verificationsApi.trigger>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/repositories/r1/verifications', expect.objectContaining({ method: 'POST' }));
	});

	it('listSchedules returns schedules', async () => {
		mockFetch({ schedules: [{ id: 'vs1' }] });
		const { verificationsApi } = await import('./api');
		const result = await verificationsApi.listSchedules('r1');
		expect(result).toEqual([{ id: 'vs1' }]);
	});

	it('createSchedule sends POST', async () => {
		const fetchFn = mockFetch({ id: 'vs1' });
		const { verificationsApi } = await import('./api');
		await verificationsApi.createSchedule('r1', { cron: '0 0 * * *' } as Parameters<typeof verificationsApi.createSchedule>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/repositories/r1/verification-schedules', expect.objectContaining({ method: 'POST' }));
	});

	it('getSchedule calls correct endpoint', async () => {
		const fetchFn = mockFetch({ id: 'vs1' });
		const { verificationsApi } = await import('./api');
		await verificationsApi.getSchedule('vs1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/verification-schedules/vs1', expect.any(Object));
	});

	it('updateSchedule sends PUT', async () => {
		const fetchFn = mockFetch({ id: 'vs1' });
		const { verificationsApi } = await import('./api');
		await verificationsApi.updateSchedule('vs1', { cron: '0 1 * * *' } as Parameters<typeof verificationsApi.updateSchedule>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/verification-schedules/vs1', expect.objectContaining({ method: 'PUT' }));
	});

	it('deleteSchedule sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { verificationsApi } = await import('./api');
		await verificationsApi.deleteSchedule('vs1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/verification-schedules/vs1', expect.objectContaining({ method: 'DELETE' }));
	});
});

describe('ssoGroupMappingsApi', () => {
	it('list returns mappings', async () => {
		mockFetch({ mappings: [{ id: 'sm1' }] });
		const { ssoGroupMappingsApi } = await import('./api');
		const result = await ssoGroupMappingsApi.list('o1');
		expect(result).toEqual([{ id: 'sm1' }]);
	});

	it('get returns mapping', async () => {
		mockFetch({ mapping: { id: 'sm1' } });
		const { ssoGroupMappingsApi } = await import('./api');
		const result = await ssoGroupMappingsApi.get('o1', 'sm1');
		expect(result).toEqual({ id: 'sm1' });
	});

	it('create sends POST', async () => {
		const fetchFn = mockFetch({ mapping: { id: 'sm1' } });
		const { ssoGroupMappingsApi } = await import('./api');
		await ssoGroupMappingsApi.create('o1', { oidc_group_name: 'eng', role: 'member' });
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/organizations/o1/sso-group-mappings', expect.objectContaining({ method: 'POST' }));
	});

	it('update sends PUT', async () => {
		const fetchFn = mockFetch({ mapping: { id: 'sm1' } });
		const { ssoGroupMappingsApi } = await import('./api');
		await ssoGroupMappingsApi.update('o1', 'sm1', { role: 'admin' });
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/organizations/o1/sso-group-mappings/sm1', expect.objectContaining({ method: 'PUT' }));
	});

	it('delete sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { ssoGroupMappingsApi } = await import('./api');
		await ssoGroupMappingsApi.delete('o1', 'sm1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/organizations/o1/sso-group-mappings/sm1', expect.objectContaining({ method: 'DELETE' }));
	});

	it('getSettings calls correct endpoint', async () => {
		const fetchFn = mockFetch({ default_role: 'member' });
		const { ssoGroupMappingsApi } = await import('./api');
		await ssoGroupMappingsApi.getSettings('o1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/organizations/o1/sso-settings', expect.any(Object));
	});

	it('updateSettings sends PUT', async () => {
		const fetchFn = mockFetch({ default_role: 'admin' });
		const { ssoGroupMappingsApi } = await import('./api');
		await ssoGroupMappingsApi.updateSettings('o1', { default_role: 'admin' });
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/organizations/o1/sso-settings', expect.objectContaining({ method: 'PUT' }));
	});

	it('getUserSSOGroups calls correct endpoint', async () => {
		const fetchFn = mockFetch({ user_id: 'u1', oidc_groups: [] });
		const { ssoGroupMappingsApi } = await import('./api');
		await ssoGroupMappingsApi.getUserSSOGroups('u1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/users/u1/sso-groups', expect.any(Object));
	});
});

describe('maintenanceApi', () => {
	it('list returns windows', async () => {
		mockFetch({ maintenance_windows: [{ id: 'mw1' }] });
		const { maintenanceApi } = await import('./api');
		const result = await maintenanceApi.list();
		expect(result).toEqual([{ id: 'mw1' }]);
	});

	it('get calls correct endpoint', async () => {
		const fetchFn = mockFetch({ id: 'mw1' });
		const { maintenanceApi } = await import('./api');
		await maintenanceApi.get('mw1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/maintenance-windows/mw1', expect.any(Object));
	});

	it('create sends POST', async () => {
		const fetchFn = mockFetch({ id: 'mw1' });
		const { maintenanceApi } = await import('./api');
		await maintenanceApi.create({ name: 'window' } as Parameters<typeof maintenanceApi.create>[0]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/maintenance-windows', expect.objectContaining({ method: 'POST' }));
	});

	it('update sends PUT', async () => {
		const fetchFn = mockFetch({ id: 'mw1' });
		const { maintenanceApi } = await import('./api');
		await maintenanceApi.update('mw1', { name: 'u' } as Parameters<typeof maintenanceApi.update>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/maintenance-windows/mw1', expect.objectContaining({ method: 'PUT' }));
	});

	it('delete sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { maintenanceApi } = await import('./api');
		await maintenanceApi.delete('mw1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/maintenance-windows/mw1', expect.objectContaining({ method: 'DELETE' }));
	});

	it('getActive calls /maintenance/active', async () => {
		const fetchFn = mockFetch({ active: false });
		const { maintenanceApi } = await import('./api');
		await maintenanceApi.getActive();
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/maintenance/active', expect.any(Object));
	});
});

describe('excludePatternsApi', () => {
	it('list returns patterns', async () => {
		mockFetch({ patterns: [{ id: 'ep1' }] });
		const { excludePatternsApi } = await import('./api');
		const result = await excludePatternsApi.list();
		expect(result).toEqual([{ id: 'ep1' }]);
	});

	it('list with category adds query param', async () => {
		const fetchFn = mockFetch({ patterns: [] });
		const { excludePatternsApi } = await import('./api');
		await excludePatternsApi.list('logs');
		expect(fetchFn).toHaveBeenCalledWith(expect.stringContaining('category=logs'), expect.any(Object));
	});

	it('get calls correct endpoint', async () => {
		const fetchFn = mockFetch({ id: 'ep1' });
		const { excludePatternsApi } = await import('./api');
		await excludePatternsApi.get('ep1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/exclude-patterns/ep1', expect.any(Object));
	});

	it('getLibrary returns built-in patterns', async () => {
		mockFetch({ patterns: [{ name: 'node_modules' }] });
		const { excludePatternsApi } = await import('./api');
		const result = await excludePatternsApi.getLibrary();
		expect(result).toEqual([{ name: 'node_modules' }]);
	});

	it('getCategories returns categories', async () => {
		mockFetch({ categories: [{ name: 'logs' }] });
		const { excludePatternsApi } = await import('./api');
		const result = await excludePatternsApi.getCategories();
		expect(result).toEqual([{ name: 'logs' }]);
	});

	it('create sends POST', async () => {
		const fetchFn = mockFetch({ id: 'ep1' });
		const { excludePatternsApi } = await import('./api');
		await excludePatternsApi.create({ pattern: '*.log' } as Parameters<typeof excludePatternsApi.create>[0]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/exclude-patterns', expect.objectContaining({ method: 'POST' }));
	});

	it('update sends PUT', async () => {
		const fetchFn = mockFetch({ id: 'ep1' });
		const { excludePatternsApi } = await import('./api');
		await excludePatternsApi.update('ep1', { pattern: '*.tmp' } as Parameters<typeof excludePatternsApi.update>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/exclude-patterns/ep1', expect.objectContaining({ method: 'PUT' }));
	});

	it('delete sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { excludePatternsApi } = await import('./api');
		await excludePatternsApi.delete('ep1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/exclude-patterns/ep1', expect.objectContaining({ method: 'DELETE' }));
	});
});

describe('drRunbooksApi', () => {
	it('list returns runbooks', async () => {
		mockFetch({ runbooks: [{ id: 'dr1' }] });
		const { drRunbooksApi } = await import('./api');
		const result = await drRunbooksApi.list();
		expect(result).toEqual([{ id: 'dr1' }]);
	});

	it('get calls correct endpoint', async () => {
		const fetchFn = mockFetch({ id: 'dr1' });
		const { drRunbooksApi } = await import('./api');
		await drRunbooksApi.get('dr1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/dr-runbooks/dr1', expect.any(Object));
	});

	it('create sends POST', async () => {
		const fetchFn = mockFetch({ id: 'dr1' });
		const { drRunbooksApi } = await import('./api');
		await drRunbooksApi.create({ name: 'DR Plan' } as Parameters<typeof drRunbooksApi.create>[0]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/dr-runbooks', expect.objectContaining({ method: 'POST' }));
	});

	it('update sends PUT', async () => {
		const fetchFn = mockFetch({ id: 'dr1' });
		const { drRunbooksApi } = await import('./api');
		await drRunbooksApi.update('dr1', { name: 'Updated' } as Parameters<typeof drRunbooksApi.update>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/dr-runbooks/dr1', expect.objectContaining({ method: 'PUT' }));
	});

	it('delete sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { drRunbooksApi } = await import('./api');
		await drRunbooksApi.delete('dr1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/dr-runbooks/dr1', expect.objectContaining({ method: 'DELETE' }));
	});

	it('activate sends POST', async () => {
		const fetchFn = mockFetch({ id: 'dr1' });
		const { drRunbooksApi } = await import('./api');
		await drRunbooksApi.activate('dr1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/dr-runbooks/dr1/activate', expect.objectContaining({ method: 'POST' }));
	});

	it('archive sends POST', async () => {
		const fetchFn = mockFetch({ id: 'dr1' });
		const { drRunbooksApi } = await import('./api');
		await drRunbooksApi.archive('dr1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/dr-runbooks/dr1/archive', expect.objectContaining({ method: 'POST' }));
	});

	it('render calls correct endpoint', async () => {
		const fetchFn = mockFetch({ html: '<p>plan</p>' });
		const { drRunbooksApi } = await import('./api');
		await drRunbooksApi.render('dr1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/dr-runbooks/dr1/render', expect.any(Object));
	});

	it('generateFromSchedule sends POST', async () => {
		const fetchFn = mockFetch({ id: 'dr1' });
		const { drRunbooksApi } = await import('./api');
		await drRunbooksApi.generateFromSchedule('s1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/dr-runbooks/s1/generate', expect.objectContaining({ method: 'POST' }));
	});

	it('getStatus calls /dr-runbooks/status', async () => {
		const fetchFn = mockFetch({ total: 5 });
		const { drRunbooksApi } = await import('./api');
		await drRunbooksApi.getStatus();
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/dr-runbooks/status', expect.any(Object));
	});

	it('listTestSchedules returns schedules', async () => {
		mockFetch({ schedules: [{ id: 'ts1' }] });
		const { drRunbooksApi } = await import('./api');
		const result = await drRunbooksApi.listTestSchedules('dr1');
		expect(result).toEqual([{ id: 'ts1' }]);
	});

	it('createTestSchedule sends POST', async () => {
		const fetchFn = mockFetch({ id: 'ts1' });
		const { drRunbooksApi } = await import('./api');
		await drRunbooksApi.createTestSchedule('dr1', { cron: '0 0 * * *' } as Parameters<typeof drRunbooksApi.createTestSchedule>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/dr-runbooks/dr1/test-schedules', expect.objectContaining({ method: 'POST' }));
	});
});

describe('drTestsApi', () => {
	it('list returns tests', async () => {
		mockFetch({ tests: [{ id: 'dt1' }] });
		const { drTestsApi } = await import('./api');
		const result = await drTestsApi.list();
		expect(result).toEqual([{ id: 'dt1' }]);
	});

	it('list with params adds query string', async () => {
		const fetchFn = mockFetch({ tests: [] });
		const { drTestsApi } = await import('./api');
		await drTestsApi.list({ runbook_id: 'dr1', status: 'passed' });
		const url = fetchFn.mock.calls[0][0] as string;
		expect(url).toContain('runbook_id=dr1');
		expect(url).toContain('status=passed');
	});

	it('get calls correct endpoint', async () => {
		const fetchFn = mockFetch({ id: 'dt1' });
		const { drTestsApi } = await import('./api');
		await drTestsApi.get('dt1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/dr-tests/dt1', expect.any(Object));
	});

	it('run sends POST', async () => {
		const fetchFn = mockFetch({ id: 'dt1' });
		const { drTestsApi } = await import('./api');
		await drTestsApi.run({ runbook_id: 'dr1' } as Parameters<typeof drTestsApi.run>[0]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/dr-tests', expect.objectContaining({ method: 'POST' }));
	});

	it('cancel sends POST with notes', async () => {
		const fetchFn = mockFetch({ id: 'dt1' });
		const { drTestsApi } = await import('./api');
		await drTestsApi.cancel('dt1', 'cancelled by user');
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/dr-tests/dt1/cancel',
			expect.objectContaining({ method: 'POST', body: JSON.stringify({ notes: 'cancelled by user' }) }),
		);
	});
});

describe('tagsApi', () => {
	it('list returns tags', async () => {
		mockFetch({ tags: [{ id: 't1' }] });
		const { tagsApi } = await import('./api');
		const result = await tagsApi.list();
		expect(result).toEqual([{ id: 't1' }]);
	});

	it('get calls correct endpoint', async () => {
		const fetchFn = mockFetch({ id: 't1' });
		const { tagsApi } = await import('./api');
		await tagsApi.get('t1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/tags/t1', expect.any(Object));
	});

	it('create sends POST', async () => {
		const fetchFn = mockFetch({ id: 't1' });
		const { tagsApi } = await import('./api');
		await tagsApi.create({ name: 'prod' } as Parameters<typeof tagsApi.create>[0]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/tags', expect.objectContaining({ method: 'POST' }));
	});

	it('update sends PUT', async () => {
		const fetchFn = mockFetch({ id: 't1' });
		const { tagsApi } = await import('./api');
		await tagsApi.update('t1', { name: 'staging' } as Parameters<typeof tagsApi.update>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/tags/t1', expect.objectContaining({ method: 'PUT' }));
	});

	it('delete sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { tagsApi } = await import('./api');
		await tagsApi.delete('t1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/tags/t1', expect.objectContaining({ method: 'DELETE' }));
	});

	it('getBackupTags returns tags', async () => {
		mockFetch({ tags: [{ id: 't1' }] });
		const { tagsApi } = await import('./api');
		const result = await tagsApi.getBackupTags('b1');
		expect(result).toEqual([{ id: 't1' }]);
	});

	it('setBackupTags sends POST', async () => {
		const fetchFn = mockFetch({ tags: [{ id: 't1' }] });
		const { tagsApi } = await import('./api');
		await tagsApi.setBackupTags('b1', { tag_ids: ['t1'] } as Parameters<typeof tagsApi.setBackupTags>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/backups/b1/tags', expect.objectContaining({ method: 'POST' }));
	});
});

describe('searchApi', () => {
	it('search builds correct query params', async () => {
		const fetchFn = mockFetch({ results: [] });
		const { searchApi } = await import('./api');
		await searchApi.search({ q: 'test', types: ['agent', 'backup'], status: 'active', tag_ids: ['t1'], date_from: '2024-01-01', date_to: '2024-12-31', size_min: 100, size_max: 1000, limit: 10 });
		const url = fetchFn.mock.calls[0][0] as string;
		expect(url).toContain('q=test');
		expect(url).toContain('types=agent%2Cbackup');
		expect(url).toContain('status=active');
		expect(url).toContain('tag_ids=t1');
		expect(url).toContain('size_min=100');
		expect(url).toContain('size_max=1000');
		expect(url).toContain('limit=10');
	});
});

describe('metricsApi', () => {
	it('getDashboardStats calls correct endpoint', async () => {
		const fetchFn = mockFetch({ total_agents: 5 });
		const { metricsApi } = await import('./api');
		await metricsApi.getDashboardStats();
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/dashboard-metrics/stats', expect.any(Object));
	});

	it('getBackupSuccessRates calls correct endpoint', async () => {
		const fetchFn = mockFetch({ rate_7d: {}, rate_30d: {} });
		const { metricsApi } = await import('./api');
		await metricsApi.getBackupSuccessRates();
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/dashboard-metrics/success-rates', expect.any(Object));
	});

	it('getStorageGrowthTrend returns trend array', async () => {
		mockFetch({ trend: [{ date: '2024-01-01' }] });
		const { metricsApi } = await import('./api');
		const result = await metricsApi.getStorageGrowthTrend(7);
		expect(result).toEqual([{ date: '2024-01-01' }]);
	});

	it('getBackupDurationTrend returns trend array', async () => {
		mockFetch({ trend: [{ date: '2024-01-01' }] });
		const { metricsApi } = await import('./api');
		const result = await metricsApi.getBackupDurationTrend(14);
		expect(result).toEqual([{ date: '2024-01-01' }]);
	});

	it('getDailyBackupStats returns stats array', async () => {
		mockFetch({ stats: [{ date: '2024-01-01' }] });
		const { metricsApi } = await import('./api');
		const result = await metricsApi.getDailyBackupStats(7);
		expect(result).toEqual([{ date: '2024-01-01' }]);
	});
});

describe('reportsApi', () => {
	it('listSchedules returns schedules', async () => {
		mockFetch({ schedules: [{ id: 'rs1' }] });
		const { reportsApi } = await import('./api');
		const result = await reportsApi.listSchedules();
		expect(result).toEqual([{ id: 'rs1' }]);
	});

	it('getSchedule calls correct endpoint', async () => {
		const fetchFn = mockFetch({ id: 'rs1' });
		const { reportsApi } = await import('./api');
		await reportsApi.getSchedule('rs1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/reports/schedules/rs1', expect.any(Object));
	});

	it('createSchedule sends POST', async () => {
		const fetchFn = mockFetch({ id: 'rs1' });
		const { reportsApi } = await import('./api');
		await reportsApi.createSchedule({ name: 'report' } as Parameters<typeof reportsApi.createSchedule>[0]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/reports/schedules', expect.objectContaining({ method: 'POST' }));
	});

	it('updateSchedule sends PUT', async () => {
		const fetchFn = mockFetch({ id: 'rs1' });
		const { reportsApi } = await import('./api');
		await reportsApi.updateSchedule('rs1', { name: 'u' } as Parameters<typeof reportsApi.updateSchedule>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/reports/schedules/rs1', expect.objectContaining({ method: 'PUT' }));
	});

	it('deleteSchedule sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { reportsApi } = await import('./api');
		await reportsApi.deleteSchedule('rs1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/reports/schedules/rs1', expect.objectContaining({ method: 'DELETE' }));
	});

	it('sendReport sends POST', async () => {
		const fetchFn = mockFetch({ message: 'sent' });
		const { reportsApi } = await import('./api');
		await reportsApi.sendReport('rs1');
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/reports/schedules/rs1/send',
			expect.objectContaining({ method: 'POST', body: JSON.stringify({ preview: false }) }),
		);
	});

	it('sendReport with preview sends preview flag', async () => {
		const fetchFn = mockFetch({ html: '<p>report</p>' });
		const { reportsApi } = await import('./api');
		await reportsApi.sendReport('rs1', true);
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/reports/schedules/rs1/send',
			expect.objectContaining({ body: JSON.stringify({ preview: true }) }),
		);
	});

	it('previewReport sends POST', async () => {
		const fetchFn = mockFetch({ html: '<p>preview</p>' });
		const { reportsApi } = await import('./api');
		await reportsApi.previewReport('weekly', 'US/Eastern');
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/reports/preview',
			expect.objectContaining({ method: 'POST', body: JSON.stringify({ frequency: 'weekly', timezone: 'US/Eastern' }) }),
		);
	});

	it('listHistory returns history', async () => {
		mockFetch({ history: [{ id: 'h1' }] });
		const { reportsApi } = await import('./api');
		const result = await reportsApi.listHistory();
		expect(result).toEqual([{ id: 'h1' }]);
	});

	it('getHistory calls correct endpoint', async () => {
		const fetchFn = mockFetch({ id: 'h1' });
		const { reportsApi } = await import('./api');
		await reportsApi.getHistory('h1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/reports/history/h1', expect.any(Object));
	});
});

describe('onboardingApi', () => {
	it('getStatus calls correct endpoint', async () => {
		const fetchFn = mockFetch({ needs_onboarding: false });
		const { onboardingApi } = await import('./api');
		await onboardingApi.getStatus();
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/onboarding/status', expect.any(Object));
	});

	it('completeStep sends POST', async () => {
		const fetchFn = mockFetch({ needs_onboarding: false });
		const { onboardingApi } = await import('./api');
		await onboardingApi.completeStep('org' as Parameters<typeof onboardingApi.completeStep>[0]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/onboarding/step/org', expect.objectContaining({ method: 'POST' }));
	});

	it('skip sends POST', async () => {
		const fetchFn = mockFetch({ needs_onboarding: false });
		const { onboardingApi } = await import('./api');
		await onboardingApi.skip();
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/onboarding/skip', expect.objectContaining({ method: 'POST' }));
	});
});

describe('costsApi', () => {
	it('getSummary calls /costs/summary', async () => {
		const fetchFn = mockFetch({ total_monthly_cost: 100 });
		const { costsApi } = await import('./api');
		await costsApi.getSummary();
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/costs/summary', expect.any(Object));
	});

	it('listRepositoryCosts calls correct endpoint', async () => {
		const fetchFn = mockFetch({ repositories: [] });
		const { costsApi } = await import('./api');
		await costsApi.listRepositoryCosts();
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/costs/repositories', expect.any(Object));
	});

	it('getRepositoryCost calls correct endpoint', async () => {
		const fetchFn = mockFetch({ cost: {} });
		const { costsApi } = await import('./api');
		await costsApi.getRepositoryCost('r1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/costs/repositories/r1', expect.any(Object));
	});

	it('getForecast passes days param', async () => {
		const fetchFn = mockFetch({ forecast: [] });
		const { costsApi } = await import('./api');
		await costsApi.getForecast(60);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/costs/forecast?days=60', expect.any(Object));
	});

	it('getHistory passes days param', async () => {
		const fetchFn = mockFetch({ history: [] });
		const { costsApi } = await import('./api');
		await costsApi.getHistory(90);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/costs/history?days=90', expect.any(Object));
	});
});

describe('pricingApi', () => {
	it('list returns pricing', async () => {
		mockFetch({ pricing: [{ id: 'pr1' }] });
		const { pricingApi } = await import('./api');
		const result = await pricingApi.list();
		expect(result).toEqual([{ id: 'pr1' }]);
	});

	it('getDefaults calls correct endpoint', async () => {
		const fetchFn = mockFetch({ defaults: {} });
		const { pricingApi } = await import('./api');
		await pricingApi.getDefaults();
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/pricing/defaults', expect.any(Object));
	});

	it('create sends POST', async () => {
		const fetchFn = mockFetch({ id: 'pr1' });
		const { pricingApi } = await import('./api');
		await pricingApi.create({ name: 'tier' } as Parameters<typeof pricingApi.create>[0]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/pricing', expect.objectContaining({ method: 'POST' }));
	});

	it('update sends PUT', async () => {
		const fetchFn = mockFetch({ id: 'pr1' });
		const { pricingApi } = await import('./api');
		await pricingApi.update('pr1', { name: 'u' } as Parameters<typeof pricingApi.update>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/pricing/pr1', expect.objectContaining({ method: 'PUT' }));
	});

	it('delete sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { pricingApi } = await import('./api');
		await pricingApi.delete('pr1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/pricing/pr1', expect.objectContaining({ method: 'DELETE' }));
	});
});

describe('costAlertsApi', () => {
	it('list returns alerts', async () => {
		mockFetch({ alerts: [{ id: 'ca1' }] });
		const { costAlertsApi } = await import('./api');
		const result = await costAlertsApi.list();
		expect(result).toEqual([{ id: 'ca1' }]);
	});

	it('get calls correct endpoint', async () => {
		const fetchFn = mockFetch({ id: 'ca1' });
		const { costAlertsApi } = await import('./api');
		await costAlertsApi.get('ca1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/cost-alerts/ca1', expect.any(Object));
	});

	it('create sends POST', async () => {
		const fetchFn = mockFetch({ id: 'ca1' });
		const { costAlertsApi } = await import('./api');
		await costAlertsApi.create({ name: 'alert' } as Parameters<typeof costAlertsApi.create>[0]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/cost-alerts', expect.objectContaining({ method: 'POST' }));
	});

	it('update sends PUT', async () => {
		const fetchFn = mockFetch({ id: 'ca1' });
		const { costAlertsApi } = await import('./api');
		await costAlertsApi.update('ca1', { name: 'u' } as Parameters<typeof costAlertsApi.update>[1]);
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/cost-alerts/ca1', expect.objectContaining({ method: 'PUT' }));
	});

	it('delete sends DELETE', async () => {
		const fetchFn = mockFetch({ message: 'ok' });
		const { costAlertsApi } = await import('./api');
		await costAlertsApi.delete('ca1');
		expect(fetchFn).toHaveBeenCalledWith('/api/v1/cost-alerts/ca1', expect.objectContaining({ method: 'DELETE' }));
	});
});
