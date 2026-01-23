import { useQuery } from '@tanstack/react-query';
import { backupsApi } from '../lib/api';

export interface BackupCalendarParams {
	month: string; // YYYY-MM format
}

export function useBackupCalendar(params: BackupCalendarParams) {
	return useQuery({
		queryKey: ['backups', 'calendar', params.month],
		queryFn: () => backupsApi.getCalendar(params.month),
		staleTime: 60 * 1000, // 1 minute
	});
}
