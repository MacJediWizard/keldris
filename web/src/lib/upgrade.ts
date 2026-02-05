import type { PlanType, UpgradeFeature } from './types';

export interface UpgradeLinkOptions {
	feature?: UpgradeFeature;
	source?: string;
	targetPlan?: PlanType;
}

/**
 * Generates an upgrade link with optional feature and source tracking
 */
export function generateUpgradeLink(options: UpgradeLinkOptions = {}): string {
	const { feature, source, targetPlan } = options;
	const params = new URLSearchParams();

	if (feature) {
		params.set('feature', feature);
	}

	if (source) {
		params.set('source', source);
	}

	if (targetPlan) {
		params.set('plan', targetPlan);
	}

	const queryString = params.toString();
	return `/organization/billing${queryString ? `?${queryString}` : ''}`;
}

/**
 * Generates a contact sales link for enterprise features
 */
export function generateContactSalesLink(
	options: UpgradeLinkOptions = {},
): string {
	const { feature, source } = options;
	const params = new URLSearchParams();

	if (feature) {
		params.set('interest', feature);
	}

	if (source) {
		params.set('source', source);
	}

	const queryString = params.toString();
	return `/organization/contact-sales${queryString ? `?${queryString}` : ''}`;
}

/**
 * Plan display names
 */
export const PLAN_NAMES: Record<PlanType, string> = {
	free: 'Free',
	starter: 'Starter',
	professional: 'Professional',
	enterprise: 'Enterprise',
};

/**
 * Plan descriptions
 */
export const PLAN_DESCRIPTIONS: Record<PlanType, string> = {
	free: 'For individuals and small projects',
	starter: 'For growing teams with basic needs',
	professional: 'For organizations needing advanced features',
	enterprise: 'For large organizations with custom requirements',
};

/**
 * Plan pricing (display strings)
 */
export const PLAN_PRICING: Record<PlanType, string> = {
	free: 'Free',
	starter: '$29/month',
	professional: '$99/month',
	enterprise: 'Contact Sales',
};

/**
 * Check if upgrade is needed to get a plan
 */
export function needsUpgrade(
	currentPlan: PlanType,
	targetPlan: PlanType,
): boolean {
	const planOrder: PlanType[] = [
		'free',
		'starter',
		'professional',
		'enterprise',
	];
	const currentIndex = planOrder.indexOf(currentPlan);
	const targetIndex = planOrder.indexOf(targetPlan);
	return targetIndex > currentIndex;
}

/**
 * Get the next plan upgrade from current
 */
export function getNextPlan(currentPlan: PlanType): PlanType | null {
	const planOrder: PlanType[] = [
		'free',
		'starter',
		'professional',
		'enterprise',
	];
	const currentIndex = planOrder.indexOf(currentPlan);
	if (currentIndex === -1 || currentIndex >= planOrder.length - 1) {
		return null;
	}
	return planOrder[currentIndex + 1];
}

/**
 * Check if a plan requires contacting sales
 */
export function requiresSales(plan: PlanType): boolean {
	return plan === 'enterprise';
}
