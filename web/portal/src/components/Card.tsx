interface CardProps {
	children: React.ReactNode;
	className?: string;
}

export function Card({ children, className = '' }: CardProps) {
	return (
		<div
			className={`bg-white dark:bg-dark-card rounded-lg shadow border border-gray-200 dark:border-dark-border ${className}`}
		>
			{children}
		</div>
	);
}

interface CardHeaderProps {
	children: React.ReactNode;
	className?: string;
}

export function CardHeader({ children, className = '' }: CardHeaderProps) {
	return (
		<div className={`px-6 py-4 border-b border-gray-200 dark:border-dark-border ${className}`}>
			{children}
		</div>
	);
}

interface CardContentProps {
	children: React.ReactNode;
	className?: string;
}

export function CardContent({ children, className = '' }: CardContentProps) {
	return <div className={`px-6 py-4 ${className}`}>{children}</div>;
}
