import type { ClassificationLevel, DataType } from '../lib/types';

interface ClassificationBadgeProps {
	level?: string;
	dataTypes?: string[];
	showDataTypes?: boolean;
	size?: 'sm' | 'md';
}

const levelColors: Record<ClassificationLevel, { bg: string; text: string; border: string }> = {
	public: {
		bg: 'bg-green-50 dark:bg-green-900/20',
		text: 'text-green-700 dark:text-green-400',
		border: 'border-green-200 dark:border-green-800',
	},
	internal: {
		bg: 'bg-blue-50 dark:bg-blue-900/20',
		text: 'text-blue-700 dark:text-blue-400',
		border: 'border-blue-200 dark:border-blue-800',
	},
	confidential: {
		bg: 'bg-yellow-50 dark:bg-yellow-900/20',
		text: 'text-yellow-700 dark:text-yellow-400',
		border: 'border-yellow-200 dark:border-yellow-800',
	},
	restricted: {
		bg: 'bg-red-50 dark:bg-red-900/20',
		text: 'text-red-700 dark:text-red-400',
		border: 'border-red-200 dark:border-red-800',
	},
};

const dataTypeLabels: Record<DataType, string> = {
	pii: 'PII',
	phi: 'PHI',
	pci: 'PCI',
	proprietary: 'Proprietary',
	general: 'General',
};

const dataTypeColors: Record<DataType, string> = {
	pii: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400',
	phi: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
	pci: 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400',
	proprietary: 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400',
	general: 'bg-gray-100 text-gray-700 dark:bg-gray-800/50 dark:text-gray-400',
};

function getLevelLabel(level: string): string {
	switch (level) {
		case 'public':
			return 'Public';
		case 'internal':
			return 'Internal';
		case 'confidential':
			return 'Confidential';
		case 'restricted':
			return 'Restricted';
		default:
			return level;
	}
}

export function ClassificationBadge({
	level = 'public',
	dataTypes,
	showDataTypes = false,
	size = 'sm',
}: ClassificationBadgeProps) {
	const normalizedLevel = (level || 'public') as ClassificationLevel;
	const colors = levelColors[normalizedLevel] || levelColors.public;
	const sizeClasses = size === 'sm' ? 'text-xs px-2 py-0.5' : 'text-sm px-2.5 py-1';

	const filteredDataTypes = (dataTypes || ['general']).filter(
		(dt) => dt !== 'general' || (dataTypes?.length === 1 && dt === 'general')
	);

	return (
		<div className="inline-flex items-center gap-1.5 flex-wrap">
			<span
				className={`inline-flex items-center font-medium rounded-full border ${colors.bg} ${colors.text} ${colors.border} ${sizeClasses}`}
			>
				{normalizedLevel === 'restricted' && (
					<svg
						className="w-3 h-3 mr-1"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
						aria-hidden="true"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
						/>
					</svg>
				)}
				{normalizedLevel === 'confidential' && (
					<svg
						className="w-3 h-3 mr-1"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
						aria-hidden="true"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
						/>
					</svg>
				)}
				{getLevelLabel(normalizedLevel)}
			</span>

			{showDataTypes &&
				filteredDataTypes.map((dt) => (
					<span
						key={dt}
						className={`inline-flex items-center font-medium rounded-full ${sizeClasses} ${dataTypeColors[dt as DataType] || dataTypeColors.general}`}
					>
						{dataTypeLabels[dt as DataType] || dt}
					</span>
				))}
		</div>
	);
}

interface ClassificationLevelSelectProps {
	value: string;
	onChange: (level: string) => void;
	disabled?: boolean;
	id?: string;
}

export function ClassificationLevelSelect({
	value,
	onChange,
	disabled = false,
	id,
}: ClassificationLevelSelectProps) {
	return (
		<select
			id={id}
			value={value || 'public'}
			onChange={(e) => onChange(e.target.value)}
			disabled={disabled}
			className="block w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-white focus:border-indigo-500 focus:ring-indigo-500 disabled:bg-gray-100 dark:disabled:bg-gray-800"
		>
			<option value="public">Public</option>
			<option value="internal">Internal</option>
			<option value="confidential">Confidential</option>
			<option value="restricted">Restricted</option>
		</select>
	);
}

interface DataTypeMultiSelectProps {
	value: string[];
	onChange: (types: string[]) => void;
	disabled?: boolean;
}

export function DataTypeMultiSelect({
	value,
	onChange,
	disabled = false,
}: DataTypeMultiSelectProps) {
	const allTypes: DataType[] = ['pii', 'phi', 'pci', 'proprietary', 'general'];

	const handleToggle = (type: string) => {
		if (value.includes(type)) {
			const newValue = value.filter((t) => t !== type);
			onChange(newValue.length > 0 ? newValue : ['general']);
		} else {
			onChange([...value.filter((t) => t !== 'general'), type]);
		}
	};

	return (
		<div className="flex flex-wrap gap-2">
			{allTypes.map((type) => (
				<button
					key={type}
					type="button"
					onClick={() => handleToggle(type)}
					disabled={disabled}
					className={`inline-flex items-center px-3 py-1.5 rounded-full text-sm font-medium transition-colors ${
						value.includes(type)
							? dataTypeColors[type]
							: 'bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400'
					} ${disabled ? 'opacity-50 cursor-not-allowed' : 'hover:opacity-80'}`}
				>
					{dataTypeLabels[type]}
				</button>
			))}
		</div>
	);
}
