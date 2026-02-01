import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';
import {
	useCurrentLicense,
	useLicensePurchaseUrl,
} from '../../hooks/useLicenses';
import type { ProFeature } from '../../lib/types';

interface FeatureGateProps {
	feature: ProFeature;
	children: ReactNode;
	fallback?: ReactNode;
}

const featureLabels: Record<ProFeature, string> = {
	sso: 'Single Sign-On',
	api_access: 'API Access',
	advanced_reporting: 'Advanced Reporting',
	custom_branding: 'Custom Branding',
	priority_support: 'Priority Support',
	backup_hooks: 'Backup Hooks',
	multi_destination: 'Multi-Destination Backups',
	unlimited_agents: 'Unlimited Agents',
	unlimited_repositories: 'Unlimited Repositories',
	unlimited_storage: 'Unlimited Storage',
};

const featureToLicenseKey: Record<ProFeature, string> = {
	sso: 'sso_enabled',
	api_access: 'api_access',
	advanced_reporting: 'advanced_reporting',
	custom_branding: 'custom_branding',
	priority_support: 'priority_support',
	backup_hooks: 'backup_hooks',
	multi_destination: 'multi_destination',
	unlimited_agents: 'max_agents',
	unlimited_repositories: 'max_repositories',
	unlimited_storage: 'max_storage_bytes',
};

function isFeatureEnabled(
	feature: ProFeature,
	features?: Record<string, boolean | number>,
): boolean {
	if (!features) return false;

	const key = featureToLicenseKey[feature];
	const value = features[key as keyof typeof features];

	// For boolean features
	if (typeof value === 'boolean') {
		return value;
	}

	// For limit-based features (unlimited means 0 or very high)
	if (typeof value === 'number') {
		// 0 or negative typically means unlimited
		return value <= 0 || value >= 999999;
	}

	return false;
}

export function FeatureGate({ feature, children, fallback }: FeatureGateProps) {
	const { data: license, isLoading } = useCurrentLicense();
	const { data: purchaseUrlData } = useLicensePurchaseUrl();

	if (isLoading) {
		return <>{children}</>;
	}

	const hasFeature = isFeatureEnabled(
		feature,
		license?.features as unknown as Record<string, boolean | number>,
	);

	if (hasFeature) {
		return <>{children}</>;
	}

	if (fallback) {
		return <>{fallback}</>;
	}

	// Default fallback: upgrade prompt
	return (
		<div className="relative">
			<div className="opacity-50 pointer-events-none select-none">
				{children}
			</div>
			<div className="absolute inset-0 flex items-center justify-center bg-gray-900/5 dark:bg-gray-900/20 rounded-lg">
				<div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-4 max-w-sm text-center">
					<div className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-indigo-100 text-indigo-700 dark:bg-indigo-900 dark:text-indigo-200 mb-3">
						Pro Feature
					</div>
					<h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
						{featureLabels[feature]}
					</h3>
					<p className="text-xs text-gray-500 dark:text-gray-400 mb-4">
						Upgrade your plan to access this feature.
					</p>
					<div className="flex flex-col gap-2">
						{purchaseUrlData?.url && (
							<a
								href={purchaseUrlData.url}
								target="_blank"
								rel="noopener noreferrer"
								className="px-3 py-1.5 bg-indigo-600 text-white text-sm rounded-lg hover:bg-indigo-700 transition-colors"
							>
								Upgrade Plan
							</a>
						)}
						<Link
							to="/organization/license"
							className="px-3 py-1.5 text-indigo-600 dark:text-indigo-400 text-sm hover:underline"
						>
							View License
						</Link>
					</div>
				</div>
			</div>
		</div>
	);
}

interface ProBadgeProps {
	feature: ProFeature;
	showLabel?: boolean;
	size?: 'sm' | 'md';
}

export function ProBadge({
	feature,
	showLabel = false,
	size = 'sm',
}: ProBadgeProps) {
	const { data: license, isLoading } = useCurrentLicense();

	if (isLoading) {
		return null;
	}

	const hasFeature = isFeatureEnabled(
		feature,
		license?.features as unknown as Record<string, boolean | number>,
	);

	if (hasFeature) {
		return null;
	}

	const sizeClasses =
		size === 'sm' ? 'px-1.5 py-0.5 text-xs' : 'px-2 py-0.5 text-sm';

	return (
		<span
			className={`inline-flex items-center rounded font-medium bg-indigo-100 text-indigo-700 dark:bg-indigo-900 dark:text-indigo-200 ${sizeClasses}`}
			title={`${featureLabels[feature]} - Pro feature`}
		>
			{showLabel ? featureLabels[feature] : 'Pro'}
		</span>
	);
}

interface FeatureDisabledButtonProps {
	feature: ProFeature;
	children: ReactNode;
	className?: string;
	onClick?: () => void;
}

export function FeatureDisabledButton({
	feature,
	children,
	className = '',
	onClick,
}: FeatureDisabledButtonProps) {
	const { data: license, isLoading } = useCurrentLicense();
	const { data: purchaseUrlData } = useLicensePurchaseUrl();

	const hasFeature = isFeatureEnabled(
		feature,
		license?.features as unknown as Record<string, boolean | number>,
	);

	if (isLoading || hasFeature) {
		return (
			<button type="button" onClick={onClick} className={className}>
				{children}
			</button>
		);
	}

	return (
		<div className="relative inline-block group">
			<button
				type="button"
				disabled
				className={`${className} opacity-50 cursor-not-allowed`}
				title={`${featureLabels[feature]} requires a Pro plan`}
			>
				{children}
				<span className="ml-1.5 inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-indigo-100 text-indigo-700 dark:bg-indigo-900 dark:text-indigo-200">
					Pro
				</span>
			</button>
			<div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 hidden group-hover:block z-10">
				<div className="bg-gray-900 text-white text-xs rounded-lg py-2 px-3 whitespace-nowrap">
					<p className="font-medium">{featureLabels[feature]}</p>
					<p className="text-gray-300 mt-1">Requires Pro plan</p>
					{purchaseUrlData?.url && (
						<a
							href={purchaseUrlData.url}
							target="_blank"
							rel="noopener noreferrer"
							className="block mt-2 text-indigo-300 hover:text-indigo-200"
						>
							Upgrade now
						</a>
					)}
					<div className="absolute top-full left-1/2 -translate-x-1/2 border-4 border-transparent border-t-gray-900" />
				</div>
			</div>
		</div>
	);
}

interface FeatureCheckProps {
	feature: ProFeature;
	children: (hasFeature: boolean, featureLabel: string) => ReactNode;
}

export function FeatureCheck({ feature, children }: FeatureCheckProps) {
	const { data: license, isLoading } = useCurrentLicense();

	if (isLoading) {
		return <>{children(false, featureLabels[feature])}</>;
	}

	const hasFeature = isFeatureEnabled(
		feature,
		license?.features as unknown as Record<string, boolean | number>,
	);

	return <>{children(hasFeature, featureLabels[feature])}</>;
}
