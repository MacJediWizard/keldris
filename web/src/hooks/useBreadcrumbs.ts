import { useMemo } from 'react';
import { useLocation, useParams } from 'react-router-dom';
import { useAgent } from './useAgents';
import { useRepositoryStats } from './useStorageStats';

export interface BreadcrumbItem {
	label: string;
	path: string;
	isCurrentPage: boolean;
}

// Static route labels mapping
const routeLabels: Record<string, string> = {
	'': 'Dashboard',
	agents: 'Agents',
	'agent-groups': 'Agent Groups',
	repositories: 'Repositories',
	schedules: 'Schedules',
	policies: 'Policies',
	templates: 'Templates',
	backups: 'Backups',
	'dr-runbooks': 'DR Runbooks',
	restore: 'Restore',
	'file-history': 'File History',
	'file-search': 'File Search',
	snapshots: 'Snapshots',
	compare: 'Compare',
	'file-diff': 'File Diff',
	alerts: 'Alerts',
	downtime: 'Downtime History',
	notifications: 'Notifications',
	'notification-rules': 'Notification Rules',
	reports: 'Reports',
	'audit-logs': 'Audit Logs',
	'legal-holds': 'Legal Holds',
	'lifecycle-policies': 'Lifecycle Policies',
	stats: 'Storage Stats',
	tags: 'Tags',
	classifications: 'Classifications',
	costs: 'Cost Estimation',
	sla: 'SLA',
	organization: 'Organization',
	members: 'Members',
	settings: 'Settings',
	sso: 'SSO Group Sync',
	maintenance: 'Maintenance',
	announcements: 'Announcements',
	'ip-allowlist': 'IP Allowlist',
	'password-policies': 'Password Policy',
	new: 'Create New',
	admin: 'Admin',
	logs: 'Server Logs',
	'rate-limits': 'Rate Limits',
	'rate-limit-configs': 'Rate Limit Configs',
	account: 'Account',
	sessions: 'Sessions',
	onboarding: 'Onboarding',
	changelog: 'Changelog',
};

function isUuid(str: string): boolean {
	return /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(
		str,
	);
}

interface UseBreadcrumbsOptions {
	agentHostname?: string;
	repositoryName?: string;
}

export function useBreadcrumbs(options?: UseBreadcrumbsOptions): {
	breadcrumbs: BreadcrumbItem[];
	isLoading: boolean;
} {
	const location = useLocation();
	const params = useParams();

	// Check if we're on a dynamic route that needs name resolution
	const agentId =
		location.pathname.startsWith('/agents/') && params.id ? params.id : null;
	const repoId =
		location.pathname.startsWith('/stats/') && params.id ? params.id : null;

	// Fetch agent name if on agent details page
	const { data: agent, isLoading: agentLoading } = useAgent(agentId ?? '');

	// Fetch repository name if on stats detail page
	const { data: repoStats, isLoading: repoLoading } = useRepositoryStats(
		repoId ?? '',
	);

	const isLoading =
		(agentId !== null && agentLoading) || (repoId !== null && repoLoading);

	const breadcrumbs = useMemo(() => {
		const pathSegments = location.pathname.split('/').filter(Boolean);
		const items: BreadcrumbItem[] = [];

		// Always add Home/Dashboard
		items.push({
			label: 'Dashboard',
			path: '/',
			isCurrentPage: pathSegments.length === 0,
		});

		// Build breadcrumb items for each segment
		let currentPath = '';
		pathSegments.forEach((segment, index) => {
			currentPath += `/${segment}`;
			const isLastSegment = index === pathSegments.length - 1;

			// Check if this segment is a dynamic ID
			if (isUuid(segment)) {
				// Use provided name or fetched name, or fallback to truncated ID
				let label = `${segment.substring(0, 8)}...`;

				if (
					agentId === segment &&
					(options?.agentHostname || agent?.hostname)
				) {
					label = options?.agentHostname || agent?.hostname || label;
				} else if (
					repoId === segment &&
					(options?.repositoryName || repoStats?.repository_name)
				) {
					label =
						options?.repositoryName || repoStats?.repository_name || label;
				}

				items.push({
					label,
					path: currentPath,
					isCurrentPage: isLastSegment,
				});
			} else {
				// Use static label from mapping
				const label = routeLabels[segment] || segment;
				items.push({
					label,
					path: currentPath,
					isCurrentPage: isLastSegment,
				});
			}
		});

		return items;
	}, [
		location.pathname,
		agentId,
		repoId,
		agent?.hostname,
		repoStats?.repository_name,
		options?.agentHostname,
		options?.repositoryName,
	]);

	return { breadcrumbs, isLoading };
}

// Hook for pages that want to provide their own dynamic name
export function useBreadcrumbsWithName(name?: string): {
	breadcrumbs: BreadcrumbItem[];
	isLoading: boolean;
} {
	const location = useLocation();

	const breadcrumbs = useMemo(() => {
		const pathSegments = location.pathname.split('/').filter(Boolean);
		const items: BreadcrumbItem[] = [];

		// Always add Home/Dashboard
		items.push({
			label: 'Dashboard',
			path: '/',
			isCurrentPage: pathSegments.length === 0,
		});

		// Build breadcrumb items for each segment
		let currentPath = '';
		pathSegments.forEach((segment, index) => {
			currentPath += `/${segment}`;
			const isLastSegment = index === pathSegments.length - 1;

			// Check if this segment is a dynamic ID
			if (isUuid(segment)) {
				// Use provided name for the last segment if it's a UUID
				const label =
					isLastSegment && name ? name : `${segment.substring(0, 8)}...`;
				items.push({
					label,
					path: currentPath,
					isCurrentPage: isLastSegment,
				});
			} else {
				// Use static label from mapping
				const label = routeLabels[segment] || segment;
				items.push({
					label,
					path: currentPath,
					isCurrentPage: isLastSegment,
				});
			}
		});

		return items;
	}, [location.pathname, name]);

	return { breadcrumbs, isLoading: false };
}
