import {
	BadgeSkeleton,
	ButtonSkeleton,
	InputSkeleton,
	Skeleton,
	StatCardSkeleton,
	TextSkeleton,
	skeletonKeys,
} from './Skeleton';

/**
 * Agent table row skeleton matching the Agents page layout
 */
export function AgentRowSkeleton() {
	return (
		<tr className="animate-pulse">
			<td className="px-6 py-4 w-12">
				<Skeleton className="h-4 w-4" />
			</td>
			<td className="px-6 py-4">
				<div className="flex items-center gap-2">
					<Skeleton className="h-4 w-4" />
					<div className="space-y-1">
						<TextSkeleton width="lg" />
						<TextSkeleton width="md" size="sm" />
					</div>
				</div>
			</td>
			<td className="px-6 py-4">
				<BadgeSkeleton />
			</td>
			<td className="px-6 py-4">
				<TextSkeleton width="md" />
			</td>
			<td className="px-6 py-4">
				<TextSkeleton width="lg" />
			</td>
			<td className="px-6 py-4 text-right">
				<ButtonSkeleton size="sm" className="inline-block" />
			</td>
		</tr>
	);
}

interface AgentListSkeletonProps {
	rows?: number;
}

/**
 * Full agent list skeleton with table structure
 */
export function AgentListSkeleton({ rows = 3 }: AgentListSkeletonProps) {
	return (
		<table className="w-full">
			<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
				<tr>
					<th className="px-6 py-3 w-12" />
					<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
						Hostname
					</th>
					<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
						Status
					</th>
					<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
						Last Seen
					</th>
					<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
						Registered
					</th>
					<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
						Actions
					</th>
				</tr>
			</thead>
			<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
				{skeletonKeys(rows, 'agent-row').map((key) => (
					<AgentRowSkeleton key={key} />
				))}
			</tbody>
		</table>
	);
}

/**
 * Schedule table row skeleton matching the Schedules page layout
 */
export function ScheduleRowSkeleton() {
	return (
		<tr className="animate-pulse">
			<td className="px-6 py-4 w-12">
				<Skeleton className="h-4 w-4" />
			</td>
			<td className="px-6 py-4">
				<div className="flex items-center gap-2">
					<Skeleton className="h-4 w-4" />
					<TextSkeleton width="lg" />
				</div>
			</td>
			<td className="px-6 py-4">
				<TextSkeleton width="md" />
			</td>
			<td className="px-6 py-4">
				<TextSkeleton width="sm" />
			</td>
			<td className="px-6 py-4">
				<BadgeSkeleton />
			</td>
			<td className="px-6 py-4 text-right">
				<ButtonSkeleton size="md" className="inline-block" />
			</td>
		</tr>
	);
}

interface ScheduleListSkeletonProps {
	rows?: number;
}

/**
 * Full schedule list skeleton with table structure
 */
export function ScheduleListSkeleton({ rows = 3 }: ScheduleListSkeletonProps) {
	return (
		<table className="w-full">
			<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
				<tr>
					<th className="px-6 py-3 w-12" />
					<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
						Name
					</th>
					<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
						Agent
					</th>
					<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
						Schedule
					</th>
					<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
						Status
					</th>
					<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
						Actions
					</th>
				</tr>
			</thead>
			<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
				{skeletonKeys(rows, 'sched-row').map((key) => (
					<ScheduleRowSkeleton key={key} />
				))}
			</tbody>
		</table>
	);
}

/**
 * Dashboard stat cards skeleton (4 cards grid)
 */
export function DashboardStatsSkeleton() {
	return (
		<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
			{skeletonKeys(4, 'stat').map((key) => (
				<StatCardSkeleton key={key} />
			))}
		</div>
	);
}

/**
 * Dashboard backup list skeleton
 */
export function DashboardBackupListSkeleton() {
	return (
		<div className="space-y-1">
			{skeletonKeys(5, 'backup').map((key) => (
				<div
					key={key}
					className="flex items-center justify-between py-3 border-b border-gray-100 dark:border-gray-700 last:border-0 animate-pulse"
				>
					<div className="space-y-2">
						<TextSkeleton width="lg" />
						<TextSkeleton width="md" size="sm" />
					</div>
					<BadgeSkeleton width="sm" />
				</div>
			))}
		</div>
	);
}

/**
 * Dashboard queue status skeleton
 */
export function DashboardQueueSkeleton() {
	return (
		<div className="grid grid-cols-2 md:grid-cols-5 gap-4 animate-pulse">
			{skeletonKeys(5, 'queue').map((key) => (
				<div key={key}>
					<TextSkeleton width="md" size="sm" className="mb-2" />
					<Skeleton className="h-8 w-12" />
				</div>
			))}
		</div>
	);
}

/**
 * Dashboard favorites section skeleton
 */
export function DashboardFavoritesSkeleton() {
	return (
		<div className="grid grid-cols-1 md:grid-cols-3 gap-4 animate-pulse">
			{skeletonKeys(3, 'fav-col').map((colKey) => (
				<div key={colKey}>
					<TextSkeleton width="md" size="sm" className="mb-3" />
					<div className="space-y-2">
						{skeletonKeys(3, `${colKey}-item`).map((itemKey) => (
							<div
								key={itemKey}
								className="flex items-center justify-between p-2 bg-gray-50 dark:bg-gray-700 rounded"
							>
								<div className="flex items-center gap-2">
									<Skeleton className="h-4 w-4" />
									<TextSkeleton width="lg" size="sm" />
								</div>
								<Skeleton className="h-2 w-2 rounded-full" />
							</div>
						))}
					</div>
				</div>
			))}
		</div>
	);
}

