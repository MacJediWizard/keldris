import type { StorageTierType } from '../lib/types';

interface StorageTierBadgeProps {
	tier: StorageTierType;
	ageDays?: number;
	showAge?: boolean;
	size?: 'sm' | 'md';
}

const tierConfig: Record<
	StorageTierType,
	{ bg: string; text: string; border: string; icon: string; label: string }
> = {
	hot: {
		bg: 'bg-red-50 dark:bg-red-900/20',
		text: 'text-red-700 dark:text-red-400',
		border: 'border-red-200 dark:border-red-800',
		icon: 'flame',
		label: 'Hot',
	},
	warm: {
		bg: 'bg-orange-50 dark:bg-orange-900/20',
		text: 'text-orange-700 dark:text-orange-400',
		border: 'border-orange-200 dark:border-orange-800',
		icon: 'sun',
		label: 'Warm',
	},
	cold: {
		bg: 'bg-blue-50 dark:bg-blue-900/20',
		text: 'text-blue-700 dark:text-blue-400',
		border: 'border-blue-200 dark:border-blue-800',
		icon: 'snowflake',
		label: 'Cold',
	},
	archive: {
		bg: 'bg-gray-50 dark:bg-gray-900/20',
		text: 'text-gray-700 dark:text-gray-400',
		border: 'border-gray-200 dark:border-gray-700',
		icon: 'archive',
		label: 'Archive',
	},
};

function TierIcon({ tier, className }: { tier: StorageTierType; className?: string }) {
	switch (tier) {
		case 'hot':
			return (
				<svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
					<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17.657 18.657A8 8 0 016.343 7.343S7 9 9 10c0-2 .5-5 2.986-7C14 5 16.09 5.777 17.656 7.343A7.975 7.975 0 0120 13a7.975 7.975 0 01-2.343 5.657z" />
					<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.879 16.121A3 3 0 1012.015 11L11 14H9c0 .768.293 1.536.879 2.121z" />
				</svg>
			);
		case 'warm':
			return (
				<svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
					<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
				</svg>
			);
		case 'cold':
			return (
				<svg className={className} fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
					<path d="M12 2l1.09 3.26L16 6l-2.91.74L12 10l-1.09-3.26L8 6l2.91-.74L12 2zM4.18 8.18l2.26 1.16L7.6 12l-1.16 2.66-2.26 1.16L6.74 13H4v-2h2.74l-2.56 2.82zM19.82 8.18l-2.26 1.16L16.4 12l1.16 2.66 2.26 1.16L17.26 13H20v-2h-2.74l2.56 2.82zM12 14l-1.09 3.26L8 18l2.91.74L12 22l1.09-3.26L16 18l-2.91-.74L12 14z" />
				</svg>
			);
		case 'archive':
			return (
				<svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
					<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
				</svg>
			);
	}
}

export function StorageTierBadge({
	tier,
	ageDays,
	showAge = false,
	size = 'sm',
}: StorageTierBadgeProps) {
	const config = tierConfig[tier] || tierConfig.hot;
	const sizeClasses = size === 'sm' ? 'text-xs px-2 py-0.5' : 'text-sm px-2.5 py-1';
	const iconSize = size === 'sm' ? 'w-3 h-3' : 'w-4 h-4';

	return (
		<span
			className={`inline-flex items-center font-medium rounded-full border ${config.bg} ${config.text} ${config.border} ${sizeClasses}`}
			title={`${config.label} storage tier${ageDays !== undefined ? ` - ${ageDays} days old` : ''}`}
		>
			<TierIcon tier={tier} className={`${iconSize} mr-1`} />
			{config.label}
			{showAge && ageDays !== undefined && (
				<span className="ml-1 opacity-75">({ageDays}d)</span>
			)}
		</span>
	);
}

interface StorageTierSelectProps {
	value: StorageTierType;
	onChange: (tier: StorageTierType) => void;
	disabled?: boolean;
	id?: string;
	excludeTiers?: StorageTierType[];
}

