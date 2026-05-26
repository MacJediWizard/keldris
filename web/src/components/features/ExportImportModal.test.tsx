import { render } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useConfigExport', () => ({
	useExportAgent: vi.fn(() => ({ mutateAsync: vi.fn() })),
	useExportSchedule: vi.fn(() => ({ mutateAsync: vi.fn() })),
	useExportRepository: vi.fn(() => ({ mutateAsync: vi.fn() })),
	useExportBundle: vi.fn(() => ({ mutateAsync: vi.fn() })),
	useImportConfig: vi.fn(() => ({ mutateAsync: vi.fn(), isPending: false })),
	useValidateImport: vi.fn(() => ({ mutateAsync: vi.fn() })),
	downloadExport: vi.fn(),
}));

vi.mock('../../hooks/useLocale', () => ({
	useLocale: () => ({
		t: (_key: string, defaultValue: string) => defaultValue,
		locale: 'en',
	}),
}));

import { ExportImportModal } from './ExportImportModal';

describe('ExportImportModal', () => {
	it('renders nothing when closed', () => {
		const { container } = render(
			<ExportImportModal isOpen={false} onClose={vi.fn()} mode="export" />,
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders modal when open in export mode', () => {
		const { container } = render(
			<ExportImportModal
				isOpen
				onClose={vi.fn()}
				mode="export"
				resourceType="agent"
				resourceId="abc"
			/>,
		);
		expect(container.firstChild).not.toBeNull();
	});

	it('renders import mode', () => {
		const { container } = render(
			<ExportImportModal isOpen onClose={vi.fn()} mode="import" />,
		);
		expect(container.firstChild).not.toBeNull();
	});
});
