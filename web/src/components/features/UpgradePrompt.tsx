import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import type { LicenseTier } from '../../lib/types';

interface UpgradePromptProps {
	feature: string;
	currentTier: LicenseTier;
	open: boolean;
	onClose: () => void;
}

export function UpgradePrompt({
	feature,
	currentTier,
	open,
	onClose,
}: UpgradePromptProps) {
	const [visible, setVisible] = useState(open);

	useEffect(() => {
		setVisible(open);
	}, [open]);

	if (!visible) return null;

	return (
		<div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
			<div className="w-full max-w-md rounded-lg bg-white p-6 shadow-xl dark:bg-dark-card dark:border dark:border-dark-border">
				<div className="mb-4 flex items-center gap-3">
					<div className="flex h-10 w-10 items-center justify-center rounded-full bg-amber-100 dark:bg-amber-900/30">
						<svg
							aria-hidden="true"
							className="h-5 w-5 text-amber-600 dark:text-amber-400"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
							/>
						</svg>
					</div>
					<h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
						Upgrade Required
					</h3>
				</div>

				<p className="mb-2 text-sm text-gray-600 dark:text-gray-400">
					The <span className="font-medium text-gray-900 dark:text-gray-100">{feature}</span>{' '}
					feature is not available on your current{' '}
					<span className="font-medium capitalize">{currentTier}</span> plan.
				</p>
				<p className="mb-6 text-sm text-gray-500 dark:text-gray-400">
					Upgrade your license to unlock this feature and more.
				</p>

				<div className="flex justify-end gap-3">
					<button
						type="button"
						onClick={onClose}
						className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
					>
						Close
					</button>
					<Link
						to="/license"
						onClick={onClose}
						className="inline-flex items-center rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500"
					>
						Upgrade License
					</Link>
				</div>
			</div>
		</div>
	);
}
import { Link } from 'react-router-dom';
import {
	FEATURE_BENEFITS,
	FEATURE_NAMES,
	FEATURE_REQUIRED_PLAN,
} from '../../hooks/usePlanLimits';
import type { PlanType, UpgradeFeature } from '../../lib/types';
import {
	PLAN_NAMES,
	PLAN_PRICING,
	generateContactSalesLink,
	generateUpgradeLink,
	requiresSales,
} from '../../lib/upgrade';

import { Link } from 'react-router-dom';
import {
	FEATURE_BENEFITS,
	FEATURE_NAMES,
	FEATURE_REQUIRED_PLAN,
} from '../../hooks/usePlanLimits';
import type { PlanType, UpgradeFeature } from '../../lib/types';
import {
	PLAN_NAMES,
	PLAN_PRICING,
	generateContactSalesLink,
	generateUpgradeLink,
	requiresSales,
} from '../../lib/upgrade';

export type UpgradePromptVariant = 'inline' | 'banner' | 'card' | 'modal';

interface UpgradePromptProps {
	feature: UpgradeFeature;
	variant?: UpgradePromptVariant;
	currentPlan?: PlanType;
	source?: string;
	onDismiss?: () => void;
	showBenefits?: boolean;
	className?: string;
}

function UpgradeIcon({ className }: { className?: string }) {
	return (
		<svg
			aria-hidden="true"
			className={className}
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
		>
			<path
				strokeLinecap="round"
				strokeLinejoin="round"
				strokeWidth={2}
				d="M5 10l7-7m0 0l7 7m-7-7v18"
			/>
		</svg>
	);
}

function SparklesIcon({ className }: { className?: string }) {
	return (
		<svg
			aria-hidden="true"
			className={className}
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
		>
			<path
				strokeLinecap="round"
				strokeLinejoin="round"
				strokeWidth={2}
				d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z"
			/>
		</svg>
	);
}

function CloseIcon({ className }: { className?: string }) {
	return (
		<svg
			aria-hidden="true"
			className={className}
			fill="currentColor"
			viewBox="0 0 20 20"
		>
			<path
				fillRule="evenodd"
				d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
				clipRule="evenodd"
			/>
		</svg>
	);
}

function UpgradeButton({
	feature,
	source,
	targetPlan,
	size = 'normal',
}: {
	feature: UpgradeFeature;
	source?: string;
	targetPlan: PlanType;
	size?: 'small' | 'normal';
}) {
	const isSales = requiresSales(targetPlan);
	const link = isSales
		? generateContactSalesLink({ feature, source })
		: generateUpgradeLink({ feature, source, targetPlan });

	const sizeClasses =
		size === 'small' ? 'px-3 py-1.5 text-sm' : 'px-4 py-2 text-sm font-medium';

	return (
		<Link
			to={link}
			className={`inline-flex items-center gap-2 ${sizeClasses} bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors`}
		>
			{isSales ? (
				<>
					<SparklesIcon className="w-4 h-4" />
					Contact Sales
				</>
			) : (
				<>
					<UpgradeIcon className="w-4 h-4" />
					Upgrade to {PLAN_NAMES[targetPlan]}
				</>
			)}
		</Link>
	);
}

