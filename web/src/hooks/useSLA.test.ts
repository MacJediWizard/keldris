import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useAcknowledgeBreach,
	useActiveSLABreaches,
	useCreateSLA,
	useCreateSLAAssignment,
	useDeleteSLA,
	useDeleteSLAAssignment,
	useOrgSLACompliance,
	useResolveBreach,
	useSLA,
	useSLAAssignments,
	useSLABreach,
	useSLABreaches,
	useSLABreachesBySLA,
	useSLACompliance,
	useSLADashboard,
	useSLAReport,
	useSLAs,
	useUpdateSLA,
} from './useSLA';

vi.mock('../lib/api', () => ({
	slaApi: {
		list: vi.fn(),
		get: vi.fn(),
		create: vi.fn(),
		update: vi.fn(),
		delete: vi.fn(),
		listAssignments: vi.fn(),
		createAssignment: vi.fn(),
		deleteAssignment: vi.fn(),
		getCompliance: vi.fn(),
		listOrgCompliance: vi.fn(),
		listBreaches: vi.fn(),
		listActiveBreaches: vi.fn(),
		listBreachesBySLA: vi.fn(),
		getBreach: vi.fn(),
		acknowledgeBreach: vi.fn(),
		resolveBreach: vi.fn(),
		getDashboard: vi.fn(),
		getReport: vi.fn(),
	},
}));

import { slaApi } from '../lib/api';

describe('useSLAs', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('lists SLAs', async () => {
		vi.mocked(slaApi.list).mockResolvedValue([]);
		const { result } = renderHook(() => useSLAs(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(slaApi.list).toHaveBeenCalledOnce();
	});
});

describe('useSLA', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches an SLA', async () => {
		vi.mocked(slaApi.get).mockResolvedValue({ id: 's-1' });
		const { result } = renderHook(() => useSLA('s-1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(slaApi.get).toHaveBeenCalledWith('s-1');
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useSLA(''), { wrapper: createWrapper() });
		expect(slaApi.get).not.toHaveBeenCalled();
	});
});

describe('useCreateSLA', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('creates an SLA', async () => {
		vi.mocked(slaApi.create).mockResolvedValue({ id: 'new' });
		const { result } = renderHook(() => useCreateSLA(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ name: 's' } as never);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(slaApi.create).toHaveBeenCalled();
	});
});

describe('useUpdateSLA', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('updates an SLA', async () => {
		vi.mocked(slaApi.update).mockResolvedValue({ id: 's-1' });
		const { result } = renderHook(() => useUpdateSLA(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ id: 's-1', data: { name: 'x' } as never });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(slaApi.update).toHaveBeenCalledWith('s-1', { name: 'x' });
	});
});

describe('useDeleteSLA', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('deletes an SLA', async () => {
		vi.mocked(slaApi.delete).mockResolvedValue({ message: 'Deleted' });
		const { result } = renderHook(() => useDeleteSLA(), {
			wrapper: createWrapper(),
		});
		result.current.mutate('s-1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(slaApi.delete).toHaveBeenCalledWith('s-1');
	});
});

describe('useSLAAssignments', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('lists assignments', async () => {
		vi.mocked(slaApi.listAssignments).mockResolvedValue([]);
		const { result } = renderHook(() => useSLAAssignments('s-1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(slaApi.listAssignments).toHaveBeenCalledWith('s-1');
	});
});

describe('useCreateSLAAssignment', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('creates an assignment', async () => {
		vi.mocked(slaApi.createAssignment).mockResolvedValue({ id: 'a-1' });
		const { result } = renderHook(() => useCreateSLAAssignment(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ slaId: 's-1', data: { id: 'x' } as never });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(slaApi.createAssignment).toHaveBeenCalledWith('s-1', { id: 'x' });
	});
});

describe('useDeleteSLAAssignment', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('deletes an assignment', async () => {
		vi.mocked(slaApi.deleteAssignment).mockResolvedValue({
			message: 'Deleted',
		});
		const { result } = renderHook(() => useDeleteSLAAssignment(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ slaId: 's-1', assignmentId: 'a-1' });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(slaApi.deleteAssignment).toHaveBeenCalledWith('s-1', 'a-1');
	});
});

describe('useSLACompliance', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches compliance', async () => {
		vi.mocked(slaApi.getCompliance).mockResolvedValue({});
		const { result } = renderHook(() => useSLACompliance('s-1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(slaApi.getCompliance).toHaveBeenCalledWith('s-1');
	});
});

describe('useOrgSLACompliance', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('lists org compliance', async () => {
		vi.mocked(slaApi.listOrgCompliance).mockResolvedValue([]);
		const { result } = renderHook(() => useOrgSLACompliance(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(slaApi.listOrgCompliance).toHaveBeenCalledOnce();
	});
});

describe('useSLABreaches', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('lists breaches', async () => {
		vi.mocked(slaApi.listBreaches).mockResolvedValue([]);
		const { result } = renderHook(() => useSLABreaches(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(slaApi.listBreaches).toHaveBeenCalledOnce();
	});
});

describe('useActiveSLABreaches', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('lists active breaches', async () => {
		vi.mocked(slaApi.listActiveBreaches).mockResolvedValue([]);
		const { result } = renderHook(() => useActiveSLABreaches(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(slaApi.listActiveBreaches).toHaveBeenCalledOnce();
	});
});

describe('useSLABreachesBySLA', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('lists breaches by SLA', async () => {
		vi.mocked(slaApi.listBreachesBySLA).mockResolvedValue([]);
		const { result } = renderHook(() => useSLABreachesBySLA('s-1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(slaApi.listBreachesBySLA).toHaveBeenCalledWith('s-1');
	});
});

describe('useSLABreach', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches a breach', async () => {
		vi.mocked(slaApi.getBreach).mockResolvedValue({ id: 'b-1' });
		const { result } = renderHook(() => useSLABreach('b-1'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(slaApi.getBreach).toHaveBeenCalledWith('b-1');
	});
});

describe('useAcknowledgeBreach', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('acknowledges a breach', async () => {
		vi.mocked(slaApi.acknowledgeBreach).mockResolvedValue({ ok: true });
		const { result } = renderHook(() => useAcknowledgeBreach(), {
			wrapper: createWrapper(),
		});
		result.current.mutate({ id: 'b-1' });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(slaApi.acknowledgeBreach).toHaveBeenCalledWith('b-1', undefined);
	});
});

describe('useResolveBreach', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('resolves a breach', async () => {
		vi.mocked(slaApi.resolveBreach).mockResolvedValue({ ok: true });
		const { result } = renderHook(() => useResolveBreach(), {
			wrapper: createWrapper(),
		});
		result.current.mutate('b-1');
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(slaApi.resolveBreach).toHaveBeenCalledWith('b-1');
	});
});

describe('useSLADashboard', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches dashboard', async () => {
		vi.mocked(slaApi.getDashboard).mockResolvedValue({});
		const { result } = renderHook(() => useSLADashboard(), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(slaApi.getDashboard).toHaveBeenCalledOnce();
	});
});

describe('useSLAReport', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches report', async () => {
		vi.mocked(slaApi.getReport).mockResolvedValue({});
		const { result } = renderHook(() => useSLAReport('2025-01'), {
			wrapper: createWrapper(),
		});
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(slaApi.getReport).toHaveBeenCalledWith('2025-01');
	});
});
