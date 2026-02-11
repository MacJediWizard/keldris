import type { ButtonHTMLAttributes, ReactNode } from 'react';

const variants = {
	primary:
		'bg-indigo-600 text-white hover:bg-indigo-700 focus:ring-indigo-500',
	secondary:
		'bg-gray-100 text-gray-900 hover:bg-gray-200 focus:ring-gray-500',
	danger: 'bg-red-600 text-white hover:bg-red-700 focus:ring-red-500',
	outline:
		'border border-gray-300 bg-white text-gray-700 hover:bg-gray-50 focus:ring-indigo-500',
} as const;

const sizes = {
	sm: 'px-3 py-1.5 text-sm',
	md: 'px-4 py-2 text-sm',
	lg: 'px-6 py-3 text-base',
} as const;

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
	variant?: keyof typeof variants;
	size?: keyof typeof sizes;
	loading?: boolean;
	children: ReactNode;
}

export function Button({
	variant = 'primary',
	size = 'md',
	loading = false,
	disabled,
	children,
	className = '',
	...props
}: ButtonProps) {
	return (
		<button
			type="button"
			disabled={disabled || loading}
			className={`inline-flex items-center justify-center rounded-md font-medium focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 ${variants[variant]} ${sizes[size]} ${className}`}
			{...props}
		>
			{loading && (
				<svg
					className="-ml-1 mr-2 h-4 w-4 animate-spin"
					fill="none"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<circle
						className="opacity-25"
						cx="12"
						cy="12"
						r="10"
						stroke="currentColor"
						strokeWidth="4"
					/>
					<path
						className="opacity-75"
						fill="currentColor"
						d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
					/>
				</svg>
			)}
			{children}
		</button>
	);
}
