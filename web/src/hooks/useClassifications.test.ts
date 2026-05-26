import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useAutoClassifySchedule,
	useBackupsByClassification,
	useClassificationDataTypes,
	useClassificationLevels,
	useClassificationRule,
	useClassificationRules,
	useClassificationSummary,
	useComplianceReport,
	useCreateClassificationRule,
	useDefaultClassificationRules,
	useDeleteClassificationRule,
	useScheduleClassification,
	useScheduleClassifications,
	useSetScheduleClassification,
	useUpdateClassificationRule,
} from './useClassifications';

vi.mock('../lib/api', () => ({
	classificationsApi: {
		getLevels: vi.fn(),
		getDataTypes: vi.fn(),
		getDefaultRules: vi.fn(),
		listRules: vi.fn(),
		getRule: vi.fn(),
		createRule: vi.fn(),
		updateRule: vi.fn(),
		deleteRule: vi.fn(),
		listScheduleClassifications: vi.fn(),
		getScheduleClassification: vi.fn(),
		setScheduleClassification: vi.fn(),
		autoClassifySchedule: vi.fn(),
		listBackupsByClassification: vi.fn(),
		getSummary: vi.fn(),
		getComplianceReport: vi.fn(),
	},
}));

import { classificationsApi } from '../lib/api';

describe('useClassificationLevels', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches levels', async () => {
		vi.mocked(classificationsApi.getLevels).mockResolvedValue([]);

		const { result } = renderHook(() => useClassificationLevels(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(classificationsApi.getLevels).toHaveBeenCalledOnce();
	});
});

describe('useClassificationDataTypes', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches data types', async () => {
		vi.mocked(classificationsApi.getDataTypes).mockResolvedValue([]);

		const { result } = renderHook(() => useClassificationDataTypes(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(classificationsApi.getDataTypes).toHaveBeenCalledOnce();
	});
});

describe('useDefaultClassificationRules', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches default rules', async () => {
		vi.mocked(classificationsApi.getDefaultRules).mockResolvedValue([]);

		const { result } = renderHook(() => useDefaultClassificationRules(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(classificationsApi.getDefaultRules).toHaveBeenCalledOnce();
	});
});

describe('useClassificationRules', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches rules', async () => {
		vi.mocked(classificationsApi.listRules).mockResolvedValue([]);

		const { result } = renderHook(() => useClassificationRules(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(classificationsApi.listRules).toHaveBeenCalledOnce();
	});
});

describe('useClassificationRule', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches a rule', async () => {
		vi.mocked(classificationsApi.getRule).mockResolvedValue({ id: 'r-1' });

		const { result } = renderHook(() => useClassificationRule('r-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(classificationsApi.getRule).toHaveBeenCalledWith('r-1');
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useClassificationRule(''), { wrapper: createWrapper() });
		expect(classificationsApi.getRule).not.toHaveBeenCalled();
	});
});

describe('useCreateClassificationRule', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('creates a rule', async () => {
		vi.mocked(classificationsApi.createRule).mockResolvedValue({ id: 'new' });

		const { result } = renderHook(() => useCreateClassificationRule(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ path: '/data' } as never);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(classificationsApi.createRule).toHaveBeenCalled();
	});
});

describe('useUpdateClassificationRule', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('updates a rule', async () => {
		vi.mocked(classificationsApi.updateRule).mockResolvedValue({ id: 'r-1' });

		const { result } = renderHook(() => useUpdateClassificationRule(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ id: 'r-1', data: { path: '/x' } as never });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(classificationsApi.updateRule).toHaveBeenCalledWith('r-1', {
			path: '/x',
		});
	});
});

describe('useDeleteClassificationRule', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('deletes a rule', async () => {
		vi.mocked(classificationsApi.deleteRule).mockResolvedValue({
			message: 'Deleted',
		});

		const { result } = renderHook(() => useDeleteClassificationRule(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('r-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(classificationsApi.deleteRule).toHaveBeenCalledWith('r-1');
	});
});

describe('useScheduleClassifications', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches schedule classifications', async () => {
		vi.mocked(classificationsApi.listScheduleClassifications).mockResolvedValue(
			[],
		);

		const { result } = renderHook(() => useScheduleClassifications(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(classificationsApi.listScheduleClassifications).toHaveBeenCalledWith(
			undefined,
		);
	});
});

describe('useScheduleClassification', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches one schedule classification', async () => {
		vi.mocked(classificationsApi.getScheduleClassification).mockResolvedValue({
			id: 's-1',
		});

		const { result } = renderHook(() => useScheduleClassification('s-1'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(classificationsApi.getScheduleClassification).toHaveBeenCalledWith(
			's-1',
		);
	});
});

describe('useSetScheduleClassification', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('sets schedule classification', async () => {
		vi.mocked(classificationsApi.setScheduleClassification).mockResolvedValue({
			ok: true,
		});

		const { result } = renderHook(() => useSetScheduleClassification(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({
			scheduleId: 's-1',
			data: { level: 'high' } as never,
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(classificationsApi.setScheduleClassification).toHaveBeenCalledWith(
			's-1',
			{ level: 'high' },
		);
	});
});

describe('useAutoClassifySchedule', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('auto-classifies a schedule', async () => {
		vi.mocked(classificationsApi.autoClassifySchedule).mockResolvedValue({
			ok: true,
		});

		const { result } = renderHook(() => useAutoClassifySchedule(), {
			wrapper: createWrapper(),
		});

		result.current.mutate('s-1');

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(classificationsApi.autoClassifySchedule).toHaveBeenCalledWith('s-1');
	});
});

describe('useBackupsByClassification', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches backups by level', async () => {
		vi.mocked(classificationsApi.listBackupsByClassification).mockResolvedValue(
			[],
		);

		const { result } = renderHook(() => useBackupsByClassification('high'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(classificationsApi.listBackupsByClassification).toHaveBeenCalledWith(
			'high',
		);
	});

	it('does not fetch when level is empty', () => {
		renderHook(() => useBackupsByClassification(''), {
			wrapper: createWrapper(),
		});
		expect(
			classificationsApi.listBackupsByClassification,
		).not.toHaveBeenCalled();
	});
});

describe('useClassificationSummary', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches summary', async () => {
		vi.mocked(classificationsApi.getSummary).mockResolvedValue({});

		const { result } = renderHook(() => useClassificationSummary(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(classificationsApi.getSummary).toHaveBeenCalledOnce();
	});
});

describe('useComplianceReport', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches compliance report', async () => {
		vi.mocked(classificationsApi.getComplianceReport).mockResolvedValue({});

		const { result } = renderHook(() => useComplianceReport(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(classificationsApi.getComplianceReport).toHaveBeenCalledOnce();
	});
});
