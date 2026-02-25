import type { SelectHTMLAttributes } from 'react';

interface SelectOption {
	value: string;
	label: string;
	disabled?: boolean;
}

interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
	label?: string;
	options: SelectOption[];
	error?: string;
	placeholder?: string;
}

export function Select({
	label,
	options,
	error,
	placeholder,
	id,
	className = '',
	...props
}: SelectProps) {
	const selectId = id || label?.toLowerCase().replace(/\s+/g, '-');

	return (
		<div>
			{label && (
				<label
					htmlFor={selectId}
					className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300"
				>
					{label}
				</label>
			)}
			<select
				id={selectId}
				className={`block w-full rounded-md border px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-offset-0 ${
					error
						? 'border-red-300 focus:border-red-500 focus:ring-red-500 dark:border-red-600 dark:bg-gray-800'
						: 'border-gray-300 focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-800 dark:text-white'
				} ${className}`}
				aria-invalid={error ? 'true' : undefined}
				aria-describedby={error ? `${selectId}-error` : undefined}
				{...props}
			>
				{placeholder && (
					<option value="" disabled>
						{placeholder}
					</option>
				)}
				{options.map((option) => (
					<option
						key={option.value}
						value={option.value}
						disabled={option.disabled}
					>
						{option.label}
					</option>
				))}
			</select>
			{error && (
				<p id={`${selectId}-error`} className="mt-1 text-sm text-red-600 dark:text-red-400">
					{error}
				</p>
			)}
		</div>
	);
}
