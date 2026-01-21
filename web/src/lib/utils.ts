// Format date to relative time (past or future)
export function formatRelativeTime(dateString: string | undefined): string {
	if (!dateString) return 'Never';

	const date = new Date(dateString);
	const now = new Date();
	const diffMs = date.getTime() - now.getTime();
	const isFuture = diffMs > 0;
	const absDiffMs = Math.abs(diffMs);
	const absDiffSeconds = Math.floor(absDiffMs / 1000);
	const absDiffMinutes = Math.floor(absDiffSeconds / 60);
	const absDiffHours = Math.floor(absDiffMinutes / 60);
	const absDiffDays = Math.floor(absDiffHours / 24);

	if (absDiffSeconds < 60) return isFuture ? 'In a moment' : 'Just now';
	if (absDiffMinutes < 60)
		return isFuture ? `In ${absDiffMinutes}m` : `${absDiffMinutes}m ago`;
	if (absDiffHours < 24)
		return isFuture ? `In ${absDiffHours}h` : `${absDiffHours}h ago`;
	if (absDiffDays < 7)
		return isFuture ? `In ${absDiffDays}d` : `${absDiffDays}d ago`;

	return date.toLocaleDateString('en-US', {
		month: 'short',
		day: 'numeric',
		year: date.getFullYear() !== now.getFullYear() ? 'numeric' : undefined,
	});
}

// Format date to relative time or absolute date
export function formatDate(dateString: string | undefined): string {
	if (!dateString) return 'Never';

	const date = new Date(dateString);
	const now = new Date();
	const diffMs = now.getTime() - date.getTime();
	const diffSeconds = Math.floor(diffMs / 1000);
	const diffMinutes = Math.floor(diffSeconds / 60);
	const diffHours = Math.floor(diffMinutes / 60);
	const diffDays = Math.floor(diffHours / 24);

	if (diffSeconds < 60) return 'Just now';
	if (diffMinutes < 60) return `${diffMinutes}m ago`;
	if (diffHours < 24) return `${diffHours}h ago`;
	if (diffDays < 7) return `${diffDays}d ago`;

	return date.toLocaleDateString('en-US', {
		month: 'short',
		day: 'numeric',
		year: date.getFullYear() !== now.getFullYear() ? 'numeric' : undefined,
	});
}

// Format date to full datetime
export function formatDateTime(dateString: string | undefined): string {
	if (!dateString) return 'N/A';

	const date = new Date(dateString);
	return date.toLocaleString('en-US', {
		month: 'short',
		day: 'numeric',
		year: 'numeric',
		hour: 'numeric',
		minute: '2-digit',
	});
}

