import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	downloadMigrationExport,
	readFileAsText,
	useGenerateExportKey,
	useMigrationExport,
	useMigrationImport,
	useValidateMigrationImport,
} from './useMigration';

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

describe('useGenerateExportKey', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('generates an export key', async () => {
		const fetchFn = mockFetch({ key: 'gen-key' });

		const { result } = renderHook(() => useGenerateExportKey(), {
			wrapper: createWrapper(),
		});

		result.current.mutate();

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/migration/export/generate-key',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useMigrationExport', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('exports migration data as a blob', async () => {
		const fetchFn = mockFetch({});

		const { result } = renderHook(() => useMigrationExport(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ include_secrets: false });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/migration/export',
			expect.objectContaining({ method: 'POST' }),
		);
		expect(result.current.data).toBeInstanceOf(Blob);
	});
});

describe('useValidateMigrationImport', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('validates migration import data', async () => {
		const fetchFn = mockFetch({ valid: true, encrypted: false });

		const { result } = renderHook(() => useValidateMigrationImport(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ data: '{}' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/migration/import/validate',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useMigrationImport', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('runs migration import', async () => {
		const fetchFn = mockFetch({
			success: true,
			dry_run: false,
			imported: {},
			skipped: {},
		});

		const { result } = renderHook(() => useMigrationImport(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ data: '{}' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/migration/import',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('downloadMigrationExport', () => {
	it('triggers a download for a blob', () => {
		const createObjectURL = vi.fn(() => 'blob:url');
		const revokeObjectURL = vi.fn();
		vi.stubGlobal('URL', {
			...URL,
			createObjectURL,
			revokeObjectURL,
		});
		const blob = new Blob(['data']);

		downloadMigrationExport(blob, false);

		expect(createObjectURL).toHaveBeenCalledWith(blob);
		expect(revokeObjectURL).toHaveBeenCalledWith('blob:url');

		vi.unstubAllGlobals();
	});
});

describe('readFileAsText', () => {
	it('reads a file as text', async () => {
		const file = new File(['hello world'], 'test.txt', {
			type: 'text/plain',
		});
		const text = await readFileAsText(file);
		expect(text).toBe('hello world');
	});
});
