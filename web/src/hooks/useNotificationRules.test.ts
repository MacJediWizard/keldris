import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCreateNotificationRule,
	useDeleteNotificationRule,
	useNotificationRule,
	useNotificationRuleEvents,
	useNotificationRuleExecutions,
	useNotificationRules,
	useTestNotificationRule,
	useUpdateNotificationRule,
} from './useNotificationRules';

function mockFetch(data: unknown, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

describe('useNotificationRules', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches notification rules', async () => {
		const fetchFn = mockFetch({ rules: [{ id: 'r1', name: 'rule-1' }] });

		const { result } = renderHook(() => useNotificationRules(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual([{ id: 'r1', name: 'rule-1' }]);
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/notification-rules',
			expect.objectContaining({ credentials: 'include' }),
		);
	});
});

describe('useNotificationRule', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches a single rule', async () => {
		mockFetch({ id: 'r1' });

		const { result } = renderHook(() => useNotificationRule('r1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual({ id: 'r1' });
	});

	it('does not fetch when id is empty', () => {
		const fetchFn = mockFetch({});

		renderHook(() => useNotificationRule(''), { wrapper: createWrapper() });

		expect(fetchFn).not.toHaveBeenCalled();
	});
});

describe('useCreateNotificationRule', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('creates a rule', async () => {
		const fetchFn = mockFetch({ id: 'r2' });

		const { result } = renderHook(() => useCreateNotificationRule(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			name: 'new',
		} as Parameters<typeof result.current.mutate>[0]);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/notification-rules',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useUpdateNotificationRule', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('updates a rule', async () => {
		const fetchFn = mockFetch({ id: 'r1' });

		const { result } = renderHook(() => useUpdateNotificationRule(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			id: 'r1',
			data: { name: 'renamed' } as Parameters<
				typeof result.current.mutate
			>[0]['data'],
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/notification-rules/r1',
			expect.objectContaining({ method: 'PUT' }),
		);
	});
});

describe('useDeleteNotificationRule', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('deletes a rule', async () => {
		const fetchFn = mockFetch({ message: 'deleted' });

		const { result } = renderHook(() => useDeleteNotificationRule(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('r1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/notification-rules/r1',
			expect.objectContaining({ method: 'DELETE' }),
		);
	});
});

describe('useTestNotificationRule', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('tests a rule', async () => {
		const fetchFn = mockFetch({ success: true });

		const { result } = renderHook(() => useTestNotificationRule(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 'r1' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/notification-rules/r1/test',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useNotificationRuleEvents', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches events for a rule', async () => {
		const fetchFn = mockFetch({ events: [] });

		const { result } = renderHook(() => useNotificationRuleEvents('r1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/notification-rules/r1/events',
			expect.objectContaining({ credentials: 'include' }),
		);
	});

	it('does not fetch when ruleId is empty', () => {
		const fetchFn = mockFetch({});

		renderHook(() => useNotificationRuleEvents(''), {
			wrapper: createWrapper(),
		});

		expect(fetchFn).not.toHaveBeenCalled();
	});
});

describe('useNotificationRuleExecutions', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches executions for a rule', async () => {
		const fetchFn = mockFetch({ executions: [] });

		const { result } = renderHook(() => useNotificationRuleExecutions('r1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/notification-rules/r1/executions',
			expect.objectContaining({ credentials: 'include' }),
		);
	});
});
