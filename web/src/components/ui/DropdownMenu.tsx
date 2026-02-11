import { type ReactNode, useEffect, useRef, useState } from 'react';

interface DropdownMenuItem {
	label: string;
	onClick: () => void;
	disabled?: boolean;
	variant?: 'default' | 'danger';
}

interface DropdownMenuProps {
	trigger: ReactNode;
	items: DropdownMenuItem[];
}

export function DropdownMenu({ trigger, items }: DropdownMenuProps) {
	const [open, setOpen] = useState(false);
	const ref = useRef<HTMLDivElement>(null);

	useEffect(() => {
		function handleClickOutside(e: MouseEvent) {
			if (ref.current && !ref.current.contains(e.target as Node)) {
				setOpen(false);
			}
		}
		if (open) {
			document.addEventListener('mousedown', handleClickOutside);
		}
		return () => document.removeEventListener('mousedown', handleClickOutside);
	}, [open]);

	return (
		<div className="relative inline-block" ref={ref}>
			<button
				type="button"
				onClick={() => setOpen(!open)}
				className="inline-flex"
			>
				{trigger}
			</button>
			{open && (
				<div
					className="absolute right-0 z-50 mt-2 min-w-[160px] rounded-md border border-gray-200 bg-white py-1 shadow-lg"
					role="menu"
				>
					{items.map((item) => (
						<button
							key={item.label}
							type="button"
							role="menuitem"
							disabled={item.disabled}
							onClick={() => {
								item.onClick();
								setOpen(false);
							}}
							className={`block w-full px-4 py-2 text-left text-sm disabled:cursor-not-allowed disabled:opacity-50 ${
								item.variant === 'danger'
									? 'text-red-600 hover:bg-red-50'
									: 'text-gray-700 hover:bg-gray-100'
							}`}
						>
							{item.label}
						</button>
					))}
				</div>
			)}
		</div>
	);
}
