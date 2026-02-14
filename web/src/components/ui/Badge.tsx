import type { ReactNode } from 'react';

const variants = {
	default: 'bg-gray-100 text-gray-800',
	success: 'bg-green-100 text-green-800',
	warning: 'bg-yellow-100 text-yellow-800',
	error: 'bg-red-100 text-red-800',
	info: 'bg-blue-100 text-blue-800',
} as const;

const sizes = {
	sm: 'px-2 py-0.5 text-xs',
	md: 'px-2.5 py-0.5 text-sm',
} as const;

interface BadgeProps {
	variant?: keyof typeof variants;
	size?: keyof typeof sizes;
	children: ReactNode;
}

export function Badge({ variant = 'default', size = 'sm', children }: BadgeProps) {
	return (
		<span
			className={`inline-flex items-center rounded-full font-medium ${variants[variant]} ${sizes[size]}`}
		>
			{children}
		</span>
	);
}
