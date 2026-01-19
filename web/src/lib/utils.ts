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
		default:
			return { label: type, className: 'bg-gray-100 text-gray-800' };
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
