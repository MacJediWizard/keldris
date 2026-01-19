import { useMutation, useQuery } from '@tanstack/react-query';
import { auditLogsApi } from '../lib/api';
import type { AuditLogFilter } from '../lib/types';

export function useAuditLogs(filter?: AuditLogFilter) {
	return useQuery({
		queryKey: ['audit-logs', filter],
		queryFn: () => auditLogsApi.list(filter),
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useAuditLog(id: string) {
	return useQuery({
		queryKey: ['audit-logs', id],
		queryFn: () => auditLogsApi.get(id),
		enabled: !!id,
	});
}

export function useExportAuditLogsCsv() {
	return useMutation({
		mutationFn: (filter?: AuditLogFilter) => auditLogsApi.exportCsv(filter),
		onSuccess: (blob) => {
			const url = window.URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `audit_logs_${new Date().toISOString().split('T')[0]}.csv`;
			document.body.appendChild(a);
			a.click();
			window.URL.revokeObjectURL(url);
			document.body.removeChild(a);
		},
	});
}

export function useExportAuditLogsJson() {
	return useMutation({
		mutationFn: (filter?: AuditLogFilter) => auditLogsApi.exportJson(filter),
		onSuccess: (blob) => {
			const url = window.URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `audit_logs_${new Date().toISOString().split('T')[0]}.json`;
			document.body.appendChild(a);
			a.click();
			window.URL.revokeObjectURL(url);
			document.body.removeChild(a);
		},
	});
}
