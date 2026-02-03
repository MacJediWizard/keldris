import { useQuery } from '@tanstack/react-query';
import { invoicesApi } from '../lib/api';

export function useInvoices() {
	return useQuery({
		queryKey: ['invoices'],
		queryFn: () => invoicesApi.list(),
	});
}

export function useInvoice(id: string) {
	return useQuery({
		queryKey: ['invoices', id],
		queryFn: () => invoicesApi.get(id),
		enabled: !!id,
	});
}
