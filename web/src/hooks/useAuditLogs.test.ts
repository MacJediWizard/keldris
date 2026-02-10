import { renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { useAuditLogs, useAuditLog, useExportAuditLogsCsv, useExportAuditLogsJson } from './useAuditLogs';
import { createWrapper } from '../test/helpers';

vi.mock('../lib/api', () => ({
	auditLogsApi: {
		list: vi.fn(),
		get: vi.fn(),
		exportCsv: vi.fn(),
		exportJson: vi.fn(),
	},
}));

import { auditLogsApi } from '../lib/api';

describe('useAuditLogs', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches audit logs', async () => {
		vi.mocked(auditLogsApi.list).mockResolvedValue({ audit_logs: [], total: 0 });
		const { result } = renderHook(() => useAuditLogs(), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('fetches with filter', async () => {
		vi.mocked(auditLogsApi.list).mockResolvedValue({ audit_logs: [], total: 0 });
		const filter = { action: 'create' };
		const { result } = renderHook(() => useAuditLogs(filter), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(auditLogsApi.list).toHaveBeenCalledWith(filter);
	});
});

describe('useAuditLog', () => {
	beforeEach(() => vi.clearAllMocks());

	it('fetches a single audit log', async () => {
		vi.mocked(auditLogsApi.get).mockResolvedValue({ id: 'log-1' });
		const { result } = renderHook(() => useAuditLog('log-1'), { wrapper: createWrapper() });
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});

	it('does not fetch when id is empty', () => {
		renderHook(() => useAuditLog(''), { wrapper: createWrapper() });
		expect(auditLogsApi.get).not.toHaveBeenCalled();
	});
});

describe('useExportAuditLogsCsv', () => {
	beforeEach(() => vi.clearAllMocks());

	it('exports CSV', async () => {
		vi.mocked(auditLogsApi.exportCsv).mockResolvedValue(new Blob());
		const { result } = renderHook(() => useExportAuditLogsCsv(), { wrapper: createWrapper() });
		result.current.mutate(undefined);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useExportAuditLogsJson', () => {
	beforeEach(() => vi.clearAllMocks());

	it('exports JSON', async () => {
		vi.mocked(auditLogsApi.exportJson).mockResolvedValue(new Blob());
		const { result } = renderHook(() => useExportAuditLogsJson(), { wrapper: createWrapper() });
		result.current.mutate(undefined);
		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