export function UpgradePromptInline({
	feature,
	source,
	className = '',
}: Omit<UpgradePromptProps, 'variant'>) {
	const targetPlan = FEATURE_REQUIRED_PLAN[feature];
	const featureName = FEATURE_NAMES[feature];

	return (
		<span
			className={`inline-flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400 ${className}`}
		>
			<SparklesIcon className="w-4 h-4 text-indigo-500" />
			<span>
				{featureName} requires{' '}
				<Link
					to={generateUpgradeLink({ feature, source, targetPlan })}
					className="text-indigo-600 dark:text-indigo-400 hover:underline font-medium"
				>
					{PLAN_NAMES[targetPlan]}
				</Link>
			</span>
		</span>
	);
}

export function UpgradePromptBanner({
	feature,
	source,
	onDismiss,
	className = '',
}: Omit<UpgradePromptProps, 'variant'>) {
	const targetPlan = FEATURE_REQUIRED_PLAN[feature];
	const featureName = FEATURE_NAMES[feature];

	return (
		<div
			className={`flex items-center justify-between gap-4 px-4 py-3 bg-indigo-50 dark:bg-indigo-900/20 border border-indigo-200 dark:border-indigo-800 rounded-lg ${className}`}
		>
			<div className="flex items-center gap-3">
				<div className="p-2 bg-indigo-100 dark:bg-indigo-800 rounded-full">
					<SparklesIcon className="w-5 h-5 text-indigo-600 dark:text-indigo-400" />
				</div>
				<div>
					<p className="text-sm font-medium text-gray-900 dark:text-white">
						Unlock {featureName}
					</p>
					<p className="text-sm text-gray-600 dark:text-gray-400">
						Upgrade to {PLAN_NAMES[targetPlan]} to access this feature
					</p>
				</div>
			</div>
			<div className="flex items-center gap-2">
				<UpgradeButton
					feature={feature}
					source={source}
					targetPlan={targetPlan}
					size="small"
				/>
				{onDismiss && (
					<button
						type="button"
						onClick={onDismiss}
						className="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 rounded-lg"
						aria-label="Dismiss"
					>
						<CloseIcon className="w-5 h-5" />
					</button>
				)}
			</div>
		</div>
	);
}

export function UpgradePromptCard({
	feature,
	source,
	showBenefits = true,
	className = '',
}: Omit<UpgradePromptProps, 'variant'>) {
	const targetPlan = FEATURE_REQUIRED_PLAN[feature];
	const featureName = FEATURE_NAMES[feature];
	const benefits = FEATURE_BENEFITS[feature];
	const isSales = requiresSales(targetPlan);

	return (
		<div
			className={`bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 shadow-sm overflow-hidden ${className}`}
		>
			<div className="p-6">
				<div className="flex items-start gap-4">
					<div className="p-3 bg-gradient-to-br from-indigo-500 to-purple-500 rounded-lg">
						<SparklesIcon className="w-6 h-6 text-white" />
					</div>
					<div className="flex-1">
						<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
							{featureName}
						</h3>
						<p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
							Available on {PLAN_NAMES[targetPlan]}
							{!isSales && ` (${PLAN_PRICING[targetPlan]})`}
						</p>
					</div>
				</div>

				{showBenefits && benefits.length > 0 && (
					<ul className="mt-4 space-y-2">
						{benefits.map((benefit) => (
							<li
								key={benefit}
								className="flex items-start gap-2 text-sm text-gray-600 dark:text-gray-400"
							>
								<svg
									aria-hidden="true"
									className="w-5 h-5 text-green-500 flex-shrink-0 mt-0.5"
									fill="currentColor"
									viewBox="0 0 20 20"
								>
									<path
										fillRule="evenodd"
										d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
										clipRule="evenodd"
									/>
								</svg>
								{benefit}
							</li>
						))}
					</ul>
				)}

				<div className="mt-6">
					<UpgradeButton
						feature={feature}
						source={source}
						targetPlan={targetPlan}
					/>
				</div>
			</div>
		</div>
	);
}

