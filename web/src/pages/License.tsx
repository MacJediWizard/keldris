import { useRef, useState } from 'react';
import { TierBadge } from '../components/features/TierBadge';
import { LoadingSpinner } from '../components/ui/LoadingSpinner';
import { useLicense, useActivateLicense, useDeactivateLicense, usePricingPlans, useStartTrial } from '../hooks/useLicense';

function formatDate(dateStr: string): string {
	if (!dateStr) return 'N/A';
	return new Date(dateStr).toLocaleDateString(undefined, {
		year: 'numeric',
		month: 'long',
		day: 'numeric',
	});
}

function formatLimit(value: number): string {
	if (value <= 0) return 'Unlimited';
	return value.toLocaleString();
}

function formatPrice(cents: number): string {
	return `$${(cents / 100).toFixed(2)}`;
}

export default function License() {
	const { data: license, isLoading, error } = useLicense();
	const { data: plans } = usePricingPlans();
	const activateMutation = useActivateLicense();
	const deactivateMutation = useDeactivateLicense();
	const startTrialMutation = useStartTrial();
	const [licenseKey, setLicenseKey] = useState('');
	const [activateError, setActivateError] = useState('');
	const [trialEmail, setTrialEmail] = useState('');
	const [trialError, setTrialError] = useState('');
	const activateFormRef = useRef<HTMLDivElement>(null);

	if (isLoading) return <LoadingSpinner />;

	if (error) {
		return (
			<div className="rounded-lg border border-red-200 bg-red-50 p-6 dark:border-red-800 dark:bg-red-900/20">
				<p className="text-red-700 dark:text-red-400">Failed to load license information.</p>
			</div>
		);
	}

	if (!license) return null;

	const isExpired =
		license.expires_at && new Date(license.expires_at) < new Date();
	const canManageFromGUI = license.license_key_source !== 'env';
	const showActivateForm = license.license_key_source === 'none' || license.tier === 'free';
	const showTrialStart = license.tier === 'free' && license.license_key_source === 'none' && !license.is_trial;
	const isTrialExpired = license.is_trial && isExpired;

	const handleStartTrial = async () => {
		setTrialError('');
		try {
			await startTrialMutation.mutateAsync({ email: trialEmail, tier: 'pro' });
			setTrialEmail('');
		} catch (err) {
			setTrialError(err instanceof Error ? err.message : 'Failed to start trial');
import { useState } from 'react';
import { useMe } from '../hooks/useAuth';
import {
	useActivateLicense,
	useCurrentLicense,
	useLicenseHistory,
	useLicensePurchaseUrl,
	useValidateLicense,
} from '../hooks/useLicenses';
import type {
	LicenseHistory,
	LicenseTier,
	OrgRole,
	ProFeature,
} from '../lib/types';

import { useState } from 'react';
import { useMe } from '../hooks/useAuth';
import {
	useActivateLicense,
	useCurrentLicense,
	useLicenseHistory,
	useLicensePurchaseUrl,
	useValidateLicense,
} from '../hooks/useLicenses';
import type {
	LicenseHistory,
	LicenseTier,
	OrgRole,
	ProFeature,
} from '../lib/types';

function formatBytes(bytes: number): string {
	if (bytes === 0) return '0 B';
	const k = 1024;
	const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
	const i = Math.floor(Math.log(bytes) / Math.log(k));
	return `${Number.parseFloat((bytes / k ** i).toFixed(2))} ${sizes[i]}`;
}

function getTierBadgeColor(tier: LicenseTier): string {
	switch (tier) {
		case 'enterprise':
			return 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-200';
		case 'professional':
			return 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-200';
		default:
			return 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-200';
	}
}

function getStatusBadgeColor(status: string): string {
	switch (status) {
		case 'active':
			return 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-200';
		case 'expiring_soon':
			return 'bg-amber-100 text-amber-700 dark:bg-amber-900 dark:text-amber-200';
		case 'expired':
			return 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-200';
		case 'grace_period':
			return 'bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-200';
		default:
			return 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-200';
	}
}

function UsageBar({
	used,
	limit,
	label,
}: {
	used: number;
	limit: number;
	label: string;
}) {
}: { used: number; limit: number; label: string }) {
	const percentage = limit > 0 ? Math.min((used / limit) * 100, 100) : 0;
	const isWarning = percentage >= 80;
	const isCritical = percentage >= 95;

	return (
		<div>
			<div className="flex justify-between text-sm mb-1">
				<span className="text-gray-600 dark:text-gray-400">{label}</span>
				<span className="font-medium text-gray-900 dark:text-white">
					{used.toLocaleString()} /{' '}
					{limit > 0 ? limit.toLocaleString() : 'Unlimited'}
				</span>
			</div>
			{limit > 0 && (
				<div className="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
					<div
						className={`h-full rounded-full transition-all ${
							isCritical
								? 'bg-red-500'
								: isWarning
									? 'bg-amber-500'
									: 'bg-indigo-500'
						}`}
						style={{ width: `${percentage}%` }}
					/>
				</div>
			)}
		</div>
	);
}

function StorageUsageBar({
	used,
	limit,
	label,
}: {
	used: number;
	limit: number;
	label: string;
}) {
}: { used: number; limit: number; label: string }) {
	const percentage = limit > 0 ? Math.min((used / limit) * 100, 100) : 0;
	const isWarning = percentage >= 80;
	const isCritical = percentage >= 95;

	return (
		<div>
			<div className="flex justify-between text-sm mb-1">
				<span className="text-gray-600 dark:text-gray-400">{label}</span>
				<span className="font-medium text-gray-900 dark:text-white">
					{formatBytes(used)} / {limit > 0 ? formatBytes(limit) : 'Unlimited'}
				</span>
			</div>
			{limit > 0 && (
				<div className="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
					<div
						className={`h-full rounded-full transition-all ${
							isCritical
								? 'bg-red-500'
								: isWarning
									? 'bg-amber-500'
									: 'bg-indigo-500'
						}`}
						style={{ width: `${percentage}%` }}
					/>
				</div>
			)}
		</div>
	);
}

function FeatureItem({
	name,
	enabled,
	proFeature,
}: {
	name: string;
	enabled: boolean;
	proFeature?: ProFeature;
}) {
}: { name: string; enabled: boolean; proFeature?: ProFeature }) {
	return (
		<div className="flex items-center justify-between py-2">
			<div className="flex items-center gap-2">
				<span className="text-gray-700 dark:text-gray-300">{name}</span>
				{proFeature && !enabled && (
					<span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-indigo-100 text-indigo-700 dark:bg-indigo-900 dark:text-indigo-200">
						Pro
					</span>
				)}
			</div>
			{enabled ? (
				<svg
					className="w-5 h-5 text-green-500"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M5 13l4 4L19 7"
					/>
				</svg>
			) : (
				<svg
					className="w-5 h-5 text-gray-400"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M6 18L18 6M6 6l12 12"
					/>
				</svg>
			)}
		</div>
	);
}

