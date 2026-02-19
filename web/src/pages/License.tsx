import { useLicense } from '../hooks/useLicense';
import { TierBadge } from '../components/features/TierBadge';
import { LoadingSpinner } from '../components/ui/LoadingSpinner';

function formatDate(dateStr: string): string {
	if (!dateStr) return 'N/A';
	return new Date(dateStr).toLocaleDateString(undefined, {
		year: 'numeric',
		month: 'long',
		day: 'numeric',
	});
}

function formatBytes(bytes: number): string {
	if (bytes === 0) return 'Unlimited';
	if (bytes < 0) return 'Unlimited';
	const units = ['B', 'KB', 'MB', 'GB', 'TB'];
	let i = 0;
	let val = bytes;
	while (val >= 1024 && i < units.length - 1) {
		val /= 1024;
		i++;
	}
	return `${val.toFixed(1)} ${units[i]}`;
}

function formatLimit(value: number): string {
	if (value <= 0) return 'Unlimited';
	return value.toLocaleString();
}

export default function License() {
	const { data: license, isLoading, error } = useLicense();

	if (isLoading) return <LoadingSpinner />;

	if (error) {
		return (
			<div className="rounded-lg border border-red-200 bg-red-50 p-6">
				<p className="text-red-700">Failed to load license information.</p>
			</div>
		);
	}

	if (!license) return null;

	const isExpired =
		license.expires_at && new Date(license.expires_at) < new Date();

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<h1 className="text-2xl font-bold text-gray-900">License</h1>
				<TierBadge tier={license.tier} className="text-sm px-3 py-1" />
			</div>

			{/* License Overview */}
			<div className="rounded-lg border bg-white p-6 shadow-sm">
				<h2 className="mb-4 text-lg font-semibold text-gray-900">
					License Details
				</h2>
				<dl className="grid grid-cols-1 gap-4 sm:grid-cols-2">
					<div>
						<dt className="text-sm font-medium text-gray-500">Tier</dt>
						<dd className="mt-1 text-sm text-gray-900 capitalize">
							{license.tier}
						</dd>
					</div>
					<div>
						<dt className="text-sm font-medium text-gray-500">Customer ID</dt>
						<dd className="mt-1 text-sm text-gray-900">
							{license.customer_id || 'N/A'}
						</dd>
					</div>
					<div>
						<dt className="text-sm font-medium text-gray-500">Issued</dt>
						<dd className="mt-1 text-sm text-gray-900">
							{formatDate(license.issued_at)}
						</dd>
					</div>
					<div>
						<dt className="text-sm font-medium text-gray-500">Expires</dt>
						<dd
							className={`mt-1 text-sm ${isExpired ? 'text-red-600 font-medium' : 'text-gray-900'}`}
						>
							{formatDate(license.expires_at)}
							{isExpired && ' (Expired)'}
						</dd>
					</div>
				</dl>
			</div>

			{/* Resource Limits */}
			<div className="rounded-lg border bg-white p-6 shadow-sm">
				<h2 className="mb-4 text-lg font-semibold text-gray-900">
					Resource Limits
				</h2>
				<div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
					<div className="rounded-lg border border-gray-200 p-4">
						<p className="text-sm font-medium text-gray-500">Agents</p>
						<p className="mt-1 text-2xl font-semibold text-gray-900">
							{formatLimit(license.limits.max_agents)}
						</p>
					</div>
					<div className="rounded-lg border border-gray-200 p-4">
						<p className="text-sm font-medium text-gray-500">Users</p>
						<p className="mt-1 text-2xl font-semibold text-gray-900">
							{formatLimit(license.limits.max_users)}
						</p>
					</div>
					<div className="rounded-lg border border-gray-200 p-4">
						<p className="text-sm font-medium text-gray-500">Organizations</p>
						<p className="mt-1 text-2xl font-semibold text-gray-900">
							{formatLimit(license.limits.max_orgs)}
						</p>
					</div>
					<div className="rounded-lg border border-gray-200 p-4">
						<p className="text-sm font-medium text-gray-500">Storage</p>
						<p className="mt-1 text-2xl font-semibold text-gray-900">
							{formatBytes(license.limits.max_storage_bytes)}
						</p>
					</div>
				</div>
			</div>

			{/* Features */}
			<div className="rounded-lg border bg-white p-6 shadow-sm">
				<h2 className="mb-4 text-lg font-semibold text-gray-900">
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
								<span className="text-sm text-gray-700 capitalize">
									{feature.replace(/_/g, ' ')}
								</span>
							</div>
						))}
					</div>
				) : (
					<p className="text-sm text-gray-500">
						No features included in the current plan.
					</p>
				)}
			</div>

			{/* Upgrade CTA for free tier */}
			{license.tier === 'free' && (
				<div className="rounded-lg border border-indigo-200 bg-indigo-50 p-6">
					<h3 className="text-lg font-semibold text-indigo-900">
						Upgrade to Pro
					</h3>
					<p className="mt-1 text-sm text-indigo-700">
						Unlock more agents, users, audit logs, API access, and more with a
						Pro or Enterprise license.
					</p>
					<p className="mt-3 text-sm text-indigo-600">
						Set the <code className="font-mono text-xs">LICENSE_KEY</code>{' '}
						environment variable on your server to activate your license.
					</p>
				</div>
			)}
		</div>
	);
}