export function UpgradePromptModal({
	feature,
	source,
	onDismiss,
	showBenefits = true,
}: Omit<UpgradePromptProps, 'variant'>) {
	const targetPlan = FEATURE_REQUIRED_PLAN[feature];
	const featureName = FEATURE_NAMES[feature];
	const benefits = FEATURE_BENEFITS[feature];
	const isSales = requiresSales(targetPlan);

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
				<div className="flex items-start justify-between gap-4 mb-4">
					<div className="flex items-center gap-3">
						<div className="p-3 bg-gradient-to-br from-indigo-500 to-purple-500 rounded-lg">
							<SparklesIcon className="w-6 h-6 text-white" />
						</div>
						<div>
							<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
								Unlock {featureName}
							</h3>
							<p className="text-sm text-gray-600 dark:text-gray-400">
								{PLAN_NAMES[targetPlan]} Plan
								{!isSales && ` - ${PLAN_PRICING[targetPlan]}`}
							</p>
						</div>
					</div>
					{onDismiss && (
						<button
							type="button"
							onClick={onDismiss}
							className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
							aria-label="Close"
						>
							<CloseIcon className="w-5 h-5" />
						</button>
					)}
				</div>

				<p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
					This feature is not available on your current plan. Upgrade to access{' '}
					{featureName.toLowerCase()} and other premium features.
				</p>

				{showBenefits && benefits.length > 0 && (
					<ul className="space-y-2 mb-6">
						{benefits.map((benefit) => (
							<li
								key={benefit}
								className="flex items-start gap-2 text-sm text-gray-600 dark:text-gray-400"
							>
								<svg
									aria-hidden="true"
									className="w-5 h-5 text-green-500 flex-shrink-0 mt-0.5"
									fill="currentColor"
									viewBox="0 0 20 20"
								>
									<path
										fillRule="evenodd"
										d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
										clipRule="evenodd"
									/>
								</svg>
								{benefit}
							</li>
						))}
					</ul>
				)}

				<div className="flex justify-end gap-3">
					{onDismiss && (
						<button
							type="button"
							onClick={onDismiss}
							className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
						>
							Maybe Later
						</button>
					)}
					<UpgradeButton
						feature={feature}
						source={source}
						targetPlan={targetPlan}
					/>
				</div>
			</div>
		</div>
	);
}

export function UpgradeLimitWarning({
	type,
	current,
	limit,
	source,
	className = '',
}: {
	type: 'agents' | 'storage';
	current: number;
	limit: number;
	source?: string;
	className?: string;
}) {
	const percentage = Math.round((current / limit) * 100);
	const isNearLimit = percentage >= 80;
	const isAtLimit = percentage >= 100;

	if (!isNearLimit) return null;

	const formatValue = (value: number) => {
		if (type === 'storage') {
			const gb = value / (1024 * 1024 * 1024);
			return gb >= 1024
				? `${(gb / 1024).toFixed(1)} TB`
				: `${gb.toFixed(1)} GB`;
		}
		return value.toString();
	};

	return (
		<div
			className={`flex items-center justify-between gap-4 px-4 py-3 rounded-lg ${
				isAtLimit
					? 'bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800'
					: 'bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800'
			} ${className}`}
		>
			<div className="flex items-center gap-3">
				<svg
					aria-hidden="true"
					className={`w-5 h-5 ${isAtLimit ? 'text-red-500' : 'text-amber-500'}`}
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
				<div>
					<p
						className={`text-sm font-medium ${isAtLimit ? 'text-red-800 dark:text-red-200' : 'text-amber-800 dark:text-amber-200'}`}
					>
						{isAtLimit
							? `${type === 'agents' ? 'Agent' : 'Storage'} limit reached`
							: `Approaching ${type === 'agents' ? 'agent' : 'storage'} limit`}
					</p>
					<p
						className={`text-sm ${isAtLimit ? 'text-red-600 dark:text-red-400' : 'text-amber-600 dark:text-amber-400'}`}
					>
						{formatValue(current)} / {formatValue(limit)} ({percentage}% used)
					</p>
				</div>
			</div>
			<UpgradeButton
				feature={type}
				source={source}
				targetPlan="starter"
				size="small"
			/>
		</div>
	);
}

export function UpgradePrompt({
	variant = 'banner',
	...props
}: UpgradePromptProps) {
	switch (variant) {
		case 'inline':
			return <UpgradePromptInline {...props} />;
		case 'banner':
			return <UpgradePromptBanner {...props} />;
		case 'card':
			return <UpgradePromptCard {...props} />;
		case 'modal':
			return <UpgradePromptModal {...props} />;
		default:
			return <UpgradePromptBanner {...props} />;
	}
}
