import type { HTMLAttributes, ReactNode } from 'react';

interface CardProps extends HTMLAttributes<HTMLDivElement> {
	children: ReactNode;
}

export function Card({ children, className = '', ...props }: CardProps) {
	return (
		<div
			className={`rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 shadow-sm ${className}`}
			{...props}
		>
			{children}
		</div>
	);
}

interface CardHeaderProps extends HTMLAttributes<HTMLDivElement> {
	children: ReactNode;
}

export function CardHeader({
	children,
	className = '',
	...props
}: CardHeaderProps) {
	return (
		<div
			className={`border-b border-gray-200 dark:border-gray-700 px-6 py-4 ${className}`}
			{...props}
		>
			{children}
		</div>
	);
}

interface CardContentProps extends HTMLAttributes<HTMLDivElement> {
	children: ReactNode;
}

export function CardContent({
	children,
	className = '',
	...props
}: CardContentProps) {
	return (
		<div className={`px-6 py-4 ${className}`} {...props}>
			{children}
		</div>
	);
}

interface CardFooterProps extends HTMLAttributes<HTMLDivElement> {
	children: ReactNode;
}

export function CardFooter({
	children,
	className = '',
	...props
}: CardFooterProps) {
	return (
		<div
			className={`border-t border-gray-200 dark:border-gray-700 px-6 py-4 ${className}`}
			{...props}
		>
			{children}
		</div>
	);
}
