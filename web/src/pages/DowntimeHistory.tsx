import { useState } from 'react';
import {
	useDowntimeEvents,
	useActiveDowntime,
	useUptimeSummary,
	useResolveDowntimeEvent,
	useMonthlyUptimeReport,
} from '../hooks/useDowntime';
import type {
	DowntimeEvent,
	DowntimeSeverity,
	ComponentType,
} from '../lib/types';
import { formatDateTime, formatDate } from '../lib/utils';

function LoadingCard() {
	return (
		<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4 animate-pulse">
			<div className="flex items-start gap-4">
				<div className="w-10 h-10 bg-gray-200 dark:bg-gray-700 rounded-full" />
				<div className="flex-1">
					<div className="h-5 w-3/4 bg-gray-200 dark:bg-gray-700 rounded mb-2" />
					<div className="h-4 w-1/2 bg-gray-200 dark:bg-gray-700 rounded mb-3" />
					<div className="h-3 w-1/4 bg-gray-200 dark:bg-gray-700 rounded" />
				</div>
			</div>
		</div>
	);
}

function getSeverityColor(severity: DowntimeSeverity) {
	switch (severity) {
		case 'critical':
			return {
				bg: 'bg-red-100',
				text: 'text-red-700',
				border: 'border-red-200',
				icon: 'text-red-600',
				dot: 'bg-red-500',
			};
		case 'warning':
			return {
				bg: 'bg-yellow-100',
				text: 'text-yellow-700',
				border: 'border-yellow-200',
				icon: 'text-yellow-600',
				dot: 'bg-yellow-500',
			};
		default:
			return {
				bg: 'bg-blue-100',
				text: 'text-blue-700',
				border: 'border-blue-200',
				icon: 'text-blue-600',
				dot: 'bg-blue-500',
			};
	}
}

function getComponentTypeLabel(type: ComponentType) {
	switch (type) {
		case 'agent':
			return 'Agent';
		case 'server':
			return 'Server';
		case 'repository':
			return 'Repository';
		case 'service':
			return 'Service';
		default:
			return type;
	}
}

function formatDuration(seconds: number) {
	if (seconds < 60) {
		return `${seconds}s`;
	}
	if (seconds < 3600) {
		const minutes = Math.floor(seconds / 60);
		const secs = seconds % 60;
		return secs > 0 ? `${minutes}m ${secs}s` : `${minutes}m`;
	}
	const hours = Math.floor(seconds / 3600);
	const minutes = Math.floor((seconds % 3600) / 60);
	return minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`;
}

function UptimeBadge({
	percent,
	label,
}: {
	percent: number;
	label: string;
}) {
	let color = 'bg-green-500';
	if (percent < 95) {
		color = 'bg-red-500';
	} else if (percent < 99) {
		color = 'bg-orange-500';
	} else if (percent < 99.9) {
		color = 'bg-yellow-500';
	}

	return (
		<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
			<div className="flex items-center gap-3">
				<div className={`w-3 h-3 ${color} rounded-full`} />
				<div>
					<div className="text-2xl font-bold text-gray-900 dark:text-white">
						{percent.toFixed(2)}%
					</div>
					<div className="text-sm text-gray-500 dark:text-gray-400">
						{label}
					</div>
				</div>
			</div>
		</div>
	);
}

interface DowntimeEventCardProps {
	event: DowntimeEvent;
	onResolve: (id: string) => void;
	isProcessing: boolean;
}

function DowntimeEventCard({
	event,
	onResolve,
	isProcessing,
}: DowntimeEventCardProps) {
	const severityColor = getSeverityColor(event.severity);
	const isActive = !event.ended_at;

	return (
		<div
			className={`bg-white rounded-lg border ${isActive ? severityColor.border : 'border-gray-200'} p-4 hover:shadow-sm transition-shadow`}
		>
			<div className="flex items-start gap-4">
				<div
					className={`flex-shrink-0 w-10 h-10 ${severityColor.bg} rounded-full flex items-center justify-center`}
				>
					{event.severity === 'critical' ? (
						<svg
							aria-hidden="true"
							className={`w-5 h-5 ${severityColor.icon}`}
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
							/>
						</svg>
					) : event.severity === 'warning' ? (
						<svg
							aria-hidden="true"
							className={`w-5 h-5 ${severityColor.icon}`}
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
							/>
						</svg>
					) : (
						<svg
							aria-hidden="true"
							className={`w-5 h-5 ${severityColor.icon}`}
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
					)}
				</div>
				<div className="flex-1 min-w-0">
					<div className="flex items-center gap-2 mb-1">
						<h3 className="font-medium text-gray-900 dark:text-white truncate">
							{event.component_name}
						</h3>
						<span
							className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${
								isActive
									? 'bg-red-100 text-red-700'
									: 'bg-green-100 text-green-700'
							}`}
						>
							<span
								className={`w-1.5 h-1.5 ${isActive ? 'bg-red-500' : 'bg-green-500'} rounded-full`}
							/>
							{isActive ? 'Active' : 'Resolved'}
						</span>
						<span
							className={`px-2 py-0.5 rounded-full text-xs font-medium ${severityColor.bg} ${severityColor.text}`}
						>
							{event.severity}
						</span>
					</div>
					{event.cause && (
						<p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
							{event.cause}
						</p>
					)}
					<div className="flex flex-wrap items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
						<span className="inline-flex items-center gap-1">
							<svg
								aria-hidden="true"
								className="w-3.5 h-3.5"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"
								/>
							</svg>
							{getComponentTypeLabel(event.component_type)}
						</span>
						<span title={formatDateTime(event.started_at)}>
							Started: {formatDate(event.started_at)}
						</span>
						{event.ended_at && (
							<span title={formatDateTime(event.ended_at)}>
								Ended: {formatDate(event.ended_at)}
							</span>
						)}
						{event.duration_seconds !== undefined && (
							<span className="font-medium">
								Duration: {formatDuration(event.duration_seconds)}
							</span>
						)}
						{isActive && (
							<span className="font-medium text-red-600">
								Duration:{' '}
								{formatDuration(
									Math.floor(
										(Date.now() - new Date(event.started_at).getTime()) / 1000,
									),
								)}
							</span>
						)}
					</div>
					{event.notes && (
						<p className="mt-2 text-sm text-gray-500 dark:text-gray-400 italic">
							{event.notes}
						</p>
					)}
				</div>
				{isActive && (
					<div className="flex-shrink-0">
						<button
							type="button"
							onClick={() => onResolve(event.id)}
							disabled={isProcessing}
							className="px-3 py-1.5 text-sm font-medium text-green-700 bg-green-100 rounded-lg hover:bg-green-200 transition-colors disabled:opacity-50"
						>
							Resolve
						</button>
					</div>
				)}
			</div>
		</div>
	);
}

