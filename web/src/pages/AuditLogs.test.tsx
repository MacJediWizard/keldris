import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { BrowserRouter } from 'react-router-dom';
import { AuditLogs } from './AuditLogs';

const mockMutateCsv = vi.fn();
const mockMutateJson = vi.fn();

vi.mock('../hooks/useAuditLogs', () => ({
	useAuditLogs: vi.fn(),
	useExportAuditLogsCsv: () => ({ mutate: mockMutateCsv, isPending: false }),
	useExportAuditLogsJson: () => ({ mutate: mockMutateJson, isPending: false }),
}));

import { useAuditLogs } from '../hooks/useAuditLogs';

function renderPage() {
	return render(
		<BrowserRouter>
			<AuditLogs />
		</BrowserRouter>,
	);
}

describe('AuditLogs', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		vi.mocked(useAuditLogs).mockReturnValue({ data: undefined, isLoading: false, isError: false } as ReturnType<typeof useAuditLogs>);
		renderPage();
		expect(screen.getByText('Audit Logs')).toBeInTheDocument();
		expect(screen.getByText('Track all user and system actions for compliance')).toBeInTheDocument();
	});

	it('shows loading state', () => {
		vi.mocked(useAuditLogs).mockReturnValue({ data: undefined, isLoading: true, isError: false } as ReturnType<typeof useAuditLogs>);
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('shows error state', () => {
		vi.mocked(useAuditLogs).mockReturnValue({ data: undefined, isLoading: false, isError: true } as ReturnType<typeof useAuditLogs>);
		renderPage();
		expect(screen.getByText('Failed to load audit logs')).toBeInTheDocument();
	});

	it('shows empty state', () => {
		vi.mocked(useAuditLogs).mockReturnValue({ data: { audit_logs: [], total_count: 0 }, isLoading: false, isError: false } as ReturnType<typeof useAuditLogs>);
		renderPage();
		expect(screen.getByText('No audit logs found')).toBeInTheDocument();
	});

	it('renders audit log rows', () => {
		vi.mocked(useAuditLogs).mockReturnValue({
			data: {
				audit_logs: [
					{ id: '1', action: 'create', resource_type: 'agent', resource_id: 'abc12345-xxxx', result: 'success', ip_address: '192.168.1.1', details: 'Created agent', created_at: '2024-06-15T12:00:00Z' },
				],
				total_count: 1,
			},
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAuditLogs>);
		renderPage();
		expect(screen.getByText('192.168.1.1')).toBeInTheDocument();
		expect(screen.getByText('Created agent')).toBeInTheDocument();
	});

	it('shows export buttons', () => {
		vi.mocked(useAuditLogs).mockReturnValue({ data: { audit_logs: [], total_count: 0 }, isLoading: false, isError: false } as ReturnType<typeof useAuditLogs>);
		renderPage();
		expect(screen.getByText('Export CSV')).toBeInTheDocument();
		expect(screen.getByText('Export JSON')).toBeInTheDocument();
	});

	it('calls export CSV on click', async () => {
		const user = userEvent.setup();
		vi.mocked(useAuditLogs).mockReturnValue({ data: { audit_logs: [], total_count: 0 }, isLoading: false, isError: false } as ReturnType<typeof useAuditLogs>);
		renderPage();
		await user.click(screen.getByText('Export CSV'));
		expect(mockMutateCsv).toHaveBeenCalled();
	});

	it('shows pagination', () => {
		vi.mocked(useAuditLogs).mockReturnValue({
			data: {
				audit_logs: [
					{ id: '1', action: 'login', resource_type: 'session', result: 'success', ip_address: '10.0.0.1', details: '', created_at: '2024-06-15T12:00:00Z' },
				],
				total_count: 100,
			},
			isLoading: false,
			isError: false,
		} as ReturnType<typeof useAuditLogs>);
		renderPage();
		expect(screen.getByText('Previous')).toBeInTheDocument();
		expect(screen.getByText('Next')).toBeInTheDocument();
		expect(screen.getByText(/Page 1 of/)).toBeInTheDocument();
	});
});
