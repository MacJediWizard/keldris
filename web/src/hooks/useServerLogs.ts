import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { serverLogsApi } from '../lib/api';
import type { ServerLogFilter } from '../lib/types';

export function useServerLogs(filter?: ServerLogFilter) {
	return useQuery({
		queryKey: ['server-logs', filter],
		queryFn: () => serverLogsApi.list(filter),
		staleTime: 5 * 1000, // 5 seconds - more frequent refresh for logs
		refetchInterval: 10 * 1000, // Auto-refresh every 10 seconds
	});
}

export function useServerLogComponents() {
	return useQuery({
		queryKey: ['server-logs', 'components'],
		queryFn: () => serverLogsApi.getComponents(),
		staleTime: 60 * 1000, // 1 minute
	});
}

export function useExportServerLogsCsv() {
	return useMutation({
		mutationFn: (filter?: ServerLogFilter) => serverLogsApi.exportCsv(filter),
		onSuccess: (blob) => {
			const url = window.URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `server_logs_${new Date().toISOString().split('T')[0]}.csv`;
			document.body.appendChild(a);
			a.click();
			window.URL.revokeObjectURL(url);
			document.body.removeChild(a);
		},
	});
}

export function useExportServerLogsJson() {
	return useMutation({
		mutationFn: (filter?: ServerLogFilter) => serverLogsApi.exportJson(filter),
		onSuccess: (blob) => {
			const url = window.URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `server_logs_${new Date().toISOString().split('T')[0]}.json`;
			document.body.appendChild(a);
			a.click();
			window.URL.revokeObjectURL(url);
			document.body.removeChild(a);
		},
	});
}

export function useClearServerLogs() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: () => serverLogsApi.clear(),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['server-logs'] });
		},
	});
}
