import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
	formatAuditAction,
	formatBytes,
	formatChartDate,
	formatCurrency,
	formatCurrencyCompact,
	formatDate,
	formatDateTime,
	formatDedupRatio,
	formatDuration,
	formatDurationMs,
	formatPercent,
	formatRelativeTime,
	formatResourceType,
	formatUptime,
	getAgentStatusColor,
	getAlertSeverityColor,
	getAlertStatusColor,
	getAlertTypeLabel,
	getAuditActionColor,
	getAuditResultColor,
	getBackupStatusColor,
	getCostColor,
	getDedupRatioColor,
	getHealthStatusColor,
	getHealthStatusLabel,
	getRepositoryTypeBadge,
	getSpaceSavedColor,
	getSuccessRateBadge,
	getSuccessRateColor,
	truncateSnapshotId,
} from './utils';

describe('formatRelativeTime', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2024-06-15T12:00:00Z'));
	});
	afterEach(() => {
		vi.useRealTimers();
	});

	it('returns "Never" for undefined', () => {
		expect(formatRelativeTime(undefined)).toBe('Never');
	});

	it('returns "Just now" for recent past times', () => {
		expect(formatRelativeTime('2024-06-15T11:59:30Z')).toBe('Just now');
	});

	it('returns minutes ago for past times within an hour', () => {
		expect(formatRelativeTime('2024-06-15T11:30:00Z')).toBe('30m ago');
	});

	it('returns hours ago for past times within a day', () => {
		expect(formatRelativeTime('2024-06-15T06:00:00Z')).toBe('6h ago');
	});

	it('returns days ago for past times within a week', () => {
		expect(formatRelativeTime('2024-06-12T12:00:00Z')).toBe('3d ago');
	});

	it('returns formatted date for older times', () => {
		const result = formatRelativeTime('2024-01-15T12:00:00Z');
		expect(result).toContain('Jan');
		expect(result).toContain('15');
	});

	it('returns "In a moment" for near future times', () => {
		expect(formatRelativeTime('2024-06-15T12:00:20Z')).toBe('In a moment');
	});

	it('returns "In Xm" for future times within an hour', () => {
		expect(formatRelativeTime('2024-06-15T12:30:00Z')).toBe('In 30m');
	});

	it('returns "In Xh" for future times within a day', () => {
		expect(formatRelativeTime('2024-06-15T18:00:00Z')).toBe('In 6h');
	});

	it('returns "In Xd" for future times within a week', () => {
		expect(formatRelativeTime('2024-06-18T12:00:00Z')).toBe('In 3d');
	});

	it('includes year for dates in a different year', () => {
		const result = formatRelativeTime('2023-06-15T12:00:00Z');
		expect(result).toContain('2023');
	});
});

describe('formatDate', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2024-06-15T12:00:00Z'));
	});
	afterEach(() => {
		vi.useRealTimers();
	});

	it('returns "Never" for undefined', () => {
		expect(formatDate(undefined)).toBe('Never');
	});

	it('returns "Just now" for very recent times', () => {
		expect(formatDate('2024-06-15T11:59:30Z')).toBe('Just now');
	});

	it('returns minutes ago', () => {
		expect(formatDate('2024-06-15T11:30:00Z')).toBe('30m ago');
	});

	it('returns hours ago', () => {
		expect(formatDate('2024-06-15T06:00:00Z')).toBe('6h ago');
	});

	it('returns days ago', () => {
		expect(formatDate('2024-06-12T12:00:00Z')).toBe('3d ago');
	});

	it('returns formatted date for older times', () => {
		const result = formatDate('2024-01-15T12:00:00Z');
		expect(result).toContain('Jan');
	});
});

describe('formatDateTime', () => {
	it('returns "N/A" for undefined', () => {
		expect(formatDateTime(undefined)).toBe('N/A');
	});

	it('formats a valid date string', () => {
		const result = formatDateTime('2024-06-15T14:30:00Z');
		expect(result).toBeTruthy();
		expect(result).not.toBe('N/A');
	});
});

