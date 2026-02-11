import type { LabelHTMLAttributes, ReactNode } from 'react';

interface LabelProps extends LabelHTMLAttributes<HTMLLabelElement> {
	children: ReactNode;
	required?: boolean;
}

export function Label({
	children,
	required,
	className = '',
	...props
}: LabelProps) {
	return (
		// biome-ignore lint/a11y/noLabelWithoutControl: Label is a reusable primitive; htmlFor is passed via props
		<label
			className={`block text-sm font-medium text-gray-700 ${className}`}
			{...props}
		>
			{children}
			{required && <span className="ml-1 text-red-500">*</span>}
		</label>
	);
}
