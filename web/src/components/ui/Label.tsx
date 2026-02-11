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
		<label
			className={`block text-sm font-medium text-gray-700 ${className}`}
			{...props}
		>
			{children}
			{required && <span className="ml-1 text-red-500">*</span>}
		</label>
	);
}