describe('formatBytes', () => {
	it('returns "N/A" for undefined', () => {
		expect(formatBytes(undefined)).toBe('N/A');
	});

	it('returns "N/A" for null', () => {
		expect(formatBytes(null as unknown as number)).toBe('N/A');
	});

	it('returns "0 B" for zero', () => {
		expect(formatBytes(0)).toBe('0 B');
	});

	it('formats bytes', () => {
		expect(formatBytes(500)).toBe('500 B');
	});

	it('formats kilobytes', () => {
		expect(formatBytes(1024)).toBe('1.0 KB');
	});

	it('formats megabytes', () => {
		expect(formatBytes(1024 * 1024)).toBe('1.0 MB');
	});

	it('formats gigabytes', () => {
		expect(formatBytes(1024 * 1024 * 1024)).toBe('1.0 GB');
	});

	it('formats terabytes', () => {
		expect(formatBytes(1024 * 1024 * 1024 * 1024)).toBe('1.0 TB');
	});

	it('formats fractional values', () => {
		expect(formatBytes(1536)).toBe('1.5 KB');
	});
});

describe('formatDuration', () => {
	it('returns "In progress" when endDate is undefined', () => {
		expect(formatDuration('2024-06-15T12:00:00Z', undefined)).toBe(
			'In progress',
		);
	});

	it('formats seconds', () => {
		expect(formatDuration('2024-06-15T12:00:00Z', '2024-06-15T12:00:30Z')).toBe(
			'30s',
		);
	});

	it('formats minutes and seconds', () => {
		expect(formatDuration('2024-06-15T12:00:00Z', '2024-06-15T12:05:30Z')).toBe(
			'5m 30s',
		);
	});

	it('formats hours and minutes', () => {
		expect(formatDuration('2024-06-15T12:00:00Z', '2024-06-15T14:30:00Z')).toBe(
			'2h 30m',
		);
	});
});

describe('truncateSnapshotId', () => {
	it('returns "N/A" for undefined', () => {
		expect(truncateSnapshotId(undefined)).toBe('N/A');
	});

	it('truncates to 8 characters', () => {
		expect(truncateSnapshotId('abcdef1234567890')).toBe('abcdef12');
	});

	it('handles short strings', () => {
		expect(truncateSnapshotId('abc')).toBe('abc');
	});
});

describe('getAgentStatusColor', () => {
	it('returns green for active', () => {
		const result = getAgentStatusColor('active');
		expect(result.bg).toBe('bg-green-100');
		expect(result.text).toBe('text-green-800');
	});

	it('returns red for offline', () => {
		const result = getAgentStatusColor('offline');
		expect(result.bg).toBe('bg-red-100');
	});

	it('returns yellow for pending', () => {
		const result = getAgentStatusColor('pending');
		expect(result.bg).toBe('bg-yellow-100');
	});

	it('returns gray for disabled', () => {
		const result = getAgentStatusColor('disabled');
		expect(result.bg).toBe('bg-gray-100');
	});

	it('returns gray for unknown status', () => {
		const result = getAgentStatusColor('unknown');
		expect(result.bg).toBe('bg-gray-100');
	});
});

describe('getBackupStatusColor', () => {
	it('returns green for completed', () => {
		expect(getBackupStatusColor('completed').bg).toBe('bg-green-100');
	});

	it('returns blue for running', () => {
		expect(getBackupStatusColor('running').bg).toBe('bg-blue-100');
	});

	it('returns red for failed', () => {
		expect(getBackupStatusColor('failed').bg).toBe('bg-red-100');
	});

	it('returns gray for canceled', () => {
		expect(getBackupStatusColor('canceled').bg).toBe('bg-gray-100');
	});

	it('returns gray for unknown', () => {
		expect(getBackupStatusColor('unknown').bg).toBe('bg-gray-100');
	});
});

