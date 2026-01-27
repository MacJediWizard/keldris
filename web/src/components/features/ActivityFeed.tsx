import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useActivityFeed, useRecentActivity } from '../../hooks/useActivity';
import { useLocale } from '../../hooks/useLocale';
import type { ActivityEvent, ActivityEventCategory } from '../../lib/types';

// Get icon and color for event category
function getCategoryStyle(category: ActivityEventCategory): {
	bg: string;
	text: string;
	icon: React.ReactNode;
} {
	switch (category) {
		case 'backup':
			return {
				bg: 'bg-blue-100 dark:bg-blue-900/30',
				text: 'text-blue-600 dark:text-blue-400',
				icon: (
					<svg
						className="w-4 h-4"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
						aria-hidden="true"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"
						/>
					</svg>
				),
			};
		case 'restore':
			return {
				bg: 'bg-purple-100 dark:bg-purple-900/30',
				text: 'text-purple-600 dark:text-purple-400',
				icon: (
					<svg
						className="w-4 h-4"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
						aria-hidden="true"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
						/>
					</svg>
				),
			};
		case 'agent':
			return {
				bg: 'bg-green-100 dark:bg-green-900/30',
				text: 'text-green-600 dark:text-green-400',
				icon: (
					<svg
						className="w-4 h-4"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
						aria-hidden="true"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
						/>
					</svg>
				),
			};
		case 'user':
			return {
				bg: 'bg-indigo-100 dark:bg-indigo-900/30',
				text: 'text-indigo-600 dark:text-indigo-400',
				icon: (
					<svg
						className="w-4 h-4"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
						aria-hidden="true"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
						/>
					</svg>
				),
			};
		case 'alert':
			return {
				bg: 'bg-red-100 dark:bg-red-900/30',
				text: 'text-red-600 dark:text-red-400',
				icon: (
					<svg
						className="w-4 h-4"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
						aria-hidden="true"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
						/>
					</svg>
				),
			};
		case 'schedule':
			return {
				bg: 'bg-amber-100 dark:bg-amber-900/30',
				text: 'text-amber-600 dark:text-amber-400',
				icon: (
					<svg
						className="w-4 h-4"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
						aria-hidden="true"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
					</svg>
				),
			};
		case 'repository':
			return {
				bg: 'bg-cyan-100 dark:bg-cyan-900/30',
				text: 'text-cyan-600 dark:text-cyan-400',
				icon: (
					<svg
						className="w-4 h-4"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
						aria-hidden="true"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"
						/>
					</svg>
				),
			};
		case 'maintenance':
			return {
				bg: 'bg-orange-100 dark:bg-orange-900/30',
				text: 'text-orange-600 dark:text-orange-400',
				icon: (
					<svg
						className="w-4 h-4"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
						aria-hidden="true"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
						/>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
						/>
					</svg>
				),
			};
		default:
			return {
				bg: 'bg-gray-100 dark:bg-gray-800',
				text: 'text-gray-600 dark:text-gray-400',
				icon: (
					<svg
						className="w-4 h-4"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
						aria-hidden="true"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
					</svg>
				),
			};
	}
}

interface ActivityEventItemProps {
	event: ActivityEvent;
	showCategory?: boolean;
}

function ActivityEventItem({
	event,
	showCategory = true,
}: ActivityEventItemProps) {
	const { formatRelativeTime } = useLocale();
	const style = getCategoryStyle(event.category);

	return (
		<div className="flex items-start gap-3 py-3 border-b border-gray-100 dark:border-gray-700 last:border-0">
			{showCategory && (
				<div
					className={`flex-shrink-0 p-2 rounded-lg ${style.bg} ${style.text}`}
				>
					{style.icon}
				</div>
			)}
			<div className="flex-1 min-w-0">
				<p className="text-sm font-medium text-gray-900 dark:text-white truncate">
					{event.title}
				</p>
				<p className="text-xs text-gray-500 dark:text-gray-400 truncate">
					{event.description}
				</p>
				<div className="flex items-center gap-2 mt-1 text-xs text-gray-400 dark:text-gray-500">
					<span>{formatRelativeTime(event.created_at)}</span>
					{event.agent_name && (
						<>
							<span>•</span>
							<span>{event.agent_name}</span>
						</>
					)}
					{event.user_name && (
						<>
							<span>•</span>
							<span>{event.user_name}</span>
						</>
					)}
				</div>
			</div>
		</div>
	);
}

function LoadingRow() {
	return (
		<div className="flex items-start gap-3 py-3 border-b border-gray-100 dark:border-gray-700 last:border-0">
			<div className="w-8 h-8 bg-gray-200 dark:bg-gray-700 rounded-lg animate-pulse" />
			<div className="flex-1 space-y-2">
				<div className="h-4 w-32 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
				<div className="h-3 w-48 bg-gray-100 dark:bg-gray-800 rounded animate-pulse" />
				<div className="h-3 w-24 bg-gray-100 dark:bg-gray-800 rounded animate-pulse" />
			</div>
		</div>
	);
}

interface ActivityFeedWidgetProps {
	limit?: number;
	showViewAll?: boolean;
	enableRealtime?: boolean;
}

