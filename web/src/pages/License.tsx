import { useState } from 'react';
import { TierBadge } from '../components/features/TierBadge';
import { LoadingSpinner } from '../components/ui/LoadingSpinner';
import { useLicense, useActivateLicense, useDeactivateLicense, usePricingPlans } from '../hooks/useLicense';

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
	const [licenseKey, setLicenseKey] = useState('');
	const [activateError, setActivateError] = useState('');

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
		} catch {
			// Error handled by mutation
		}
	};

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">License</h1>
				<TierBadge tier={license.tier} className="text-sm px-3 py-1" />
			</div>

			{/* License Key Entry Form */}
			{showActivateForm && (
				<div className="rounded-lg border border-indigo-200 bg-indigo-50 p-6 dark:border-indigo-800 dark:bg-indigo-900/20">
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
					<h2 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">
						Available Plans
					</h2>
					<div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
						{plans.map((plan) => (
							<div
								key={plan.id}
								className="rounded-lg border border-gray-200 p-4 dark:border-dark-border"
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
							</div>
						))}
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
		</div>
	);
}