function getHistoryActionLabel(action: LicenseHistory['action']): string {
	switch (action) {
		case 'created':
			return 'License Created';
		case 'renewed':
			return 'License Renewed';
		case 'upgraded':
			return 'Plan Upgraded';
		case 'downgraded':
			return 'Plan Downgraded';
		case 'expired':
			return 'License Expired';
		case 'activated':
			return 'License Activated';
		default:
			return action;
	}
}

function getHistoryActionColor(action: LicenseHistory['action']): string {
	switch (action) {
		case 'created':
		case 'activated':
			return 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-200';
		case 'renewed':
		case 'upgraded':
			return 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-200';
		case 'downgraded':
			return 'bg-amber-100 text-amber-700 dark:bg-amber-900 dark:text-amber-200';
		case 'expired':
			return 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-200';
		default:
			return 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-200';
	}
}

export function License() {
	const { data: user } = useMe();
	const { data: license, isLoading } = useCurrentLicense();
	const { data: historyData, isLoading: historyLoading } = useLicenseHistory();
	const { data: purchaseUrlData } = useLicensePurchaseUrl();
	const validateLicense = useValidateLicense();
	const activateLicense = useActivateLicense();

	const [licenseKey, setLicenseKey] = useState('');
	const [showActivateForm, setShowActivateForm] = useState(false);
	const [validationResult, setValidationResult] = useState<{
		valid: boolean;
		tier?: LicenseTier;
		valid_until?: string;
		error?: string;
	} | null>(null);

	const currentUserRole = (user?.current_org_role ?? 'member') as OrgRole;
	const isOwner = currentUserRole === 'owner';
	const canManageLicense = isOwner || currentUserRole === 'admin';

	const handleValidate = async () => {
		if (!licenseKey.trim()) return;
		try {
			const result = await validateLicense.mutateAsync(licenseKey);
			setValidationResult(result);
		} catch {
			setValidationResult({
				valid: false,
				error: 'Failed to validate license key',
			});
		}
	};

	const handleActivate = async () => {
		setActivateError('');
		try {
			await activateMutation.mutateAsync(licenseKey);
			setLicenseKey('');
		} catch (err) {
			setActivateError(err instanceof Error ? err.message : 'Activation failed');
		}
	};

	const handleDeactivate = async () => {
		if (!confirm('Are you sure you want to deactivate your license? You will revert to the Free tier.')) {
			return;
		}
		try {
			await deactivateMutation.mutateAsync();
		if (!licenseKey.trim() || !validationResult?.valid) return;
		try {
			await activateLicense.mutateAsync({ license_key: licenseKey });
			setLicenseKey('');
			setValidationResult(null);
			setShowActivateForm(false);
		} catch {
			// Error handled by mutation
		}
	};

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">License</h1>
				<div className="flex items-center gap-2">
					{license.is_trial && (
						<span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300">
							Trial
						</span>
					)}
					<TierBadge tier={license.tier} className="text-sm px-3 py-1" />
				</div>
			</div>

			{/* Active Trial Banner */}
			{license.is_trial && !isTrialExpired && (
				<div className="rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-900/20">
					<div className="flex items-center justify-between">
						<div>
							<p className="font-medium text-amber-800 dark:text-amber-300">
								Trial &mdash; {license.trial_days_left} day{license.trial_days_left !== 1 ? 's' : ''} remaining
							</p>
							<p className="mt-1 text-sm text-amber-700 dark:text-amber-400">
								Upgrade to keep your {license.tier} features after the trial ends.
							</p>
						</div>
						<button
							type="button"
							onClick={() => activateFormRef.current?.scrollIntoView({ behavior: 'smooth' })}
							className="rounded-md bg-amber-600 px-4 py-2 text-sm font-medium text-white hover:bg-amber-700"
						>
							Upgrade Now
						</button>
					</div>
				</div>
			)}

			{/* Expired Trial Banner */}
			{isTrialExpired && (
				<div className="rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-800 dark:bg-red-900/20">
					<div className="flex items-center justify-between">
						<div>
							<p className="font-medium text-red-800 dark:text-red-300">
								Trial expired
							</p>
							<p className="mt-1 text-sm text-red-700 dark:text-red-400">
								Your trial has ended. Enter a license key to continue with Pro or Enterprise features.
							</p>
						</div>
						<button
							type="button"
							onClick={() => activateFormRef.current?.scrollIntoView({ behavior: 'smooth' })}
							className="rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"
						>
							Upgrade Now
						</button>
					</div>
				</div>
			)}

			{/* Start Free Trial */}
			{showTrialStart && (
				<div className="rounded-lg border border-emerald-200 bg-emerald-50 p-6 dark:border-emerald-800 dark:bg-emerald-900/20">
					<h2 className="text-lg font-semibold text-emerald-900 dark:text-emerald-300">
						Start Free 14-Day Trial
					</h2>
					<p className="mt-1 text-sm text-emerald-700 dark:text-emerald-400">
						Try all Pro features free for 14 days. No credit card required.
					</p>
					<div className="mt-4 flex gap-3">
						<input
							type="email"
							value={trialEmail}
							onChange={(e) => setTrialEmail(e.target.value)}
							placeholder="Enter your email..."
							className="flex-1 rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500 dark:border-gray-600 dark:bg-dark-card dark:text-gray-100"
						/>
						<button
							type="button"
							onClick={handleStartTrial}
							disabled={!trialEmail.trim() || startTrialMutation.isPending}
							className="rounded-md bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-700 disabled:opacity-50 disabled:cursor-not-allowed"
						>
							{startTrialMutation.isPending ? 'Starting...' : 'Start 14-Day Trial'}
						</button>
					</div>
					{trialError && (
						<p className="mt-2 text-sm text-red-600 dark:text-red-400">{trialError}</p>
					)}
				</div>
			)}

			{/* License Key Entry Form */}
			{showActivateForm && (
				<div ref={activateFormRef} className="rounded-lg border border-indigo-200 bg-indigo-50 p-6 dark:border-indigo-800 dark:bg-indigo-900/20">
					<h2 className="text-lg font-semibold text-indigo-900 dark:text-indigo-300">
						Activate License
					</h2>
					<p className="mt-1 text-sm text-indigo-700 dark:text-indigo-400">
						Enter your license key to unlock Pro or Enterprise features.
					</p>
					<div className="mt-4 flex gap-3">
						<input
							type="text"
							value={licenseKey}
							onChange={(e) => setLicenseKey(e.target.value)}
							placeholder="Enter your license key..."
							className="flex-1 rounded-md border border-gray-300 px-3 py-2 text-sm font-mono focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-dark-card dark:text-gray-100"
						/>
						<button
							type="button"
							onClick={handleActivate}
							disabled={!licenseKey.trim() || activateMutation.isPending}
							className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed"
						>
							{activateMutation.isPending ? 'Activating...' : 'Activate'}
						</button>
					</div>
					{activateError && (
						<p className="mt-2 text-sm text-red-600 dark:text-red-400">{activateError}</p>
					)}
				</div>
			)}

			{/* License Overview */}
			<div className="rounded-lg border bg-white p-6 shadow-sm dark:border-dark-border dark:bg-dark-card">
				<div className="flex items-center justify-between mb-4">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
						License Details
					</h2>
					{canManageFromGUI && license.license_key_source === 'database' && license.tier !== 'free' && (
						<button
							type="button"
							onClick={handleDeactivate}
							disabled={deactivateMutation.isPending}
							className="rounded-md border border-red-300 px-3 py-1.5 text-sm font-medium text-red-700 hover:bg-red-50 disabled:opacity-50 dark:border-red-700 dark:text-red-400 dark:hover:bg-red-900/20"
						>
							{deactivateMutation.isPending ? 'Deactivating...' : 'Deactivate'}
						</button>
					)}
				</div>
				<dl className="grid grid-cols-1 gap-4 sm:grid-cols-2">
					<div>
						<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Tier</dt>
						<dd className="mt-1 text-sm text-gray-900 dark:text-gray-100 capitalize">
							{license.tier}
						</dd>
					</div>
					<div>
						<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Customer</dt>
						<dd className="mt-1 text-sm text-gray-900 dark:text-gray-100">
							{license.customer_name || license.customer_id || 'N/A'}
						</dd>
					</div>
					<div>
						<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Issued</dt>
						<dd className="mt-1 text-sm text-gray-900 dark:text-gray-100">
							{formatDate(license.issued_at)}
						</dd>
					</div>
					<div>
						<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Expires</dt>
						<dd
							className={`mt-1 text-sm ${isExpired ? 'text-red-600 font-medium dark:text-red-400' : 'text-gray-900 dark:text-gray-100'}`}
						>
							{formatDate(license.expires_at)}
							{isExpired && ' (Expired)'}
						</dd>
					</div>
					<div>
						<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Key Source</dt>
						<dd className="mt-1 text-sm text-gray-900 dark:text-gray-100 capitalize">
							{license.license_key_source === 'none' ? 'Not configured' : license.license_key_source}
						</dd>
					</div>
				</dl>
			</div>

			{/* Resource Limits */}
			<div className="rounded-lg border bg-white p-6 shadow-sm dark:border-dark-border dark:bg-dark-card">
				<h2 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">
					Resource Limits
				</h2>
				<div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
					<div className="rounded-lg border border-gray-200 p-4 dark:border-dark-border">
						<p className="text-sm font-medium text-gray-500 dark:text-gray-400">Agents</p>
						<p className="mt-1 text-2xl font-semibold text-gray-900 dark:text-gray-100">
							{formatLimit(license.limits.max_agents)}
						</p>
					</div>
					<div className="rounded-lg border border-gray-200 p-4 dark:border-dark-border">
						<p className="text-sm font-medium text-gray-500 dark:text-gray-400">Users</p>
						<p className="mt-1 text-2xl font-semibold text-gray-900 dark:text-gray-100">
							{formatLimit(license.limits.max_users)}
						</p>
					</div>
					<div className="rounded-lg border border-gray-200 p-4 dark:border-dark-border">
						<p className="text-sm font-medium text-gray-500 dark:text-gray-400">Organizations</p>
						<p className="mt-1 text-2xl font-semibold text-gray-900 dark:text-gray-100">
							{formatLimit(license.limits.max_orgs)}
						</p>
					</div>
				</div>
			</div>

			{/* Features */}
			<div className="rounded-lg border bg-white p-6 shadow-sm dark:border-dark-border dark:bg-dark-card">
				<h2 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">
					Included Features
				</h2>
				{license.features.length > 0 ? (
					<div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
						{license.features.map((feature) => (
							<div key={feature} className="flex items-center gap-2">
								<svg
									aria-hidden="true"
									className="h-4 w-4 text-green-500"
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
								<span className="text-sm text-gray-700 dark:text-gray-300 capitalize">
									{feature.replace(/_/g, ' ')}
								</span>
							</div>
						))}
					</div>
				) : (
					<p className="text-sm text-gray-500 dark:text-gray-400">
						No features included in the current plan.
					</p>
				)}
			</div>

			{/* Upgrade Section */}
			{license.tier === 'free' && plans && plans.length > 0 && (
				<div className="rounded-lg border bg-white p-6 shadow-sm dark:border-dark-border dark:bg-dark-card">
					<h2 className="mb-2 text-lg font-semibold text-gray-900 dark:text-gray-100">
						Available Plans
					</h2>
					<p className="mb-4 text-sm text-gray-500 dark:text-gray-400">
						Purchase a plan to get a license key, then enter it above to activate.
					</p>
					<div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
						{plans.map((plan) => (
							<div
								key={plan.id}
								className="flex flex-col rounded-lg border border-gray-200 p-4 dark:border-dark-border"
							>
								<h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 capitalize">
									{plan.name}
								</h3>
								<p className="mt-1 text-2xl font-bold text-indigo-600 dark:text-indigo-400">
									{formatPrice(plan.base_price_cents)}
									<span className="text-sm font-normal text-gray-500 dark:text-gray-400">/mo</span>
								</p>
								<ul className="mt-3 space-y-1 text-sm text-gray-600 dark:text-gray-400">
									<li>{plan.included_agents} agents included</li>
									<li>{plan.included_servers} servers included</li>
									{plan.agent_price_cents > 0 && (
										<li>{formatPrice(plan.agent_price_cents)}/extra agent</li>
									)}
								</ul>
								<button
									type="button"
									onClick={() => activateFormRef.current?.scrollIntoView({ behavior: 'smooth' })}
									className="mt-4 w-full rounded-md border border-indigo-300 px-3 py-2 text-sm font-medium text-indigo-700 hover:bg-indigo-50 dark:border-indigo-600 dark:text-indigo-400 dark:hover:bg-indigo-900/20"
								>
									Enter License Key
								</button>
							</div>
						))}
	const handleCancelActivate = () => {
		setLicenseKey('');
		setValidationResult(null);
		setShowActivateForm(false);
	};

	if (isLoading) {
		return (
			<div className="space-y-6">
				<div>
					<div className="h-8 w-48 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
					<div className="h-4 w-64 bg-gray-200 dark:bg-gray-700 rounded animate-pulse mt-2" />
				</div>
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
					<div className="space-y-4">
						{[1, 2, 3].map((i) => (
							<div
								key={i}
								className="h-16 bg-gray-200 dark:bg-gray-700 rounded animate-pulse"
							/>
						))}
					</div>
				</div>
			</div>
		);
	}

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						License
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-1">
						Manage your organization's license and usage
					</p>
				</div>
				{purchaseUrlData?.url && (
					<a
						href={purchaseUrlData.url}
						target="_blank"
						rel="noopener noreferrer"
						className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors flex items-center gap-2"
					>
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
								d="M3 3h2l.4 2M7 13h10l4-8H5.4M7 13L5.4 5M7 13l-2.293 2.293c-.63.63-.184 1.707.707 1.707H17m0 0a2 2 0 100 4 2 2 0 000-4zm-8 2a2 2 0 11-4 0 2 2 0 014 0z"
							/>
						</svg>
						Upgrade Plan
					</a>
				)}
			</div>

			{/* Current License Info */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						Current License
					</h2>
					{canManageLicense && !showActivateForm && (
						<button
							type="button"
							onClick={() => setShowActivateForm(true)}
							className="text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 text-sm font-medium"
						>
							Enter New License Key
						</button>
					)}
				</div>

				<div className="p-6">
					{license ? (
						<div className="space-y-6">
							<div className="grid grid-cols-1 md:grid-cols-3 gap-6">
								<div>
									<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
										Plan
									</dt>
									<dd className="mt-1">
										<span
											className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium capitalize ${getTierBadgeColor(license.tier)}`}
										>
											{license.tier}
										</span>
									</dd>
								</div>
								<div>
									<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
										Status
									</dt>
									<dd className="mt-1">
										<span
											className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium capitalize ${getStatusBadgeColor(license.status)}`}
										>
											{license.status.replace('_', ' ')}
										</span>
									</dd>
								</div>
								<div>
									<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
										Expires
									</dt>
									<dd className="mt-1 text-sm text-gray-900 dark:text-white">
										{new Date(license.valid_until).toLocaleDateString()}
									</dd>
								</div>
							</div>

							{license.status === 'grace_period' && (
								<div className="bg-orange-50 dark:bg-orange-900/20 border border-orange-200 dark:border-orange-800 rounded-lg p-4">
									<div className="flex items-start gap-3">
										<svg
											className="w-5 h-5 text-orange-600 dark:text-orange-400 mt-0.5"
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
										<div>
											<h3 className="font-medium text-orange-800 dark:text-orange-200">
												Grace Period Active
											</h3>
											<p className="text-sm text-orange-700 dark:text-orange-300 mt-1">
												Your license has expired but you have{' '}
												{license.grace_period_days} days remaining in the grace
												period. Please renew to avoid service interruption.
											</p>
										</div>
									</div>
								</div>
							)}
						</div>
					) : (
						<div className="text-center py-8">
							<svg
								className="mx-auto h-12 w-12 text-gray-400"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
								aria-hidden="true"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"
								/>
							</svg>
							<h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-white">
								No license found
							</h3>
							<p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
								You are using the free plan with limited features.
							</p>
							{canManageLicense && (
								<button
									type="button"
									onClick={() => setShowActivateForm(true)}
									className="mt-4 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
								>
									Enter License Key
								</button>
							)}
						</div>
					)}
				</div>
			</div>

			{/* Activate License Form */}
			{showActivateForm && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
					<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
						<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
							Activate License
						</h2>
					</div>
					<div className="p-6 space-y-4">
						<div>
							<label
								htmlFor="licenseKey"
								className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							>
								License Key
							</label>
							<input
								type="text"
								id="licenseKey"
								value={licenseKey}
								onChange={(e) => {
									setLicenseKey(e.target.value);
									setValidationResult(null);
								}}
								placeholder="Enter your license key"
								className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono"
							/>
						</div>

						{validationResult && (
							<div
								className={`p-4 rounded-lg ${
									validationResult.valid
										? 'bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800'
										: 'bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800'
								}`}
							>
								{validationResult.valid ? (
									<div className="flex items-start gap-3">
										<svg
											className="w-5 h-5 text-green-500 mt-0.5"
											fill="none"
											stroke="currentColor"
											viewBox="0 0 24 24"
											aria-hidden="true"
										>
											<path
												strokeLinecap="round"
												strokeLinejoin="round"
												strokeWidth={2}
												d="M5 13l4 4L19 7"
											/>
										</svg>
										<div>
											<h3 className="font-medium text-green-800 dark:text-green-200">
												Valid License Key
											</h3>
											<p className="text-sm text-green-700 dark:text-green-300 mt-1">
												Plan:{' '}
												<span className="capitalize">
													{validationResult.tier}
												</span>
												{validationResult.valid_until && (
													<>
														{' '}
														| Expires:{' '}
														{new Date(
															validationResult.valid_until,
														).toLocaleDateString()}
													</>
												)}
											</p>
										</div>
									</div>
								) : (
									<div className="flex items-start gap-3">
										<svg
											className="w-5 h-5 text-red-500 mt-0.5"
											fill="none"
											stroke="currentColor"
											viewBox="0 0 24 24"
											aria-hidden="true"
										>
											<path
												strokeLinecap="round"
												strokeLinejoin="round"
												strokeWidth={2}
												d="M6 18L18 6M6 6l12 12"
											/>
										</svg>
										<div>
											<h3 className="font-medium text-red-800 dark:text-red-200">
												Invalid License Key
											</h3>
											<p className="text-sm text-red-700 dark:text-red-300 mt-1">
												{validationResult.error ??
													'The license key is not valid.'}
											</p>
										</div>
									</div>
								)}
							</div>
						)}

						{activateLicense.isError && (
							<p className="text-sm text-red-600 dark:text-red-400">
								Failed to activate license. Please try again.
							</p>
						)}

						<div className="flex justify-end gap-3">
							<button
								type="button"
								onClick={handleCancelActivate}
								className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
							>
								Cancel
							</button>
							{!validationResult?.valid ? (
								<button
									type="button"
									onClick={handleValidate}
									disabled={!licenseKey.trim() || validateLicense.isPending}
									className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
								>
									{validateLicense.isPending ? 'Validating...' : 'Validate'}
								</button>
							) : (
								<button
									type="button"
									onClick={handleActivate}
									disabled={activateLicense.isPending}
									className="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors disabled:opacity-50"
								>
									{activateLicense.isPending
										? 'Activating...'
										: 'Activate License'}
								</button>
							)}
						</div>
					</div>
				</div>
			)}

			{/* Env var notice for env-configured licenses */}
			{license.license_key_source === 'env' && (
				<div className="rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-900/20">
					<p className="text-sm text-amber-700 dark:text-amber-400">
						Your license key is configured via the <code className="font-mono text-xs">LICENSE_KEY</code> environment variable.
						To manage your license from this page, remove the environment variable and restart.
					</p>
				</div>
			)}
			{/* Usage Section */}
			{license?.usage && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
					<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
						<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
							Usage
						</h2>
					</div>
					<div className="p-6 space-y-6">
						<UsageBar
							label="Agents"
							used={license.usage.agents_used}
							limit={license.usage.agents_limit}
						/>
						<UsageBar
							label="Repositories"
							used={license.usage.repositories_used}
							limit={license.usage.repositories_limit}
						/>
						<StorageUsageBar
							label="Storage"
							used={license.usage.storage_used_bytes}
							limit={license.usage.storage_limit_bytes}
						/>
					</div>
				</div>
			)}

			{/* Features Section */}
			{license?.features && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
					<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
						<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
							Features
						</h2>
					</div>
					<div className="p-6">
						<div className="grid grid-cols-1 md:grid-cols-2 gap-x-8 divide-y md:divide-y-0 divide-gray-200 dark:divide-gray-700">
							<div className="space-y-1">
								<FeatureItem
									name="Single Sign-On (SSO)"
									enabled={license.features.sso_enabled}
									proFeature="sso"
								/>
								<FeatureItem
									name="API Access"
									enabled={license.features.api_access}
									proFeature="api_access"
								/>
								<FeatureItem
									name="Advanced Reporting"
									enabled={license.features.advanced_reporting}
									proFeature="advanced_reporting"
								/>
								<FeatureItem
									name="Custom Branding"
									enabled={license.features.custom_branding}
									proFeature="custom_branding"
								/>
							</div>
							<div className="space-y-1 pt-2 md:pt-0">
								<FeatureItem
									name="Priority Support"
									enabled={license.features.priority_support}
									proFeature="priority_support"
								/>
								<FeatureItem
									name="Backup Hooks"
									enabled={license.features.backup_hooks}
									proFeature="backup_hooks"
								/>
								<FeatureItem
									name="Multi-Destination Backups"
									enabled={license.features.multi_destination}
									proFeature="multi_destination"
								/>
							</div>
						</div>
					</div>
				</div>
			)}

			{/* License History */}
			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
					<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
						License History
					</h2>
				</div>
				<div className="p-6">
					{historyLoading ? (
						<div className="space-y-4">
							{[1, 2, 3].map((i) => (
								<div
									key={i}
									className="h-12 bg-gray-200 dark:bg-gray-700 rounded animate-pulse"
								/>
							))}
						</div>
					) : historyData?.history && historyData.history.length > 0 ? (
						<div className="space-y-4">
							{historyData.history.map((entry) => (
								<div
									key={entry.id}
									className="flex items-start gap-4 py-3 border-b border-gray-200 dark:border-gray-700 last:border-0"
								>
									<div className="flex-1">
										<div className="flex items-center gap-2">
											<span
												className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${getHistoryActionColor(entry.action)}`}
											>
												{getHistoryActionLabel(entry.action)}
											</span>
											{entry.new_tier && entry.previous_tier && (
												<span className="text-sm text-gray-500 dark:text-gray-400">
													{entry.previous_tier} → {entry.new_tier}
												</span>
											)}
										</div>
										{entry.notes && (
											<p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
												{entry.notes}
											</p>
										)}
									</div>
									<div className="text-sm text-gray-500 dark:text-gray-400">
										{new Date(entry.created_at).toLocaleDateString()}
									</div>
								</div>
							))}
						</div>
					) : (
						<div className="text-center py-8">
							<svg
								className="mx-auto h-12 w-12 text-gray-400"
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
							<h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-white">
								No license history
							</h3>
							<p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
								License changes will appear here.
							</p>
						</div>
					)}
				</div>
			</div>
		</div>
	);
}
