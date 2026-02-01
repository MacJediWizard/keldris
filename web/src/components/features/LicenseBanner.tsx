import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useLicensePurchaseUrl, useLicenseWarnings } from '../../hooks/useLicenses';
import type { LicenseLimitsWarning } from '../../lib/types';

function formatLimitType(type: LicenseLimitsWarning['type']): string {
	switch (type) {
		case 'agents':
			return 'agents';
		case 'repositories':
			return 'repositories';
		case 'storage':
			return 'storage';
		default:
			return type;
	}
}

export function LicenseBanner() {
	const { data: warningsData, isLoading } = useLicenseWarnings();
	const { data: purchaseUrlData } = useLicensePurchaseUrl();
	const [dismissedExpiry, setDismissedExpiry] = useState(false);
	const [dismissedLimits, setDismissedLimits] = useState<string[]>([]);

	if (isLoading) {
		return null;
	}

	const warnings = warningsData?.warnings;
	if (!warnings) {
		return null;
	}

	const { expiration, limits } = warnings;
	const purchaseUrl = purchaseUrlData?.url;

	// Check for expired license (blocking modal)
	if (expiration?.is_expired && !expiration.is_in_grace_period) {
		return (
			<div className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center z-50">
				<div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-lg w-full mx-4 p-6">
					<div className="flex items-center gap-3 text-red-600 dark:text-red-400 mb-4">
						<svg
							className="w-8 h-8"
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
						<h2 className="text-xl font-bold">License Expired</h2>
					</div>

					<p className="text-gray-600 dark:text-gray-400 mb-6">
						Your license has expired and the grace period has ended. Please renew
						your license to continue using all features.
					</p>

					<div className="flex flex-col sm:flex-row gap-3">
						{purchaseUrl && (
							<a
								href={purchaseUrl}
								target="_blank"
								rel="noopener noreferrer"
								className="flex-1 px-4 py-2 bg-red-600 text-white text-center rounded-lg hover:bg-red-700 transition-colors"
							>
								Renew License
							</a>
						)}
						<Link
							to="/organization/license"
							className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 text-center rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
						>
							Enter License Key
						</Link>
					</div>
				</div>
			</div>
		);
	}

	// Grace period banner (urgent but dismissible)
	if (expiration?.is_in_grace_period && !dismissedExpiry) {
		const daysRemaining = expiration.grace_period_ends_at
			? Math.ceil(
					(new Date(expiration.grace_period_ends_at).getTime() - Date.now()) /
						(1000 * 60 * 60 * 24),
				)
			: 0;

		return (
			<div className="bg-orange-50 dark:bg-orange-900/30 border-b border-orange-200 dark:border-orange-800">
				<div className="max-w-7xl mx-auto py-3 px-4 sm:px-6 lg:px-8">
					<div className="flex items-center justify-between flex-wrap">
						<div className="flex-1 flex items-center">
							<span className="flex p-2 rounded-lg bg-orange-100 dark:bg-orange-800">
								<svg
									className="w-5 h-5 text-orange-600 dark:text-orange-300"
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
							</span>
							<p className="ml-3 font-medium text-orange-700 dark:text-orange-200">
								<span className="font-bold">License Expired - Grace Period Active.</span>{' '}
								{daysRemaining > 0
									? `You have ${daysRemaining} ${daysRemaining === 1 ? 'day' : 'days'} remaining to renew.`
									: 'Grace period ends today.'}
							</p>
						</div>
						<div className="flex gap-2">
							{purchaseUrl && (
								<a
									href={purchaseUrl}
									target="_blank"
									rel="noopener noreferrer"
									className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-orange-600 hover:bg-orange-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-orange-500"
								>
									Renew Now
								</a>
							)}
							<button
								type="button"
								onClick={() => setDismissedExpiry(true)}
								className="inline-flex items-center px-4 py-2 text-sm font-medium text-orange-700 dark:text-orange-200 hover:text-orange-900 dark:hover:text-orange-100"
							>
								Dismiss
							</button>
						</div>
					</div>
				</div>
			</div>
		);
	}

	// Expiring soon banner
	if (
		expiration &&
		!expiration.is_expired &&
		expiration.days_until_expiry <= 30 &&
		!dismissedExpiry
	) {
		return (
			<div className="bg-amber-50 dark:bg-amber-900/30 border-b border-amber-200 dark:border-amber-800">
				<div className="max-w-7xl mx-auto py-3 px-4 sm:px-6 lg:px-8">
					<div className="flex items-center justify-between flex-wrap">
						<div className="flex-1 flex items-center">
							<span className="flex p-2 rounded-lg bg-amber-100 dark:bg-amber-800">
								<svg
									className="w-5 h-5 text-amber-600 dark:text-amber-300"
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
							</span>
							<p className="ml-3 font-medium text-amber-700 dark:text-amber-200">
								Your license will expire in {expiration.days_until_expiry}{' '}
								{expiration.days_until_expiry === 1 ? 'day' : 'days'}.
							</p>
						</div>
						<div className="flex gap-2">
							{purchaseUrl && (
								<a
									href={purchaseUrl}
									target="_blank"
									rel="noopener noreferrer"
									className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-amber-700 bg-amber-100 hover:bg-amber-200 dark:bg-amber-800 dark:text-amber-200 dark:hover:bg-amber-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-amber-500"
								>
									Renew Now
								</a>
							)}
							<button
								type="button"
								onClick={() => setDismissedExpiry(true)}
								className="inline-flex items-center px-4 py-2 text-sm font-medium text-amber-700 dark:text-amber-200 hover:text-amber-900 dark:hover:text-amber-100"
							>
								Dismiss
							</button>
						</div>
					</div>
				</div>
			</div>
		);
	}

	// Limits warning banner (show critical limits approaching)
	const criticalLimits = limits.filter(
		(l) => l.percentage >= 80 && !dismissedLimits.includes(l.type),
	);

	if (criticalLimits.length > 0) {
		const mostCritical = criticalLimits.reduce((prev, curr) =>
			curr.percentage > prev.percentage ? curr : prev,
		);
		const isCritical = mostCritical.percentage >= 95;

		return (
			<div
				className={`${
					isCritical
						? 'bg-red-50 dark:bg-red-900/30 border-b border-red-200 dark:border-red-800'
						: 'bg-amber-50 dark:bg-amber-900/30 border-b border-amber-200 dark:border-amber-800'
				}`}
			>
				<div className="max-w-7xl mx-auto py-3 px-4 sm:px-6 lg:px-8">
					<div className="flex items-center justify-between flex-wrap">
						<div className="flex-1 flex items-center">
							<span
								className={`flex p-2 rounded-lg ${
									isCritical
										? 'bg-red-100 dark:bg-red-800'
										: 'bg-amber-100 dark:bg-amber-800'
								}`}
							>
								<svg
									className={`w-5 h-5 ${
										isCritical
											? 'text-red-600 dark:text-red-300'
											: 'text-amber-600 dark:text-amber-300'
									}`}
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
									aria-hidden="true"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M13 10V3L4 14h7v7l9-11h-7z"
									/>
								</svg>
							</span>
							<p
								className={`ml-3 font-medium ${
									isCritical
										? 'text-red-700 dark:text-red-200'
										: 'text-amber-700 dark:text-amber-200'
								}`}
							>
								{isCritical ? (
									<span className="font-bold">Limit Reached! </span>
								) : null}
								You're using {Math.round(mostCritical.percentage)}% of your{' '}
								{formatLimitType(mostCritical.type)} limit ({mostCritical.current} /{' '}
								{mostCritical.limit}).
								{criticalLimits.length > 1 && (
									<span>
										{' '}
										+{criticalLimits.length - 1} more{' '}
										{criticalLimits.length - 1 === 1 ? 'limit' : 'limits'} approaching.
									</span>
								)}
							</p>
						</div>
						<div className="flex gap-2">
							{purchaseUrl && (
								<a
									href={purchaseUrl}
									target="_blank"
									rel="noopener noreferrer"
									className={`inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md ${
										isCritical
											? 'text-white bg-red-600 hover:bg-red-700'
											: 'text-amber-700 bg-amber-100 hover:bg-amber-200 dark:bg-amber-800 dark:text-amber-200 dark:hover:bg-amber-700'
									} focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-amber-500`}
								>
									Upgrade Plan
								</a>
							)}
							<button
								type="button"
								onClick={() =>
									setDismissedLimits([
										...dismissedLimits,
										...criticalLimits.map((l) => l.type),
									])
								}
								className={`inline-flex items-center px-4 py-2 text-sm font-medium ${
									isCritical
										? 'text-red-700 dark:text-red-200 hover:text-red-900 dark:hover:text-red-100'
										: 'text-amber-700 dark:text-amber-200 hover:text-amber-900 dark:hover:text-amber-100'
								}`}
							>
								Dismiss
							</button>
						</div>
					</div>
				</div>
			</div>
		);
	}

	return null;
}
