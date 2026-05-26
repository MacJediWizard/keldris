import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useAgentImport,
	useAgentImportPreview,
	useAgentImportTemplate,
	useAgentImportTemplateDownload,
	useAgentImportTokensExport,
	useAgentRegistrationScript,
} from './useAgentImport';

function mockFetch(data: unknown, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
		blob: () => Promise.resolve(new Blob([JSON.stringify(data)])),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

describe('useAgentImportPreview', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('previews import', async () => {
		mockFetch({ rows: [], errors: [] });

		const { result } = renderHook(() => useAgentImportPreview(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ file: new File(['x'], 'test.csv') });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useAgentImport', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('imports agents', async () => {
		mockFetch({ imported: 5, results: [] });

		const { result } = renderHook(() => useAgentImport(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ file: new File(['x'], 'test.csv') });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useAgentImportTemplate', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches template', async () => {
		mockFetch({ template: 'hostname,group' });

		const { result } = renderHook(() => useAgentImportTemplate(), {
			wrapper: createWrapper(),
		});

		result.current.mutate(undefined);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useAgentImportTemplateDownload', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
		vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:url');
		vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => undefined);
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('downloads template blob', async () => {
		mockFetch({ ok: true });

		const { result } = renderHook(() => useAgentImportTemplateDownload(), {
			wrapper: createWrapper(),
		});

		result.current.mutate();

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useAgentRegistrationScript', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('generates script', async () => {
		mockFetch({ script: 'curl ...' });

		const { result } = renderHook(() => useAgentRegistrationScript(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ hostname: 'h1', registrationCode: 'code' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});

describe('useAgentImportTokensExport', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
		vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:url');
		vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => undefined);
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('exports tokens', async () => {
		mockFetch({ ok: true });

		const { result } = renderHook(() => useAgentImportTokensExport(), {
			wrapper: createWrapper(),
		});

		result.current.mutate([]);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
	});
});