function TimelineView({ events }: { events: DowntimeEvent[] }) {
	return (
		<div className="relative">
			<div className="absolute left-4 top-0 bottom-0 w-0.5 bg-gray-200 dark:bg-gray-700" />
			<div className="space-y-6">
				{events.map((event, index) => {
					const severityColor = getSeverityColor(event.severity);
					const isActive = !event.ended_at;
					return (
						<div key={event.id} className="relative pl-10">
							<div
								className={`absolute left-2 w-5 h-5 -translate-x-1/2 ${severityColor.bg} rounded-full flex items-center justify-center border-2 border-white dark:border-gray-900`}
							>
								<div className={`w-2 h-2 ${severityColor.dot} rounded-full`} />
							</div>
							<div
								className={`bg-white dark:bg-gray-800 rounded-lg border ${isActive ? severityColor.border : 'border-gray-200 dark:border-gray-700'} p-4`}
							>
								<div className="flex items-start justify-between gap-4">
									<div>
										<div className="flex items-center gap-2 mb-1">
											<h4 className="font-medium text-gray-900 dark:text-white">
												{event.component_name}
											</h4>
											<span
												className={`px-2 py-0.5 rounded-full text-xs font-medium ${severityColor.bg} ${severityColor.text}`}
											>
												{event.severity}
											</span>
										</div>
										{event.cause && (
											<p className="text-sm text-gray-600 dark:text-gray-400">
												{event.cause}
											</p>
										)}
									</div>
									<div className="text-right text-xs text-gray-500 dark:text-gray-400">
										<div>{formatDateTime(event.started_at)}</div>
										{event.duration_seconds !== undefined && (
											<div className="font-medium">
												{formatDuration(event.duration_seconds)}
											</div>
										)}
									</div>
								</div>
							</div>
						</div>
					);
				})}
			</div>
		</div>
	);
}