describe('getRepositoryTypeBadge', () => {
	it('returns correct badge for local', () => {
		expect(getRepositoryTypeBadge('local').label).toBe('Local');
	});

	it('returns correct badge for s3', () => {
		expect(getRepositoryTypeBadge('s3').label).toBe('S3');
	});

	it('returns correct badge for b2', () => {
		expect(getRepositoryTypeBadge('b2').label).toBe('B2');
	});

	it('returns correct badge for sftp', () => {
		expect(getRepositoryTypeBadge('sftp').label).toBe('SFTP');
	});

	it('returns correct badge for rest', () => {
		expect(getRepositoryTypeBadge('rest').label).toBe('REST');
	});

	it('returns correct badge for dropbox', () => {
		expect(getRepositoryTypeBadge('dropbox').label).toBe('Dropbox');
	});

	it('returns type as label for unknown types', () => {
		expect(getRepositoryTypeBadge('gcs').label).toBe('gcs');
	});
});

describe('getAlertSeverityColor', () => {
	it('returns red for critical', () => {
		expect(getAlertSeverityColor('critical').bg).toBe('bg-red-50');
	});

	it('returns yellow for warning', () => {
		expect(getAlertSeverityColor('warning').bg).toBe('bg-yellow-50');
	});

	it('returns blue for info', () => {
		expect(getAlertSeverityColor('info').bg).toBe('bg-blue-50');
	});

	it('returns gray for unknown', () => {
		expect(getAlertSeverityColor('unknown').bg).toBe('bg-gray-50');
	});
});

describe('getAlertStatusColor', () => {
	it('returns red for active', () => {
		expect(getAlertStatusColor('active').bg).toBe('bg-red-100');
	});

	it('returns yellow for acknowledged', () => {
		expect(getAlertStatusColor('acknowledged').bg).toBe('bg-yellow-100');
	});

	it('returns green for resolved', () => {
		expect(getAlertStatusColor('resolved').bg).toBe('bg-green-100');
	});

	it('returns gray for unknown', () => {
		expect(getAlertStatusColor('unknown').bg).toBe('bg-gray-100');
	});
});

describe('getAlertTypeLabel', () => {
	it('returns "Agent Offline" for agent_offline', () => {
		expect(getAlertTypeLabel('agent_offline')).toBe('Agent Offline');
	});

	it('returns "Backup SLA" for backup_sla', () => {
		expect(getAlertTypeLabel('backup_sla')).toBe('Backup SLA');
	});

	it('returns "Storage Usage" for storage_usage', () => {
		expect(getAlertTypeLabel('storage_usage')).toBe('Storage Usage');
	});

	it('returns "Agent Health Warning" for agent_health_warning', () => {
		expect(getAlertTypeLabel('agent_health_warning')).toBe(
			'Agent Health Warning',
		);
	});

	it('returns "Agent Health Critical" for agent_health_critical', () => {
		expect(getAlertTypeLabel('agent_health_critical')).toBe(
			'Agent Health Critical',
		);
	});

	it('returns the type itself for unknown types', () => {
		expect(getAlertTypeLabel('custom_type')).toBe('custom_type');
	});
});

describe('getAuditActionColor', () => {
	it('returns green for create', () => {
		expect(getAuditActionColor('create').bg).toBe('bg-green-100');
	});

	it('returns blue for update', () => {
		expect(getAuditActionColor('update').bg).toBe('bg-blue-100');
	});

	it('returns red for delete', () => {
		expect(getAuditActionColor('delete').bg).toBe('bg-red-100');
	});

	it('returns gray for read', () => {
		expect(getAuditActionColor('read').bg).toBe('bg-gray-100');
	});

	it('returns indigo for login', () => {
		expect(getAuditActionColor('login').bg).toBe('bg-indigo-100');
	});

	it('returns purple for logout', () => {
		expect(getAuditActionColor('logout').bg).toBe('bg-purple-100');
	});

	it('returns cyan for backup', () => {
		expect(getAuditActionColor('backup').bg).toBe('bg-cyan-100');
	});

	it('returns orange for restore', () => {
		expect(getAuditActionColor('restore').bg).toBe('bg-orange-100');
	});

	it('returns gray for unknown', () => {
		expect(getAuditActionColor('unknown').bg).toBe('bg-gray-100');
	});
});

