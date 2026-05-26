import { describe, expect, it } from 'vitest';
import {
	PLAN_NAMES,
	PLAN_PRICING,
	generateContactSalesLink,
	generateUpgradeLink,
	getNextPlan,
	needsUpgrade,
	requiresSales,
} from './upgrade';

describe('generateUpgradeLink', () => {
	it('returns base path with no params', () => {
		expect(generateUpgradeLink()).toBe('/organization/license');
	});

	it('includes feature param', () => {
		expect(generateUpgradeLink({ feature: 'sso' })).toBe(
			'/organization/license?feature=sso',
		);
	});

	it('includes all params', () => {
		const url = generateUpgradeLink({
			feature: 'sso',
			source: 'banner',
			targetPlan: 'professional',
		});
		expect(url).toContain('feature=sso');
		expect(url).toContain('source=banner');
		expect(url).toContain('plan=professional');
	});
});

describe('generateContactSalesLink', () => {
	it('returns base path with no params', () => {
		expect(generateContactSalesLink()).toBe('/organization/license');
	});

	it('maps feature to interest param', () => {
		expect(generateContactSalesLink({ feature: 'sso' })).toBe(
			'/organization/license?interest=sso',
		);
	});

	it('includes source', () => {
		const url = generateContactSalesLink({
			feature: 'sso',
			source: 'modal',
		});
		expect(url).toContain('interest=sso');
		expect(url).toContain('source=modal');
	});
});

describe('needsUpgrade', () => {
	it('returns true when target is higher tier', () => {
		expect(needsUpgrade('free', 'professional')).toBe(true);
		expect(needsUpgrade('starter', 'enterprise')).toBe(true);
	});

	it('returns false when target is same or lower tier', () => {
		expect(needsUpgrade('professional', 'free')).toBe(false);
		expect(needsUpgrade('professional', 'professional')).toBe(false);
	});
});

describe('getNextPlan', () => {
	it('returns next tier in order', () => {
		expect(getNextPlan('free')).toBe('starter');
		expect(getNextPlan('starter')).toBe('professional');
		expect(getNextPlan('professional')).toBe('enterprise');
	});

	it('returns null at top tier', () => {
		expect(getNextPlan('enterprise')).toBeNull();
	});
});

describe('requiresSales', () => {
	it('returns true for enterprise only', () => {
		expect(requiresSales('enterprise')).toBe(true);
		expect(requiresSales('professional')).toBe(false);
		expect(requiresSales('starter')).toBe(false);
		expect(requiresSales('free')).toBe(false);
	});
});

describe('PLAN_NAMES + PLAN_PRICING', () => {
	it('PLAN_NAMES has entry for each tier', () => {
		expect(PLAN_NAMES.free).toBe('Free');
		expect(PLAN_NAMES.starter).toBe('Starter');
		expect(PLAN_NAMES.professional).toBe('Professional');
		expect(PLAN_NAMES.enterprise).toBe('Enterprise');
	});

	it('PLAN_PRICING enterprise tier says Contact Sales', () => {
		expect(PLAN_PRICING.enterprise).toBe('Contact Sales');
	});
});
