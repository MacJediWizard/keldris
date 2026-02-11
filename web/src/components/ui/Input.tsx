import type { InputHTMLAttributes } from 'react';

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
	label?: string;
	error?: string;
	helperText?: string;
}

export function Input({
	label,
	error,
	helperText,
	id,
	className = '',
	...props
}: InputProps) {
	const inputId = id || label?.toLowerCase().replace(/\s+/g, '-');

	return (
		<div>
			{label && (
				<label
					htmlFor={inputId}
					className="mb-1 block text-sm font-medium text-gray-700"
				>
					{label}
				</label>
			)}
			<input
				id={inputId}
				className={`block w-full rounded-md border px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-offset-0 ${
					error
						? 'border-red-300 text-red-900 placeholder-red-300 focus:border-red-500 focus:ring-red-500'
						: 'border-gray-300 text-gray-900 placeholder-gray-400 focus:border-indigo-500 focus:ring-indigo-500'
				} ${className}`}
				aria-invalid={error ? 'true' : undefined}
				aria-describedby={
					error
						? `${inputId}-error`
						: helperText
							? `${inputId}-helper`
							: undefined
				}
				{...props}
			/>
			{error && (
				<p id={`${inputId}-error`} className="mt-1 text-sm text-red-600">
					{error}
				</p>
			)}
			{!error && helperText && (
				<p id={`${inputId}-helper`} className="mt-1 text-sm text-gray-500">
					{helperText}
				</p>
			)}
		</div>
	);
}
