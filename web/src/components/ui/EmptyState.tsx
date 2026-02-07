import type { ReactNode } from 'react';
import { HelpTooltip } from './HelpTooltip';

interface EmptyStateProps {
	icon: ReactNode;
	title: string;
	description: string;
	action?: {
		label: string;
		onClick: () => void;
	};
	help?: {
		content: string;
		title?: string;
		docsUrl?: string;
	};
	variant?: 'default' | 'compact';
	children?: ReactNode;
}

export function EmptyState({
	icon,
	title,
	description,
	action,
	help,
	variant = 'default',
	children,
}: EmptyStateProps) {
	const isCompact = variant === 'compact';

	return (
		<div
			className={`text-center ${isCompact ? 'py-6' : 'py-12'} text-gray-500 dark:text-gray-400`}
		>
			<div
				className={`mx-auto ${isCompact ? 'w-12 h-12' : 'w-16 h-16'} text-gray-300 dark:text-gray-600`}
			>
				{icon}
			</div>
			<div className="flex items-center justify-center gap-1.5 mt-4">
				<h3
					className={`${isCompact ? 'text-base' : 'text-lg'} font-medium text-gray-900 dark:text-white`}
				>
					{title}
				</h3>
				{help && (
					<HelpTooltip
						content={help.content}
						title={help.title}
						docsUrl={help.docsUrl}
					/>
				)}
			</div>
			<p className={`mt-2 ${isCompact ? 'text-sm' : ''}`}>{description}</p>
			{children && <div className="mt-4">{children}</div>}
			{action && (
				<button
					type="button"
					onClick={action.onClick}
					className={`mt-6 inline-flex items-center gap-2 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors ${isCompact ? 'text-sm' : ''}`}
				>
					<svg
						aria-hidden="true"
						className="w-5 h-5"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M12 4v16m8-8H4"
						/>
					</svg>
					{action.label}
				</button>
			)}
		</div>
	);
}

// Pre-built empty state variants for common use cases

interface EmptyStateNoAgentsProps {
	onAddAgent: () => void;
	actionLabel?: string;
	title?: string;
	description?: string;
}

export function EmptyStateNoAgents({
	onAddAgent,
	actionLabel = 'Add Your First Agent',
	title = 'No agents connected',
	description = 'Agents run on your servers to perform backups. Install an agent to get started.',
}: EmptyStateNoAgentsProps) {
	return (
		<EmptyState
			icon={
				<svg
					className="w-full h-full"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={1.5}
						d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
					/>
				</svg>
			}
			title={title}
			description={description}
			action={{
				label: actionLabel,
				onClick: onAddAgent,
			}}
			help={{
				title: 'What are agents?',
				content:
					'Agents are lightweight programs that run on your servers. They connect to Cairo to receive backup schedules and execute backup operations securely.',
				docsUrl: '/docs/agents',
			}}
		/>
	);
}

interface EmptyStateNoBackupsProps {
	onCreateSchedule: () => void;
}

export function EmptyStateNoBackups({
	onCreateSchedule,
}: EmptyStateNoBackupsProps) {
	return (
		<EmptyState
			icon={
				<svg
					className="w-full h-full"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={1.5}
						d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"
					/>
				</svg>
			}
			title="No backups yet"
			description="Backups will appear here once you create a schedule and agents start running."
			action={{
				label: 'Create a Schedule',
				onClick: onCreateSchedule,
			}}
			help={{
				title: 'How backups work',
				content:
					'Backups are created automatically based on your schedules. Each backup captures a snapshot of your data that can be restored later.',
			}}
		/>
	);
}

interface EmptyStateNoSchedulesProps {
	onCreateSchedule: () => void;
	showCronExamples?: boolean;
}

