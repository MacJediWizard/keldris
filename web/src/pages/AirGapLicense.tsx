import { Link } from 'react-router-dom';
import { Card, CardContent, CardHeader } from '../components/ui/Card';
import { ErrorMessage } from '../components/ui/ErrorMessage';
import { LoadingSpinner } from '../components/ui/LoadingSpinner';
import { useAirGapStatus } from '../hooks/useAirGap';
import { useLocale } from '../hooks/useLocale';

function StatusBadge({
	variant,
	children,
}: {
	variant: 'success' | 'warning';
	children: React.ReactNode;
}) {
	const colors = {
		success: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
		warning: 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400',
	};
	return (
		<span
			className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${colors[variant]}`}
		>
			{children}
		</span>
	);
}

export default function AirGapLicensePage() {
	const { t } = useLocale();
	const { data: status, isLoading, error } = useAirGapStatus();

	if (isLoading) return <LoadingSpinner />;
	if (error) return <ErrorMessage message={t('airGap.failedToLoadStatus')} />;

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
					{t('airGap.title')}
				</h1>
				<StatusBadge variant={status?.enabled ? 'warning' : 'success'}>
					{status?.enabled
						? t('airGap.airGapEnabled')
						: t('airGap.airGapDisabled')}
				</StatusBadge>
			</div>

			<Card>
				<CardHeader>
					<h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
						{t('airGap.modeStatus')}
					</h2>
				</CardHeader>
				<CardContent>
					{status?.enabled ? (
						<>
							<p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
								Air-gap mode is active. Some features that require internet access are disabled.
								Your license key is verified locally using cryptographic signature validation.
							</p>
							{status.disabled_features?.length > 0 && (
								<div>
									<h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
										{t('airGap.disabledFeatures')}
									</h3>
									<ul className="space-y-2">
										{status.disabled_features.map((feature) => (
											<li
												key={feature.name}
												className="flex items-start gap-2 text-sm"
											>
												<svg
													aria-hidden="true"
													className="w-4 h-4 text-amber-500 mt-0.5 flex-shrink-0"
													fill="none"
													stroke="currentColor"
													viewBox="0 0 24 24"
												>
													<path
														strokeLinecap="round"
														strokeLinejoin="round"
														strokeWidth={2}
														d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z"
													/>
												</svg>
												<div>
													<span className="font-medium text-gray-900 dark:text-gray-100">
														{feature.name.replace(/_/g, ' ')}
													</span>
													<p className="text-gray-500 dark:text-gray-400">{feature.reason}</p>
												</div>
											</li>
										))}
									</ul>
								</div>
							)}
						</>
					) : (
						<p className="text-sm text-gray-500 dark:text-gray-400">
							{t('airGap.notInAirGapMode')} Your license is validated online with the license server.
							To enable air-gap mode, set the <code className="font-mono text-xs bg-gray-100 dark:bg-gray-700 px-1 py-0.5 rounded">AIR_GAP_MODE=true</code> environment variable.
						</p>
					)}
				</CardContent>
			</Card>

			<div className="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-border dark:bg-dark-card">
				<p className="text-sm text-gray-600 dark:text-gray-400">
					License management is available on the{' '}
					<Link to="/license" className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400">
						License page
					</Link>.
					In air-gap mode, your license key is verified locally using Ed25519 signature validation.
					If the key expires, a 30-day grace period allows continued operation.
				</p>
			</div>
		</div>
	);
}
