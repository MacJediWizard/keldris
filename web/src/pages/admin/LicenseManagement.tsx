import { useState } from 'react';
import { AirGapStatusCard } from '../../components/features/AirGapIndicator';
import {
	useApplyUpdate,
	useDownloadRenewalRequest,
	useLicenseStatus,
	useUpdatePackages,
	useUpdateRevocationList,
	useUploadLicense,
} from '../../hooks/useAirGap';
import { useMe } from '../../hooks/useAuth';

export function LicenseManagement() {
	const { data: user } = useMe();
	const { data: license, isLoading: licenseLoading } = useLicenseStatus();
	const { data: packages, isLoading: packagesLoading } = useUpdatePackages();

	const uploadLicense = useUploadLicense();
	const downloadRenewal = useDownloadRenewalRequest();
	const updateRevocations = useUpdateRevocationList();
	const applyUpdate = useApplyUpdate();

	const [licenseFile, setLicenseFile] = useState<string>('');
	const [revocationFile, setRevocationFile] = useState<string>('');
	const [showUploadModal, setShowUploadModal] = useState(false);
	const [showRevocationModal, setShowRevocationModal] = useState(false);

	const isSuperuser = user?.is_superuser;

	const handleFileUpload = (
		e: React.ChangeEvent<HTMLInputElement>,
		setter: (value: string) => void,
	) => {
		const file = e.target.files?.[0];
		if (file) {
			const reader = new FileReader();
			reader.onload = (event) => {
				setter(event.target?.result as string);
			};
			reader.readAsText(file);
		}
	};

	const handleLicenseUpload = () => {
		if (!licenseFile) return;
		uploadLicense.mutate(licenseFile, {
			onSuccess: () => {
				setShowUploadModal(false);
				setLicenseFile('');
			},
		});
	};

	const handleRevocationUpload = () => {
		if (!revocationFile) return;
		updateRevocations.mutate(revocationFile, {
			onSuccess: () => {
				setShowRevocationModal(false);
				setRevocationFile('');
			},
		});
	};

	const formatBytes = (bytes: number) => {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	};

	if (licenseLoading) {
		return (
			<div className="p-6 animate-pulse">
				<div className="h-8 bg-gray-200 rounded w-1/4 mb-6" />
				<div className="h-64 bg-gray-200 rounded" />
			</div>
		);
	}

	return (
		<div className="space-y-6">
			{/* Header */}
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">
						License Management
					</h1>
					<p className="text-gray-500 mt-1">
						Manage your Keldris Enterprise license for air-gapped operation
					</p>
				</div>
			</div>

			{/* Air-Gap Status Card */}
			<AirGapStatusCard />

			{/* License Features */}
			{license?.features && license.features.length > 0 && (
				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<h2 className="text-lg font-semibold text-gray-900 mb-4">
						Licensed Features
					</h2>
					<div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
						{license.features.map((feature) => (
							<div
								key={feature}
								className="flex items-center gap-2 px-3 py-2 bg-green-50 border border-green-200 rounded-lg"
							>
								<svg
									className="w-4 h-4 text-green-600"
									fill="currentColor"
									viewBox="0 0 20 20"
								>
									<path
										fillRule="evenodd"
										d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
										clipRule="evenodd"
									/>
								</svg>
								<span className="text-sm text-green-800 capitalize">
									{feature.replace(/_/g, ' ')}
								</span>
							</div>
						))}
					</div>
				</div>
			)}

			{/* License Actions */}
			{isSuperuser && (
				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<h2 className="text-lg font-semibold text-gray-900 mb-4">
						License Actions
					</h2>
					<div className="flex flex-wrap gap-4">
						<button
							type="button"
							onClick={() => setShowUploadModal(true)}
							className="inline-flex items-center gap-2 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
						>
							<svg
								className="w-5 h-5"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"
								/>
							</svg>
							Upload New License
						</button>

						<button
							type="button"
							onClick={() => downloadRenewal.mutate()}
							disabled={downloadRenewal.isPending}
							className="inline-flex items-center gap-2 px-4 py-2 bg-white border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors disabled:opacity-50"
						>
							<svg
								className="w-5 h-5"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
								/>
							</svg>
							{downloadRenewal.isPending
								? 'Generating...'
								: 'Download Renewal Request'}
						</button>

						<button
							type="button"
							onClick={() => setShowRevocationModal(true)}
							className="inline-flex items-center gap-2 px-4 py-2 bg-white border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors"
						>
							<svg
								className="w-5 h-5"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
								/>
							</svg>
							Update Revocation List
						</button>
					</div>
				</div>
			)}

			{/* Offline Update Packages */}
			{isSuperuser && (
				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<h2 className="text-lg font-semibold text-gray-900 mb-4">
						Offline Update Packages
					</h2>
					<p className="text-sm text-gray-600 mb-4">
						Place update packages in the configured update directory to make
						them available here.
					</p>

					{packagesLoading ? (
						<div className="animate-pulse space-y-2">
							<div className="h-12 bg-gray-200 rounded" />
							<div className="h-12 bg-gray-200 rounded" />
						</div>
					) : packages?.packages && packages.packages.length > 0 ? (
						<div className="overflow-x-auto">
							<table className="w-full">
								<thead>
									<tr className="text-left text-xs font-medium text-gray-500 uppercase tracking-wider border-b border-gray-200">
										<th className="pb-3">Package</th>
										<th className="pb-3">Size</th>
										<th className="pb-3">Added</th>
										<th className="pb-3" />
									</tr>
								</thead>
								<tbody className="divide-y divide-gray-100">
									{packages.packages.map((pkg) => (
										<tr key={pkg.filename} className="hover:bg-gray-50">
											<td className="py-3">
												<div className="flex items-center gap-2">
													<svg
														className="w-5 h-5 text-gray-400"
														fill="none"
														stroke="currentColor"
														viewBox="0 0 24 24"
													>
														<path
															strokeLinecap="round"
															strokeLinejoin="round"
															strokeWidth={2}
															d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"
														/>
													</svg>
													<span className="font-medium text-gray-900">
														{pkg.filename}
													</span>
													{pkg.version && (
														<span className="px-2 py-0.5 text-xs bg-indigo-100 text-indigo-700 rounded">
															{pkg.version}
														</span>
													)}
												</div>
											</td>
											<td className="py-3 text-sm text-gray-600">
												{formatBytes(pkg.size)}
											</td>
											<td className="py-3 text-sm text-gray-600">
												{new Date(pkg.created_at).toLocaleDateString()}
											</td>
											<td className="py-3 text-right">
												<button
													type="button"
													onClick={() => {
														if (
															window.confirm(
																`Apply update ${pkg.filename}? The system will restart.`,
															)
														) {
															applyUpdate.mutate(pkg.filename);
														}
													}}
													disabled={applyUpdate.isPending}
													className="px-3 py-1.5 text-sm font-medium text-white bg-indigo-600 rounded hover:bg-indigo-700 disabled:opacity-50 transition-colors"
												>
													{applyUpdate.isPending ? 'Applying...' : 'Apply'}
												</button>
											</td>
										</tr>
									))}
								</tbody>
							</table>
						</div>
					) : (
						<div className="text-center py-8 text-gray-500">
							<svg
								className="w-12 h-12 mx-auto text-gray-300 mb-3"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"
								/>
							</svg>
							<p>No update packages found</p>
							<p className="text-sm mt-1">
								Upload packages to the configured update directory
							</p>
						</div>
					)}
				</div>
			)}

			{/* Manual Renewal Instructions */}
			<div className="bg-blue-50 border border-blue-200 rounded-lg p-6">
				<h3 className="text-lg font-semibold text-blue-900 mb-3">
					Manual License Renewal
				</h3>
				<div className="prose prose-sm prose-blue">
					<p className="text-blue-800">
						To renew your license in an air-gapped environment:
					</p>
					<ol className="text-blue-700 list-decimal list-inside space-y-2 mt-3">
						<li>
							Click "Download Renewal Request" to generate a renewal request
							file
						</li>
						<li>
							Transfer the file to a system with internet access (e.g., via USB
							drive)
						</li>
						<li>
							Upload the file to your Keldris account portal at{' '}
							<code className="bg-blue-100 px-1 rounded">
								license.keldris.io
							</code>
						</li>
						<li>Download the new license file from the portal</li>
						<li>
							Transfer the new license file back to this system and upload it
							using "Upload New License"
						</li>
					</ol>
				</div>
			</div>

			{/* Upload License Modal */}
			{showUploadModal && (
				<div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
					<div className="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
						<h3 className="text-lg font-semibold text-gray-900 mb-4">
							Upload New License
						</h3>
						<div className="space-y-4">
							<div>
								<label
									htmlFor="license-file"
									className="block text-sm font-medium text-gray-700 mb-2"
								>
									License File
								</label>
								<input
									type="file"
									id="license-file"
									accept=".json,.lic"
									onChange={(e) => handleFileUpload(e, setLicenseFile)}
									className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								/>
							</div>
							{licenseFile && (
								<div className="p-3 bg-gray-50 rounded border border-gray-200">
									<p className="text-sm text-gray-600">
										License file loaded ({licenseFile.length} bytes)
									</p>
								</div>
							)}
							{uploadLicense.isError && (
								<div className="p-3 bg-red-50 border border-red-200 rounded">
									<p className="text-sm text-red-700">
										{uploadLicense.error instanceof Error
											? uploadLicense.error.message
											: 'Failed to upload license'}
									</p>
								</div>
							)}
						</div>
						<div className="flex justify-end gap-3 mt-6">
							<button
								type="button"
								onClick={() => {
									setShowUploadModal(false);
									setLicenseFile('');
								}}
								className="px-4 py-2 text-gray-700 hover:text-gray-900"
							>
								Cancel
							</button>
							<button
								type="button"
								onClick={handleLicenseUpload}
								disabled={!licenseFile || uploadLicense.isPending}
								className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 disabled:opacity-50 transition-colors"
							>
								{uploadLicense.isPending ? 'Uploading...' : 'Upload License'}
							</button>
						</div>
					</div>
				</div>
			)}

			{/* Update Revocation List Modal */}
			{showRevocationModal && (
				<div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
					<div className="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
						<h3 className="text-lg font-semibold text-gray-900 mb-4">
							Update Revocation List
						</h3>
						<p className="text-sm text-gray-600 mb-4">
							Upload a signed revocation list to invalidate compromised
							licenses.
						</p>
						<div className="space-y-4">
							<div>
								<label
									htmlFor="revocation-file"
									className="block text-sm font-medium text-gray-700 mb-2"
								>
									Revocation List File
								</label>
								<input
									type="file"
									id="revocation-file"
									accept=".json"
									onChange={(e) => handleFileUpload(e, setRevocationFile)}
									className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
								/>
							</div>
							{updateRevocations.isError && (
								<div className="p-3 bg-red-50 border border-red-200 rounded">
									<p className="text-sm text-red-700">
										{updateRevocations.error instanceof Error
											? updateRevocations.error.message
											: 'Failed to update revocation list'}
									</p>
								</div>
							)}
						</div>
						<div className="flex justify-end gap-3 mt-6">
							<button
								type="button"
								onClick={() => {
									setShowRevocationModal(false);
									setRevocationFile('');
								}}
								className="px-4 py-2 text-gray-700 hover:text-gray-900"
							>
								Cancel
							</button>
							<button
								type="button"
								onClick={handleRevocationUpload}
								disabled={!revocationFile || updateRevocations.isPending}
								className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 disabled:opacity-50 transition-colors"
							>
								{updateRevocations.isPending ? 'Updating...' : 'Update List'}
							</button>
						</div>
					</div>
				</div>
			)}
		</div>
	);
}
