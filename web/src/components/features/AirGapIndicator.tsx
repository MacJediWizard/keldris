import { Link } from 'react-router-dom';
import { useAirGap, useLicenseStatus } from '../../hooks/useAirGap';

interface AirGapIndicatorProps {
	showDetails?: boolean;
	className?: string;
}

/**
 * Air-gap mode indicator badge.
 * Shows when the system is operating in offline/air-gapped mode.
 */
export function AirGapIndicator({
	showDetails = false,
	className = '',
}: AirGapIndicatorProps) {
	const { isAirGapMode, licenseValid, isLoading } = useAirGap();

	if (isLoading || !isAirGapMode) {
		return null;
	}

	return (
		<Link
			to="/admin/license"
			className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium transition-colors ${
				licenseValid
					? 'bg-slate-100 text-slate-700 hover:bg-slate-200 border border-slate-200'
					: 'bg-amber-100 text-amber-800 hover:bg-amber-200 border border-amber-300'
			} ${className}`}
			title={
				licenseValid
					? 'Air-gapped mode - No external network access'
					: 'Air-gapped mode - License issue detected'
			}
		>
			{/* Air-gap icon (cloud with slash) */}
			<svg
				aria-hidden="true"
				className="w-3.5 h-3.5"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					strokeLinecap="round"
					strokeLinejoin="round"
					strokeWidth={2}
					d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z"
				/>
				<path
					strokeLinecap="round"
					strokeLinejoin="round"
					strokeWidth={2}
					d="M4 4l16 16"
				/>
			</svg>
			<span>Air-Gapped</span>
			{showDetails && !licenseValid && (
				<span className="ml-1 px-1.5 py-0.5 bg-amber-200 rounded text-[10px]">
					Check License
				</span>
			)}
		</Link>
	);
}

/**
 * Detailed air-gap status card for admin pages.
 */
export function AirGapStatusCard() {
	const { isAirGapMode, disableExternalLinks, offlineDocsVersion, isLoading } =
		useAirGap();
	const { data: license, isLoading: licenseLoading } = useLicenseStatus();

	if (isLoading || licenseLoading) {
		return (
			<div className="bg-white rounded-lg border border-gray-200 p-6 animate-pulse">
				<div className="h-6 bg-gray-200 rounded w-1/3 mb-4" />
				<div className="space-y-3">
					<div className="h-4 bg-gray-200 rounded w-1/2" />
					<div className="h-4 bg-gray-200 rounded w-2/3" />
				</div>
			</div>
		);
	}

	if (!isAirGapMode) {
		return (
			<div className="bg-white rounded-lg border border-gray-200 p-6">
				<div className="flex items-center gap-3 mb-4">
					<div className="p-2 bg-blue-100 rounded-lg">
						<svg
							aria-hidden="true"
							className="w-5 h-5 text-blue-600"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z"
							/>
						</svg>
					</div>
					<div>
						<h3 className="font-semibold text-gray-900">Connected Mode</h3>
						<p className="text-sm text-gray-500">
							System is operating with network access
						</p>
					</div>
				</div>
				<div className="text-sm text-gray-600">
					<p>
						The system can connect to external services for updates, telemetry,
						and documentation.
					</p>
				</div>
			</div>
		);
	}

	return (
		<div className="bg-white rounded-lg border border-slate-200 p-6">
			<div className="flex items-center gap-3 mb-4">
				<div className="p-2 bg-slate-100 rounded-lg">
					<svg
						aria-hidden="true"
						className="w-5 h-5 text-slate-600"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z"
						/>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M4 4l16 16"
						/>
					</svg>
				</div>
				<div>
					<h3 className="font-semibold text-gray-900">Air-Gapped Mode</h3>
					<p className="text-sm text-gray-500">
						Operating in offline/isolated environment
					</p>
				</div>
			</div>

			<div className="space-y-4">
				{/* License Status */}
				<div className="flex items-center justify-between py-2 border-b border-gray-100">
					<span className="text-sm text-gray-600">License Status</span>
					{license?.valid ? (
						<span className="inline-flex items-center gap-1 text-sm text-green-700 bg-green-50 px-2 py-0.5 rounded">
							<svg
								aria-hidden="true"
								className="w-4 h-4"
								fill="currentColor"
								viewBox="0 0 20 20"
							>
								<path
									fillRule="evenodd"
									d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
									clipRule="evenodd"
								/>
							</svg>
							Valid
						</span>
					) : (
						<span className="inline-flex items-center gap-1 text-sm text-amber-700 bg-amber-50 px-2 py-0.5 rounded">
							<svg
								aria-hidden="true"
								className="w-4 h-4"
								fill="currentColor"
								viewBox="0 0 20 20"
							>
								<path
									fillRule="evenodd"
									d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
									clipRule="evenodd"
								/>
							</svg>
							{license?.error || 'Invalid'}
						</span>
					)}
				</div>

				{/* License Expiry */}
				{license?.expires_at && (
					<div className="flex items-center justify-between py-2 border-b border-gray-100">
						<span className="text-sm text-gray-600">Expires</span>
						<span
							className={`text-sm ${
								(license.days_until_expiry ?? 0) <= 30
									? 'text-amber-600 font-medium'
									: 'text-gray-900'
							}`}
						>
							{new Date(license.expires_at).toLocaleDateString()}
							{license.days_until_expiry !== undefined &&
								` (${license.days_until_expiry} days)`}
						</span>
					</div>
				)}

				{/* License Type */}
				{license?.type && (
					<div className="flex items-center justify-between py-2 border-b border-gray-100">
						<span className="text-sm text-gray-600">License Type</span>
						<span className="text-sm font-medium text-gray-900 capitalize">
							{license.type}
						</span>
					</div>
				)}

				{/* External Links */}
				<div className="flex items-center justify-between py-2 border-b border-gray-100">
					<span className="text-sm text-gray-600">External Links</span>
					<span
						className={`text-sm ${disableExternalLinks ? 'text-red-600' : 'text-green-600'}`}
					>
						{disableExternalLinks ? 'Blocked' : 'Allowed'}
					</span>
				</div>

				{/* Offline Docs */}
				{offlineDocsVersion && (
					<div className="flex items-center justify-between py-2 border-b border-gray-100">
						<span className="text-sm text-gray-600">Offline Documentation</span>
						<span className="text-sm text-gray-900">v{offlineDocsVersion}</span>
					</div>
				)}

				{/* Organization */}
				{license?.organization && (
					<div className="flex items-center justify-between py-2">
						<span className="text-sm text-gray-600">Organization</span>
						<span className="text-sm font-medium text-gray-900">
							{license.organization}
						</span>
					</div>
				)}
			</div>

			{/* Actions */}
			<div className="mt-6 pt-4 border-t border-gray-100">
				<Link
					to="/admin/license"
					className="text-sm text-indigo-600 hover:text-indigo-800 font-medium"
				>
					Manage License &rarr;
				</Link>
			</div>
		</div>
	);
}

/**
 * External link wrapper that respects air-gap mode settings.
 */
interface ExternalLinkProps
	extends React.AnchorHTMLAttributes<HTMLAnchorElement> {
	href: string;
	children: React.ReactNode;
	fallbackMessage?: string;
}

export function ExternalLink({
	href,
	children,
	fallbackMessage = 'External links are disabled in air-gapped mode',
	...props
}: ExternalLinkProps) {
	const { shouldBlockExternalLink } = useAirGap();

	if (shouldBlockExternalLink(href)) {
		return (
			<span
				className="text-gray-400 cursor-not-allowed"
				title={fallbackMessage}
				{...props}
			>
				{children}
				<svg
					aria-hidden="true"
					className="inline-block w-3 h-3 ml-1 opacity-50"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636"
					/>
				</svg>
			</span>
		);
	}

	return (
		<a href={href} target="_blank" rel="noopener noreferrer" {...props}>
			{children}
		</a>
	);
}
