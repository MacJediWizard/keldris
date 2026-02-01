import { useQuery } from '@tanstack/react-query';
import { licenseApi } from '../lib/api';
import type {
	FeatureCheckResult,
	FeatureInfo,
	LicenseFeature,
	LicenseInfo,
	LicenseTier,
	TierInfo,
} from '../lib/types';

// Feature definitions for client-side reference
const FEATURE_TIERS: Record<LicenseFeature, LicenseTier> = {
	oidc: 'pro',
	audit_logs: 'pro',
	multi_org: 'enterprise',
	sla_tracking: 'enterprise',
	white_label: 'enterprise',
};

const TIER_ORDER: Record<LicenseTier, number> = {
	free: 0,
	pro: 1,
	professional: 1,
	enterprise: 2,
};

// Hook to get current license info
export function useLicense() {
	return useQuery({
		queryKey: ['license'],
		queryFn: licenseApi.getLicense,
		staleTime: 5 * 60 * 1000, // 5 minutes
	});
}

// Hook to check if a specific feature is available
export function useFeatureCheck(feature: LicenseFeature) {
	return useQuery({
		queryKey: ['license', 'features', feature, 'check'],
		queryFn: () => licenseApi.checkFeature(feature),
		staleTime: 5 * 60 * 1000, // 5 minutes
	});
}

// Hook to get all feature definitions
export function useFeatures() {
	return useQuery({
		queryKey: ['license', 'features'],
		queryFn: licenseApi.getFeatures,
		staleTime: 30 * 60 * 1000, // 30 minutes - rarely changes
	});
}

// Hook to get all tier definitions
export function useTiers() {
	return useQuery({
		queryKey: ['license', 'tiers'],
		queryFn: licenseApi.getTiers,
		staleTime: 30 * 60 * 1000, // 30 minutes - rarely changes
	});
}

// Simplified hook that returns just boolean for feature availability
export function useFeature(feature: LicenseFeature): {
	isEnabled: boolean;
	isLoading: boolean;
	currentTier: LicenseTier | undefined;
	requiredTier: LicenseTier;
	upgradeMessage: string | undefined;
} {
	const { data, isLoading } = useFeatureCheck(feature);

	return {
		isEnabled: data?.enabled ?? false,
		isLoading,
		currentTier: data?.current_tier,
		requiredTier: data?.required_tier ?? FEATURE_TIERS[feature],
		upgradeMessage: data?.upgrade_info?.message,
	};
}

// Client-side feature check (useful for immediate checks without API call)
export function canAccessFeature(
	currentTier: LicenseTier,
	feature: LicenseFeature,
): boolean {
	const requiredTier = FEATURE_TIERS[feature];
	return TIER_ORDER[currentTier] >= TIER_ORDER[requiredTier];
}

// Get required tier for a feature
export function getRequiredTier(feature: LicenseFeature): LicenseTier {
	return FEATURE_TIERS[feature];
}

// Get tier display name
export function getTierDisplayName(tier: LicenseTier): string {
	switch (tier) {
		case 'free':
			return 'Free';
		case 'pro':
		case 'professional':
			return 'Professional';
		case 'enterprise':
			return 'Enterprise';
		default:
			return tier;
	}
}

// Get feature display name
export function getFeatureDisplayName(feature: LicenseFeature): string {
	switch (feature) {
		case 'oidc':
			return 'OIDC Authentication';
		case 'audit_logs':
			return 'Audit Logs';
		case 'multi_org':
			return 'Multiple Organizations';
		case 'sla_tracking':
			return 'SLA Tracking';
		case 'white_label':
			return 'White-Label';
		default:
			return feature;
	}
}

// Generate upgrade message for a feature
export function getUpgradeMessage(feature: LicenseFeature): string {
	const requiredTier = getRequiredTier(feature);
	const tierName = getTierDisplayName(requiredTier);
	const featureName = getFeatureDisplayName(feature);
	return `Upgrade to ${tierName} to access ${featureName}`;
}

// Type guard for checking if tier meets minimum requirement
export function tierMeetsRequirement(
	currentTier: LicenseTier,
	requiredTier: LicenseTier,
): boolean {
	return TIER_ORDER[currentTier] >= TIER_ORDER[requiredTier];
}

// Hook for components that need to show upgrade prompts
export function useUpgradePrompt(feature: LicenseFeature): {
	shouldShowPrompt: boolean;
	isLoading: boolean;
	featureName: string;
	requiredTier: LicenseTier;
	tierName: string;
	message: string;
} {
	const { isEnabled, isLoading, requiredTier, upgradeMessage } =
		useFeature(feature);

	return {
		shouldShowPrompt: !isLoading && !isEnabled,
		isLoading,
		featureName: getFeatureDisplayName(feature),
		requiredTier,
		tierName: getTierDisplayName(requiredTier),
		message: upgradeMessage ?? getUpgradeMessage(feature),
	};
}

// All available features
export const ALL_FEATURES: LicenseFeature[] = [
	'oidc',
	'audit_logs',
	'multi_org',
	'sla_tracking',
	'white_label',
];

// Features by tier
export const PRO_FEATURES: LicenseFeature[] = ['oidc', 'audit_logs'];
export const ENTERPRISE_FEATURES: LicenseFeature[] = [
	'multi_org',
	'sla_tracking',
	'white_label',
];

// Re-export types for convenience
export type {
	FeatureCheckResult,
	FeatureInfo,
	LicenseFeature,
	LicenseInfo,
	LicenseTier,
	TierInfo,
};
