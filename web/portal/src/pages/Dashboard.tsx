import { Link } from 'react-router-dom';
import { Card, CardContent, CardHeader } from '../components/Card';
import { StatusBadge } from '../components/StatusBadge';
import { useMe } from '../hooks/useAuth';
import { useInvoices } from '../hooks/useInvoices';
import { useLicenses } from '../hooks/useLicenses';

export function Dashboard() {
	const { data: customer } = useMe();
	const { data: licensesData } = useLicenses();
	const { data: invoicesData } = useInvoices();

	const licenses = licensesData?.licenses || [];
	const invoices = invoicesData?.invoices || [];

	const activeLicenses = licenses.filter((l) => l.status === 'active').length;
	const unpaidInvoices = invoices.filter(
		(i) => i.status !== 'paid' && i.status !== 'cancelled',
	).length;

	return (
		<div className="space-y-6">
			<div>
				<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
					Welcome, {customer?.name || 'Customer'}
				</h1>
				<p className="mt-1 text-gray-600 dark:text-gray-400">
					Manage your licenses and view invoices
				</p>
			</div>

			{/* Stats */}
			<div className="grid grid-cols-1 md:grid-cols-3 gap-6">
				<Card>
					<CardContent>
						<div className="text-sm font-medium text-gray-500 dark:text-gray-400">
							Active Licenses
						</div>
						<div className="mt-1 text-3xl font-semibold text-gray-900 dark:text-white">
							{activeLicenses}
						</div>
					</CardContent>
				</Card>
				<Card>
					<CardContent>
						<div className="text-sm font-medium text-gray-500 dark:text-gray-400">
							Total Licenses
						</div>
						<div className="mt-1 text-3xl font-semibold text-gray-900 dark:text-white">
							{licenses.length}
						</div>
					</CardContent>
				</Card>
				<Card>
					<CardContent>
						<div className="text-sm font-medium text-gray-500 dark:text-gray-400">
							Unpaid Invoices
						</div>
						<div className="mt-1 text-3xl font-semibold text-gray-900 dark:text-white">
							{unpaidInvoices}
						</div>
					</CardContent>
				</Card>
			</div>

			{/* Recent Licenses */}
			<Card>
				<CardHeader>
					<div className="flex justify-between items-center">
						<h2 className="text-lg font-medium text-gray-900 dark:text-white">
							Recent Licenses
						</h2>
						<Link
							to="/licenses"
							className="text-sm text-blue-600 hover:text-blue-500"
						>
							View all
						</Link>
					</div>
				</CardHeader>
				<CardContent className="p-0">
					{licenses.length === 0 ? (
						<div className="px-6 py-4 text-gray-500 dark:text-gray-400">
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
										Type
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Status
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Expires
									</th>
								</tr>
							</thead>
							<tbody className="bg-white dark:bg-dark-card divide-y divide-gray-200 dark:divide-dark-border">
								{licenses.slice(0, 5).map((license) => (
									<tr key={license.id}>
										<td className="px-6 py-4 whitespace-nowrap">
											<Link
												to={`/licenses/${license.id}`}
												className="text-blue-600 hover:text-blue-500"
											>
												{license.product_name}
											</Link>
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
									</tr>
								))}
							</tbody>
						</table>
					)}
				</CardContent>
			</Card>

			{/* Recent Invoices */}
			<Card>
				<CardHeader>
					<div className="flex justify-between items-center">
						<h2 className="text-lg font-medium text-gray-900 dark:text-white">
							Recent Invoices
						</h2>
						<Link
							to="/invoices"
							className="text-sm text-blue-600 hover:text-blue-500"
						>
							View all
						</Link>
					</div>
				</CardHeader>
				<CardContent className="p-0">
					{invoices.length === 0 ? (
						<div className="px-6 py-4 text-gray-500 dark:text-gray-400">
							No invoices found
						</div>
					) : (
						<table className="min-w-full divide-y divide-gray-200 dark:divide-dark-border">
							<thead className="bg-gray-50 dark:bg-dark-card">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Invoice
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Amount
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Status
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Date
									</th>
								</tr>
							</thead>
							<tbody className="bg-white dark:bg-dark-card divide-y divide-gray-200 dark:divide-dark-border">
								{invoices.slice(0, 5).map((invoice) => (
									<tr key={invoice.id}>
										<td className="px-6 py-4 whitespace-nowrap">
											<Link
												to={`/invoices/${invoice.id}`}
												className="text-blue-600 hover:text-blue-500"
											>
												{invoice.invoice_number}
											</Link>
										</td>
										<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
											{invoice.currency} {(invoice.total / 100).toFixed(2)}
										</td>
										<td className="px-6 py-4 whitespace-nowrap">
											<StatusBadge status={invoice.status} variant="invoice" />
										</td>
										<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
											{new Date(invoice.created_at).toLocaleDateString()}
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
