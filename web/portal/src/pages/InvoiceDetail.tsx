import { Link, useParams } from 'react-router-dom';
import { Card, CardContent, CardHeader } from '../components/Card';
import { StatusBadge } from '../components/StatusBadge';
import { useInvoice } from '../hooks/useInvoices';

export function InvoiceDetail() {
	const { id } = useParams<{ id: string }>();
	const { data, isLoading, error } = useInvoice(id || '');

	if (isLoading) {
		return (
			<div className="flex justify-center py-8">
				<div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
			</div>
		);
	}

	if (error || !data) {
		return (
			<div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-red-600 dark:text-red-400 px-4 py-3 rounded-md">
				Failed to load invoice
			</div>
		);
	}

	const { invoice, items } = data;

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<Link
						to="/invoices"
						className="text-sm text-blue-600 hover:text-blue-500"
					>
						&larr; Back to Invoices
					</Link>
					<h1 className="mt-2 text-2xl font-bold text-gray-900 dark:text-white">
						Invoice {invoice.invoice_number}
					</h1>
				</div>
				<StatusBadge status={invoice.status} variant="invoice" />
			</div>

			<div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
				<Card className="lg:col-span-2">
					<CardHeader>
						<h2 className="text-lg font-medium text-gray-900 dark:text-white">
							Line Items
						</h2>
					</CardHeader>
					<CardContent className="p-0">
						<table className="min-w-full divide-y divide-gray-200 dark:divide-dark-border">
							<thead className="bg-gray-50 dark:bg-dark-card">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Description
									</th>
									<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Qty
									</th>
									<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Unit Price
									</th>
									<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Total
									</th>
								</tr>
							</thead>
							<tbody className="bg-white dark:bg-dark-card divide-y divide-gray-200 dark:divide-dark-border">
								{items.map((item) => (
									<tr key={item.id}>
										<td className="px-6 py-4 text-sm text-gray-900 dark:text-white">
											{item.description}
										</td>
										<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400 text-right">
											{item.quantity}
										</td>
										<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400 text-right">
											{invoice.currency} {(item.unit_price / 100).toFixed(2)}
										</td>
										<td className="px-6 py-4 text-sm text-gray-900 dark:text-white text-right font-medium">
											{invoice.currency} {(item.total / 100).toFixed(2)}
										</td>
									</tr>
								))}
							</tbody>
							<tfoot className="bg-gray-50 dark:bg-dark-card">
								<tr>
									<td
										colSpan={3}
										className="px-6 py-3 text-sm text-gray-500 dark:text-gray-400 text-right"
									>
										Subtotal
									</td>
									<td className="px-6 py-3 text-sm text-gray-900 dark:text-white text-right font-medium">
										{invoice.currency} {(invoice.subtotal / 100).toFixed(2)}
									</td>
								</tr>
								<tr>
									<td
										colSpan={3}
										className="px-6 py-3 text-sm text-gray-500 dark:text-gray-400 text-right"
									>
										Tax
									</td>
									<td className="px-6 py-3 text-sm text-gray-900 dark:text-white text-right font-medium">
										{invoice.currency} {(invoice.tax / 100).toFixed(2)}
									</td>
								</tr>
								<tr>
									<td
										colSpan={3}
										className="px-6 py-3 text-sm font-medium text-gray-900 dark:text-white text-right"
									>
										Total
									</td>
									<td className="px-6 py-3 text-lg text-gray-900 dark:text-white text-right font-bold">
										{invoice.currency} {(invoice.total / 100).toFixed(2)}
									</td>
								</tr>
							</tfoot>
						</table>
					</CardContent>
				</Card>

				<Card>
					<CardHeader>
						<h2 className="text-lg font-medium text-gray-900 dark:text-white">
							Invoice Details
						</h2>
					</CardHeader>
					<CardContent>
						<dl className="space-y-4">
							<div>
								<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
									Invoice Number
								</dt>
								<dd className="mt-1 text-sm text-gray-900 dark:text-white">
									{invoice.invoice_number}
								</dd>
							</div>
							<div>
								<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
									Issue Date
								</dt>
								<dd className="mt-1 text-sm text-gray-900 dark:text-white">
									{new Date(invoice.created_at).toLocaleDateString()}
								</dd>
							</div>
							{invoice.due_date && (
								<div>
									<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
										Due Date
									</dt>
									<dd className="mt-1 text-sm text-gray-900 dark:text-white">
										{new Date(invoice.due_date).toLocaleDateString()}
									</dd>
								</div>
							)}
							{invoice.paid_at && (
								<div>
									<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
										Paid Date
									</dt>
									<dd className="mt-1 text-sm text-gray-900 dark:text-white">
										{new Date(invoice.paid_at).toLocaleDateString()}
									</dd>
								</div>
							)}
							{invoice.payment_method && (
								<div>
									<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
										Payment Method
									</dt>
									<dd className="mt-1 text-sm text-gray-900 dark:text-white capitalize">
										{invoice.payment_method.replace('_', ' ')}
									</dd>
								</div>
							)}
							{invoice.billing_address && (
								<div>
									<dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
										Billing Address
									</dt>
									<dd className="mt-1 text-sm text-gray-900 dark:text-white whitespace-pre-line">
										{invoice.billing_address}
									</dd>
								</div>
							)}
						</dl>

						{invoice.status !== 'paid' && invoice.status !== 'cancelled' && (
							<div className="mt-6 pt-6 border-t border-gray-200 dark:border-dark-border">
								<div className="text-sm text-gray-500 dark:text-gray-400 mb-3">
									Amount Due
								</div>
								<div className="text-2xl font-bold text-gray-900 dark:text-white">
									{invoice.currency}{' '}
									{((invoice.total - invoice.amount_paid) / 100).toFixed(2)}
								</div>
							</div>
						)}
					</CardContent>
				</Card>
			</div>

			{invoice.notes && (
				<Card>
					<CardHeader>
						<h2 className="text-lg font-medium text-gray-900 dark:text-white">
							Notes
						</h2>
					</CardHeader>
					<CardContent>
						<p className="text-sm text-gray-700 dark:text-gray-300">
							{invoice.notes}
						</p>
					</CardContent>
				</Card>
			)}
		</div>
	);
}
