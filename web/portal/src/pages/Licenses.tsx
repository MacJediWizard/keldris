import { Link } from 'react-router-dom';
import { Card, CardContent, CardHeader } from '../components/Card';
import { StatusBadge } from '../components/StatusBadge';
import { useLicenses } from '../hooks/useLicenses';

export function Licenses() {
	const { data, isLoading, error } = useLicenses();

	if (isLoading) {
		return (
			<div className="flex justify-center py-8">
				<div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
			</div>
		);
	}

	if (error) {
		return (
			<div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-red-600 dark:text-red-400 px-4 py-3 rounded-md">
				Failed to load licenses
			</div>
		);
	}

	const licenses = data?.licenses || [];

	return (
		<div className="space-y-6">
			<div>
				<h1 className="text-2xl font-bold text-gray-900 dark:text-white">Licenses</h1>
				<p className="mt-1 text-gray-600 dark:text-gray-400">
					View and download your license keys
				</p>
			</div>

			<Card>
				<CardHeader>
					<h2 className="text-lg font-medium text-gray-900 dark:text-white">
						Your Licenses
					</h2>
				</CardHeader>
				<CardContent className="p-0">
					{licenses.length === 0 ? (
						<div className="px-6 py-8 text-center text-gray-500 dark:text-gray-400">
							No licenses found
						</div>
					) : (
						<table className="min-w-full divide-y divide-gray-200 dark:divide-dark-border">
							<thead className="bg-gray-50 dark:bg-dark-card">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Product
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										License Key
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Type
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Status
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Expires
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Actions
									</th>
								</tr>
							</thead>
							<tbody className="bg-white dark:bg-dark-card divide-y divide-gray-200 dark:divide-dark-border">
								{licenses.map((license) => (
									<tr key={license.id}>
										<td className="px-6 py-4 whitespace-nowrap">
											<div className="text-sm font-medium text-gray-900 dark:text-white">
												{license.product_name}
											</div>
										</td>
										<td className="px-6 py-4 whitespace-nowrap">
											<code className="text-sm text-gray-600 dark:text-gray-400 bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded">
												{license.license_key}
											</code>
										</td>
										<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400 capitalize">
											{license.license_type}
										</td>
										<td className="px-6 py-4 whitespace-nowrap">
											<StatusBadge status={license.status} />
										</td>
										<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
											{license.expires_at
												? new Date(license.expires_at).toLocaleDateString()
												: 'Never'}
										</td>
										<td className="px-6 py-4 whitespace-nowrap text-sm">
											<Link
												to={`/licenses/${license.id}`}
												className="text-blue-600 hover:text-blue-500"
											>
												View Details
											</Link>
										</td>
									</tr>
								))}
							</tbody>
						</table>
					)}
				</CardContent>
			</Card>
		</div>
	);
}
