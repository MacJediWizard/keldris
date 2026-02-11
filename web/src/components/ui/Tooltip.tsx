import { type ReactNode, useState } from 'react';

interface TooltipProps {
	content: string;
	children: ReactNode;
	position?: 'top' | 'bottom' | 'left' | 'right';
}

const positionClasses = {
	top: 'bottom-full left-1/2 -translate-x-1/2 mb-2',
	bottom: 'top-full left-1/2 -translate-x-1/2 mt-2',
	left: 'right-full top-1/2 -translate-y-1/2 mr-2',
	right: 'left-full top-1/2 -translate-y-1/2 ml-2',
} as const;

export function Tooltip({ content, children, position = 'top' }: TooltipProps) {
	const [visible, setVisible] = useState(false);

	return (
		<div
			className="relative inline-block"
			onMouseEnter={() => setVisible(true)}
			onMouseLeave={() => setVisible(false)}
		>
			{children}
			{visible && (
				<div
					role="tooltip"
					className={`absolute z-50 whitespace-nowrap rounded bg-gray-900 px-2 py-1 text-xs text-white shadow-lg ${positionClasses[position]}`}
				>
					{content}
				</div>
			)}
		</div>
	);
}
