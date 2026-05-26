import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useActivateLicense,
	useCompleteSetup,
	useConfigureOIDC,
	useConfigureSMTP,
	useCreateFirstOrganization,
	useCreateSuperuser,
	useRerunConfigureOIDC,
	useRerunConfigureSMTP,
	useRerunStatus,
	useRerunUpdateLicense,
	useSetupStatus,
	useSkipOIDC,
	useSkipSMTP,
	useStartTrial,
	useTestDatabase,
} from './useSetup';

vi.mock('../lib/api', () => ({
	setupApi: {
		getStatus: vi.fn(),
		testDatabase: vi.fn(),
		createSuperuser: vi.fn(),
		configureSMTP: vi.fn(),
		skipSMTP: vi.fn(),
		configureOIDC: vi.fn(),
		skipOIDC: vi.fn(),
		activateLicense: vi.fn(),
		startTrial: vi.fn(),
		createOrganization: vi.fn(),
		completeSetup: vi.fn(),
		getRerunStatus: vi.fn(),
		rerunConfigureSMTP: vi.fn(),
		rerunConfigureOIDC: vi.fn(),
		rerunUpdateLicense: vi.fn(),
	},
}));

import { setupApi } from '../lib/api';

describe('useSetupStatus', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches setup status', async () => {
		vi.mocked(setupApi.getStatus).mockResolvedValue({ ready: true });

		const { result } = renderHook(() => useSetupStatus(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(setupApi.getStatus).toHaveBeenCalledOnce();
	});
});

describe('useTestDatabase', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('tests database', async () => {
		vi.mocked(setupApi.testDatabase).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useTestDatabase(), {
			wrapper: createWrapper(),
		});

		result.current.mutate();

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(setupApi.testDatabase).toHaveBeenCalledOnce();
	});
});

describe('useCreateSuperuser', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('creates a superuser', async () => {
		vi.mocked(setupApi.createSuperuser).mockResolvedValue({ id: 'u-1' });

		const { result } = renderHook(() => useCreateSuperuser(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ email: 'a@b.com', password: 'x' } as never);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(setupApi.createSuperuser).toHaveBeenCalled();
	});
});

describe('useConfigureSMTP', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('configures SMTP', async () => {
		vi.mocked(setupApi.configureSMTP).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useConfigureSMTP(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ host: 'smtp.example.com' } as never);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(setupApi.configureSMTP).toHaveBeenCalled();
	});
});

describe('useSkipSMTP', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('skips SMTP', async () => {
		vi.mocked(setupApi.skipSMTP).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useSkipSMTP(), {
			wrapper: createWrapper(),
		});

		result.current.mutate();

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(setupApi.skipSMTP).toHaveBeenCalledOnce();
	});
});

describe('useConfigureOIDC', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('configures OIDC', async () => {
		vi.mocked(setupApi.configureOIDC).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useConfigureOIDC(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ issuer: 'https://x' } as never);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(setupApi.configureOIDC).toHaveBeenCalled();
	});
});

describe('useSkipOIDC', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('skips OIDC', async () => {
		vi.mocked(setupApi.skipOIDC).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useSkipOIDC(), {
			wrapper: createWrapper(),
		});

		result.current.mutate();

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(setupApi.skipOIDC).toHaveBeenCalledOnce();
	});
});

describe('useActivateLicense', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('activates a license', async () => {
		vi.mocked(setupApi.activateLicense).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useActivateLicense(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ key: 'X' } as never);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(setupApi.activateLicense).toHaveBeenCalled();
	});
});

describe('useStartTrial', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('starts a trial', async () => {
		vi.mocked(setupApi.startTrial).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useStartTrial(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ email: 'a@b.com' } as never);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(setupApi.startTrial).toHaveBeenCalled();
	});
});

describe('useCreateFirstOrganization', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('creates first organization', async () => {
		vi.mocked(setupApi.createOrganization).mockResolvedValue({ id: 'o-1' });

		const { result } = renderHook(() => useCreateFirstOrganization(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ name: 'org' } as never);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(setupApi.createOrganization).toHaveBeenCalled();
	});
});

describe('useCompleteSetup', () => {
	const originalLocation = window.location;

	beforeEach(() => {
		vi.clearAllMocks();
		// Replace window.location so the mutation's redirect doesn't navigate jsdom.
		Object.defineProperty(window, 'location', {
			configurable: true,
			value: { href: '' },
		});
	});

	afterEach(() => {
		Object.defineProperty(window, 'location', {
			configurable: true,
			value: originalLocation,
		});
	});

	it('completes setup', async () => {
		vi.mocked(setupApi.completeSetup).mockResolvedValue({ redirect: '/done' });

		const { result } = renderHook(() => useCompleteSetup(), {
			wrapper: createWrapper(),
		});

		result.current.mutate();

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(setupApi.completeSetup).toHaveBeenCalledOnce();
		expect(window.location.href).toBe('/done');
	});
});

describe('useRerunStatus', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches rerun status', async () => {
		vi.mocked(setupApi.getRerunStatus).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useRerunStatus(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(setupApi.getRerunStatus).toHaveBeenCalledOnce();
	});
});

describe('useRerunConfigureSMTP', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('re-configures SMTP', async () => {
		vi.mocked(setupApi.rerunConfigureSMTP).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useRerunConfigureSMTP(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ host: 'x' } as never);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(setupApi.rerunConfigureSMTP).toHaveBeenCalled();
	});
});

describe('useRerunConfigureOIDC', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('re-configures OIDC', async () => {
		vi.mocked(setupApi.rerunConfigureOIDC).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useRerunConfigureOIDC(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ issuer: 'https://x' } as never);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(setupApi.rerunConfigureOIDC).toHaveBeenCalled();
	});
});

describe('useRerunUpdateLicense', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('re-updates license', async () => {
		vi.mocked(setupApi.rerunUpdateLicense).mockResolvedValue({ ok: true });

		const { result } = renderHook(() => useRerunUpdateLicense(), {
			wrapper: createWrapper(),
		});

		result.current.mutate({ key: 'X' } as never);

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(setupApi.rerunUpdateLicense).toHaveBeenCalled();
	});
});