/**
 * Full dashboard page skeleton
 */
export function DashboardSkeleton() {
	return (
		<div className="space-y-6">
			{/* Header */}
			<div>
				<TextSkeleton width="lg" size="xl" className="mb-2" />
				<TextSkeleton width="xl" size="base" />
			</div>

			{/* Stats cards */}
			<DashboardStatsSkeleton />

			{/* Queue status */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
				<div className="flex items-center justify-between mb-4">
					<TextSkeleton width="lg" size="lg" />
					<TextSkeleton width="md" size="sm" />
				</div>
				<DashboardQueueSkeleton />
			</div>

			{/* Main content grid */}
			<div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
				{/* Recent backups */}
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
					<TextSkeleton width="lg" size="lg" className="mb-4" />
					<DashboardBackupListSkeleton />
				</div>

				{/* System status */}
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
					<TextSkeleton width="lg" size="lg" className="mb-4" />
					<div className="space-y-4">
						{skeletonKeys(3, 'status').map((key) => (
							<div key={key} className="flex items-center justify-between">
								<TextSkeleton width="md" />
								<BadgeSkeleton width="sm" />
							</div>
						))}
					</div>
				</div>

				{/* Calendar */}
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
					<TextSkeleton width="lg" size="lg" className="mb-4" />
					<Skeleton className="h-48 w-full" />
				</div>
			</div>

			{/* Storage efficiency */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
				<div className="flex items-center justify-between mb-4">
					<TextSkeleton width="xl" size="lg" />
					<TextSkeleton width="sm" size="sm" />
				</div>
				<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
					{skeletonKeys(4, 'storage').map((key) => (
						<div key={key} className="animate-pulse">
							<TextSkeleton width="md" size="sm" className="mb-2" />
							<Skeleton className="h-8 w-20 mb-1" />
							<TextSkeleton width="lg" size="sm" />
						</div>
					))}
				</div>
			</div>
		</div>
	);
}

/**
 * Schedule form modal skeleton
 */
export function ScheduleFormSkeleton() {
	return (
		<div className="space-y-6 animate-pulse">
			{/* Basic info */}
			<div className="space-y-4">
				<InputSkeleton />
				<InputSkeleton />
				<InputSkeleton />
			</div>

			{/* Paths section */}
			<div>
				<TextSkeleton width="md" size="sm" className="mb-2" />
				<Skeleton className="h-24 w-full" />
			</div>

			{/* Schedule section */}
			<div className="space-y-4">
				<TextSkeleton width="lg" size="lg" />
				<InputSkeleton />
				<div className="flex gap-4">
					<InputSkeleton className="flex-1" />
					<InputSkeleton className="flex-1" />
				</div>
			</div>

			{/* Retention section */}
			<div className="space-y-4">
				<TextSkeleton width="lg" size="lg" />
				<div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
					{skeletonKeys(5, 'retention').map((key) => (
						<InputSkeleton key={key} />
					))}
				</div>
			</div>

			{/* Actions */}
			<div className="flex justify-end gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
				<ButtonSkeleton size="md" />
				<ButtonSkeleton size="lg" />
			</div>
		</div>
	);
}

/**
 * Generic list loading skeleton for any table
 */
interface GenericListSkeletonProps {
	rows?: number;
	columns?: number;
	showCheckbox?: boolean;
	showActions?: boolean;
}

export function GenericListSkeleton({
	rows = 5,
	columns = 5,
	showCheckbox = true,
	showActions = true,
}: GenericListSkeletonProps) {
	const contentCols = columns - (showCheckbox ? 1 : 0) - (showActions ? 1 : 0);

	return (
		<table className="w-full">
			<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
				<tr>
					{showCheckbox && <th className="px-6 py-3 w-12" />}
					{skeletonKeys(contentCols, 'th').map((key) => (
						<th
							key={key}
							className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
						>
							<TextSkeleton width="md" size="xs" />
						</th>
					))}
					{showActions && (
						<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
							<TextSkeleton width="sm" size="xs" className="ml-auto" />
						</th>
					)}
				</tr>
			</thead>
			<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
				{skeletonKeys(rows, 'row').map((rowKey) => (
					<tr key={rowKey} className="animate-pulse">
						{showCheckbox && (
							<td className="px-6 py-4 w-12">
								<Skeleton className="h-4 w-4" />
							</td>
						)}
						{skeletonKeys(contentCols, `${rowKey}-col`).map(
							(colKey, colIdx) => (
								<td key={colKey} className="px-6 py-4">
									{colIdx === 0 ? (
										<div className="space-y-1">
											<TextSkeleton width="lg" />
											<TextSkeleton width="md" size="sm" />
										</div>
									) : colIdx === columns - 3 ? (
										<BadgeSkeleton />
									) : (
										<TextSkeleton width={colIdx % 2 === 0 ? 'md' : 'lg'} />
									)}
								</td>
							),
						)}
						{showActions && (
							<td className="px-6 py-4 text-right">
								<ButtonSkeleton size="sm" className="inline-block" />
							</td>
						)}
					</tr>
				))}
			</tbody>
		</table>
	);
}