// Mini widget for dashboard
export function ActivityFeedWidget({
	limit = 5,
	showViewAll = true,
	enableRealtime = true,
}: ActivityFeedWidgetProps) {
	const { data: recentEvents, isLoading } = useRecentActivity(limit);
	const { events: liveEvents, isConnected } = useActivityFeed({
		enabled: enableRealtime,
		maxEvents: limit,
	});

	// Merge live events with recent events, removing duplicates
	const allEvents = enableRealtime
		? [...liveEvents, ...(recentEvents ?? [])]
				.filter(
					(event, index, self) =>
						self.findIndex((e) => e.id === event.id) === index,
				)
				.slice(0, limit)
		: (recentEvents ?? []);

	return (
		<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
			<div className="flex items-center justify-between mb-4">
				<div className="flex items-center gap-2">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						Activity Feed
					</h2>
					{enableRealtime && (
						<span
							className={`w-2 h-2 rounded-full ${
								isConnected ? 'bg-green-500 animate-pulse' : 'bg-gray-400'
							}`}
							title={isConnected ? 'Live updates enabled' : 'Connecting...'}
						/>
					)}
				</div>
				{showViewAll && (
					<Link
						to="/activity"
						className="text-sm text-indigo-600 hover:text-indigo-800 dark:text-indigo-400 dark:hover:text-indigo-300"
					>
						View All
					</Link>
				)}
			</div>

			{isLoading ? (
				<div>
					<LoadingRow />
					<LoadingRow />
					<LoadingRow />
				</div>
			) : allEvents.length === 0 ? (
				<div className="text-center py-8 text-gray-500 dark:text-gray-400">
					<svg
						aria-hidden="true"
						className="w-12 h-12 mx-auto mb-3 text-gray-300 dark:text-gray-600"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
						/>
					</svg>
					<p>No activity yet</p>
					<p className="text-sm">Events will appear here as they happen</p>
				</div>
			) : (
				<div>
					{allEvents.map((event) => (
						<ActivityEventItem key={event.id} event={event} />
					))}
				</div>
			)}
		</div>
	);
}

// Category filter pills
const CATEGORIES: { value: ActivityEventCategory | 'all'; label: string }[] = [
	{ value: 'all', label: 'All' },
	{ value: 'backup', label: 'Backups' },
	{ value: 'restore', label: 'Restores' },
	{ value: 'agent', label: 'Agents' },
	{ value: 'user', label: 'Users' },
	{ value: 'alert', label: 'Alerts' },
	{ value: 'schedule', label: 'Schedules' },
	{ value: 'repository', label: 'Repositories' },
	{ value: 'maintenance', label: 'Maintenance' },
	{ value: 'system', label: 'System' },
];

interface ActivityFeedFullProps {
	enableRealtime?: boolean;
}

// Full activity feed component for dedicated page
export function ActivityFeedFull({
	enableRealtime = true,
}: ActivityFeedFullProps) {
	const [selectedCategory, setSelectedCategory] = useState<
		ActivityEventCategory | 'all'
	>('all');

	const { data: recentEvents, isLoading, isFetching } = useRecentActivity(50);
	const {
		events: liveEvents,
		isConnected,
		clearEvents,
	} = useActivityFeed({
		enabled: enableRealtime,
		categories: selectedCategory === 'all' ? undefined : [selectedCategory],
		maxEvents: 100,
	});

	// Merge live events with recent events
	const filteredRecent =
		selectedCategory === 'all'
			? recentEvents
			: recentEvents?.filter((e) => e.category === selectedCategory);

	const allEvents = enableRealtime
		? [...liveEvents, ...(filteredRecent ?? [])].filter(
				(event, index, self) =>
					self.findIndex((e) => e.id === event.id) === index,
			)
		: (filteredRecent ?? []);

	return (
		<div className="space-y-4">
			{/* Header */}
			<div className="flex items-center justify-between">
				<div className="flex items-center gap-3">
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Activity Feed
					</h1>
					{enableRealtime && (
						<div className="flex items-center gap-2 text-sm">
							<span
								className={`w-2 h-2 rounded-full ${
									isConnected ? 'bg-green-500 animate-pulse' : 'bg-gray-400'
								}`}
							/>
							<span className="text-gray-500 dark:text-gray-400">
								{isConnected ? 'Live' : 'Connecting...'}
							</span>
						</div>
					)}
				</div>
				{liveEvents.length > 0 && (
					<button
						type="button"
						onClick={clearEvents}
						className="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
					>
						Clear new events
					</button>
				)}
			</div>

			{/* Category filter */}
			<div className="flex flex-wrap gap-2">
				{CATEGORIES.map((cat) => (
					<button
						type="button"
						key={cat.value}
						onClick={() => setSelectedCategory(cat.value)}
						className={`px-3 py-1.5 rounded-full text-sm font-medium transition-colors ${
							selectedCategory === cat.value
								? 'bg-indigo-600 text-white'
								: 'bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600'
						}`}
					>
						{cat.label}
					</button>
				))}
			</div>

			{/* Events list */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				{isLoading ? (
					<div className="p-4">
						<LoadingRow />
						<LoadingRow />
						<LoadingRow />
						<LoadingRow />
						<LoadingRow />
					</div>
				) : allEvents.length === 0 ? (
					<div className="text-center py-12 text-gray-500 dark:text-gray-400">
						<svg
							aria-hidden="true"
							className="w-16 h-16 mx-auto mb-4 text-gray-300 dark:text-gray-600"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
							/>
						</svg>
						<p className="text-lg font-medium">No activity found</p>
						<p className="text-sm">
							{selectedCategory === 'all'
								? 'Events will appear here as they happen'
								: `No ${selectedCategory} events found`}
						</p>
					</div>
				) : (
					<div className="divide-y divide-gray-100 dark:divide-gray-700">
						{allEvents.map((event, index) => (
							<div
								key={event.id}
								className={`p-4 ${
									index < liveEvents.length
										? 'bg-indigo-50/50 dark:bg-indigo-900/10'
										: ''
								}`}
							>
								<ActivityEventItem event={event} />
							</div>
						))}
					</div>
				)}
			</div>

			{/* Loading indicator for refetch */}
			{isFetching && !isLoading && (
				<div className="text-center text-sm text-gray-500 dark:text-gray-400">
					Refreshing...
				</div>
			)}
		</div>
	);
}
