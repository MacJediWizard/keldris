import type { LicenseTier } from '../../lib/types';

interface TierBadgeProps {
	tier: LicenseTier;
	className?: string;
}

const tierStyles: Record<LicenseTier, string> = {
	free: 'bg-gray-100 text-gray-700',
	pro: 'bg-blue-100 text-blue-700',
	professional: 'bg-indigo-100 text-indigo-700',
	enterprise: 'bg-purple-100 text-purple-700',
};

const tierLabels: Record<LicenseTier, string> = {
	free: 'Free',
	pro: 'Pro',
	professional: 'Professional',
	enterprise: 'Enterprise',
};

export function TierBadge({ tier, className = '' }: TierBadgeProps) {
	return (
		<span
			className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${tierStyles[tier]} ${className}`}
		>
			{tierLabels[tier]}
		</span>
	);
}
