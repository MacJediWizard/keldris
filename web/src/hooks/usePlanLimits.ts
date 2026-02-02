import { useMemo } from 'react';
import type {
	PlanFeatures,
	PlanLimits,
	PlanType,
	UpgradeFeature,
} from '../lib/types';
import { useMe } from './useAuth';
import { useCurrentOrganization } from './useOrganizations';

// Default plan configurations
const PLAN_LIMITS: Record<PlanType, PlanLimits> = {
	free: {
		agent_limit: 3,
		storage_quota_bytes: 10 * 1024 * 1024 * 1024, // 10 GB
		backup_retention_days: 30,
		concurrent_backups: 1,
	},
	starter: {
		agent_limit: 10,
		storage_quota_bytes: 100 * 1024 * 1024 * 1024, // 100 GB
		backup_retention_days: 90,
		concurrent_backups: 3,
	},
	professional: {
		agent_limit: 50,
		storage_quota_bytes: 1024 * 1024 * 1024 * 1024, // 1 TB
		backup_retention_days: 365,
		concurrent_backups: 10,
	},
	enterprise: {
		agent_limit: undefined, // Unlimited
		storage_quota_bytes: undefined, // Unlimited
		backup_retention_days: undefined, // Unlimited
		concurrent_backups: undefined, // Unlimited
	},
};

const PLAN_FEATURES: Record<PlanType, PlanFeatures> = {
	free: {
		sso_enabled: false,
		api_access: false,
		advanced_reporting: false,
		audit_logs: false,
		custom_branding: false,
		priority_support: false,
		geo_replication: false,
		lifecycle_policies: false,
		legal_holds: false,
	},
	starter: {
		sso_enabled: false,
		api_access: true,
		advanced_reporting: false,
		audit_logs: false,
		custom_branding: false,
		priority_support: false,
		geo_replication: false,
		lifecycle_policies: false,
		legal_holds: false,
	},
	professional: {
		sso_enabled: true,
		api_access: true,
		advanced_reporting: true,
		audit_logs: true,
		custom_branding: true,
		priority_support: false,
		geo_replication: false,
		lifecycle_policies: true,
		legal_holds: false,
	},
	enterprise: {
		sso_enabled: true,
		api_access: true,
		advanced_reporting: true,
		audit_logs: true,
		custom_branding: true,
		priority_support: true,
		geo_replication: true,
		lifecycle_policies: true,
		legal_holds: true,
	},
};

// Feature names for display
export const FEATURE_NAMES: Record<UpgradeFeature, string> = {
	agents: 'More Agents',
	storage: 'More Storage',
	sso: 'Single Sign-On (SSO)',
	api_access: 'API Access',
	advanced_reporting: 'Advanced Reporting',
	audit_logs: 'Audit Logs',
	custom_branding: 'Custom Branding',
	priority_support: 'Priority Support',
	geo_replication: 'Geo-Replication',
	lifecycle_policies: 'Lifecycle Policies',
	legal_holds: 'Legal Holds',
};

// Features benefits for upgrade prompts
export const FEATURE_BENEFITS: Record<UpgradeFeature, string[]> = {
	agents: [
		'Connect more backup agents',
		'Scale your backup infrastructure',
		'Manage distributed systems',
	],
	storage: [
		'Store more backup data',
		'Longer retention periods',
		'Archive more versions',
	],
	sso: [
		'Centralized authentication',
		'Automatic user provisioning',
		'Enhanced security compliance',
	],
	api_access: [
		'Automate backup operations',
		'Build custom integrations',
		'Programmatic access to data',
	],
	advanced_reporting: [
		'Detailed backup analytics',
		'Custom report generation',
		'Trend analysis and insights',
	],
	audit_logs: [
		'Track all user actions',
		'Compliance reporting',
		'Security monitoring',
	],
	custom_branding: [
		'White-label the interface',
		'Custom logo and colors',
		'Professional appearance',
	],
	priority_support: [
		'24/7 dedicated support',
		'Faster response times',
		'Direct access to engineers',
	],
	geo_replication: [
		'Cross-region redundancy',
		'Disaster recovery',
		'Lower latency worldwide',
	],
	lifecycle_policies: [
		'Automated retention rules',
		'Cost optimization',
		'Storage tier management',
	],
	legal_holds: [
		'Preserve data for litigation',
		'Compliance requirements',
		'Prevent accidental deletion',
	],
};

