import { useState } from 'react';
import { Link } from 'react-router-dom';
import {
	useCostAlerts,
	useCostForecast,
	useCostSummary,
	useCreateCostAlert,
	useDeleteCostAlert,
	useUpdateCostAlert,
} from '../hooks/useCostEstimation';
import type { CostAlert, CreateCostAlertRequest } from '../lib/types';
import {
	formatBytes,
	formatCurrency,
	formatDate,
	getRepositoryTypeBadge,
} from '../lib/utils';

interface StatCardProps {
	title: string;
	value: string;
	subtitle: string;
	icon: React.ReactNode;
	isLoading?: boolean;
	valueClassName?: string;
}

function StatCard({
	title,
	value,
	subtitle,
	icon,
	isLoading,
	valueClassName = 'text-gray-900',
}: StatCardProps) {
	return (
		<div className="bg-white rounded-lg border border-gray-200 p-6">
			<div className="flex items-center justify-between">
				<div>
					<p className="text-sm font-medium text-gray-600">{title}</p>
					<p className={`text-2xl font-bold mt-1 ${valueClassName}`}>
						{isLoading ? (
							<span className="inline-block w-16 h-7 bg-gray-200 rounded animate-pulse" />
						) : (
							value
						)}
					</p>
					<p className="text-sm text-gray-500 mt-1">{subtitle}</p>
				</div>
				<div className="p-3 bg-indigo-50 rounded-lg text-indigo-600">
					{icon}
				</div>
			</div>
		</div>
	);
}

function CostForecastChart() {
	const { data, isLoading } = useCostForecast(30);

	if (isLoading) {
		return (
			<div className="h-48 flex items-center justify-center">
				<div className="animate-pulse text-gray-400">Loading forecast...</div>
			</div>
		);
	}

	if (!data?.forecasts || data.forecasts.length === 0) {
		return (
			<div className="h-48 flex items-center justify-center text-gray-500">
				<div className="text-center">
					<svg
						aria-hidden="true"
						className="w-12 h-12 mx-auto mb-3 text-gray-300"
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
					<p>Insufficient data for forecasting</p>
					<p className="text-sm">Need more historical data points</p>
				</div>
			</div>
		);
	}

	const maxCost = Math.max(
		data.current_monthly_cost,
		...data.forecasts.map((f) => f.projected_cost),
	);

	return (
		<div className="space-y-4">
			<div className="flex items-center justify-between text-sm text-gray-600 mb-2">
				<span>Current: {formatCurrency(data.current_monthly_cost)}/mo</span>
				<span>
					Growth Rate: {((data.monthly_growth_rate || 0) * 100).toFixed(1)}%/mo
				</span>
			</div>
			<div className="space-y-3">
				{data.forecasts.map((forecast) => {
					const widthPercent =
						maxCost > 0 ? (forecast.projected_cost / maxCost) * 100 : 0;
					return (
						<div key={forecast.period}>
							<div className="flex justify-between text-sm mb-1">
								<span className="font-medium">{forecast.period}</span>
								<span className="text-gray-600">
									{formatCurrency(forecast.projected_cost)}/mo
								</span>
							</div>
							<div className="h-4 bg-gray-100 rounded-full overflow-hidden">
								<div
									className="h-full bg-indigo-500 rounded-full transition-all duration-500"
									style={{ width: `${widthPercent}%` }}
								/>
							</div>
							<div className="text-xs text-gray-500 mt-1">
								{forecast.projected_size_gb.toFixed(1)} GB projected
							</div>
						</div>
					);
				})}
			</div>
		</div>
	);
}

function CostBreakdownByType({ byType }: { byType: Record<string, number> }) {
	const entries = Object.entries(byType).filter(([, cost]) => cost > 0);

	if (entries.length === 0) {
		return <p className="text-gray-500 text-sm">No cost data available</p>;
	}

	const total = entries.reduce((sum, [, cost]) => sum + cost, 0);

	return (
		<div className="space-y-3">
			{entries.map(([type, cost]) => {
				const badge = getRepositoryTypeBadge(type);
				const percent = total > 0 ? (cost / total) * 100 : 0;
				return (
					<div key={type} className="flex items-center justify-between">
						<div className="flex items-center gap-2">
							<span
								className={`px-2 py-0.5 rounded text-xs font-medium ${badge.className}`}
							>
								{badge.label}
							</span>
						</div>
						<div className="flex items-center gap-4">
							<div className="w-24 h-2 bg-gray-100 rounded-full overflow-hidden">
								<div
									className="h-full bg-indigo-500 rounded-full"
									style={{ width: `${percent}%` }}
								/>
							</div>
							<span className="text-sm font-medium text-gray-900 w-20 text-right">
								{formatCurrency(cost)}
							</span>
						</div>
					</div>
				);
			})}
		</div>
	);
}

