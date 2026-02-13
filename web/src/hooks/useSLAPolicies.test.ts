import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useCreateSLAPolicy,
	useDeleteSLAPolicy,
	useSLAPolicies,
	useSLAPolicy,
	useSLAPolicyHistory,
	useSLAPolicyStatus,
} from './useSLAPolicies';

vi.mock('../lib/api', () => ({
	slaPoliciesApi: {
		list: vi.fn(),
		get: vi.fn(),
		create: vi.fn(),
		update: vi.fn(),
		delete: vi.fn(),
		getStatus: vi.fn(),
		getHistory: vi.fn(),
	},
}));

import { slaPoliciesApi } from '../lib/api';

const mockPolicies = [
	{
		id: 'policy-1',
		name: 'Gold SLA',
		target_rpo_hours: 24,
		target_rto_hours: 4,
		target_success_rate: 99.5,
		enabled: true,
	},
];

describe('useSLAPolicies', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches all SLA policies', async () => {
		vi.mocked(slaPoliciesApi.list).mockResolvedValue(mockPolicies);

		const { result } = renderHook(() => useSLAPolicies(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockPolicies);
	});
});

describe('useSLAPolicy', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches a single SLA policy', async () => {
		vi.mocked(slaPoliciesApi.get).mockResolvedValue(mockPolicies[0]);

		const { result } = renderHook(() => useSLAPolicy('policy-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(slaPoliciesApi.get).toHaveBeenCalledWith('policy-1');
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useSLAPolicy(''), {
			wrapper: createWrapper(),
		});
		expect(slaPoliciesApi.get).not.toHaveBeenCalled();
	});
});

describe('useCreateSLAPolicy', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('creates an SLA policy', async () => {
		vi.mocked(slaPoliciesApi.create).mockResolvedValue({
			id: 'policy-2',
			name: 'Silver SLA',
		});

		const { result } = renderHook(() => useCreateSLAPolicy(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			name: 'Silver SLA',
			target_rpo_hours: 48,
			target_rto_hours: 8,
			target_success_rate: 95,
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useDeleteSLAPolicy', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('deletes an SLA policy', async () => {
		vi.mocked(slaPoliciesApi.delete).mockResolvedValue({
			message: 'Deleted',
		});

		const { result } = renderHook(() => useDeleteSLAPolicy(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('policy-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(slaPoliciesApi.delete).toHaveBeenCalledWith('policy-1');
	});
});

describe('useSLAPolicyStatus', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches SLA policy status', async () => {
		const mockStatus = {
			policy_id: 'policy-1',
			current_rpo_hours: 2.5,
			current_rto_hours: 2.5,
			success_rate: 99.8,
			compliant: true,
			calculated_at: '2024-01-01T00:00:00Z',
		};
		vi.mocked(slaPoliciesApi.getStatus).mockResolvedValue(mockStatus);

		const { result } = renderHook(() => useSLAPolicyStatus('policy-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockStatus);
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useSLAPolicyStatus(''), {
			wrapper: createWrapper(),
		});
		expect(slaPoliciesApi.getStatus).not.toHaveBeenCalled();
	});
});

describe('useSLAPolicyHistory', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches SLA policy history', async () => {
		const mockHistory = [
			{
				id: 'snap-1',
				policy_id: 'policy-1',
				rpo_hours: 2.5,
				rto_hours: 2.5,
				success_rate: 99.8,
				compliant: true,
				calculated_at: '2024-01-01T00:00:00Z',
			},
		];
		vi.mocked(slaPoliciesApi.getHistory).mockResolvedValue(mockHistory);

		const { result } = renderHook(() => useSLAPolicyHistory('policy-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockHistory);
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useSLAPolicyHistory(''), {
			wrapper: createWrapper(),
		});
		expect(slaPoliciesApi.getHistory).not.toHaveBeenCalled();
	});
});