describe('getAuditResultColor', () => {
	it('returns green for success', () => {
		expect(getAuditResultColor('success').bg).toBe('bg-green-100');
	});

	it('returns red for failure', () => {
		expect(getAuditResultColor('failure').bg).toBe('bg-red-100');
	});

	it('returns yellow for denied', () => {
		expect(getAuditResultColor('denied').bg).toBe('bg-yellow-100');
	});

	it('returns gray for unknown', () => {
		expect(getAuditResultColor('unknown').bg).toBe('bg-gray-100');
	});
});

describe('formatAuditAction', () => {
	it('capitalizes first letter', () => {
		expect(formatAuditAction('create')).toBe('Create');
		expect(formatAuditAction('delete')).toBe('Delete');
	});
});

describe('formatResourceType', () => {
	it('splits and capitalizes words', () => {
		expect(formatResourceType('backup_schedule')).toBe('Backup Schedule');
		expect(formatResourceType('agent')).toBe('Agent');
		expect(formatResourceType('notification_channel')).toBe(
			'Notification Channel',
		);
	});
});

describe('formatDedupRatio', () => {
	it('returns "N/A" for undefined', () => {
		expect(formatDedupRatio(undefined)).toBe('N/A');
	});

	it('returns "N/A" for zero', () => {
		expect(formatDedupRatio(0)).toBe('N/A');
	});

	it('formats ratio with one decimal', () => {
		expect(formatDedupRatio(2.5)).toBe('2.5x');
	});
});

describe('formatPercent', () => {
	it('returns "N/A" for undefined', () => {
		expect(formatPercent(undefined)).toBe('N/A');
	});

	it('formats percentage with one decimal', () => {
		expect(formatPercent(45.25)).toBe('45.3%');
	});
});

describe('getDedupRatioColor', () => {
	it('returns green for high ratio', () => {
		expect(getDedupRatioColor(3)).toBe('text-green-600');
	});

	it('returns blue for medium ratio', () => {
		expect(getDedupRatioColor(2)).toBe('text-blue-600');
	});

	it('returns yellow for low ratio', () => {
		expect(getDedupRatioColor(1.5)).toBe('text-yellow-600');
	});

	it('returns gray for very low ratio', () => {
		expect(getDedupRatioColor(1)).toBe('text-gray-600');
	});
});

describe('getSpaceSavedColor', () => {
	it('returns green for high percentage', () => {
		expect(getSpaceSavedColor(70)).toBe('text-green-600');
	});

	it('returns blue for medium percentage', () => {
		expect(getSpaceSavedColor(50)).toBe('text-blue-600');
	});

	it('returns yellow for low percentage', () => {
		expect(getSpaceSavedColor(30)).toBe('text-yellow-600');
	});

	it('returns gray for very low percentage', () => {
		expect(getSpaceSavedColor(10)).toBe('text-gray-600');
	});
});

describe('formatChartDate', () => {
	it('formats to month and day', () => {
		const result = formatChartDate('2024-06-15T12:00:00Z');
		expect(result).toContain('Jun');
		expect(result).toContain('15');
	});
});

describe('formatDurationMs', () => {
	it('returns "N/A" for undefined', () => {
		expect(formatDurationMs(undefined)).toBe('N/A');
	});

	it('formats milliseconds', () => {
		expect(formatDurationMs(500)).toBe('500ms');
	});

	it('formats seconds', () => {
		expect(formatDurationMs(5000)).toBe('5s');
	});

	it('formats minutes and seconds', () => {
		expect(formatDurationMs(65000)).toBe('1m 5s');
	});

	it('formats hours and minutes', () => {
		expect(formatDurationMs(3660000)).toBe('1h 1m');
	});
});

