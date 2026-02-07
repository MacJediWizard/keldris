interface SkeletonProps {
	className?: string;
}

/**
 * Base skeleton component with pulse animation
 */
export function Skeleton({ className = '' }: SkeletonProps) {
	return (
		<div
			className={`animate-pulse bg-gray-200 dark:bg-gray-700 rounded ${className}`}
		/>
	);
}

/**
 * Generate stable keys for skeleton list items
 */
export function skeletonKeys(count: number, prefix = 'sk'): string[] {
	return Array.from({ length: count }, (_, i) => `${prefix}-${i}`);
}

interface TextSkeletonProps {
	width?: 'xs' | 'sm' | 'md' | 'lg' | 'xl' | 'full';
	size?: 'xs' | 'sm' | 'base' | 'lg' | 'xl';
	className?: string;
}

const textWidths = {
	xs: 'w-12',
	sm: 'w-20',
	md: 'w-32',
	lg: 'w-48',
	xl: 'w-64',
	full: 'w-full',
};

const textHeights = {
	xs: 'h-3',
	sm: 'h-3.5',
	base: 'h-4',
	lg: 'h-5',
	xl: 'h-6',
};

/**
 * Text skeleton that matches text line heights
 */
export function TextSkeleton({
	width = 'md',
	size = 'base',
	className = '',
}: TextSkeletonProps) {
	return (
		<Skeleton
			className={`${textWidths[width]} ${textHeights[size]} ${className}`}
		/>
	);
}

interface AvatarSkeletonProps {
	size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
	className?: string;
}

const avatarSizes = {
	xs: 'h-6 w-6',
	sm: 'h-8 w-8',
	md: 'h-10 w-10',
	lg: 'h-12 w-12',
	xl: 'h-16 w-16',
};

/**
 * Circular skeleton for avatars and icons
 */
export function AvatarSkeleton({
	size = 'md',
	className = '',
}: AvatarSkeletonProps) {
	return (
		<Skeleton className={`${avatarSizes[size]} rounded-full ${className}`} />
	);
}

interface BadgeSkeletonProps {
	width?: 'sm' | 'md' | 'lg';
	className?: string;
}

const badgeWidths = {
	sm: 'w-12',
	md: 'w-16',
	lg: 'w-20',
};

/**
 * Pill-shaped skeleton for status badges
 */
export function BadgeSkeleton({
	width = 'md',
	className = '',
}: BadgeSkeletonProps) {
	return (
		<Skeleton
			className={`h-6 ${badgeWidths[width]} rounded-full ${className}`}
		/>
	);
}

interface ButtonSkeletonProps {
	size?: 'sm' | 'md' | 'lg';
	className?: string;
}

const buttonSizes = {
	sm: 'h-8 w-16',
	md: 'h-10 w-24',
	lg: 'h-12 w-32',
};

/**
 * Button-shaped skeleton
 */
export function ButtonSkeleton({
	size = 'md',
	className = '',
}: ButtonSkeletonProps) {
	return <Skeleton className={`${buttonSizes[size]} ${className}`} />;
}

interface CardSkeletonProps {
	showHeader?: boolean;
	showFooter?: boolean;
	lines?: number;
	className?: string;
}

/**
 * Card skeleton with optional header and footer
 */
export function CardSkeleton({
	showHeader = true,
	showFooter = false,
	lines = 3,
	className = '',
}: CardSkeletonProps) {
	return (
		<div
			className={`bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 ${className}`}
		>
			{showHeader && (
				<div className="flex items-center justify-between mb-4">
					<TextSkeleton width="lg" size="lg" />
					<ButtonSkeleton size="sm" />
				</div>
			)}
			<div className="space-y-3">
				{skeletonKeys(lines, 'card-line').map((key, i) => (
					<TextSkeleton
						key={key}
						width={i === lines - 1 ? 'lg' : 'full'}
						size="base"
					/>
				))}
			</div>
			{showFooter && (
				<div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
					<div className="flex justify-end gap-3">
						<ButtonSkeleton size="sm" />
						<ButtonSkeleton size="md" />
					</div>
				</div>
			)}
		</div>
	);
}

interface TableRowSkeletonProps {
	columns: number;
	showCheckbox?: boolean;
	showActions?: boolean;
	className?: string;
}

/**
 * Table row skeleton with configurable columns
 */
export function TableRowSkeleton({
	columns,
	showCheckbox = false,
	showActions = false,
	className = '',
}: TableRowSkeletonProps) {
	const contentCols = columns - (showCheckbox ? 1 : 0) - (showActions ? 1 : 0);

	return (
		<tr className={`animate-pulse ${className}`}>
			{showCheckbox && (
				<td className="px-6 py-4 w-12">
					<Skeleton className="h-4 w-4" />
				</td>
			)}
			{skeletonKeys(contentCols, 'col').map((key, i) => (
				<td key={key} className="px-6 py-4">
					{i === 0 ? (
						<div className="space-y-1">
							<TextSkeleton width="lg" />
							<TextSkeleton width="md" size="sm" />
						</div>
					) : i === 1 ? (
						<BadgeSkeleton />
					) : (
						<TextSkeleton width={i % 2 === 0 ? 'md' : 'lg'} />
					)}
				</td>
			))}
			{showActions && (
				<td className="px-6 py-4 text-right">
					<ButtonSkeleton size="sm" className="inline-block" />
				</td>
			)}
		</tr>
	);
}

interface StatCardSkeletonProps {
	className?: string;
}

/**
 * Skeleton for dashboard stat cards
 */
export function StatCardSkeleton({ className = '' }: StatCardSkeletonProps) {
	return (
		<div
			className={`bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 ${className}`}
		>
			<div className="flex items-center justify-between">
				<div className="space-y-2">
					<TextSkeleton width="lg" size="sm" />
					<Skeleton className="h-8 w-16" />
					<TextSkeleton width="md" size="sm" />
				</div>
				<AvatarSkeleton size="lg" />
			</div>
		</div>
	);
}

interface InputSkeletonProps {
	showLabel?: boolean;
	className?: string;
}

/**
 * Form input skeleton with optional label
 */
export function InputSkeleton({
	showLabel = true,
	className = '',
}: InputSkeletonProps) {
	return (
		<div className={className}>
			{showLabel && <TextSkeleton width="md" size="sm" className="mb-2" />}
			<Skeleton className="h-10 w-full" />
		</div>
	);
}

interface FormSectionSkeletonProps {
	fields?: number;
	className?: string;
}

/**
 * Form section skeleton with multiple fields
 */
export function FormSectionSkeleton({
	fields = 4,
	className = '',
}: FormSectionSkeletonProps) {
	return (
		<div className={`space-y-4 ${className}`}>
			{skeletonKeys(fields, 'field').map((key) => (
				<InputSkeleton key={key} />
			))}
		</div>
	);
}
