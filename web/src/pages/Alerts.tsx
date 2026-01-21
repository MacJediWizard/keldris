import { useState } from 'react';
import {
	useAcknowledgeAlert,
	useAlerts,
	useResolveAlert,
} from '../hooks/useAlerts';
import type { Alert, AlertStatus } from '../lib/types';
import {
	formatDate,
	formatDateTime,
	getAlertSeverityColor,
	getAlertStatusColor,
	getAlertTypeLabel,
} from '../lib/utils';

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

interface AlertCardProps {
	alert: Alert;
	onAcknowledge: (id: string) => void;
	onResolve: (id: string) => void;
	isProcessing: boolean;
}

function AlertCard({
	alert,
	onAcknowledge,
	onResolve,
	isProcessing,
}: AlertCardProps) {
	const severityColor = getAlertSeverityColor(alert.severity);
	const statusColor = getAlertStatusColor(alert.status);

	return (
		<div
			className={`bg-white rounded-lg border ${severityColor.border} p-4 hover:shadow-sm transition-shadow`}
		>
			<div className="flex items-start gap-4">
				<div
					className={`flex-shrink-0 w-10 h-10 ${severityColor.bg} rounded-full flex items-center justify-center`}
				>
					{alert.severity === 'critical' ? (
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
					) : alert.severity === 'warning' ? (
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
							{alert.title}
						</h3>
						<span
							className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}
						>
							<span className={`w-1.5 h-1.5 ${statusColor.dot} rounded-full`} />
							{alert.status}
						</span>
					</div>
					<p className="text-sm text-gray-600 dark:text-gray-400 mb-2">{alert.message}</p>
					<div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400 dark:text-gray-400">
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
							{getAlertTypeLabel(alert.type)}
						</span>
						<span title={formatDateTime(alert.created_at)}>
							{formatDate(alert.created_at)}
						</span>
						{alert.acknowledged_at && (
							<span
								title={`Acknowledged: ${formatDateTime(alert.acknowledged_at)}`}
							>
								Ack'd {formatDate(alert.acknowledged_at)}
							</span>
						)}
						{alert.resolved_at && (
							<span title={`Resolved: ${formatDateTime(alert.resolved_at)}`}>
								Resolved {formatDate(alert.resolved_at)}
							</span>
						)}
					</div>
				</div>
				{alert.status !== 'resolved' && (
					<div className="flex-shrink-0 flex items-center gap-2">
						{alert.status === 'active' && (
							<button
								type="button"
								onClick={() => onAcknowledge(alert.id)}
								disabled={isProcessing}
								className="px-3 py-1.5 text-sm font-medium text-yellow-700 bg-yellow-100 rounded-lg hover:bg-yellow-200 transition-colors disabled:opacity-50"
							>
								Acknowledge
							</button>
						)}
						<button
							type="button"
							onClick={() => onResolve(alert.id)}
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

export function Alerts() {
	const [statusFilter, setStatusFilter] = useState<AlertStatus | 'all'>('all');
	const [severityFilter, setSeverityFilter] = useState<string>('all');

	const { data: alerts, isLoading, isError } = useAlerts();
	const acknowledgeAlert = useAcknowledgeAlert();
	const resolveAlert = useResolveAlert();

	const filteredAlerts = alerts?.filter((alert) => {
		const matchesStatus =
			statusFilter === 'all' || alert.status === statusFilter;
		const matchesSeverity =
			severityFilter === 'all' || alert.severity === severityFilter;
		return matchesStatus && matchesSeverity;
	});

	const activeCount = alerts?.filter((a) => a.status === 'active').length ?? 0;
	const acknowledgedCount =
		alerts?.filter((a) => a.status === 'acknowledged').length ?? 0;
	const resolvedCount =
		alerts?.filter((a) => a.status === 'resolved').length ?? 0;

	const handleAcknowledge = (id: string) => {
		acknowledgeAlert.mutate(id);
	};

	const handleResolve = (id: string) => {
		resolveAlert.mutate(id);
	};

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">Alerts</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">Monitor and manage system alerts</p>
				</div>
			</div>

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
							<div className="text-sm text-gray-500 dark:text-gray-400">Active</div>
						</div>
					</div>
				</button>

				<button
					type="button"
					onClick={() =>
						setStatusFilter(
							statusFilter === 'acknowledged' ? 'all' : 'acknowledged',
						)
					}
					className={`p-4 rounded-lg border transition-colors ${
						statusFilter === 'acknowledged'
							? 'bg-yellow-50 border-yellow-200'
							: 'bg-white border-gray-200 hover:bg-gray-50'
					}`}
				>
					<div className="flex items-center gap-3">
						<div className="p-2 bg-yellow-100 rounded-lg">
							<svg
								aria-hidden="true"
								className="w-5 h-5 text-yellow-600"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
								/>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
								/>
							</svg>
						</div>
						<div className="text-left">
							<div className="text-2xl font-bold text-gray-900 dark:text-white">
								{acknowledgedCount}
							</div>
							<div className="text-sm text-gray-500 dark:text-gray-400">Acknowledged</div>
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
							<div className="text-sm text-gray-500 dark:text-gray-400">Resolved</div>
						</div>
					</div>
				</button>
			</div>

			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="p-4 border-b border-gray-200 dark:border-gray-700">
					<div className="flex items-center gap-4">
						<select
							value={statusFilter}
							onChange={(e) =>
								setStatusFilter(e.target.value as AlertStatus | 'all')
							}
							className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="all">All Status</option>
							<option value="active">Active</option>
							<option value="acknowledged">Acknowledged</option>
							<option value="resolved">Resolved</option>
						</select>
						<select
							value={severityFilter}
							onChange={(e) => setSeverityFilter(e.target.value)}
							className="px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
						>
							<option value="all">All Severity</option>
							<option value="critical">Critical</option>
							<option value="warning">Warning</option>
							<option value="info">Info</option>
						</select>
					</div>
				</div>

				<div className="p-4">
					{isError ? (
						<div className="py-12 text-center text-red-500 dark:text-red-400 dark:text-red-400">
							<p className="font-medium">Failed to load alerts</p>
							<p className="text-sm">Please try refreshing the page</p>
						</div>
					) : isLoading ? (
						<div className="space-y-4">
							<LoadingCard />
							<LoadingCard />
							<LoadingCard />
						</div>
					) : filteredAlerts && filteredAlerts.length > 0 ? (
						<div className="space-y-4">
							{filteredAlerts.map((alert) => (
								<AlertCard
									key={alert.id}
									alert={alert}
									onAcknowledge={handleAcknowledge}
									onResolve={handleResolve}
									isProcessing={
										acknowledgeAlert.isPending || resolveAlert.isPending
									}
								/>
							))}
						</div>
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
								{statusFilter === 'all' && severityFilter === 'all'
									? 'No alerts'
									: 'No matching alerts'}
							</h3>
							<p>
								{statusFilter === 'all' && severityFilter === 'all'
									? 'Everything is running smoothly'
									: 'Try adjusting your filters'}
							</p>
						</div>
					)}
				</div>
			</div>
		</div>
	);
}