export function StorageTierSelect({
	value,
	onChange,
	disabled = false,
	id,
	excludeTiers = [],
}: StorageTierSelectProps) {
	const allTiers: StorageTierType[] = ['hot', 'warm', 'cold', 'archive'];
	const availableTiers = allTiers.filter((t) => !excludeTiers.includes(t));

	return (
		<select
			id={id}
			value={value}
			onChange={(e) => onChange(e.target.value as StorageTierType)}
			disabled={disabled}
			className="block w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-white focus:border-indigo-500 focus:ring-indigo-500 disabled:bg-gray-100 dark:disabled:bg-gray-800"
		>
			{availableTiers.map((tier) => (
				<option key={tier} value={tier}>
					{tierConfig[tier].label}
				</option>
			))}
		</select>
	);
}

interface TierCostIndicatorProps {
	tier: StorageTierType;
	monthlyCost: number;
	size?: 'sm' | 'md';
}

export function TierCostIndicator({
	tier,
	monthlyCost,
	size = 'sm',
}: TierCostIndicatorProps) {
	const sizeClasses = size === 'sm' ? 'text-xs' : 'text-sm';

	const formatCost = (cost: number): string => {
		if (cost < 0.01) return '<$0.01';
		return `$${cost.toFixed(2)}`;
	};

	return (
		<div className={`flex items-center gap-2 ${sizeClasses}`}>
			<StorageTierBadge tier={tier} size={size} />
			<span className="text-gray-500 dark:text-gray-400">
				{formatCost(monthlyCost)}/mo
			</span>
		</div>
	);
}

interface ColdRestoreStatusBadgeProps {
	status: string;
	estimatedReadyAt?: string;
	size?: 'sm' | 'md';
}

const restoreStatusConfig: Record<
	string,
	{ bg: string; text: string; label: string }
> = {
	pending: {
		bg: 'bg-gray-100 dark:bg-gray-800',
		text: 'text-gray-700 dark:text-gray-400',
		label: 'Pending',
	},
	warming: {
		bg: 'bg-yellow-100 dark:bg-yellow-900/30',
		text: 'text-yellow-700 dark:text-yellow-400',
		label: 'Warming',
	},
	ready: {
		bg: 'bg-green-100 dark:bg-green-900/30',
		text: 'text-green-700 dark:text-green-400',
		label: 'Ready',
	},
	restoring: {
		bg: 'bg-blue-100 dark:bg-blue-900/30',
		text: 'text-blue-700 dark:text-blue-400',
		label: 'Restoring',
	},
	completed: {
		bg: 'bg-green-100 dark:bg-green-900/30',
		text: 'text-green-700 dark:text-green-400',
		label: 'Completed',
	},
	failed: {
		bg: 'bg-red-100 dark:bg-red-900/30',
		text: 'text-red-700 dark:text-red-400',
		label: 'Failed',
	},
	expired: {
		bg: 'bg-gray-100 dark:bg-gray-800',
		text: 'text-gray-500 dark:text-gray-500',
		label: 'Expired',
	},
};

export function ColdRestoreStatusBadge({
	status,
	estimatedReadyAt,
	size = 'sm',
}: ColdRestoreStatusBadgeProps) {
	const config = restoreStatusConfig[status] || restoreStatusConfig.pending;
	const sizeClasses = size === 'sm' ? 'text-xs px-2 py-0.5' : 'text-sm px-2.5 py-1';

	const formatTime = (dateStr: string): string => {
		const date = new Date(dateStr);
		const now = new Date();
		const diffMs = date.getTime() - now.getTime();
		const diffMins = Math.round(diffMs / 60000);

		if (diffMins < 60) return `${diffMins}m`;
		const diffHours = Math.round(diffMins / 60);
		if (diffHours < 24) return `${diffHours}h`;
		const diffDays = Math.round(diffHours / 24);
		return `${diffDays}d`;
	};

	return (
		<span
			className={`inline-flex items-center gap-1.5 font-medium rounded-full ${config.bg} ${config.text} ${sizeClasses}`}
		>
			{status === 'warming' && (
				<span className="w-1.5 h-1.5 rounded-full bg-yellow-500 animate-pulse" />
			)}
			{config.label}
			{status === 'warming' && estimatedReadyAt && (
				<span className="opacity-75">({formatTime(estimatedReadyAt)})</span>
			)}
		</span>
	);
}
