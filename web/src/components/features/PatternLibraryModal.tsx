import { useMemo, useState } from 'react';
import {
	useCreateExcludePattern,
	useExcludePatternCategories,
	useExcludePatternsLibrary,
} from '../../hooks/useExcludePatterns';
import type {
	BuiltInPattern,
	CategoryInfo,
	ExcludePatternCategory,
} from '../../lib/types';

interface PatternLibraryModalProps {
	isOpen: boolean;
	onClose: () => void;
	onAddPatterns: (patterns: string[]) => void;
	existingPatterns?: string[];
}

const categoryIcons: Record<string, React.ReactNode> = {
	os: (
		<svg
			className="w-4 h-4"
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
			aria-hidden="true"
		>
			<path
				strokeLinecap="round"
				strokeLinejoin="round"
				strokeWidth={2}
				d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
			/>
		</svg>
	),
	ide: (
		<svg
			className="w-4 h-4"
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
			aria-hidden="true"
		>
			<path
				strokeLinecap="round"
				strokeLinejoin="round"
				strokeWidth={2}
				d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"
			/>
		</svg>
	),
	language: (
		<svg
			className="w-4 h-4"
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
			aria-hidden="true"
		>
			<path
				strokeLinecap="round"
				strokeLinejoin="round"
				strokeWidth={2}
				d="M3 5h12M9 3v2m1.048 9.5A18.022 18.022 0 016.412 9m6.088 9h7M11 21l5-10 5 10M12.751 5C11.783 10.77 8.07 15.61 3 18.129"
			/>
		</svg>
	),
	build: (
		<svg
			className="w-4 h-4"
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
			aria-hidden="true"
		>
			<path
				strokeLinecap="round"
				strokeLinejoin="round"
				strokeWidth={2}
				d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
			/>
		</svg>
	),
	cache: (
		<svg
			className="w-4 h-4"
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
			aria-hidden="true"
		>
			<path
				strokeLinecap="round"
				strokeLinejoin="round"
				strokeWidth={2}
				d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4"
			/>
		</svg>
	),
	temp: (
		<svg
			className="w-4 h-4"
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
			aria-hidden="true"
		>
			<path
				strokeLinecap="round"
				strokeLinejoin="round"
				strokeWidth={2}
				d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
			/>
		</svg>
	),
	logs: (
		<svg
			className="w-4 h-4"
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
			aria-hidden="true"
		>
			<path
				strokeLinecap="round"
				strokeLinejoin="round"
				strokeWidth={2}
				d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
			/>
		</svg>
	),
	security: (
		<svg
			className="w-4 h-4"
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
			aria-hidden="true"
		>
			<path
				strokeLinecap="round"
				strokeLinejoin="round"
				strokeWidth={2}
				d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
			/>
		</svg>
	),
	database: (
		<svg
			className="w-4 h-4"
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
			aria-hidden="true"
		>
			<path
				strokeLinecap="round"
				strokeLinejoin="round"
				strokeWidth={2}
				d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"
			/>
		</svg>
	),
	container: (
		<svg
			className="w-4 h-4"
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
			aria-hidden="true"
		>
			<path
				strokeLinecap="round"
				strokeLinejoin="round"
				strokeWidth={2}
				d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"
			/>
		</svg>
	),
};

