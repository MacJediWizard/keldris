/**
 * Shared skeleton loading card.
 *
 * Provides several built-in variants that match the existing inline patterns
 * used across the codebase. If none fit, pass `children` for a custom layout
 * (the outer card wrapper + animate-pulse is still applied).
 *
 * Variants:
 *   "stat"     – label / big value / small subtitle  (default)
 *   "stat-sm"  – label / value only (no subtitle)
 *   "alert"    – circle icon + multi-line text
 *   "template" – icon block + title/description + body lines
 *   "repo"     – header row with badge + subtitle
 *   "sla"      – title + 3-column grid of tall bars
 *   "health"   – label / value / subtitle (same as stat but different spacing)
 */

export type LoadingCardVariant =
	| 'stat'
	| 'stat-sm'
	| 'alert'
	| 'template'
	| 'repo'
	| 'sla'
	| 'health';

interface LoadingCardProps {
	variant?: LoadingCardVariant;
	className?: string;
	children?: React.ReactNode;
}

export function LoadingCard({
	variant = 'stat',
	className,
	children,
}: LoadingCardProps) {
	const base =
		'bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 animate-pulse';
	const wrapperClass = className ? `${base} ${className}` : base;

	if (children) {
		return <div className={wrapperClass}>{children}</div>;
	}

	switch (variant) {
		case 'stat':
			return (
				<div className={`${wrapperClass} p-6`}>
					<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded mb-2" />
					<div className="h-8 w-32 bg-gray-200 dark:bg-gray-700 rounded mb-1" />
					<div className="h-3 w-20 bg-gray-100 dark:bg-gray-700 rounded" />
				</div>
			);

		case 'stat-sm':
			return (
				<div className={`${wrapperClass} p-6`}>
					<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded mb-2" />
					<div className="h-8 w-16 bg-gray-200 dark:bg-gray-700 rounded" />
				</div>
			);

		case 'alert':
			return (
				<div className={`${wrapperClass} p-4`}>
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

		case 'template':
			return (
				<div
					className={`bg-white dark:bg-gray-800 rounded-lg shadow animate-pulse p-6 ${className ?? ''}`}
				>
					<div className="flex items-center gap-4 mb-4">
						<div className="w-12 h-12 bg-gray-200 dark:bg-gray-700 rounded-lg" />
						<div>
							<div className="h-5 w-32 bg-gray-200 dark:bg-gray-700 rounded mb-2" />
							<div className="h-3 w-48 bg-gray-200 dark:bg-gray-700 rounded" />
						</div>
					</div>
					<div className="h-4 w-full bg-gray-200 dark:bg-gray-700 rounded mb-2" />
					<div className="h-4 w-3/4 bg-gray-200 dark:bg-gray-700 rounded" />
				</div>
			);

		case 'repo':
			return (
				<div className={`${wrapperClass} p-6`}>
					<div className="flex items-start justify-between mb-4">
						<div className="h-5 w-32 bg-gray-200 dark:bg-gray-700 rounded" />
						<div className="h-6 w-12 bg-gray-200 dark:bg-gray-700 rounded-full" />
					</div>
					<div className="h-4 w-24 bg-gray-100 dark:bg-gray-700 rounded" />
				</div>
			);

		case 'sla':
			return (
				<div className={`${wrapperClass} p-6`}>
					<div className="h-5 w-3/4 bg-gray-200 dark:bg-gray-700 rounded mb-4" />
					<div className="grid grid-cols-3 gap-4">
						<div className="h-12 bg-gray-200 dark:bg-gray-700 rounded" />
						<div className="h-12 bg-gray-200 dark:bg-gray-700 rounded" />
						<div className="h-12 bg-gray-200 dark:bg-gray-700 rounded" />
					</div>
				</div>
			);

		case 'health':
			return (
				<div className={`${wrapperClass} p-4`}>
					<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded" />
					<div className="mt-2 h-8 w-32 bg-gray-200 dark:bg-gray-700 rounded" />
					<div className="mt-1 h-4 w-20 bg-gray-200 dark:bg-gray-700 rounded" />
				</div>
			);

		default:
			return (
				<div className={`${wrapperClass} p-6`}>
					<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded mb-2" />
					<div className="h-8 w-32 bg-gray-200 dark:bg-gray-700 rounded mb-1" />
					<div className="h-3 w-20 bg-gray-100 dark:bg-gray-700 rounded" />
				</div>
			);
	}
}
