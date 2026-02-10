import { renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
	useAlerts,
	useActiveAlerts,
	useAlertCount,
	useAlert,
	useAcknowledgeAlert,
	useResolveAlert,
	useAlertRules,
	useCreateAlertRule,
	useDeleteAlertRule,
} from './useAlerts';
import { createWrapper } from '../test/helpers';

vi.mock('../lib/api', () => ({
	alertsApi: {
		list: vi.fn(),
		listActive: vi.fn(),
		count: vi.fn(),
		get: vi.fn(),
		acknowledge: vi.fn(),
		resolve: vi.fn(),
	},
	alertRulesApi: {
		list: vi.fn(),
		get: vi.fn(),
		create: vi.fn(),
		update: vi.fn(),
		delete: vi.fn(),
	},
}));

import { alertsApi, alertRulesApi } from '../lib/api';

const mockAlerts = [
	{
		id: 'alert-1',
		type: 'agent_offline',
		severity: 'critical',
		status: 'active',
		message: 'Agent server-1 is offline',
	},
];

describe('useAlerts', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches all alerts', async () => {
		vi.mocked(alertsApi.list).mockResolvedValue(mockAlerts);

		const { result } = renderHook(() => useAlerts(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockAlerts);
	});
});

describe('useActiveAlerts', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches active alerts', async () => {
		vi.mocked(alertsApi.listActive).mockResolvedValue(mockAlerts);

		const { result } = renderHook(() => useActiveAlerts(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(alertsApi.listActive).toHaveBeenCalled();
	});
});

describe('useAlertCount', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches alert count', async () => {
		vi.mocked(alertsApi.count).mockResolvedValue(5);

		const { result } = renderHook(() => useAlertCount(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toBe(5);
	});
});

describe('useAlert', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches a single alert', async () => {
		vi.mocked(alertsApi.get).mockResolvedValue(mockAlerts[0]);

		const { result } = renderHook(() => useAlert('alert-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(alertsApi.get).toHaveBeenCalledWith('alert-1');
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useAlert(''), {
			wrapper: createWrapper(),
		});
		expect(alertsApi.get).not.toHaveBeenCalled();
	});
});

describe('useAcknowledgeAlert', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('acknowledges an alert', async () => {
		vi.mocked(alertsApi.acknowledge).mockResolvedValue({
			...mockAlerts[0],
			status: 'acknowledged',
		});

		const { result } = renderHook(() => useAcknowledgeAlert(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('alert-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(alertsApi.acknowledge).toHaveBeenCalledWith('alert-1');
	});
});

describe('useResolveAlert', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('resolves an alert', async () => {
		vi.mocked(alertsApi.resolve).mockResolvedValue({
			...mockAlerts[0],
			status: 'resolved',
		});

		const { result } = renderHook(() => useResolveAlert(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('alert-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(alertsApi.resolve).toHaveBeenCalledWith('alert-1');
	});
});

describe('useAlertRules', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches alert rules', async () => {
		const mockRules = [{ id: 'rule-1', name: 'Test Rule' }];
		vi.mocked(alertRulesApi.list).mockResolvedValue(mockRules);

		const { result } = renderHook(() => useAlertRules(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockRules);
	});
});

describe('useCreateAlertRule', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('creates an alert rule', async () => {
		vi.mocked(alertRulesApi.create).mockResolvedValue({
			id: 'rule-1',
			name: 'New Rule',
		});

		const { result } = renderHook(() => useCreateAlertRule(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			name: 'New Rule',
			type: 'agent_offline',
		} as Parameters<typeof alertRulesApi.create>[0]);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteAlertRule', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('deletes an alert rule', async () => {
		vi.mocked(alertRulesApi.delete).mockResolvedValue({ message: 'Deleted' });

		const { result } = renderHook(() => useDeleteAlertRule(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('rule-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(alertRulesApi.delete).toHaveBeenCalledWith('rule-1');
	});
});