function CostAlertForm({
	alert,
	onSave,
	onCancel,
}: {
	alert?: CostAlert;
	onSave: (data: CreateCostAlertRequest) => void;
	onCancel: () => void;
}) {
	const [name, setName] = useState(alert?.name || '');
	const [threshold, setThreshold] = useState(
		alert?.monthly_threshold?.toString() || '100',
	);
	const [enabled, setEnabled] = useState(alert?.enabled ?? true);
	const [notifyOnExceed, setNotifyOnExceed] = useState(
		alert?.notify_on_exceed ?? true,
	);
	const [notifyOnForecast, setNotifyOnForecast] = useState(
		alert?.notify_on_forecast ?? false,
	);
	const [forecastMonths, setForecastMonths] = useState(
		alert?.forecast_months?.toString() || '3',
	);

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();
		onSave({
			name,
			monthly_threshold: Number.parseFloat(threshold),
			enabled,
			notify_on_exceed: notifyOnExceed,
			notify_on_forecast: notifyOnForecast,
			forecast_months: Number.parseInt(forecastMonths, 10),
		});
	};

	return (
		<form onSubmit={handleSubmit} className="space-y-4">
			<div>
				<label
					htmlFor="alert-name"
					className="block text-sm font-medium text-gray-700 mb-1"
				>
					Alert Name
				</label>
				<input
					id="alert-name"
					type="text"
					value={name}
					onChange={(e) => setName(e.target.value)}
					className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
					placeholder="e.g., Monthly Cost Limit"
					required
				/>
			</div>
			<div>
				<label
					htmlFor="monthly-threshold"
					className="block text-sm font-medium text-gray-700 mb-1"
				>
					Monthly Threshold ($)
				</label>
				<input
					id="monthly-threshold"
					type="number"
					value={threshold}
					onChange={(e) => setThreshold(e.target.value)}
					className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
					min="0"
					step="0.01"
					required
				/>
			</div>
			<div className="space-y-2">
				<label className="flex items-center gap-2">
					<input
						type="checkbox"
						checked={enabled}
						onChange={(e) => setEnabled(e.target.checked)}
						className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
					/>
					<span className="text-sm text-gray-700">Enabled</span>
				</label>
				<label className="flex items-center gap-2">
					<input
						type="checkbox"
						checked={notifyOnExceed}
						onChange={(e) => setNotifyOnExceed(e.target.checked)}
						className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
					/>
					<span className="text-sm text-gray-700">
						Notify when current cost exceeds threshold
					</span>
				</label>
				<label className="flex items-center gap-2">
					<input
						type="checkbox"
						checked={notifyOnForecast}
						onChange={(e) => setNotifyOnForecast(e.target.checked)}
						className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
					/>
					<span className="text-sm text-gray-700">
						Notify when forecasted cost exceeds threshold
					</span>
				</label>
			</div>
			{notifyOnForecast && (
				<div>
					<label
						htmlFor="forecast-period"
						className="block text-sm font-medium text-gray-700 mb-1"
					>
						Forecast Period (months)
					</label>
					<select
						id="forecast-period"
						value={forecastMonths}
						onChange={(e) => setForecastMonths(e.target.value)}
						className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
					>
						<option value="3">3 months</option>
						<option value="6">6 months</option>
						<option value="12">12 months</option>
					</select>
				</div>
			)}
			<div className="flex justify-end gap-2 pt-4">
				<button
					type="button"
					onClick={onCancel}
					className="px-4 py-2 text-gray-700 border border-gray-300 rounded-md hover:bg-gray-50"
				>
					Cancel
				</button>
				<button
					type="submit"
					className="px-4 py-2 bg-indigo-600 text-white rounded-md hover:bg-indigo-700"
				>
					{alert ? 'Update' : 'Create'} Alert
				</button>
			</div>
		</form>
	);
}

