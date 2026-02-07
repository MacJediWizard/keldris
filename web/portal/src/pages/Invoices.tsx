import { Link } from 'react-router-dom';
import { Card, CardContent, CardHeader } from '../components/Card';
import { StatusBadge } from '../components/StatusBadge';
import { useInvoices } from '../hooks/useInvoices';

export function Invoices() {
	const { data, isLoading, error } = useInvoices();

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
				Failed to load invoices
			</div>
		);
	}

	const invoices = data?.invoices || [];

	return (
		<div className="space-y-6">
			<div>
				<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
					Invoices
				</h1>
				<p className="mt-1 text-gray-600 dark:text-gray-400">
					View your billing history
				</p>
			</div>

			<Card>
				<CardHeader>
					<h2 className="text-lg font-medium text-gray-900 dark:text-white">
						Your Invoices
					</h2>
				</CardHeader>
				<CardContent className="p-0">
					{invoices.length === 0 ? (
						<div className="px-6 py-8 text-center text-gray-500 dark:text-gray-400">
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
										Date
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Amount
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Status
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Due Date
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Actions
									</th>
								</tr>
							</thead>
							<tbody className="bg-white dark:bg-dark-card divide-y divide-gray-200 dark:divide-dark-border">
								{invoices.map((invoice) => (
									<tr key={invoice.id}>
										<td className="px-6 py-4 whitespace-nowrap">
											<div className="text-sm font-medium text-gray-900 dark:text-white">
												{invoice.invoice_number}
											</div>
										</td>
										<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
											{new Date(invoice.created_at).toLocaleDateString()}
										</td>
										<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
											{invoice.currency} {(invoice.total / 100).toFixed(2)}
										</td>
										<td className="px-6 py-4 whitespace-nowrap">
											<StatusBadge status={invoice.status} variant="invoice" />
										</td>
										<td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
											{invoice.due_date
												? new Date(invoice.due_date).toLocaleDateString()
												: '-'}
										</td>
										<td className="px-6 py-4 whitespace-nowrap text-sm">
											<Link
												to={`/invoices/${invoice.id}`}
												className="text-blue-600 hover:text-blue-500"
											>
												View
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
