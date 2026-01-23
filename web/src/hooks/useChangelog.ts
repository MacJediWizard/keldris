import { useQuery } from '@tanstack/react-query';
import { changelogApi } from '../lib/api';

export function useChangelog() {
	return useQuery({
		queryKey: ['changelog'],
		queryFn: changelogApi.list,
		staleTime: 5 * 60 * 1000, // 5 minutes
	});
}

export function useChangelogEntry(version: string) {
	return useQuery({
		queryKey: ['changelog', version],
		queryFn: () => changelogApi.get(version),
		enabled: !!version,
		staleTime: 5 * 60 * 1000,
	});
}

// Hook to check if there's a new version available
export function useNewVersionAvailable() {
	const { data } = useChangelog();

	if (!data) return { hasNewVersion: false, latestVersion: null };

	// Find the latest non-unreleased version
	const latestEntry = data.entries.find((e) => !e.is_unreleased);
	const currentVersion = data.current_version;

	if (!latestEntry || !currentVersion) {
		return { hasNewVersion: false, latestVersion: null };
	}

	// Simple version comparison (assumes semver-like format)
	const hasNewVersion = latestEntry.version !== currentVersion;

	return {
		hasNewVersion,
		latestVersion: latestEntry.version,
		currentVersion,
	};
}

// Hook to get the latest version's changes (for "What's New" modal)
export function useLatestChanges() {
	const { data, isLoading } = useChangelog();

	if (isLoading || !data) {
		return { latestEntry: null, isLoading };
	}

	// Find the first non-unreleased version
	const latestEntry = data.entries.find((e) => !e.is_unreleased);

	return { latestEntry, isLoading, currentVersion: data.current_version };
}
