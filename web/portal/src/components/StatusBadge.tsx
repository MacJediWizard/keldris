interface StatusBadgeProps {
	status: string;
	variant?: 'license' | 'invoice';
}

export function StatusBadge({ status, variant = 'license' }: StatusBadgeProps) {
	const getColors = () => {
		if (variant === 'invoice') {
			switch (status) {
				case 'paid':
					return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
				case 'sent':
					return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200';
				case 'overdue':
					return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
				case 'cancelled':
				case 'refunded':
					return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200';
				default:
					return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200';
			}
		}

		switch (status) {
			case 'active':
				return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
			case 'expired':
				return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200';
			case 'revoked':
			case 'suspended':
				return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
			default:
				return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200';
		}
	};

	return (
		<span
			className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium capitalize ${getColors()}`}
		>
			{status}
		</span>
	);
}
