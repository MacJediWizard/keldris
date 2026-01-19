import { useQuery } from '@tanstack/react-query';
import { backupsApi } from '../lib/api';

export interface BackupsFilter {
	agent_id?: string;
	schedule_id?: string;
	status?: string;
}

export function useBackups(filter?: BackupsFilter) {
	return useQuery({
		queryKey: ['backups', filter],
		queryFn: () => backupsApi.list(filter),
		staleTime: 30 * 1000, // 30 seconds
	});
}

export function useBackup(id: string) {
	return useQuery({
		queryKey: ['backups', id],
		queryFn: () => backupsApi.get(id),
		enabled: !!id,
	});
}