function CostAlertsSection() {
	const { data: alerts, isLoading } = useCostAlerts();
	const createAlert = useCreateCostAlert();
	const updateAlert = useUpdateCostAlert();
	const deleteAlert = useDeleteCostAlert();
	const [showForm, setShowForm] = useState(false);
	const [editingAlert, setEditingAlert] = useState<CostAlert | null>(null);

	const handleSave = (data: CreateCostAlertRequest) => {
		if (editingAlert) {
			updateAlert.mutate(
				{ id: editingAlert.id, data },
				{
					onSuccess: () => {
						setEditingAlert(null);
						setShowForm(false);
					},
				},
			);
		} else {
			createAlert.mutate(data, {
				onSuccess: () => setShowForm(false),
			});
		}
	};

	if (showForm || editingAlert) {
		return (
			<div className="bg-white rounded-lg border border-gray-200 p-6">
				<h2 className="text-lg font-semibold text-gray-900 mb-4">
					{editingAlert ? 'Edit Cost Alert' : 'New Cost Alert'}
				</h2>
				<CostAlertForm
					alert={editingAlert || undefined}
					onSave={handleSave}
					onCancel={() => {
						setShowForm(false);
						setEditingAlert(null);
					}}
				/>
			</div>
		);
	}

	return (
		<div className="bg-white rounded-lg border border-gray-200 p-6">
			<div className="flex items-center justify-between mb-4">
				<h2 className="text-lg font-semibold text-gray-900">Cost Alerts</h2>
				<button
					type="button"
					onClick={() => setShowForm(true)}
					className="px-3 py-1.5 text-sm bg-indigo-600 text-white rounded-md hover:bg-indigo-700"
				>
					Add Alert
				</button>
			</div>
			{isLoading ? (
				<div className="space-y-3">
					{[1, 2].map((i) => (
						<div key={i} className="h-16 bg-gray-100 rounded animate-pulse" />
					))}
				</div>
			) : !alerts || alerts.length === 0 ? (
				<div className="text-center py-8 text-gray-500">
					<svg
						aria-hidden="true"
						className="w-12 h-12 mx-auto mb-3 text-gray-300"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
						/>
					</svg>
					<p>No cost alerts configured</p>
					<p className="text-sm">
						Set up alerts to get notified when costs exceed thresholds
					</p>
				</div>
			) : (
				<div className="space-y-3">
					{alerts.map((alert) => (
						<div
							key={alert.id}
							className="flex items-center justify-between p-4 bg-gray-50 rounded-lg"
						>
							<div>
								<div className="flex items-center gap-2">
									<span className="font-medium text-gray-900">
										{alert.name}
									</span>
									<span
										className={`px-2 py-0.5 text-xs rounded-full ${
											alert.enabled
												? 'bg-green-100 text-green-800'
												: 'bg-gray-100 text-gray-600'
										}`}
									>
										{alert.enabled ? 'Active' : 'Disabled'}
									</span>
								</div>
								<p className="text-sm text-gray-500 mt-1">
									Alert when cost exceeds{' '}
									{formatCurrency(alert.monthly_threshold)}/mo
									{alert.notify_on_forecast &&
										` or forecast exceeds in ${alert.forecast_months} months`}
								</p>
								{alert.last_triggered_at && (
									<p className="text-xs text-gray-400 mt-1">
										Last triggered: {formatDate(alert.last_triggered_at)}
									</p>
								)}
							</div>
							<div className="flex items-center gap-2">
								<button
									type="button"
									onClick={() => setEditingAlert(alert)}
									className="p-1.5 text-gray-400 hover:text-gray-600 rounded"
									aria-label="Edit alert"
								>
									<svg
										aria-hidden="true"
										className="w-4 h-4"
										fill="none"
										stroke="currentColor"
										viewBox="0 0 24 24"
									>
										<path
											strokeLinecap="round"
											strokeLinejoin="round"
											strokeWidth={2}
											d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
										/>
									</svg>
								</button>
								<button
									type="button"
									onClick={() => {
										if (
											confirm('Are you sure you want to delete this alert?')
										) {
											deleteAlert.mutate(alert.id);
										}
									}}
									className="p-1.5 text-gray-400 hover:text-red-600 rounded"
									aria-label="Delete alert"
								>
									<svg
										aria-hidden="true"
										className="w-4 h-4"
										fill="none"
										stroke="currentColor"
										viewBox="0 0 24 24"
									>
										<path
											strokeLinecap="round"
											strokeLinejoin="round"
											strokeWidth={2}
											d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
										/>
									</svg>
								</button>
							</div>
						</div>
					))}
				</div>
			)}
		</div>
	);
}

