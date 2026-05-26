import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCreateWebhookEndpoint,
	useDeleteWebhookEndpoint,
	useRetryWebhookDelivery,
	useTestWebhookEndpoint,
	useUpdateWebhookEndpoint,
	useWebhookDeliveries,
	useWebhookDelivery,
	useWebhookEndpoint,
	useWebhookEndpointDeliveries,
	useWebhookEndpoints,
	useWebhookEventTypes,
} from './useWebhooks';

function mockFetch(data: unknown, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

describe('useWebhookEventTypes', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches webhook event types', async () => {
		const fetchFn = mockFetch({ event_types: [] });

		const { result } = renderHook(() => useWebhookEventTypes(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/webhooks/event-types',
			expect.any(Object),
		);
	});
});

describe('useWebhookEndpoints', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches webhook endpoints', async () => {
		const fetchFn = mockFetch({ endpoints: [{ id: 'wh-1' }] });

		const { result } = renderHook(() => useWebhookEndpoints(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual([{ id: 'wh-1' }]);
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/webhooks/endpoints',
			expect.any(Object),
		);
	});
});

describe('useWebhookEndpoint', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches a single webhook endpoint', async () => {
		mockFetch({ id: 'wh-1' });

		const { result } = renderHook(() => useWebhookEndpoint('wh-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual({ id: 'wh-1' });
	});

	it('does not fetch when id is empty', () => {
		const fetchFn = mockFetch({});

		renderHook(() => useWebhookEndpoint(''), { wrapper: createWrapper() });

		expect(fetchFn).not.toHaveBeenCalled();
	});
});

describe('useCreateWebhookEndpoint', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('creates a webhook endpoint', async () => {
		const fetchFn = mockFetch({ id: 'wh-1' });

		const { result } = renderHook(() => useCreateWebhookEndpoint(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			name: 'wh',
			url: 'https://x.example',
			event_types: ['backup.completed'],
			enabled: true,
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/webhooks/endpoints',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useUpdateWebhookEndpoint', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('updates a webhook endpoint', async () => {
		const fetchFn = mockFetch({ id: 'wh-1' });

		const { result } = renderHook(() => useUpdateWebhookEndpoint(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 'wh-1', data: { enabled: false } });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/webhooks/endpoints/wh-1',
			expect.objectContaining({ method: 'PUT' }),
		);
	});
});

describe('useDeleteWebhookEndpoint', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('deletes a webhook endpoint', async () => {
		const fetchFn = mockFetch({ message: 'ok' });

		const { result } = renderHook(() => useDeleteWebhookEndpoint(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('wh-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/webhooks/endpoints/wh-1',
			expect.objectContaining({ method: 'DELETE' }),
		);
	});
});

describe('useTestWebhookEndpoint', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('tests a webhook endpoint', async () => {
		const fetchFn = mockFetch({ success: true });

		const { result } = renderHook(() => useTestWebhookEndpoint(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 'wh-1' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/webhooks/endpoints/wh-1/test',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useWebhookDeliveries', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches webhook deliveries with defaults', async () => {
		const fetchFn = mockFetch({ deliveries: [] });

		const { result } = renderHook(() => useWebhookDeliveries(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/webhooks/deliveries?limit=50&offset=0',
			expect.any(Object),
		);
	});
});

describe('useWebhookEndpointDeliveries', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches deliveries for an endpoint', async () => {
		const fetchFn = mockFetch({ deliveries: [] });

		const { result } = renderHook(() => useWebhookEndpointDeliveries('wh-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/webhooks/endpoints/wh-1/deliveries?limit=50&offset=0',
			expect.any(Object),
		);
	});

	it('does not fetch when endpointId is empty', () => {
		const fetchFn = mockFetch({});

		renderHook(() => useWebhookEndpointDeliveries(''), {
			wrapper: createWrapper(),
		});

		expect(fetchFn).not.toHaveBeenCalled();
	});
});

describe('useWebhookDelivery', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches a single delivery', async () => {
		mockFetch({ id: 'd-1' });

		const { result } = renderHook(() => useWebhookDelivery('d-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual({ id: 'd-1' });
	});
});

describe('useRetryWebhookDelivery', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('retries a delivery', async () => {
		const fetchFn = mockFetch({ message: 'ok' });

		const { result } = renderHook(() => useRetryWebhookDelivery(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('d-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/webhooks/deliveries/d-1/retry',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});
