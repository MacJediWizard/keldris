import { renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createWrapper } from '../test/helpers';
import {
	FEATURE_BENEFITS,
	FEATURE_NAMES,
	FEATURE_REQUIRED_PLAN,
	usePlanLimits,
} from './usePlanLimits';

// Mock all three hooks that usePlanLimits depends on
vi.mock('./useAuth', () => ({
	useMe: vi.fn(),
}));

vi.mock('./useLicense', () => ({
	useLicense: vi.fn(),
}));

vi.mock('./useOrganizations', () => ({
	useCurrentOrganization: vi.fn(),
}));

import { useMe } from './useAuth';
import { useLicense } from './useLicense';
import { useCurrentOrganization } from './useOrganizations';

function mockHookDefaults(overrides?: {
	orgLoading?: boolean;
	userLoading?: boolean;
	licenseLoading?: boolean;
	licenseTier?: string;
	licenseFeatures?: string[];
}) {
	vi.mocked(useCurrentOrganization).mockReturnValue({
		isLoading: overrides?.orgLoading ?? false,
		data: { id: 'org-1', name: 'Test Org' },
	} as ReturnType<typeof useCurrentOrganization>);

	vi.mocked(useMe).mockReturnValue({
		isLoading: overrides?.userLoading ?? false,
		data: { id: 'user-1', email: 'test@example.com' },
	} as ReturnType<typeof useMe>);

	vi.mocked(useLicense).mockReturnValue({
		isLoading: overrides?.licenseLoading ?? false,
		data: {
			tier: overrides?.licenseTier ?? 'free',
			features: overrides?.licenseFeatures ?? [],
			customer_id: 'cust-1',
			expires_at: '2026-12-31T00:00:00Z',
			issued_at: '2026-01-01T00:00:00Z',
			limits: {},
			license_key_source: 'database' as const,
			is_trial: false,
		},
	} as ReturnType<typeof useLicense>);
}