export function PatternLibraryModal({
	isOpen,
	onClose,
	onAddPatterns,
	existingPatterns = [],
}: PatternLibraryModalProps) {
	const [selectedCategory, setSelectedCategory] =
		useState<ExcludePatternCategory | null>(null);
	const [selectedPatterns, setSelectedPatterns] = useState<Set<string>>(
		new Set(),
	);
	const [showCustomForm, setShowCustomForm] = useState(false);
	const [customName, setCustomName] = useState('');
	const [customDescription, setCustomDescription] = useState('');
	const [customPatterns, setCustomPatterns] = useState('');
	const [customCategory, setCustomCategory] =
		useState<ExcludePatternCategory>('temp');

	const { data: library, isLoading: libraryLoading } =
		useExcludePatternsLibrary();
	const { data: categories } = useExcludePatternCategories();
	const createPattern = useCreateExcludePattern();

	// Group patterns by category
	const patternsByCategory = useMemo(() => {
		if (!library) return new Map<ExcludePatternCategory, BuiltInPattern[]>();
		const grouped = new Map<ExcludePatternCategory, BuiltInPattern[]>();
		for (const pattern of library) {
			const existing = grouped.get(pattern.category) || [];
			grouped.set(pattern.category, [...existing, pattern]);
		}
		return grouped;
	}, [library]);

	const existingPatternsSet = useMemo(
		() => new Set(existingPatterns),
		[existingPatterns],
	);

	const togglePattern = (patternName: string) => {
		const newSelected = new Set(selectedPatterns);
		if (newSelected.has(patternName)) {
			newSelected.delete(patternName);
		} else {
			newSelected.add(patternName);
		}
		setSelectedPatterns(newSelected);
	};

	const getSelectedPatternsArray = (): string[] => {
		if (!library) return [];
		const result: string[] = [];
		for (const pattern of library) {
			if (selectedPatterns.has(pattern.name)) {
				for (const p of pattern.patterns) {
					if (!existingPatternsSet.has(p) && !result.includes(p)) {
						result.push(p);
					}
				}
			}
		}
		return result;
	};

	const handleAddSelected = () => {
		const patternsToAdd = getSelectedPatternsArray();
		if (patternsToAdd.length > 0) {
			onAddPatterns(patternsToAdd);
		}
		setSelectedPatterns(new Set());
		onClose();
	};

	const handleAddSinglePattern = (pattern: BuiltInPattern) => {
		const patternsToAdd = pattern.patterns.filter(
			(p: string) => !existingPatternsSet.has(p),
		);
		if (patternsToAdd.length > 0) {
			onAddPatterns(patternsToAdd);
		}
	};

	const handleSaveCustom = async () => {
		const patterns = customPatterns
			.split('\n')
			.map((p) => p.trim())
			.filter((p) => p);

		if (customName && patterns.length > 0) {
			try {
				await createPattern.mutateAsync({
					name: customName,
					description: customDescription,
					patterns,
					category: customCategory,
				});
				onAddPatterns(patterns);
				setShowCustomForm(false);
				setCustomName('');
				setCustomDescription('');
				setCustomPatterns('');
				setCustomCategory('temp');
				onClose();
			} catch {
				// Error handled by mutation
			}
		}
	};

	if (!isOpen) return null;

	const displayedPatterns = selectedCategory
		? patternsByCategory.get(selectedCategory) || []
		: library || [];

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white rounded-lg w-full max-w-4xl mx-4 max-h-[90vh] flex flex-col">
				<div className="p-6 border-b border-gray-200 flex items-center justify-between">
					<div>
						<h3 className="text-lg font-semibold text-gray-900">
							Exclude Patterns Library
						</h3>
						<p className="text-sm text-gray-500 mt-1">
							Select patterns to exclude from your backups
						</p>
					</div>
					<button
						type="button"
						onClick={onClose}
						className="text-gray-400 hover:text-gray-500"
					>
						<svg
							className="w-6 h-6"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M6 18L18 6M6 6l12 12"
							/>
						</svg>
					</button>
				</div>

				<div className="flex-1 flex overflow-hidden">
					{/* Categories sidebar */}
					<div className="w-56 border-r border-gray-200 overflow-y-auto p-4">
						<button
							type="button"
							onClick={() => setSelectedCategory(null)}
							className={`w-full text-left px-3 py-2 rounded-lg mb-2 transition-colors ${
								selectedCategory === null
									? 'bg-indigo-50 text-indigo-700'
									: 'text-gray-700 hover:bg-gray-50'
							}`}
						>
							All Categories
						</button>
						<div className="space-y-1">
							{categories?.map((cat: CategoryInfo) => (
								<button
									key={cat.id}
									type="button"
									onClick={() => setSelectedCategory(cat.id)}
									className={`w-full text-left px-3 py-2 rounded-lg transition-colors flex items-center gap-2 ${
										selectedCategory === cat.id
											? 'bg-indigo-50 text-indigo-700'
											: 'text-gray-700 hover:bg-gray-50'
									}`}
								>
									{categoryIcons[cat.id] || (
										<span className="w-4 h-4 rounded bg-gray-200" />
									)}
									<span className="text-sm">{cat.name}</span>
								</button>
							))}
						</div>
						<div className="border-t border-gray-200 mt-4 pt-4">
							<button
								type="button"
								onClick={() => setShowCustomForm(true)}
								className="w-full text-left px-3 py-2 rounded-lg text-indigo-600 hover:bg-indigo-50 transition-colors flex items-center gap-2"
							>
								<svg
									className="w-4 h-4"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
									aria-hidden="true"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M12 4v16m8-8H4"
									/>
								</svg>
								<span className="text-sm">Create Custom</span>
							</button>
						</div>
					</div>

					{/* Pattern list */}
					<div className="flex-1 overflow-y-auto p-4">
						{showCustomForm ? (
							<div className="space-y-4">
								<div className="flex items-center justify-between mb-4">
									<h4 className="font-medium text-gray-900">
										Create Custom Pattern
									</h4>
									<button
										type="button"
										onClick={() => setShowCustomForm(false)}
										className="text-sm text-gray-500 hover:text-gray-700"
									>
										Cancel
									</button>
								</div>
								<div>
									<label
										htmlFor="custom-name"
										className="block text-sm font-medium text-gray-700 mb-1"
									>
										Name
									</label>
									<input
										type="text"
										id="custom-name"
										value={customName}
										onChange={(e) => setCustomName(e.target.value)}
										placeholder="My Custom Patterns"
										className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
									/>
								</div>
								<div>
									<label
										htmlFor="custom-description"
										className="block text-sm font-medium text-gray-700 mb-1"
									>
										Description
									</label>
									<input
										type="text"
										id="custom-description"
										value={customDescription}
										onChange={(e) => setCustomDescription(e.target.value)}
										placeholder="Patterns for my project"
										className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
									/>
								</div>
								<div>
									<label
										htmlFor="custom-category"
										className="block text-sm font-medium text-gray-700 mb-1"
									>
										Category
									</label>
									<select
										id="custom-category"
										value={customCategory}
										onChange={(e) =>
											setCustomCategory(
												e.target.value as ExcludePatternCategory,
											)
										}
										className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
									>
										{categories?.map((cat: CategoryInfo) => (
											<option key={cat.id} value={cat.id}>
												{cat.name}
											</option>
										))}
									</select>
								</div>
								<div>
									<label
										htmlFor="custom-patterns"
										className="block text-sm font-medium text-gray-700 mb-1"
									>
										Patterns (one per line)
									</label>
									<textarea
										id="custom-patterns"
										value={customPatterns}
										onChange={(e) => setCustomPatterns(e.target.value)}
										placeholder="*.log&#10;temp/*&#10;.cache/"
										rows={5}
										className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono text-sm"
									/>
								</div>
								<button
									type="button"
									onClick={handleSaveCustom}
									disabled={
										!customName ||
										!customPatterns.trim() ||
										createPattern.isPending
									}
									className="w-full px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
								>
									{createPattern.isPending ? 'Saving...' : 'Save and Add'}
								</button>
							</div>
						) : libraryLoading ? (
							<div className="flex items-center justify-center py-12">
								<div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600" />
							</div>
						) : (
							<div className="space-y-3">
								{displayedPatterns.map((pattern: BuiltInPattern) => {
									const isSelected = selectedPatterns.has(pattern.name);
									const alreadyAdded = pattern.patterns.every((p: string) =>
										existingPatternsSet.has(p),
									);
									const partiallyAdded =
										!alreadyAdded &&
										pattern.patterns.some((p: string) =>
											existingPatternsSet.has(p),
										);

									return (
										<div
											key={pattern.name}
											className={`p-4 rounded-lg border transition-colors ${
												isSelected
													? 'border-indigo-500 bg-indigo-50'
													: alreadyAdded
														? 'border-gray-200 bg-gray-50 opacity-60'
														: 'border-gray-200 hover:border-gray-300'
											}`}
										>
											<div className="flex items-start justify-between">
												<div className="flex-1">
													<div className="flex items-center gap-2">
														<h5 className="font-medium text-gray-900">
															{pattern.name}
														</h5>
														{alreadyAdded && (
															<span className="text-xs bg-green-100 text-green-700 px-2 py-0.5 rounded">
																Added
															</span>
														)}
														{partiallyAdded && (
															<span className="text-xs bg-yellow-100 text-yellow-700 px-2 py-0.5 rounded">
																Partially added
															</span>
														)}
													</div>
													<p className="text-sm text-gray-500 mt-0.5">
														{pattern.description}
													</p>
													<div className="mt-2 flex flex-wrap gap-1">
														{pattern.patterns
															.slice(0, 5)
															.map((p: string, i: number) => (
																<code
																	// biome-ignore lint/suspicious/noArrayIndexKey: Static display of pattern strings
																	key={i}
																	className={`text-xs px-1.5 py-0.5 rounded ${
																		existingPatternsSet.has(p)
																			? 'bg-green-100 text-green-700'
																			: 'bg-gray-100 text-gray-600'
																	}`}
																>
																	{p}
																</code>
															))}
														{pattern.patterns.length > 5 && (
															<span className="text-xs text-gray-500">
																+{pattern.patterns.length - 5} more
															</span>
														)}
													</div>
												</div>
												<div className="flex items-center gap-2 ml-4">
													{!alreadyAdded && (
														<>
															<button
																type="button"
																onClick={() => togglePattern(pattern.name)}
																className={`p-2 rounded-lg transition-colors ${
																	isSelected
																		? 'bg-indigo-100 text-indigo-700'
																		: 'bg-gray-100 text-gray-600 hover:bg-gray-200'
																}`}
																title={isSelected ? 'Deselect' : 'Select'}
															>
																<svg
																	className="w-5 h-5"
																	fill={isSelected ? 'currentColor' : 'none'}
																	stroke="currentColor"
																	viewBox="0 0 24 24"
																	aria-hidden="true"
																>
																	<path
																		strokeLinecap="round"
																		strokeLinejoin="round"
																		strokeWidth={2}
																		d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
																	/>
																</svg>
															</button>
															<button
																type="button"
																onClick={() => handleAddSinglePattern(pattern)}
																className="px-3 py-1.5 text-sm bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
															>
																Add
															</button>
														</>
													)}
												</div>
											</div>
										</div>
									);
								})}
							</div>
						)}
					</div>
				</div>

				{/* Footer */}
				<div className="p-4 border-t border-gray-200 flex items-center justify-between">
					<div className="text-sm text-gray-500">
						{selectedPatterns.size > 0 && (
							<span>
								{selectedPatterns.size} pattern
								{selectedPatterns.size !== 1 ? 's' : ''} selected (
								{getSelectedPatternsArray().length} new exclude
								{getSelectedPatternsArray().length !== 1 ? 's' : ''})
							</span>
						)}
					</div>
					<div className="flex gap-3">
						<button
							type="button"
							onClick={onClose}
							className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
						>
							Cancel
						</button>
						{selectedPatterns.size > 0 && (
							<button
								type="button"
								onClick={handleAddSelected}
								className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
							>
								Add Selected
							</button>
						)}
					</div>
				</div>
			</div>
		</div>
	);
}