export function CostEstimation() {
	const { data: summary, isLoading: summaryLoading } = useCostSummary();

	return (
		<div className="space-y-6">
			<div>
				<h1 className="text-2xl font-bold text-gray-900">Cost Estimation</h1>
				<p className="text-gray-600 mt-1">
					Monitor and forecast your cloud storage costs
				</p>
			</div>

			<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
				<StatCard
					title="Monthly Cost"
					value={formatCurrency(summary?.total_monthly_cost)}
					subtitle="Current estimated cost"
					isLoading={summaryLoading}
					valueClassName="text-indigo-600"
					icon={
						<svg
							aria-hidden="true"
							className="w-6 h-6"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
					}
				/>
				<StatCard
					title="Yearly Cost"
					value={formatCurrency(summary?.total_yearly_cost)}
					subtitle="Projected annual cost"
					isLoading={summaryLoading}
					icon={
						<svg
							aria-hidden="true"
							className="w-6 h-6"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
							/>
						</svg>
					}
				/>
				<StatCard
					title="Total Storage"
					value={`${(summary?.total_storage_size_gb || 0).toFixed(1)} GB`}
					subtitle="Across all repositories"
					isLoading={summaryLoading}
					icon={
						<svg
							aria-hidden="true"
							className="w-6 h-6"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"
							/>
						</svg>
					}
				/>
				<StatCard
					title="Repositories"
					value={String(summary?.repository_count || 0)}
					subtitle="Tracked repositories"
					isLoading={summaryLoading}
					icon={
						<svg
							aria-hidden="true"
							className="w-6 h-6"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"
							/>
						</svg>
					}
				/>
			</div>

			<div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<div className="flex items-center justify-between mb-4">
						<h2 className="text-lg font-semibold text-gray-900">
							Cost Forecast
						</h2>
					</div>
					<CostForecastChart />
				</div>

				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<h2 className="text-lg font-semibold text-gray-900 mb-4">
						Cost by Storage Type
					</h2>
					{summaryLoading ? (
						<div className="space-y-3">
							{[1, 2, 3].map((i) => (
								<div
									key={i}
									className="h-8 bg-gray-100 rounded animate-pulse"
								/>
							))}
						</div>
					) : (
						<CostBreakdownByType byType={summary?.by_type || {}} />
					)}
				</div>
			</div>

			<div className="bg-white rounded-lg border border-gray-200 p-6">
				<div className="flex items-center justify-between mb-4">
					<h2 className="text-lg font-semibold text-gray-900">
						Cost per Repository
					</h2>
					<Link
						to="/stats"
						className="text-sm text-indigo-600 hover:text-indigo-800"
					>
						View Storage Stats
					</Link>
				</div>
				{summaryLoading ? (
					<div className="space-y-3">
						{[1, 2, 3].map((i) => (
							<div key={i} className="h-16 bg-gray-100 rounded animate-pulse" />
						))}
					</div>
				) : !summary?.repositories || summary.repositories.length === 0 ? (
					<div className="text-center py-8 text-gray-500">
						<svg
							aria-hidden="true"
							className="w-12 h-12 mx-auto mb-3 text-gray-300"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"
							/>
						</svg>
						<p>No repositories with cost data</p>
						<p className="text-sm">
							Create repositories and run backups to see cost estimates
						</p>
					</div>
				) : (
					<div className="overflow-x-auto">
						<table className="min-w-full">
							<thead>
								<tr className="border-b border-gray-200">
									<th className="text-left py-3 px-4 text-sm font-medium text-gray-500">
										Repository
									</th>
									<th className="text-left py-3 px-4 text-sm font-medium text-gray-500">
										Type
									</th>
									<th className="text-right py-3 px-4 text-sm font-medium text-gray-500">
										Storage
									</th>
									<th className="text-right py-3 px-4 text-sm font-medium text-gray-500">
										Cost/GB
									</th>
									<th className="text-right py-3 px-4 text-sm font-medium text-gray-500">
										Monthly
									</th>
									<th className="text-right py-3 px-4 text-sm font-medium text-gray-500">
										Yearly
									</th>
								</tr>
							</thead>
							<tbody>
								{summary.repositories.map((repo) => {
									const typeBadge = getRepositoryTypeBadge(
										repo.repository_type,
									);
									return (
										<tr
											key={repo.repository_id}
											className="border-b border-gray-100 hover:bg-gray-50"
										>
											<td className="py-3 px-4">
												<span className="font-medium text-gray-900">
													{repo.repository_name}
												</span>
											</td>
											<td className="py-3 px-4">
												<span
													className={`px-2 py-0.5 rounded text-xs font-medium ${typeBadge.className}`}
												>
													{typeBadge.label}
												</span>
											</td>
											<td className="py-3 px-4 text-right text-gray-600">
												{formatBytes(repo.storage_size_bytes)}
											</td>
											<td className="py-3 px-4 text-right text-gray-600">
												{formatCurrency(repo.cost_per_gb)}
											</td>
											<td className="py-3 px-4 text-right font-medium text-gray-900">
												{formatCurrency(repo.monthly_cost)}
											</td>
											<td className="py-3 px-4 text-right text-gray-600">
												{formatCurrency(repo.yearly_cost)}
											</td>
										</tr>
									);
								})}
							</tbody>
							<tfoot>
								<tr className="bg-gray-50">
									<td
										colSpan={4}
										className="py-3 px-4 font-medium text-gray-900"
									>
										Total
									</td>
									<td className="py-3 px-4 text-right font-bold text-indigo-600">
										{formatCurrency(summary.total_monthly_cost)}
									</td>
									<td className="py-3 px-4 text-right font-medium text-gray-900">
										{formatCurrency(summary.total_yearly_cost)}
									</td>
								</tr>
							</tfoot>
						</table>
					</div>
				)}
			</div>

			<CostAlertsSection />
		</div>
	);
}
