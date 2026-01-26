import { useEffect, useRef, useState } from 'react';

interface HelpTooltipProps {
	content: string;
	title?: string;
	docsUrl?: string;
	position?: 'top' | 'bottom' | 'left' | 'right';
	size?: 'sm' | 'md';
}

export function HelpTooltip({
	content,
	title,
	docsUrl,
	position = 'top',
	size = 'sm',
}: HelpTooltipProps) {
	const [isVisible, setIsVisible] = useState(false);
	const [tooltipPosition, setTooltipPosition] = useState(position);
	const triggerRef = useRef<HTMLButtonElement>(null);
	const tooltipRef = useRef<HTMLDivElement>(null);

	useEffect(() => {
		if (isVisible && triggerRef.current && tooltipRef.current) {
			const triggerRect = triggerRef.current.getBoundingClientRect();
			const tooltipRect = tooltipRef.current.getBoundingClientRect();
			const viewportWidth = window.innerWidth;
			const viewportHeight = window.innerHeight;

			let newPosition = position;

			// Check if tooltip would overflow and adjust position
			if (position === 'top' && triggerRect.top < tooltipRect.height + 8) {
				newPosition = 'bottom';
			} else if (
				position === 'bottom' &&
				triggerRect.bottom + tooltipRect.height + 8 > viewportHeight
			) {
				newPosition = 'top';
			} else if (
				position === 'left' &&
				triggerRect.left < tooltipRect.width + 8
			) {
				newPosition = 'right';
			} else if (
				position === 'right' &&
				triggerRect.right + tooltipRect.width + 8 > viewportWidth
			) {
				newPosition = 'left';
			}

			setTooltipPosition(newPosition);
		}
	}, [isVisible, position]);

	const iconSize = size === 'sm' ? 'w-4 h-4' : 'w-5 h-5';

	const positionClasses = {
		top: 'bottom-full left-1/2 -translate-x-1/2 mb-2',
		bottom: 'top-full left-1/2 -translate-x-1/2 mt-2',
		left: 'right-full top-1/2 -translate-y-1/2 mr-2',
		right: 'left-full top-1/2 -translate-y-1/2 ml-2',
	};

	const arrowClasses = {
		top: 'top-full left-1/2 -translate-x-1/2 border-t-gray-800 dark:border-t-gray-700 border-l-transparent border-r-transparent border-b-transparent',
		bottom:
			'bottom-full left-1/2 -translate-x-1/2 border-b-gray-800 dark:border-b-gray-700 border-l-transparent border-r-transparent border-t-transparent',
		left: 'left-full top-1/2 -translate-y-1/2 border-l-gray-800 dark:border-l-gray-700 border-t-transparent border-b-transparent border-r-transparent',
		right:
			'right-full top-1/2 -translate-y-1/2 border-r-gray-800 dark:border-r-gray-700 border-t-transparent border-b-transparent border-l-transparent',
	};

	// Simple markdown-like parsing for bold and code
	const parseContent = (text: string) => {
		const parts = text.split(/(\*\*[^*]+\*\*|`[^`]+`)/g);
		return parts.map((part, index) => {
			if (part.startsWith('**') && part.endsWith('**')) {
				return (
					// biome-ignore lint/suspicious/noArrayIndexKey: Static text parsing, order never changes
					<strong key={index} className="font-semibold text-white">
						{part.slice(2, -2)}
					</strong>
				);
			}
			if (part.startsWith('`') && part.endsWith('`')) {
				return (
					// biome-ignore lint/suspicious/noArrayIndexKey: Static text parsing, order never changes
					<code key={index} className="px-1 py-0.5 bg-gray-700 dark:bg-gray-600 rounded text-xs font-mono">
						{part.slice(1, -1)}
					</code>
				);
			}
			return part;
		});
	};

	return (
		<span className="relative inline-flex items-center">
			<button
				ref={triggerRef}
				type="button"
				className="text-gray-400 hover:text-gray-500 dark:text-gray-500 dark:hover:text-gray-400 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-1 rounded-full"
				onMouseEnter={() => setIsVisible(true)}
				onMouseLeave={() => setIsVisible(false)}
				onFocus={() => setIsVisible(true)}
				onBlur={() => setIsVisible(false)}
				aria-label={title || 'Help'}
			>
				<svg
					className={iconSize}
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				</svg>
			</button>
			{isVisible && (
				<div
					ref={tooltipRef}
					role="tooltip"
					className={`absolute z-50 w-64 px-3 py-2 text-sm text-gray-100 bg-gray-800 dark:bg-gray-700 rounded-lg shadow-lg ${positionClasses[tooltipPosition]}`}
				>
					{title && <div className="font-medium text-white mb-1">{title}</div>}
					<div className="text-gray-300 leading-relaxed">
						{parseContent(content)}
					</div>
					{docsUrl && (
						<a
							href={docsUrl}
							target="_blank"
							rel="noopener noreferrer"
							className="inline-flex items-center gap-1 mt-2 text-xs text-indigo-400 hover:text-indigo-300"
							onClick={(e) => e.stopPropagation()}
						>
							Learn more
							<svg
								className="w-3 h-3"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
								aria-hidden="true"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
								/>
							</svg>
						</a>
					)}
					<span
						className={`absolute w-0 h-0 border-4 ${arrowClasses[tooltipPosition]}`}
					/>
				</div>
			)}
		</span>
	);
}

interface FormLabelWithHelpProps {
	htmlFor: string;
	label: string;
	helpContent: string;
	helpTitle?: string;
	docsUrl?: string;
	required?: boolean;
}

export function FormLabelWithHelp({
	htmlFor,
	label,
	helpContent,
	helpTitle,
	docsUrl,
	required,
}: FormLabelWithHelpProps) {
	return (
		<label
			htmlFor={htmlFor}
			className="flex items-center gap-1.5 text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
		>
			{label}
			{required && <span className="text-red-500">*</span>}
			<HelpTooltip content={helpContent} title={helpTitle} docsUrl={docsUrl} />
		</label>
	);
}

interface StatusBadgeWithHelpProps {
	status: string;
	statusColor: {
		bg: string;
		text: string;
		dot: string;
	};
	helpContent: string;
	helpTitle?: string;
}

export function StatusBadgeWithHelp({
	status,
	statusColor,
	helpContent,
	helpTitle,
}: StatusBadgeWithHelpProps) {
	return (
		<span className="inline-flex items-center gap-1.5">
			<span
				className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}
			>
				<span className={`w-1.5 h-1.5 ${statusColor.dot} rounded-full`} />
				{status}
			</span>
			<HelpTooltip content={helpContent} title={helpTitle} position="right" />
		</span>
	);
}

interface DashboardWidgetHelpProps {
	title: string;
	helpContent: string;
	docsUrl?: string;
	children: React.ReactNode;
}

export function DashboardWidgetHelp({
	title,
	helpContent,
	docsUrl,
	children,
}: DashboardWidgetHelpProps) {
	return (
		<div className="flex items-center justify-between">
			<div className="flex items-center gap-1.5">
				<h2 className="text-lg font-semibold text-gray-900 dark:text-white">
					{title}
				</h2>
				<HelpTooltip content={helpContent} docsUrl={docsUrl} size="md" />
			</div>
			{children}
		</div>
	);
}
