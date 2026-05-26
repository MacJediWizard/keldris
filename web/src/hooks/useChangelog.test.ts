import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	useChangelog,
	useChangelogEntry,
	useLatestChanges,
	useNewVersionAvailable,
} from './useChangelog';

vi.mock('../lib/api', () => ({
	changelogApi: {
		list: vi.fn(),
		get: vi.fn(),
	},
}));

import { changelogApi } from '../lib/api';

const mockChangelog = {
	current_version: '1.0.0',
	entries: [
		{ version: '1.1.0', is_unreleased: false, changes: ['change'] },
		{ version: '1.0.0', is_unreleased: false, changes: ['initial'] },
	],
};

describe('useChangelog', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches changelog', async () => {
		vi.mocked(changelogApi.list).mockResolvedValue(mockChangelog);

		const { result } = renderHook(() => useChangelog(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(result.current.data).toEqual(mockChangelog);
	});
});

describe('useChangelogEntry', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('fetches a single entry', async () => {
		vi.mocked(changelogApi.get).mockResolvedValue({
			version: '1.0.0',
			changes: [],
		});

		const { result } = renderHook(() => useChangelogEntry('1.0.0'), {
			wrapper: createWrapper(),
		});

		await waitFor(() => expect(result.current.isSuccess).toBe(true));
		expect(changelogApi.get).toHaveBeenCalledWith('1.0.0');
	});

	it('does not fetch when version is empty', () => {
		renderHook(() => useChangelogEntry(''), { wrapper: createWrapper() });
		expect(changelogApi.get).not.toHaveBeenCalled();
	});
});

describe('useNewVersionAvailable', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('reports new version when latest differs from current', async () => {
		vi.mocked(changelogApi.list).mockResolvedValue(mockChangelog);

		const { result } = renderHook(() => useNewVersionAvailable(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => {
			expect(result.current.hasNewVersion).toBe(true);
		});
		expect(result.current.latestVersion).toBe('1.1.0');
	});

	it('returns no new version when data is missing', () => {
		vi.mocked(changelogApi.list).mockResolvedValue(
			undefined as unknown as typeof mockChangelog,
		);

		const { result } = renderHook(() => useNewVersionAvailable(), {
			wrapper: createWrapper(),
		});

		expect(result.current.hasNewVersion).toBe(false);
		expect(result.current.latestVersion).toBeNull();
	});
});

describe('useLatestChanges', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('returns the latest non-unreleased entry', async () => {
		vi.mocked(changelogApi.list).mockResolvedValue(mockChangelog);

		const { result } = renderHook(() => useLatestChanges(), {
			wrapper: createWrapper(),
		});

		await waitFor(() => {
			expect(result.current.latestEntry).toBeTruthy();
		});
		expect(result.current.latestEntry?.version).toBe('1.1.0');
	});
});