// Format bytes to human readable size
export function formatBytes(bytes: number | undefined): string {
	if (bytes === undefined || bytes === null) return 'N/A';
	if (bytes === 0) return '0 B';

	const units = ['B', 'KB', 'MB', 'GB', 'TB'];
	const k = 1024;
	const i = Math.floor(Math.log(bytes) / Math.log(k));
	const value = bytes / k ** i;

	return `${value.toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

// Format duration between two dates
export function formatDuration(
	startDate: string,
	endDate: string | undefined,
): string {
	if (!endDate) return 'In progress';

	const start = new Date(startDate);
	const end = new Date(endDate);
	const diffMs = end.getTime() - start.getTime();
	const diffSeconds = Math.floor(diffMs / 1000);
	const diffMinutes = Math.floor(diffSeconds / 60);
	const diffHours = Math.floor(diffMinutes / 60);

	if (diffSeconds < 60) return `${diffSeconds}s`;
	if (diffMinutes < 60) return `${diffMinutes}m ${diffSeconds % 60}s`;
	return `${diffHours}h ${diffMinutes % 60}m`;
}

// Truncate snapshot ID for display
export function truncateSnapshotId(id: string | undefined): string {
	if (!id) return 'N/A';
	return id.substring(0, 8);
}

// Get status color classes for badges
export function getAgentStatusColor(status: string): {
	bg: string;
	text: string;
	dot: string;
} {
	switch (status) {
		case 'active':
			return {
				bg: 'bg-green-100',
				text: 'text-green-800',
				dot: 'bg-green-500',
			};
		case 'offline':
			return { bg: 'bg-red-100', text: 'text-red-800', dot: 'bg-red-500' };
		case 'pending':
			return {
				bg: 'bg-yellow-100',
				text: 'text-yellow-800',
				dot: 'bg-yellow-500',
			};
		case 'disabled':
			return { bg: 'bg-gray-100', text: 'text-gray-600', dot: 'bg-gray-400' };
		default:
			return { bg: 'bg-gray-100', text: 'text-gray-600', dot: 'bg-gray-400' };
	}
}

export function getBackupStatusColor(status: string): {
	bg: string;
	text: string;
	dot: string;
} {
	switch (status) {
		case 'completed':
			return {
				bg: 'bg-green-100',
				text: 'text-green-800',
				dot: 'bg-green-500',
			};
		case 'running':
			return {
				bg: 'bg-blue-100',
				text: 'text-blue-800',
				dot: 'bg-blue-500',
			};
		case 'failed':
			return { bg: 'bg-red-100', text: 'text-red-800', dot: 'bg-red-500' };
		case 'canceled':
			return { bg: 'bg-gray-100', text: 'text-gray-600', dot: 'bg-gray-400' };
		default:
			return { bg: 'bg-gray-100', text: 'text-gray-600', dot: 'bg-gray-400' };
	}
}

export function getRepositoryTypeBadge(type: string): {
	label: string;
	className: string;
} {
	switch (type) {
		case 'local':
			return { label: 'Local', className: 'bg-gray-100 text-gray-800' };
		case 's3':
			return { label: 'S3', className: 'bg-orange-100 text-orange-800' };
		case 'b2':
			return { label: 'B2', className: 'bg-blue-100 text-blue-800' };
		case 'sftp':
			return { label: 'SFTP', className: 'bg-purple-100 text-purple-800' };
		case 'rest':
			return { label: 'REST', className: 'bg-indigo-100 text-indigo-800' };
		case 'dropbox':
			return { label: 'Dropbox', className: 'bg-sky-100 text-sky-800' };
		default:
			return { label: type, className: 'bg-gray-100 text-gray-800' };
	}
}

export function getAlertSeverityColor(severity: string): {
	bg: string;
	text: string;
	border: string;
	icon: string;
} {
	switch (severity) {
		case 'critical':
			return {
				bg: 'bg-red-50',
				text: 'text-red-800',
				border: 'border-red-200',
				icon: 'text-red-500',
			};
		case 'warning':
			return {
				bg: 'bg-yellow-50',
				text: 'text-yellow-800',
				border: 'border-yellow-200',
				icon: 'text-yellow-500',
			};
		case 'info':
			return {
				bg: 'bg-blue-50',
				text: 'text-blue-800',
				border: 'border-blue-200',
				icon: 'text-blue-500',
			};
		default:
			return {
				bg: 'bg-gray-50',
				text: 'text-gray-800',
				border: 'border-gray-200',
				icon: 'text-gray-500',
			};
	}
}

export function getAlertStatusColor(status: string): {
	bg: string;
	text: string;
	dot: string;
} {
	switch (status) {
		case 'active':
			return { bg: 'bg-red-100', text: 'text-red-800', dot: 'bg-red-500' };
		case 'acknowledged':
			return {
				bg: 'bg-yellow-100',
				text: 'text-yellow-800',
				dot: 'bg-yellow-500',
			};
		case 'resolved':
			return {
				bg: 'bg-green-100',
				text: 'text-green-800',
				dot: 'bg-green-500',
			};
		default:
			return { bg: 'bg-gray-100', text: 'text-gray-600', dot: 'bg-gray-400' };
	}
}

export function getAlertTypeLabel(type: string): string {
	switch (type) {
		case 'agent_offline':
			return 'Agent Offline';
		case 'backup_sla':
			return 'Backup SLA';
		case 'storage_usage':
			return 'Storage Usage';
		case 'agent_health_warning':
			return 'Agent Health Warning';
		case 'agent_health_critical':
			return 'Agent Health Critical';
		default:
			return type;
	}
}

// Audit log utilities
export function getAuditActionColor(action: string): {
	bg: string;
	text: string;
} {
	switch (action) {
		case 'create':
			return { bg: 'bg-green-100', text: 'text-green-800' };
		case 'update':
			return { bg: 'bg-blue-100', text: 'text-blue-800' };
		case 'delete':
			return { bg: 'bg-red-100', text: 'text-red-800' };
		case 'read':
			return { bg: 'bg-gray-100', text: 'text-gray-800' };
		case 'login':
			return { bg: 'bg-indigo-100', text: 'text-indigo-800' };
		case 'logout':
			return { bg: 'bg-purple-100', text: 'text-purple-800' };
		case 'backup':
			return { bg: 'bg-cyan-100', text: 'text-cyan-800' };
		case 'restore':
			return { bg: 'bg-orange-100', text: 'text-orange-800' };
		default:
			return { bg: 'bg-gray-100', text: 'text-gray-600' };
	}
}

export function getAuditResultColor(result: string): {
	bg: string;
	text: string;
	dot: string;
} {
	switch (result) {
		case 'success':
			return {
				bg: 'bg-green-100',
				text: 'text-green-800',
				dot: 'bg-green-500',
			};
		case 'failure':
			return { bg: 'bg-red-100', text: 'text-red-800', dot: 'bg-red-500' };
		case 'denied':
			return {
				bg: 'bg-yellow-100',
				text: 'text-yellow-800',
				dot: 'bg-yellow-500',
			};
		default:
			return { bg: 'bg-gray-100', text: 'text-gray-600', dot: 'bg-gray-400' };
	}
}

export function formatAuditAction(action: string): string {
	return action.charAt(0).toUpperCase() + action.slice(1);
}

export function formatResourceType(type: string): string {
	return type
		.split('_')
		.map((word) => word.charAt(0).toUpperCase() + word.slice(1))
		.join(' ');
}

// Format deduplication ratio (e.g., 2.5x)
export function formatDedupRatio(ratio: number | undefined): string {
	if (ratio === undefined || ratio === null || ratio === 0) return 'N/A';
	return `${ratio.toFixed(1)}x`;
}

// Format percentage (e.g., 45.2%)
export function formatPercent(percent: number | undefined): string {
	if (percent === undefined || percent === null) return 'N/A';
	return `${percent.toFixed(1)}%`;
}

// Get color class based on dedup ratio quality
export function getDedupRatioColor(ratio: number): string {
	if (ratio >= 3) return 'text-green-600';
	if (ratio >= 2) return 'text-blue-600';
	if (ratio >= 1.5) return 'text-yellow-600';
	return 'text-gray-600';
}

// Get color class based on space saved percentage
export function getSpaceSavedColor(percent: number): string {
	if (percent >= 70) return 'text-green-600';
	if (percent >= 50) return 'text-blue-600';
	if (percent >= 30) return 'text-yellow-600';
	return 'text-gray-600';
}

// Format a date for chart axis labels
export function formatChartDate(dateString: string): string {
	const date = new Date(dateString);
	return date.toLocaleDateString('en-US', {
		month: 'short',
		day: 'numeric',
	});
}

// Format duration in milliseconds to human readable
export function formatDurationMs(ms: number | undefined): string {
	if (ms === undefined || ms === null) return 'N/A';
	if (ms < 1000) return `${ms}ms`;

	const seconds = Math.floor(ms / 1000);
	const minutes = Math.floor(seconds / 60);
	const hours = Math.floor(minutes / 60);

	if (hours > 0) return `${hours}h ${minutes % 60}m`;
	if (minutes > 0) return `${minutes}m ${seconds % 60}s`;
	return `${seconds}s`;
}

// Get success rate color based on percentage
export function getSuccessRateColor(percent: number): string {
	if (percent >= 95) return 'text-green-600';
	if (percent >= 80) return 'text-yellow-600';
	if (percent >= 50) return 'text-orange-600';
	return 'text-red-600';
}

// Get success rate badge classes based on percentage
export function getSuccessRateBadge(percent: number): {
	bg: string;
	text: string;
} {
	if (percent >= 95) return { bg: 'bg-green-100', text: 'text-green-800' };
	if (percent >= 80) return { bg: 'bg-yellow-100', text: 'text-yellow-800' };
	if (percent >= 50) return { bg: 'bg-orange-100', text: 'text-orange-800' };
	return { bg: 'bg-red-100', text: 'text-red-800' };
}

// Get health status color classes for badges
export function getHealthStatusColor(status: string): {
	bg: string;
	text: string;
	dot: string;
	icon: string;
} {
	switch (status) {
		case 'healthy':
			return {
				bg: 'bg-green-100',
				text: 'text-green-800',
				dot: 'bg-green-500',
				icon: 'text-green-500',
			};
		case 'warning':
			return {
				bg: 'bg-yellow-100',
				text: 'text-yellow-800',
				dot: 'bg-yellow-500',
				icon: 'text-yellow-500',
			};
		case 'critical':
			return {
				bg: 'bg-red-100',
				text: 'text-red-800',
				dot: 'bg-red-500',
				icon: 'text-red-500',
			};
		case 'unknown':
		default:
			return {
				bg: 'bg-gray-100',
				text: 'text-gray-600',
				dot: 'bg-gray-400',
				icon: 'text-gray-400',
			};
	}
}

// Get health status label
export function getHealthStatusLabel(status: string): string {
	switch (status) {
		case 'healthy':
			return 'Healthy';
		case 'warning':
			return 'Warning';
		case 'critical':
			return 'Critical';
		case 'unknown':
		default:
			return 'Unknown';
	}
}

// Format uptime from seconds
export function formatUptime(seconds: number | undefined): string {
	if (seconds === undefined || seconds === null) return 'N/A';

	const days = Math.floor(seconds / 86400);
	const hours = Math.floor((seconds % 86400) / 3600);
	const minutes = Math.floor((seconds % 3600) / 60);

	if (days > 0) return `${days}d ${hours}h`;
	if (hours > 0) return `${hours}h ${minutes}m`;
	return `${minutes}m`;
}