export function DowntimeHistory() {
	const [viewMode, setViewMode] = useState<'list' | 'timeline'>('list');
	const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'resolved'>('all');
	const [severityFilter, setSeverityFilter] = useState<DowntimeSeverity | 'all'>('all');
	const [componentFilter, setComponentFilter] = useState<ComponentType | 'all'>('all');
	const [selectedMonth, setSelectedMonth] = useState(() => {
		const now = new Date();
		return { year: now.getFullYear(), month: now.getMonth() + 1 };
	});

	const { data: events, isLoading, isError } = useDowntimeEvents(100, 0);
	const { data: activeEvents } = useActiveDowntime();
	const { data: summary } = useUptimeSummary();
	const { data: monthlyReport } = useMonthlyUptimeReport(
		selectedMonth.year,
		selectedMonth.month,
	);
	const resolveEvent = useResolveDowntimeEvent();

	const filteredEvents = events?.filter((event) => {
		const matchesStatus =
			statusFilter === 'all' ||
			(statusFilter === 'active' && !event.ended_at) ||
			(statusFilter === 'resolved' && event.ended_at);
		const matchesSeverity =
			severityFilter === 'all' || event.severity === severityFilter;
		const matchesComponent =
			componentFilter === 'all' || event.component_type === componentFilter;
		return matchesStatus && matchesSeverity && matchesComponent;
	});

	const activeCount = activeEvents?.length ?? 0;
	const resolvedCount = events?.filter((e) => e.ended_at).length ?? 0;

	const handleResolve = (id: string) => {
		resolveEvent.mutate({ id });
	};

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Downtime History
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						Track and display historical outages and uptime statistics
					</p>
				</div>
				<div className="flex items-center gap-2">
					<button
						type="button"
						onClick={() => setViewMode('list')}
						className={`px-3 py-2 text-sm font-medium rounded-lg transition-colors ${
							viewMode === 'list'
								? 'bg-indigo-600 text-white'
								: 'bg-white text-gray-700 border border-gray-300 hover:bg-gray-50'
						}`}
					>
						List View
					</button>
					<button
						type="button"
						onClick={() => setViewMode('timeline')}
						className={`px-3 py-2 text-sm font-medium rounded-lg transition-colors ${
							viewMode === 'timeline'
								? 'bg-indigo-600 text-white'
								: 'bg-white text-gray-700 border border-gray-300 hover:bg-gray-50'
						}`}
					>
						Timeline
					</button>
				</div>
			</div>

			{/* Uptime Badges */}
			<div className="grid grid-cols-1 md:grid-cols-4 gap-4">
				<UptimeBadge
					percent={summary?.overall_uptime_7d ?? 100}
					label="7-Day Uptime"
				/>
				<UptimeBadge
					percent={summary?.overall_uptime_30d ?? 100}
					label="30-Day Uptime"
				/>
				<UptimeBadge
					percent={summary?.overall_uptime_90d ?? 100}
					label="90-Day Uptime"
				/>
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
					<div className="flex items-center gap-3">
						<div
							className={`w-3 h-3 ${activeCount > 0 ? 'bg-red-500 animate-pulse' : 'bg-green-500'} rounded-full`}
						/>
						<div>
							<div className="text-2xl font-bold text-gray-900 dark:text-white">
								{activeCount}
							</div>
							<div className="text-sm text-gray-500 dark:text-gray-400">
								Active Incidents
							</div>
						</div>
					</div>
				</div>
			</div>

			{/* Summary Stats */}
			<div className="grid grid-cols-1 md:grid-cols-3 gap-4">
				<button
					type="button"
					onClick={() =>
						setStatusFilter(statusFilter === 'active' ? 'all' : 'active')
					}
					className={`p-4 rounded-lg border transition-colors ${
						statusFilter === 'active'
							? 'bg-red-50 border-red-200'
							: 'bg-white border-gray-200 hover:bg-gray-50'
					}`}
				>
					<div className="flex items-center gap-3">
						<div className="p-2 bg-red-100 rounded-lg">
							<svg
								aria-hidden="true"
								className="w-5 h-5 text-red-600"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
								/>
							</svg>
						</div>
						<div className="text-left">
							<div className="text-2xl font-bold text-gray-900 dark:text-white">
								{activeCount}
							</div>
							<div className="text-sm text-gray-500 dark:text-gray-400">
								Active Outages
							</div>
						</div>
					</div>
				</button>

				<button
					type="button"
					onClick={() =>
						setStatusFilter(statusFilter === 'resolved' ? 'all' : 'resolved')
					}
					className={`p-4 rounded-lg border transition-colors ${
						statusFilter === 'resolved'
							? 'bg-green-50 border-green-200'
							: 'bg-white border-gray-200 hover:bg-gray-50'
					}`}
				>
					<div className="flex items-center gap-3">
						<div className="p-2 bg-green-100 rounded-lg">
							<svg
								aria-hidden="true"
								className="w-5 h-5 text-green-600"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M5 13l4 4L19 7"
								/>
							</svg>
						</div>
						<div className="text-left">
							<div className="text-2xl font-bold text-gray-900 dark:text-white">
								{resolvedCount}
							</div>
							<div className="text-sm text-gray-500 dark:text-gray-400">
								Resolved
							</div>
						</div>
					</div>
				</button>

				<div className="p-4 rounded-lg border border-gray-200 bg-white">
					<div className="flex items-center gap-3">
						<div className="p-2 bg-blue-100 rounded-lg">
							<svg
								aria-hidden="true"
								className="w-5 h-5 text-blue-600"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
								/>
							</svg>
						</div>
						<div className="text-left">
							<div className="text-2xl font-bold text-gray-900 dark:text-white">
								{summary?.total_components ?? 0}
							</div>
							<div className="text-sm text-gray-500 dark:text-gray-400">
								Monitored Components
							</div>
						</div>
					</div>
				</div>
			</div>

			{/* Monthly Report Section */}
			{monthlyReport && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
						Monthly Report - {monthlyReport.month}
					</h2>
					<div className="grid grid-cols-1 md:grid-cols-3 gap-6">
						<div>
							<div className="text-3xl font-bold text-gray-900 dark:text-white">
								{monthlyReport.overall_uptime.toFixed(2)}%
							</div>
							<div className="text-sm text-gray-500 dark:text-gray-400">
								Overall Uptime
							</div>
						</div>
						<div>
							<div className="text-3xl font-bold text-gray-900 dark:text-white">
								{formatDuration(monthlyReport.total_downtime_seconds)}
							</div>
							<div className="text-sm text-gray-500 dark:text-gray-400">
								Total Downtime
							</div>
						</div>
						<div>
							<div className="text-3xl font-bold text-gray-900 dark:text-white">
								{monthlyReport.incident_count}
							</div>
							<div className="text-sm text-gray-500 dark:text-gray-400">
								Incidents
							</div>
						</div>
					</div>
				</div>
			)}

			{/* Events List/Timeline */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="p-4 border-b border-gray-200 dark:border-gray-700">
					<div className="flex flex-wrap items-center gap-4">
						<select
							value={statusFilter}
							onChange={(e) =>
								setStatusFilter(e.target.value as 'all' | 'active' | 'resolved')
							}
							className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="all">All Status</option>
							<option value="active">Active</option>
							<option value="resolved">Resolved</option>
						</select>
						<select
							value={severityFilter}
							onChange={(e) =>
								setSeverityFilter(e.target.value as DowntimeSeverity | 'all')
							}
							className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="all">All Severity</option>
							<option value="critical">Critical</option>
							<option value="warning">Warning</option>
							<option value="info">Info</option>
						</select>
						<select
							value={componentFilter}
							onChange={(e) =>
								setComponentFilter(e.target.value as ComponentType | 'all')
							}
							className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="all">All Components</option>
							<option value="agent">Agent</option>
							<option value="server">Server</option>
							<option value="repository">Repository</option>
							<option value="service">Service</option>
						</select>
					</div>
				</div>

				<div className="p-4">
					{isError ? (
						<div className="py-12 text-center text-red-500 dark:text-red-400">
							<p className="font-medium">Failed to load downtime events</p>
							<p className="text-sm">Please try refreshing the page</p>
						</div>
					) : isLoading ? (
						<div className="space-y-4">
							<LoadingCard />
							<LoadingCard />
							<LoadingCard />
						</div>
					) : filteredEvents && filteredEvents.length > 0 ? (
						viewMode === 'timeline' ? (
							<TimelineView events={filteredEvents} />
						) : (
							<div className="space-y-4">
								{filteredEvents.map((event) => (
									<DowntimeEventCard
										key={event.id}
										event={event}
										onResolve={handleResolve}
										isProcessing={resolveEvent.isPending}
									/>
								))}
							</div>
						)
					) : (
						<div className="py-12 text-center text-gray-500 dark:text-gray-400">
							<svg
								aria-hidden="true"
								className="w-16 h-16 mx-auto mb-4 text-gray-300"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
								/>
							</svg>
							<h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
								{statusFilter === 'all' &&
								severityFilter === 'all' &&
								componentFilter === 'all'
									? 'No downtime events'
									: 'No matching events'}
							</h3>
							<p>
								{statusFilter === 'all' &&
								severityFilter === 'all' &&
								componentFilter === 'all'
									? 'All systems are running smoothly'
									: 'Try adjusting your filters'}
							</p>
						</div>
					)}
				</div>
			</div>
		</div>
	);
}
