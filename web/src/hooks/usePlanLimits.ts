import { useMemo } from 'react';
import type {
	LicenseTier,
	PlanFeatures,
	PlanLimits,
	PlanType,
	UpgradeFeature,
} from '../lib/types';
import { useMe } from './useAuth';
import { useLicense } from './useLicense';
import { useCurrentOrganization } from './useOrganizations';

// Map backend tier names to frontend plan types.
// Backend uses "pro", frontend uses "professional".
function tierToPlan(tier: LicenseTier | undefined): PlanType {
	switch (tier) {
		case 'enterprise':
			return 'enterprise';
		case 'pro':
		case 'professional':
			return 'professional';
		default:
			return 'free';
	}
}

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

// Feature names for display — must cover every backend feature string
// so that 402 responses render correctly in UpgradePrompt.
export const FEATURE_NAMES: Record<UpgradeFeature, string> = {
	// Frontend-only (limit-based)
	agents: 'More Agents',
	storage: 'More Storage',
	// Pro tier features (backend names)
	oidc: 'OIDC Authentication',
	api_access: 'API Access',
	audit_logs: 'Audit Logs',
	notification_slack: 'Slack Notifications',
	notification_teams: 'Teams Notifications',
	notification_pagerduty: 'PagerDuty Notifications',
	notification_discord: 'Discord Notifications',
	storage_s3: 'S3 Storage',
	storage_b2: 'Backblaze B2 Storage',
	storage_sftp: 'SFTP Storage',
	docker_backup: 'Docker Backup',
	multi_repo: 'Multiple Repositories',
	custom_reports: 'Custom Reports',
	custom_retention: 'Custom Retention',
	// Enterprise tier features (backend names)
	white_label: 'White-Label Branding',
	multi_org: 'Multiple Organizations',
	sla_tracking: 'SLA Tracking',
	air_gap: 'Air-Gapped Deployment',
	dr_runbooks: 'DR Runbooks',
	dr_tests: 'DR Testing',
	sso_sync: 'SSO Directory Sync',
	rbac: 'Role-Based Access Control',
	geo_replication: 'Geo-Replication',
	ransomware_protection: 'Ransomware Protection',
	legal_holds: 'Legal Holds',
	priority_support: 'Priority Support',
	// Frontend aliases
	sso: 'Single Sign-On (SSO)',
	advanced_reporting: 'Advanced Reporting',
	custom_branding: 'Custom Branding',
	lifecycle_policies: 'Lifecycle Policies',
};

// Features benefits for upgrade prompts
export const FEATURE_BENEFITS: Record<UpgradeFeature, string[]> = {
	// Frontend-only (limit-based)
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
	// Pro tier features
	oidc: [
		'Centralized authentication',
		'OpenID Connect providers',
		'Enhanced security compliance',
	],
	api_access: [
		'Automate backup operations',
		'Build custom integrations',
		'Programmatic access to data',
	],
	audit_logs: [
		'Track all user actions',
		'Compliance reporting',
		'Security monitoring',
	],
	notification_slack: [
		'Real-time Slack alerts',
		'Channel-based notifications',
		'Custom notification rules',
	],
	notification_teams: [
		'Microsoft Teams alerts',
		'Channel-based notifications',
		'Custom notification rules',
	],
	notification_pagerduty: [
		'PagerDuty incident alerts',
		'On-call routing',
		'Escalation policies',
	],
	notification_discord: [
		'Discord channel alerts',
		'Real-time notifications',
		'Custom notification rules',
	],
	storage_s3: [
		'S3-compatible storage',
		'Cloud backup destinations',
		'Cross-region storage',
	],
	storage_b2: [
		'Backblaze B2 storage',
		'Cost-effective cloud backup',
		'Unlimited scalability',
	],
	storage_sftp: [
		'SFTP server destinations',
		'Remote backup storage',
		'Flexible storage options',
	],
	docker_backup: [
		'Docker container backups',
		'Volume snapshot support',
		'Container-aware protection',
	],
	multi_repo: [
		'Multiple backup repositories',
		'Storage redundancy',
		'Flexible backup routing',
	],
	custom_reports: [
		'Custom report generation',
		'Scheduled report delivery',
		'Advanced analytics',
	],
	custom_retention: [
		'Custom retention policies',
		'Flexible scheduling',
		'Cost optimization',
	],
	// Enterprise tier features
	white_label: [
		'White-label the interface',
		'Custom logo and colors',
		'Professional appearance',
	],
	multi_org: [
		'Multiple organizations',
		'Centralized management',
		'Tenant isolation',
	],
	sla_tracking: [
		'SLA monitoring',
		'Compliance reporting',
		'Automated tracking',
	],
	air_gap: [
		'Air-gapped deployment',
		'No internet required',
		'Offline license validation',
	],
	dr_runbooks: [
		'DR runbook automation',
		'Recovery workflows',
		'Step-by-step procedures',
	],
	dr_tests: [
		'Automated DR testing',
		'Recovery validation',
		'Compliance verification',
	],
	sso_sync: [
		'Directory synchronization',
		'Automated user provisioning',
		'Group-based access',
	],
	rbac: [
		'Fine-grained permissions',
		'Custom roles',
		'Team-based access control',
	],
	geo_replication: [
		'Cross-region redundancy',
		'Disaster recovery',
		'Lower latency worldwide',
	],
	ransomware_protection: [
		'Immutable backups',
		'Anomaly detection',
		'Recovery assurance',
	],
	legal_holds: [
		'Preserve data for litigation',
		'Compliance requirements',
		'Prevent accidental deletion',
	],
	priority_support: [
		'24/7 dedicated support',
		'Faster response times',
		'Direct access to engineers',
	],
	// Frontend aliases
	sso: [
		'Centralized authentication',
		'Automatic user provisioning',
		'Enhanced security compliance',
	],
	advanced_reporting: [
		'Detailed backup analytics',
		'Custom report generation',
		'Trend analysis and insights',
	],
	custom_branding: [
		'White-label the interface',
		'Custom logo and colors',
		'Professional appearance',
	],
	lifecycle_policies: [
		'Automated retention rules',
		'Cost optimization',
		'Storage tier management',
	],
};