describe('usePlanLimits', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	// --- Loading states ---

	describe('isLoading', () => {
		it('returns true when org is loading', () => {
			mockHookDefaults({ orgLoading: true });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.isLoading).toBe(true);
		});

		it('returns true when user is loading', () => {
			mockHookDefaults({ userLoading: true });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.isLoading).toBe(true);
		});

		it('returns true when license is loading', () => {
			mockHookDefaults({ licenseLoading: true });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.isLoading).toBe(true);
		});

		it('returns false when all are loaded', () => {
			mockHookDefaults();

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.isLoading).toBe(false);
		});
	});

	// --- Tier mapping ---

	describe('planType mapping', () => {
		it('maps undefined tier to free', () => {
			mockHookDefaults({ licenseTier: undefined });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.planType).toBe('free');
		});

		it('maps free tier to free plan', () => {
			mockHookDefaults({ licenseTier: 'free' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.planType).toBe('free');
		});

		it('maps pro tier to professional plan', () => {
			mockHookDefaults({ licenseTier: 'pro' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.planType).toBe('professional');
		});

		it('maps professional tier to professional plan', () => {
			mockHookDefaults({ licenseTier: 'professional' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.planType).toBe('professional');
		});

		it('maps enterprise tier to enterprise plan', () => {
			mockHookDefaults({ licenseTier: 'enterprise' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.planType).toBe('enterprise');
		});

		it('maps unknown tier to free plan', () => {
			mockHookDefaults({ licenseTier: 'unknown-tier' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.planType).toBe('free');
		});
	});

	// --- Plan limits per tier ---

	describe('limits for each plan tier', () => {
		it('returns free plan limits', () => {
			mockHookDefaults({ licenseTier: 'free' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.limits.agent_limit).toBe(3);
			expect(result.current.limits.server_limit).toBe(1);
			expect(result.current.limits.storage_quota_bytes).toBe(
				10 * 1024 * 1024 * 1024,
			);
			expect(result.current.limits.backup_retention_days).toBe(30);
			expect(result.current.limits.concurrent_backups).toBe(1);
		});

		it('returns professional plan limits', () => {
			mockHookDefaults({ licenseTier: 'professional' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.limits.agent_limit).toBe(100);
			expect(result.current.limits.server_limit).toBe(10);
			expect(result.current.limits.storage_quota_bytes).toBe(
				1024 * 1024 * 1024 * 1024,
			);
			expect(result.current.limits.backup_retention_days).toBe(365);
			expect(result.current.limits.concurrent_backups).toBe(10);
		});

		it('returns enterprise plan limits (all unlimited)', () => {
			mockHookDefaults({ licenseTier: 'enterprise' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.limits.agent_limit).toBeUndefined();
			expect(result.current.limits.server_limit).toBeUndefined();
			expect(result.current.limits.storage_quota_bytes).toBeUndefined();
			expect(result.current.limits.backup_retention_days).toBeUndefined();
			expect(result.current.limits.concurrent_backups).toBeUndefined();
		});
	});

	// --- Plan features per tier ---

	describe('features for each plan tier', () => {
		it('returns free plan features (all disabled)', () => {
			mockHookDefaults({ licenseTier: 'free' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.features.sso_enabled).toBe(false);
			expect(result.current.features.api_access).toBe(false);
			expect(result.current.features.advanced_reporting).toBe(false);
			expect(result.current.features.audit_logs).toBe(false);
			expect(result.current.features.custom_branding).toBe(false);
			expect(result.current.features.priority_support).toBe(false);
			expect(result.current.features.geo_replication).toBe(false);
			expect(result.current.features.lifecycle_policies).toBe(false);
			expect(result.current.features.legal_holds).toBe(false);
		});

		it('returns professional plan features', () => {
			mockHookDefaults({ licenseTier: 'professional' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.features.sso_enabled).toBe(true);
			expect(result.current.features.api_access).toBe(true);
			expect(result.current.features.advanced_reporting).toBe(true);
			expect(result.current.features.audit_logs).toBe(true);
			expect(result.current.features.custom_branding).toBe(false);
			expect(result.current.features.priority_support).toBe(false);
		});

		it('returns enterprise plan features (all enabled)', () => {
			mockHookDefaults({ licenseTier: 'enterprise' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.features.sso_enabled).toBe(true);
			expect(result.current.features.api_access).toBe(true);
			expect(result.current.features.advanced_reporting).toBe(true);
			expect(result.current.features.audit_logs).toBe(true);
			expect(result.current.features.custom_branding).toBe(true);
			expect(result.current.features.priority_support).toBe(true);
			expect(result.current.features.geo_replication).toBe(true);
			expect(result.current.features.lifecycle_policies).toBe(true);
			expect(result.current.features.legal_holds).toBe(true);
		});
	});

	// --- hasFeature ---

	describe('hasFeature', () => {
		it('returns false for sso_enabled on free plan (static fallback)', () => {
			mockHookDefaults({ licenseTier: 'free', licenseFeatures: [] });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.hasFeature('sso_enabled')).toBe(false);
		});

		it('returns true for sso_enabled on professional plan (static fallback)', () => {
			mockHookDefaults({
				licenseTier: 'professional',
				licenseFeatures: [],
			});

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.hasFeature('sso_enabled')).toBe(true);
		});

		it('checks backend feature list when available', () => {
			mockHookDefaults({
				licenseTier: 'free',
				licenseFeatures: ['oidc', 'api_access'],
			});

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			// sso_enabled maps to ['oidc', 'sso_sync'] - oidc is in the list
			expect(result.current.hasFeature('sso_enabled')).toBe(true);
			// api_access maps to ['api_access']
			expect(result.current.hasFeature('api_access')).toBe(true);
			// audit_logs not in the license features
			expect(result.current.hasFeature('audit_logs')).toBe(false);
		});

		it('returns true for sso_enabled when sso_sync is in backend features', () => {
			mockHookDefaults({
				licenseTier: 'free',
				licenseFeatures: ['sso_sync'],
			});

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.hasFeature('sso_enabled')).toBe(true);
		});

		it('maps custom_branding to white_label backend feature', () => {
			mockHookDefaults({
				licenseTier: 'free',
				licenseFeatures: ['white_label'],
			});

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.hasFeature('custom_branding')).toBe(true);
		});

		it('maps lifecycle_policies to custom_retention backend feature', () => {
			mockHookDefaults({
				licenseTier: 'free',
				licenseFeatures: ['custom_retention'],
			});

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.hasFeature('lifecycle_policies')).toBe(true);
		});

		it('maps advanced_reporting to custom_reports backend feature', () => {
			mockHookDefaults({
				licenseTier: 'free',
				licenseFeatures: ['custom_reports'],
			});

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.hasFeature('advanced_reporting')).toBe(true);
		});
	});

	// --- canAddAgents ---

	describe('canAddAgents', () => {
		it('returns true when under limit (free plan, usage 0)', () => {
			mockHookDefaults({ licenseTier: 'free' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			// Free plan has agent_limit=3, usage.agentCount=0
			expect(result.current.canAddAgents()).toBe(true);
			expect(result.current.canAddAgents(3)).toBe(true);
		});

		it('returns false when adding would exceed limit', () => {
			mockHookDefaults({ licenseTier: 'free' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			// Free plan has agent_limit=3, trying to add 4
			expect(result.current.canAddAgents(4)).toBe(false);
		});

		it('returns true for enterprise plan (unlimited)', () => {
			mockHookDefaults({ licenseTier: 'enterprise' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.canAddAgents(1000)).toBe(true);
		});

		it('defaults to count=1 when no argument', () => {
			mockHookDefaults({ licenseTier: 'free' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.canAddAgents()).toBe(true);
		});
	});

	// --- canAddStorage ---

	describe('canAddStorage', () => {
		it('returns true when under storage limit', () => {
			mockHookDefaults({ licenseTier: 'free' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			// Free: 10 GB, usage 0
			expect(result.current.canAddStorage(1024)).toBe(true);
		});

		it('returns false when adding would exceed storage limit', () => {
			mockHookDefaults({ licenseTier: 'free' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			// Free: 10 GB, trying to add 11 GB
			const elevenGB = 11 * 1024 * 1024 * 1024;
			expect(result.current.canAddStorage(elevenGB)).toBe(false);
		});

		it('returns true for enterprise plan (unlimited storage)', () => {
			mockHookDefaults({ licenseTier: 'enterprise' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			const petabyte = 1024 * 1024 * 1024 * 1024 * 1024;
			expect(result.current.canAddStorage(petabyte)).toBe(true);
		});
	});

	// --- isAtAgentLimit ---

	describe('isAtAgentLimit', () => {
		it('returns false when under limit (usage is 0)', () => {
			mockHookDefaults({ licenseTier: 'free' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			// Free plan, 0 agents used, limit is 3
			expect(result.current.isAtAgentLimit()).toBe(false);
		});

		it('returns false for enterprise plan (unlimited)', () => {
			mockHookDefaults({ licenseTier: 'enterprise' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.isAtAgentLimit()).toBe(false);
		});
	});

	// --- isAtStorageLimit ---

	describe('isAtStorageLimit', () => {
		it('returns false when under limit (usage is 0)', () => {
			mockHookDefaults({ licenseTier: 'free' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.isAtStorageLimit()).toBe(false);
		});

		it('returns false for enterprise plan (unlimited)', () => {
			mockHookDefaults({ licenseTier: 'enterprise' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.isAtStorageLimit()).toBe(false);
		});
	});

	// --- getAgentLimitRemaining ---

	describe('getAgentLimitRemaining', () => {
		it('returns remaining count for free plan', () => {
			mockHookDefaults({ licenseTier: 'free' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			// Free plan: limit=3, usage=0
			expect(result.current.getAgentLimitRemaining()).toBe(3);
		});

		it('returns undefined for enterprise plan (unlimited)', () => {
			mockHookDefaults({ licenseTier: 'enterprise' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.getAgentLimitRemaining()).toBeUndefined();
		});

		it('returns professional plan remaining', () => {
			mockHookDefaults({ licenseTier: 'professional' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			// Professional: limit=100, usage=0
			expect(result.current.getAgentLimitRemaining()).toBe(100);
		});
	});

	// --- getStorageLimitRemaining ---

	describe('getStorageLimitRemaining', () => {
		it('returns remaining bytes for free plan', () => {
			mockHookDefaults({ licenseTier: 'free' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			// Free plan: 10 GB, usage=0
			expect(result.current.getStorageLimitRemaining()).toBe(
				10 * 1024 * 1024 * 1024,
			);
		});

		it('returns undefined for enterprise plan (unlimited)', () => {
			mockHookDefaults({ licenseTier: 'enterprise' });

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.getStorageLimitRemaining()).toBeUndefined();
		});
	});

	// --- getUpgradePlanFor ---

	describe('getUpgradePlanFor', () => {
		it('returns starter for agents feature', () => {
			mockHookDefaults();

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.getUpgradePlanFor('agents')).toBe('starter');
		});

		it('returns professional for oidc feature', () => {
			mockHookDefaults();

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.getUpgradePlanFor('oidc')).toBe('professional');
		});

		it('returns enterprise for white_label feature', () => {
			mockHookDefaults();

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.getUpgradePlanFor('white_label')).toBe(
				'enterprise',
			);
		});

		it('returns enterprise for geo_replication feature', () => {
			mockHookDefaults();

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.getUpgradePlanFor('geo_replication')).toBe(
				'enterprise',
			);
		});
	});

	// --- usage ---

	describe('usage', () => {
		it('returns zero usage by default', () => {
			mockHookDefaults();

			const { result } = renderHook(() => usePlanLimits(), {
				wrapper: createWrapper(),
			});

			expect(result.current.usage.agentCount).toBe(0);
			expect(result.current.usage.storageUsedBytes).toBe(0);
		});
	});
});

// --- Static export tests ---

describe('FEATURE_NAMES', () => {
	it('contains a name for every UpgradeFeature key in FEATURE_REQUIRED_PLAN', () => {
		for (const key of Object.keys(FEATURE_REQUIRED_PLAN)) {
			expect(FEATURE_NAMES[key as keyof typeof FEATURE_NAMES]).toBeDefined();
		}
	});

	it('returns human-readable strings', () => {
		expect(FEATURE_NAMES.oidc).toBe('OIDC Authentication');
		expect(FEATURE_NAMES.agents).toBe('More Agents');
		expect(FEATURE_NAMES.white_label).toBe('White-Label Branding');
	});
});

describe('FEATURE_BENEFITS', () => {
	it('contains benefits for every UpgradeFeature key in FEATURE_REQUIRED_PLAN', () => {
		for (const key of Object.keys(FEATURE_REQUIRED_PLAN)) {
			const benefits = FEATURE_BENEFITS[key as keyof typeof FEATURE_BENEFITS];
			expect(benefits).toBeDefined();
			expect(Array.isArray(benefits)).toBe(true);
			expect(benefits.length).toBeGreaterThanOrEqual(1);
		}
	});
});

describe('FEATURE_REQUIRED_PLAN', () => {
	it('maps all pro-tier features to professional', () => {
		const proFeatures = [
			'oidc',
			'api_access',
			'audit_logs',
			'notification_slack',
			'notification_teams',
			'notification_pagerduty',
			'notification_discord',
			'storage_s3',
			'storage_b2',
			'storage_sftp',
			'storage_dropbox',
			'storage_rest',
			'docker_backup',
			'multi_repo',
			'custom_reports',
			'custom_retention',
		] as const;

		for (const feature of proFeatures) {
			expect(FEATURE_REQUIRED_PLAN[feature]).toBe('professional');
		}
	});

	it('maps all enterprise-tier features to enterprise', () => {
		const enterpriseFeatures = [
			'white_label',
			'multi_org',
			'sla_tracking',
			'air_gap',
			'dr_runbooks',
			'dr_tests',
			'sso_sync',
			'rbac',
			'geo_replication',
			'ransomware_protection',
			'legal_holds',
			'priority_support',
		] as const;

		for (const feature of enterpriseFeatures) {
			expect(FEATURE_REQUIRED_PLAN[feature]).toBe('enterprise');
		}
	});

	it('maps limit-based features to starter', () => {
		expect(FEATURE_REQUIRED_PLAN.agents).toBe('starter');
		expect(FEATURE_REQUIRED_PLAN.storage).toBe('starter');
	});
});