export function EmptyStateNoSchedules({
	onCreateSchedule,
	showCronExamples = true,
}: EmptyStateNoSchedulesProps) {
	return (
		<EmptyState
			icon={
				<svg
					className="w-full h-full"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={1.5}
						d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				</svg>
			}
			title="No schedules configured"
			description="Schedules define when and what to backup. Create a schedule to automate your backups."
			action={{
				label: 'Set Up Backup Schedule',
				onClick: onCreateSchedule,
			}}
			help={{
				title: 'About schedules',
				content:
					'Schedules use cron expressions to define when backups run. You can set up daily, weekly, or custom intervals, and configure retention policies to manage storage.',
			}}
		>
			{showCronExamples && (
				<div className="bg-gray-50 dark:bg-gray-700 rounded-lg p-4 max-w-md mx-auto text-left space-y-2">
					<p className="text-sm font-medium text-gray-700 dark:text-gray-300">
						Common schedules:
					</p>
					<div className="text-sm text-gray-600 dark:text-gray-400 space-y-1">
						<p>
							<span className="font-mono bg-gray-200 dark:bg-gray-600 px-1 rounded">
								0 2 * * *
							</span>{' '}
							— Daily at 2 AM
						</p>
						<p>
							<span className="font-mono bg-gray-200 dark:bg-gray-600 px-1 rounded">
								0 */6 * * *
							</span>{' '}
							— Every 6 hours
						</p>
						<p>
							<span className="font-mono bg-gray-200 dark:bg-gray-600 px-1 rounded">
								0 3 * * 0
							</span>{' '}
							— Weekly on Sunday
						</p>
					</div>
				</div>
			)}
		</EmptyState>
	);
}

interface EmptyStateNoSearchResultsProps {
	query: string;
	onClearSearch?: () => void;
}

export function EmptyStateNoSearchResults({
	query,
	onClearSearch,
}: EmptyStateNoSearchResultsProps) {
	return (
		<EmptyState
			icon={
				<svg
					className="w-full h-full"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={1.5}
						d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
					/>
				</svg>
			}
			title={`No results for "${query}"`}
			description="Try different search terms or check your spelling."
			action={
				onClearSearch
					? {
							label: 'Clear Search',
							onClick: onClearSearch,
						}
					: undefined
			}
			help={{
				title: 'Search tips',
				content:
					'You can search by name, hostname, or ID. Use filters to narrow down results by type or date range.',
			}}
			variant="compact"
		/>
	);
}

interface EmptyStateNoGroupsProps {
	onCreateGroup: () => void;
}

export function EmptyStateNoGroups({ onCreateGroup }: EmptyStateNoGroupsProps) {
	return (
		<EmptyState
			icon={
				<svg
					className="w-full h-full"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={1.5}
						d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"
					/>
				</svg>
			}
			title="No agent groups"
			description="Groups help organize your agents by environment, location, or purpose."
			action={{
				label: 'Create Your First Group',
				onClick: onCreateGroup,
			}}
			help={{
				title: 'Why use groups?',
				content:
					'Agent groups make it easier to manage large deployments. You can apply policies and schedules to entire groups instead of individual agents.',
			}}
		/>
	);
}

interface EmptyStateNoRepositoriesProps {
	onCreateRepository: () => void;
}

export function EmptyStateNoRepositories({
	onCreateRepository,
}: EmptyStateNoRepositoriesProps) {
	return (
		<EmptyState
			icon={
				<svg
					className="w-full h-full"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={1.5}
						d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"
					/>
				</svg>
			}
			title="No repositories"
			description="Repositories store your backup data. Create a repository to define where backups are saved."
			action={{
				label: 'Add Repository',
				onClick: onCreateRepository,
			}}
			help={{
				title: 'What are repositories?',
				content:
					'Repositories are storage locations for your backups. They can be local paths, S3 buckets, or other supported backends. Data is encrypted and deduplicated.',
			}}
		/>
	);
}

interface EmptyStateNoPoliciesProps {
	onCreatePolicy: () => void;
}

export function EmptyStateNoPolicies({
	onCreatePolicy,
}: EmptyStateNoPoliciesProps) {
	return (
		<EmptyState
			icon={
				<svg
					className="w-full h-full"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={1.5}
						d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
					/>
				</svg>
			}
			title="No policies defined"
			description="Policies are templates that define backup settings. Use them to standardize configurations across schedules."
			action={{
				label: 'Create Policy',
				onClick: onCreatePolicy,
			}}
			help={{
				title: 'About policies',
				content:
					'Policies define reusable backup configurations including paths, retention rules, and schedule patterns. Apply them to multiple schedules for consistency.',
			}}
		/>
	);
}
