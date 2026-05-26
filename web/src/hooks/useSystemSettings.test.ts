import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useOIDCSettings,
	useSMTPSettings,
	useSecuritySettings,
	useSettingsAuditLog,
	useStorageDefaultSettings,
	useSystemSettings,
	useTestOIDC,
	useTestSMTP,
	useUpdateOIDCSettings,
	useUpdateSMTPSettings,
	useUpdateSecuritySettings,
	useUpdateStorageDefaultSettings,
} from './useSystemSettings';

function mockFetch(data: unknown, ok = true, status = 200) {
	const fn = vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(data),
	} as unknown as Response);
	vi.stubGlobal('fetch', fn);
	return fn;
}

describe('useSystemSettings', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches all system settings', async () => {
		const fetchFn = mockFetch({ smtp: null });

		const { result } = renderHook(() => useSystemSettings(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/system-settings',
			expect.any(Object),
		);
	});
});

describe('useSMTPSettings', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches SMTP settings', async () => {
		const fetchFn = mockFetch({ host: 'smtp.test' });

		const { result } = renderHook(() => useSMTPSettings(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/system-settings/smtp',
			expect.any(Object),
		);
	});
});

describe('useUpdateSMTPSettings', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('updates SMTP settings', async () => {
		const fetchFn = mockFetch({ host: 'smtp.test' });

		const { result } = renderHook(() => useUpdateSMTPSettings(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ host: 'smtp.test', port: 587 });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/system-settings/smtp',
			expect.objectContaining({ method: 'PUT' }),
		);
	});
});

describe('useTestSMTP', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('tests SMTP settings', async () => {
		const fetchFn = mockFetch({ success: true });

		const { result } = renderHook(() => useTestSMTP(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ recipient: 'a@b.com' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/system-settings/smtp/test',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useOIDCSettings', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches OIDC settings', async () => {
		const fetchFn = mockFetch({ enabled: false });

		const { result } = renderHook(() => useOIDCSettings(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/system-settings/oidc',
			expect.any(Object),
		);
	});
});

describe('useUpdateOIDCSettings', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('updates OIDC settings', async () => {
		const fetchFn = mockFetch({ enabled: true });

		const { result } = renderHook(() => useUpdateOIDCSettings(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ enabled: true });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/system-settings/oidc',
			expect.objectContaining({ method: 'PUT' }),
		);
	});
});

describe('useTestOIDC', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('tests OIDC settings', async () => {
		const fetchFn = mockFetch({ success: true });

		const { result } = renderHook(() => useTestOIDC(), {
			wrapper: createWrapper(),
		});

		result.current.mutate();

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/system-settings/oidc/test',
			expect.objectContaining({ method: 'POST' }),
		);
	});
});

describe('useStorageDefaultSettings', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches storage defaults', async () => {
		const fetchFn = mockFetch({ default_backend: 'local' });

		const { result } = renderHook(() => useStorageDefaultSettings(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/system-settings/storage',
			expect.any(Object),
		);
	});
});

describe('useUpdateStorageDefaultSettings', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('updates storage defaults', async () => {
		const fetchFn = mockFetch({ default_backend: 's3' });

		const { result } = renderHook(() => useUpdateStorageDefaultSettings(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ default_backend: 's3' });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/system-settings/storage',
			expect.objectContaining({ method: 'PUT' }),
		);
	});
});

describe('useSecuritySettings', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches security settings', async () => {
		const fetchFn = mockFetch({ mfa_required: false });

		const { result } = renderHook(() => useSecuritySettings(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/system-settings/security',
			expect.any(Object),
		);
	});
});

describe('useUpdateSecuritySettings', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('updates security settings', async () => {
		const fetchFn = mockFetch({ mfa_required: true });

		const { result } = renderHook(() => useUpdateSecuritySettings(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ mfa_required: true });

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/system-settings/security',
			expect.objectContaining({ method: 'PUT' }),
		);
	});
});

describe('useSettingsAuditLog', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('fetches settings audit log with defaults', async () => {
		const fetchFn = mockFetch({ logs: [] });

		const { result } = renderHook(() => useSettingsAuditLog(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/v1/system-settings/audit-log?limit=50&offset=0',
			expect.any(Object),
		);
	});
});
