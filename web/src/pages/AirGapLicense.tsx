import { Link } from 'react-router-dom';
import { Card, CardContent, CardHeader } from '../components/ui/Card';
import { ErrorMessage } from '../components/ui/ErrorMessage';
import { LoadingSpinner } from '../components/ui/LoadingSpinner';
import { useAirGapStatus } from '../hooks/useAirGap';
import { useCallback, useRef, useState } from 'react';
import { Button } from '../components/ui/Button';
import { Card, CardContent, CardHeader } from '../components/ui/Card';
import { ErrorMessage } from '../components/ui/ErrorMessage';
import { LoadingSpinner } from '../components/ui/LoadingSpinner';
import { useAirGapStatus, useUploadLicense } from '../hooks/useAirGap';
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
	variant: 'success' | 'warning' | 'danger';
	children: React.ReactNode;
}) {
	const colors = {
		success: 'bg-green-100 text-green-800',
		warning: 'bg-amber-100 text-amber-800',
		danger: 'bg-red-100 text-red-800',
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
	const { t, formatDateTime } = useLocale();
	const { data: status, isLoading, error } = useAirGapStatus();
	const uploadLicense = useUploadLicense();
	const fileInputRef = useRef<HTMLInputElement>(null);
	const [uploadError, setUploadError] = useState<string | null>(null);

	const handleFileUpload = useCallback(
		async (event: React.ChangeEvent<HTMLInputElement>) => {
			const file = event.target.files?.[0];
			if (!file) return;

			setUploadError(null);

			try {
				const buffer = await file.arrayBuffer();
				await uploadLicense.mutateAsync(buffer);
			} catch (err) {
				setUploadError(
					err instanceof Error ? err.message : t('errors.generic'),
				);
			}

			if (fileInputRef.current) {
				fileInputRef.current.value = '';
			}
		},
		[uploadLicense, t],
	);

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
		<div className="p-6">
			<div className="mb-6">
				<h1 className="text-2xl font-bold text-gray-900">
					{t('airGap.title')}
				</h1>
				<p className="mt-1 text-sm text-gray-500">{t('airGap.subtitle')}</p>
			</div>

			<div className="grid gap-6 md:grid-cols-2">
				<Card>
					<CardHeader>
						<div className="flex items-center justify-between">
							<h2 className="text-lg font-semibold text-gray-900">
								{t('airGap.modeStatus')}
							</h2>
							<StatusBadge variant={status?.enabled ? 'warning' : 'success'}>
								{status?.enabled
									? t('airGap.airGapEnabled')
									: t('airGap.airGapDisabled')}
							</StatusBadge>
						</div>
					</CardHeader>
					<CardContent>
						{status?.enabled && status.disabled_features?.length > 0 && (
							<div>
								<h3 className="text-sm font-medium text-gray-700 mb-2">
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
												<span className="font-medium text-gray-900">
													{feature.name.replace(/_/g, ' ')}
												</span>
												<p className="text-gray-500">{feature.reason}</p>
											</div>
										</li>
									))}
								</ul>
							</div>
						)}
						{!status?.enabled && (
							<p className="text-sm text-gray-500">
								{t('airGap.notInAirGapMode')}
							</p>
						)}
					</CardContent>
				</Card>

				<Card>
					<CardHeader>
						<h2 className="text-lg font-semibold text-gray-900">
							{t('airGap.offlineLicense')}
						</h2>
					</CardHeader>
					<CardContent>
						{status?.license ? (
							<div className="space-y-3">
								<div className="flex items-center justify-between">
									<span className="text-sm text-gray-500">
										{t('common.status')}
									</span>
									<StatusBadge
										variant={status.license.valid ? 'success' : 'danger'}
									>
										{status.license.valid
											? t('airGap.licenseValid')
											: t('airGap.licenseExpired')}
									</StatusBadge>
								</div>
								<div className="flex items-center justify-between">
									<span className="text-sm text-gray-500">
										{t('airGap.customerIdLabel')}
									</span>
									<span className="text-sm font-medium text-gray-900">
										{status.license.customer_id}
									</span>
								</div>
								<div className="flex items-center justify-between">
									<span className="text-sm text-gray-500">
										{t('airGap.tierLabel')}
									</span>
									<span className="text-sm font-medium text-gray-900 capitalize">
										{status.license.tier}
									</span>
								</div>
								<div className="flex items-center justify-between">
									<span className="text-sm text-gray-500">
										{t('airGap.expiresLabel')}
									</span>
									<span className="text-sm text-gray-900">
										{formatDateTime(status.license.expires_at)}
									</span>
								</div>
								<div className="flex items-center justify-between">
									<span className="text-sm text-gray-500">
										{t('airGap.issuedLabel')}
									</span>
									<span className="text-sm text-gray-900">
										{formatDateTime(status.license.issued_at)}
									</span>
								</div>
							</div>
						) : (
							<p className="text-sm text-gray-500 mb-4">
								{t('airGap.noLicense')}
							</p>
						)}

						<div className="mt-4 pt-4 border-t border-gray-100">
							<input
								ref={fileInputRef}
								type="file"
								accept=".json,.license"
								onChange={handleFileUpload}
								className="hidden"
							/>
							<Button
								onClick={() => fileInputRef.current?.click()}
								disabled={uploadLicense.isPending}
							>
								{uploadLicense.isPending
									? t('airGap.uploading')
									: t('airGap.uploadLicense')}
							</Button>
							{uploadError && (
								<p className="mt-2 text-sm text-red-600">{uploadError}</p>
							)}
							{uploadLicense.isSuccess && (
								<p className="mt-2 text-sm text-green-600">
									{t('airGap.licenseUploaded')}
								</p>
							)}
						</div>
					</CardContent>
				</Card>
			</div>
		</div>
	);
}
