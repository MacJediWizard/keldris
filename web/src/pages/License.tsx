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