// Plan that unlocks each feature — must match backend tier mapping.
// Backend "pro" tier = frontend "professional" plan.
export const FEATURE_REQUIRED_PLAN: Record<UpgradeFeature, PlanType> = {
	// Frontend-only (limit-based)
	agents: 'starter',
	storage: 'starter',
	// Pro tier features → "professional"
	oidc: 'professional',
	api_access: 'professional',
	audit_logs: 'professional',
	notification_slack: 'professional',
	notification_teams: 'professional',
	notification_pagerduty: 'professional',
	notification_discord: 'professional',
	storage_s3: 'professional',
	storage_b2: 'professional',
	storage_sftp: 'professional',
	docker_backup: 'professional',
	multi_repo: 'professional',
	custom_reports: 'professional',
	custom_retention: 'professional',
	// Enterprise tier features → "enterprise"
	white_label: 'enterprise',
	multi_org: 'enterprise',
	sla_tracking: 'enterprise',
	air_gap: 'enterprise',
	dr_runbooks: 'enterprise',
	dr_tests: 'enterprise',
	sso_sync: 'enterprise',
	rbac: 'enterprise',
	geo_replication: 'enterprise',
	ransomware_protection: 'enterprise',
	legal_holds: 'enterprise',
	priority_support: 'enterprise',
	// Frontend aliases
	sso: 'professional',
	advanced_reporting: 'professional',
	custom_branding: 'enterprise',
	lifecycle_policies: 'professional',
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
	const { data: license, isLoading: licenseLoading } = useLicense();

	const isLoading = orgLoading || userLoading || licenseLoading;

	// Read the actual tier from the license and map to frontend plan type.
	const planType: PlanType = useMemo(() => {
		return tierToPlan(license?.tier);
	}, [license?.tier]);

	const limits = useMemo(() => PLAN_LIMITS[planType], [planType]);
	const features = useMemo(() => PLAN_FEATURES[planType], [planType]);

	// Use actual license feature list from backend when available
	const licenseFeatures = useMemo(
		() => new Set(license?.features ?? []),
		[license?.features],
	);

	const usage = useMemo(
		() => ({
			agentCount: 0,
			storageUsedBytes: 0,
		}),
		[],
	);

	const hasFeature = (feature: keyof PlanFeatures): boolean => {
		// If we have a license with features, check the backend list directly.
		// This handles cases where the backend feature names don't match
		// the PlanFeatures keys (e.g. "legal_holds" in backend features array).
		if (licenseFeatures.size > 0) {
			// Map PlanFeatures key to backend feature name
			const featureMap: Record<keyof PlanFeatures, string[]> = {
				sso_enabled: ['oidc', 'sso_sync'],
				api_access: ['api_access'],
				advanced_reporting: ['custom_reports'],
				audit_logs: ['audit_logs'],
				custom_branding: ['white_label'],
				priority_support: ['priority_support'],
				geo_replication: ['geo_replication'],
				lifecycle_policies: ['custom_retention'],
				legal_holds: ['legal_holds'],
			};
			const backendNames = featureMap[feature] ?? [feature];
			return backendNames.some((name) => licenseFeatures.has(name));
		}
		// Fallback to static plan features
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
