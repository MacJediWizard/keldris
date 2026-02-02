import { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { Card, CardContent, CardHeader } from '../components/Card';
import { StatusBadge } from '../components/StatusBadge';
import { useLicense } from '../hooks/useLicenses';
import { licensesApi } from '../lib/api';

export function LicenseDetail() {
	const { id } = useParams<{ id: string }>();
	const { data: license, isLoading, error } = useLicense(id || '');
	const [downloading, setDownloading] = useState(false);
	const [copied, setCopied] = useState(false);

	if (isLoading) {
		return (
			<div className="flex justify-center py-8">
				<div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
			</div>
		);
	}

	if (error || !license) {
		return (
			<div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-red-600 dark:text-red-400 px-4 py-3 rounded-md">
				Failed to load license
			</div>
		);
	}

	const handleDownload = async () => {
		if (!id) return;
		setDownloading(true);
		try {
			const data = await licensesApi.download(id);
			const blob = new Blob([JSON.stringify(data, null, 2)], {
				type: 'application/json',
			});
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `license-${license.license_key}.json`;
			document.body.appendChild(a);
			a.click();
			document.body.removeChild(a);
			URL.revokeObjectURL(url);
		} catch (err) {
			console.error('Failed to download license:', err);
		} finally {
			setDownloading(false);
		}
	};

	const handleCopy = async () => {
		try {
			await navigator.clipboard.writeText(license.license_key);
			setCopied(true);
			setTimeout(() => setCopied(false), 2000);
		} catch (err) {
			console.error('Failed to copy:', err);
		}
	};

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<Link
						to="/licenses"
						className="text-sm text-blue-600 hover:text-blue-500"
					>
						&larr; Back to Licenses
					</Link>
					<h1 className="mt-2 text-2xl font-bold text-gray-900 dark:text-white">
						{license.product_name}
					</h1>
				</div>
				<StatusBadge status={license.status} />
			</div>

			<div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
				<Card>
					<CardHeader>
						<h2 className="text-lg font-medium text-gray-900 dark:text-white">
							License Key
						</h2>
					</CardHeader>
					<CardContent>
						<div className="flex items-center space-x-2">
							<code className="flex-1 text-lg font-mono text-gray-900 dark:text-white bg-gray-100 dark:bg-gray-800 px-4 py-3 rounded-lg">
								{license.license_key}
							</code>
							<button
								type="button"
								onClick={handleCopy}
								className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-800 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-700"
							>
								{copied ? 'Copied!' : 'Copy'}
							</button>
						</div>
						<div className="mt-4">
							<button
								type="button"
								onClick={handleDownload}
								disabled={downloading || license.status !== 'active'}
								className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
							>
								{downloading ? 'Downloading...' : 'Download License File'}
							</button>
						</div>
					</CardContent>
				</Card>

				<Card>
					<CardHeader>
						<h2 className="text-lg font-medium text-gray-900 dark:text-white">
							License Details
						</h2>
					</CardHeader>
					<CardContent>
						<dl className="space-y-4">
							<div>
								<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
									License Type
								</dt>
								<dd className="mt-1 text-sm text-gray-900 dark:text-white capitalize">
									{license.license_type}
								</dd>
							</div>
							<div>
								<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
									Issued At
								</dt>
								<dd className="mt-1 text-sm text-gray-900 dark:text-white">
									{new Date(license.issued_at).toLocaleDateString()}
								</dd>
							</div>
							<div>
								<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
									Expires At
								</dt>
								<dd className="mt-1 text-sm text-gray-900 dark:text-white">
									{license.expires_at
										? new Date(license.expires_at).toLocaleDateString()
										: 'Never (Perpetual)'}
								</dd>
							</div>
							{license.max_agents && (
								<div>
									<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
										Max Agents
									</dt>
									<dd className="mt-1 text-sm text-gray-900 dark:text-white">
										{license.max_agents}
									</dd>
								</div>
							)}
							{license.max_repos && (
								<div>
									<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
										Max Repositories
									</dt>
									<dd className="mt-1 text-sm text-gray-900 dark:text-white">
										{license.max_repos}
									</dd>
								</div>
							)}
							{license.max_storage_gb && (
								<div>
									<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
										Max Storage
									</dt>
									<dd className="mt-1 text-sm text-gray-900 dark:text-white">
										{license.max_storage_gb} GB
									</dd>
								</div>
							)}
						</dl>
					</CardContent>
				</Card>
			</div>

			{license.features && license.features.length > 0 && (
				<Card>
					<CardHeader>
						<h2 className="text-lg font-medium text-gray-900 dark:text-white">
							Included Features
						</h2>
					</CardHeader>
					<CardContent>
						<ul className="grid grid-cols-2 md:grid-cols-3 gap-2">
							{license.features.map((feature) => (
								<li
									key={feature}
									className="flex items-center text-sm text-gray-700 dark:text-gray-300"
								>
									<svg
										className="w-4 h-4 mr-2 text-green-500"
										fill="currentColor"
										viewBox="0 0 20 20"
										aria-hidden="true"
									>
										<path
											fillRule="evenodd"
											d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
											clipRule="evenodd"
										/>
									</svg>
									{feature}
								</li>
							))}
						</ul>
					</CardContent>
				</Card>
			)}
		</div>
	);
}