// Plan that unlocks each feature
export const FEATURE_REQUIRED_PLAN: Record<UpgradeFeature, PlanType> = {
	agents: 'starter',
	storage: 'starter',
	sso: 'professional',
	api_access: 'starter',
	advanced_reporting: 'professional',
	audit_logs: 'professional',
	custom_branding: 'professional',
	priority_support: 'enterprise',
	geo_replication: 'enterprise',
	lifecycle_policies: 'professional',
	legal_holds: 'enterprise',
};

export interface UsePlanLimitsResult {
	isLoading: boolean;
	planType: PlanType;
	limits: PlanLimits;
	features: PlanFeatures;
	usage: {
		agentCount: number;
		storageUsedBytes: number;
	};
	hasFeature: (feature: keyof PlanFeatures) => boolean;
	canAddAgents: (count?: number) => boolean;
	canAddStorage: (bytes: number) => boolean;
	getAgentLimitRemaining: () => number | undefined;
	getStorageLimitRemaining: () => number | undefined;
	isAtAgentLimit: () => boolean;
	isAtStorageLimit: () => boolean;
	getUpgradePlanFor: (feature: UpgradeFeature) => PlanType;
}

export function usePlanLimits(): UsePlanLimitsResult {
	const { isLoading: orgLoading } = useCurrentOrganization();
	const { isLoading: userLoading } = useMe();

	const isLoading = orgLoading || userLoading;

	// Default to free plan if no data is available
	// In a real implementation, this would come from the organization data
	// via an API call that returns the plan info
	const planType: PlanType = useMemo(() => {
		// This would typically come from org.organization.plan_type
		// For now, return 'free' as default
		return 'free';
	}, []);

	const limits = useMemo(() => PLAN_LIMITS[planType], [planType]);
	const features = useMemo(() => PLAN_FEATURES[planType], [planType]);

	// Mock usage data - in reality this would come from the API
	const usage = useMemo(
		() => ({
			agentCount: 0,
			storageUsedBytes: 0,
		}),
		[],
	);

	const hasFeature = (feature: keyof PlanFeatures): boolean => {
		return features[feature];
	};

	const canAddAgents = (count = 1): boolean => {
		if (limits.agent_limit === undefined) return true; // Unlimited
		return usage.agentCount + count <= limits.agent_limit;
	};

	const canAddStorage = (bytes: number): boolean => {
		if (limits.storage_quota_bytes === undefined) return true; // Unlimited
		return usage.storageUsedBytes + bytes <= limits.storage_quota_bytes;
	};

	const getAgentLimitRemaining = (): number | undefined => {
		if (limits.agent_limit === undefined) return undefined;
		return Math.max(0, limits.agent_limit - usage.agentCount);
	};

	const getStorageLimitRemaining = (): number | undefined => {
		if (limits.storage_quota_bytes === undefined) return undefined;
		return Math.max(0, limits.storage_quota_bytes - usage.storageUsedBytes);
	};

	const isAtAgentLimit = (): boolean => {
		if (limits.agent_limit === undefined) return false;
		return usage.agentCount >= limits.agent_limit;
	};

	const isAtStorageLimit = (): boolean => {
		if (limits.storage_quota_bytes === undefined) return false;
		return usage.storageUsedBytes >= limits.storage_quota_bytes;
	};

	const getUpgradePlanFor = (feature: UpgradeFeature): PlanType => {
		return FEATURE_REQUIRED_PLAN[feature];
	};

	return {
		isLoading,
		planType,
		limits,
		features,
		usage,
		hasFeature,
		canAddAgents,
		canAddStorage,
		getAgentLimitRemaining,
		getStorageLimitRemaining,
		isAtAgentLimit,
		isAtStorageLimit,
		getUpgradePlanFor,
	};
}