describe('getSuccessRateColor', () => {
	it('returns green for high rate', () => {
		expect(getSuccessRateColor(95)).toBe('text-green-600');
	});

	it('returns yellow for medium rate', () => {
		expect(getSuccessRateColor(80)).toBe('text-yellow-600');
	});

	it('returns orange for low rate', () => {
		expect(getSuccessRateColor(50)).toBe('text-orange-600');
	});

	it('returns red for very low rate', () => {
		expect(getSuccessRateColor(30)).toBe('text-red-600');
	});
});

describe('getSuccessRateBadge', () => {
	it('returns green for high rate', () => {
		expect(getSuccessRateBadge(95).bg).toBe('bg-green-100');
	});

	it('returns yellow for medium rate', () => {
		expect(getSuccessRateBadge(80).bg).toBe('bg-yellow-100');
	});

	it('returns orange for low rate', () => {
		expect(getSuccessRateBadge(50).bg).toBe('bg-orange-100');
	});

	it('returns red for very low rate', () => {
		expect(getSuccessRateBadge(30).bg).toBe('bg-red-100');
	});
});

describe('getHealthStatusColor', () => {
	it('returns green for healthy', () => {
		expect(getHealthStatusColor('healthy').bg).toBe('bg-green-100');
	});

	it('returns yellow for warning', () => {
		expect(getHealthStatusColor('warning').bg).toBe('bg-yellow-100');
	});

	it('returns red for critical', () => {
		expect(getHealthStatusColor('critical').bg).toBe('bg-red-100');
	});

	it('returns gray for unknown', () => {
		expect(getHealthStatusColor('unknown').bg).toBe('bg-gray-100');
	});
});

describe('getHealthStatusLabel', () => {
	it('returns "Healthy" for healthy', () => {
		expect(getHealthStatusLabel('healthy')).toBe('Healthy');
	});

	it('returns "Warning" for warning', () => {
		expect(getHealthStatusLabel('warning')).toBe('Warning');
	});

	it('returns "Critical" for critical', () => {
		expect(getHealthStatusLabel('critical')).toBe('Critical');
	});

	it('returns "Unknown" for unknown', () => {
		expect(getHealthStatusLabel('other')).toBe('Unknown');
	});
});

describe('formatUptime', () => {
	it('returns "N/A" for undefined', () => {
		expect(formatUptime(undefined)).toBe('N/A');
	});

	it('formats minutes', () => {
		expect(formatUptime(300)).toBe('5m');
	});

	it('formats hours and minutes', () => {
		expect(formatUptime(3660)).toBe('1h 1m');
	});

	it('formats days and hours', () => {
		expect(formatUptime(90000)).toBe('1d 1h');
	});
});

describe('formatCurrency', () => {
	it('returns "$0.00" for undefined', () => {
		expect(formatCurrency(undefined)).toBe('$0.00');
	});

	it('formats currency value', () => {
		expect(formatCurrency(42.5)).toBe('$42.50');
	});

	it('formats large values', () => {
		const result = formatCurrency(1234.56);
		expect(result).toContain('1,234.56');
	});
});

describe('formatCurrencyCompact', () => {
	it('returns "$0" for undefined', () => {
		expect(formatCurrencyCompact(undefined)).toBe('$0');
	});

	it('formats small amounts normally', () => {
		expect(formatCurrencyCompact(42.5)).toBe('$42.50');
	});

	it('formats large amounts compactly', () => {
		const result = formatCurrencyCompact(5000);
		expect(result).toContain('$5');
	});
});

describe('getCostColor', () => {
	it('returns red for high cost', () => {
		expect(getCostColor(200)).toBe('text-red-600');
	});

	it('returns yellow for medium cost', () => {
		expect(getCostColor(100)).toBe('text-yellow-600');
	});

	it('returns green for low cost', () => {
		expect(getCostColor(50)).toBe('text-green-600');
	});

	it('returns gray for zero cost', () => {
		expect(getCostColor(0)).toBe('text-gray-600');
	});

	it('uses custom threshold', () => {
		expect(getCostColor(50, 50)).toBe('text-yellow-600');
	});
});
