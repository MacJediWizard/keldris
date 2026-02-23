import { useCallback, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { Button } from '../components/ui/Button';
import { Card, CardContent, CardHeader } from '../components/ui/Card';
import { ErrorMessage } from '../components/ui/ErrorMessage';
import { LoadingSpinner } from '../components/ui/LoadingSpinner';
import {
	useAirGapStatus,
	useLicenseStatus,
	useUploadLicense,
} from '../hooks/useAirGap';
import { useLocale } from '../hooks/useLocale';

function StatusBadge({
	variant,
	children,
}: {
	variant: 'success' | 'warning' | 'danger';
	children: React.ReactNode;
}) {
	const colors = {
		success:
			'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
		warning:
			'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400',
		danger: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
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
	const { data: licenseStatus } = useLicenseStatus();
	const uploadLicense = useUploadLicense();
	const fileInputRef = useRef<HTMLInputElement>(null);
	const [uploadError, setUploadError] = useState<string | null>(null);

	const handleFileUpload = useCallback(
		async (event: React.ChangeEvent<HTMLInputElement>) => {
			const file = event.target.files?.[0];
			if (!file) return;

			setUploadError(null);

			try {
				const text = await file.text();
				await uploadLicense.mutateAsync(text);
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

	const isAirGapMode = status?.airgap_mode ?? false;

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
						{t('airGap.title')}
					</h1>
					<p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
						{t('airGap.subtitle')}
					</p>
				</div>
				<StatusBadge variant={isAirGapMode ? 'warning' : 'success'}>
					{isAirGapMode
						? t('airGap.airGapEnabled')
						: t('airGap.airGapDisabled')}
				</StatusBadge>
			</div>

			<div className="grid gap-6 md:grid-cols-2">
				<Card>
					<CardHeader>
						<h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
							{t('airGap.modeStatus')}
						</h2>
					</CardHeader>
					<CardContent>
						{isAirGapMode ? (
							<>
								<p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
									Air-gap mode is active. Some features that require internet
									access are disabled. Your license key is verified locally
									using cryptographic signature validation.
								</p>
								<div className="space-y-2">
									{status?.disable_update_checker && (
										<div className="flex items-start gap-2 text-sm">
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
											<span className="text-gray-700 dark:text-gray-300">
												Update checker disabled
											</span>
										</div>
									)}
									{status?.disable_telemetry && (
										<div className="flex items-start gap-2 text-sm">
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
											<span className="text-gray-700 dark:text-gray-300">
												Telemetry disabled
											</span>
										</div>
									)}
									{status?.disable_external_links && (
										<div className="flex items-start gap-2 text-sm">
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
											<span className="text-gray-700 dark:text-gray-300">
												External links disabled
											</span>
										</div>
									)}
								</div>
							</>
						) : (
							<p className="text-sm text-gray-500 dark:text-gray-400">
								{t('airGap.notInAirGapMode')} Your license is validated online
								with the license server. To enable air-gap mode, set the{' '}
								<code className="font-mono text-xs bg-gray-100 dark:bg-gray-700 px-1 py-0.5 rounded">
									AIR_GAP_MODE=true
								</code>{' '}
								environment variable.
							</p>
						)}
					</CardContent>
				</Card>

				<Card>
					<CardHeader>
						<h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
							{t('airGap.offlineLicense')}
						</h2>
					</CardHeader>
					<CardContent>
						{licenseStatus ? (
							<div className="space-y-3">
								<div className="flex items-center justify-between">
									<span className="text-sm text-gray-500 dark:text-gray-400">
										{t('common.status')}
									</span>
									<StatusBadge
										variant={licenseStatus.valid ? 'success' : 'danger'}
									>
										{licenseStatus.valid
											? t('airGap.licenseValid')
											: t('airGap.licenseExpired')}
									</StatusBadge>
								</div>
								{licenseStatus.organization && (
									<div className="flex items-center justify-between">
										<span className="text-sm text-gray-500 dark:text-gray-400">
											Organization
										</span>
										<span className="text-sm font-medium text-gray-900 dark:text-gray-100">
											{licenseStatus.organization}
										</span>
									</div>
								)}
								{licenseStatus.type && (
									<div className="flex items-center justify-between">
										<span className="text-sm text-gray-500 dark:text-gray-400">
											{t('airGap.tierLabel')}
										</span>
										<span className="text-sm font-medium text-gray-900 dark:text-gray-100 capitalize">
											{licenseStatus.type}
										</span>
									</div>
								)}
								{licenseStatus.expires_at && (
									<div className="flex items-center justify-between">
										<span className="text-sm text-gray-500 dark:text-gray-400">
											{t('airGap.expiresLabel')}
										</span>
										<span className="text-sm text-gray-900 dark:text-gray-100">
											{new Date(licenseStatus.expires_at).toLocaleDateString()}
										</span>
									</div>
								)}
								{licenseStatus.days_until_expiry != null && (
									<div className="flex items-center justify-between">
										<span className="text-sm text-gray-500 dark:text-gray-400">
											Days until expiry
										</span>
										<span
											className={`text-sm font-medium ${
												licenseStatus.days_until_expiry <= 30
													? 'text-amber-600 dark:text-amber-400'
													: 'text-gray-900 dark:text-gray-100'
											}`}
										>
											{licenseStatus.days_until_expiry}
										</span>
									</div>
								)}
							</div>
						) : (
							<p className="text-sm text-gray-500 dark:text-gray-400 mb-4">
								{t('airGap.noLicense')}
							</p>
						)}

						<div className="mt-4 pt-4 border-t border-gray-100 dark:border-gray-700">
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

			<div className="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-gray-700 dark:bg-gray-800">
				<p className="text-sm text-gray-600 dark:text-gray-400">
					License management is available on the{' '}
					<Link
						to="/license"
						className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400"
					>
						License page
					</Link>
					. In air-gap mode, your license key is verified locally using Ed25519
					signature validation. If the key expires, a 30-day grace period allows
					continued operation.
				</p>
			</div>
		</div>
	);
}
