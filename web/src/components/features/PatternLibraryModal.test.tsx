import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const mockCreateMutateAsync = vi.fn();

vi.mock('../../hooks/useExcludePatterns', () => ({
	useExcludePatternsLibrary: vi.fn(),
	useExcludePatternCategories: vi.fn(),
	useCreateExcludePattern: vi.fn(() => ({
		mutateAsync: mockCreateMutateAsync,
		isPending: false,
	})),
}));

import {
	useCreateExcludePattern,
	useExcludePatternCategories,
	useExcludePatternsLibrary,
} from '../../hooks/useExcludePatterns';
import { PatternLibraryModal } from './PatternLibraryModal';

const mockOnClose = vi.fn();
const mockOnAddPatterns = vi.fn();

const sampleLibrary = [
	{
		name: 'Node.js',
		description: 'Node.js files',
		category: 'language' as const,
		patterns: ['node_modules/', '.npm/', 'package-lock.json'],
	},
	{
		name: 'Git',
		description: 'Git files',
		category: 'ide' as const,
		patterns: ['.git/', '.gitignore'],
	},
	{
		name: 'Temp Files',
		description: 'Temporary files',
		category: 'temp' as const,
		patterns: ['*.tmp', '*.swp', '.DS_Store'],
	},
	{
		name: 'OS Caches',
		description: 'OS cache files',
		category: 'cache' as const,
		patterns: ['.cache/', 'Thumbs.db'],
	},
];

const sampleCategories = [
	{
		id: 'language' as const,
		name: 'Languages',
		description: 'Programming language patterns',
	},
	{ id: 'ide' as const, name: 'IDE/Editor', description: 'IDE patterns' },
	{ id: 'temp' as const, name: 'Temporary', description: 'Temp files' },
	{ id: 'cache' as const, name: 'Caches', description: 'Cache patterns' },
];

function setupMocks(overrides?: {
	library?: unknown[];
	categories?: unknown[];
	libraryLoading?: boolean;
}) {
	vi.mocked(useExcludePatternsLibrary).mockReturnValue({
		data: (overrides?.library ?? sampleLibrary) as ReturnType<
			typeof useExcludePatternsLibrary
		>['data'],
		isLoading: overrides?.libraryLoading ?? false,
	} as ReturnType<typeof useExcludePatternsLibrary>);
	vi.mocked(useExcludePatternCategories).mockReturnValue({
		data: (overrides?.categories ?? sampleCategories) as ReturnType<
			typeof useExcludePatternCategories
		>['data'],
	} as ReturnType<typeof useExcludePatternCategories>);
}

describe('PatternLibraryModal', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		vi.mocked(useCreateExcludePattern).mockReturnValue({
			mutateAsync: mockCreateMutateAsync,
			isPending: false,
		} as unknown as ReturnType<typeof useCreateExcludePattern>);
		setupMocks();
	});

	it('renders nothing when not open', () => {
		const { container } = render(
			<PatternLibraryModal
				isOpen={false}
				onClose={mockOnClose}
				onAddPatterns={mockOnAddPatterns}
			/>,
		);
		expect(container.innerHTML).toBe('');
	});

	it('renders modal when open', () => {
		render(
			<PatternLibraryModal
				isOpen={true}
				onClose={mockOnClose}
				onAddPatterns={mockOnAddPatterns}
			/>,
		);
		expect(screen.getByText('Exclude Patterns Library')).toBeInTheDocument();
		expect(
			screen.getByText('Select patterns to exclude from your backups'),
		).toBeInTheDocument();
	});

	it('shows All Categories button in sidebar', () => {
		render(
			<PatternLibraryModal
				isOpen={true}
				onClose={mockOnClose}
				onAddPatterns={mockOnAddPatterns}
			/>,
		);
		expect(screen.getByText('All Categories')).toBeInTheDocument();
	});

	it('shows category buttons in sidebar', () => {
		render(
			<PatternLibraryModal
				isOpen={true}
				onClose={mockOnClose}
				onAddPatterns={mockOnAddPatterns}
			/>,
		);
		expect(screen.getByText('Languages')).toBeInTheDocument();
		expect(screen.getByText('IDE/Editor')).toBeInTheDocument();
		expect(screen.getByText('Temporary')).toBeInTheDocument();
		expect(screen.getByText('Caches')).toBeInTheDocument();
	});

	it('shows all patterns when no category selected', () => {
		render(
			<PatternLibraryModal
				isOpen={true}
				onClose={mockOnClose}
				onAddPatterns={mockOnAddPatterns}
			/>,
		);
		expect(screen.getByText('Node.js')).toBeInTheDocument();
		expect(screen.getByText('Git')).toBeInTheDocument();
		expect(screen.getByText('Temp Files')).toBeInTheDocument();
		expect(screen.getByText('OS Caches')).toBeInTheDocument();
	});

	it('filters patterns by category', async () => {
		const user = userEvent.setup();
		render(
			<PatternLibraryModal
				isOpen={true}
				onClose={mockOnClose}
				onAddPatterns={mockOnAddPatterns}
			/>,
		);
		await user.click(screen.getByText('Languages'));
		expect(screen.getByText('Node.js')).toBeInTheDocument();
		expect(screen.queryByText('Git')).not.toBeInTheDocument();
	});

	it('shows Add button on each pattern', () => {
		render(
			<PatternLibraryModal
				isOpen={true}
				onClose={mockOnClose}
				onAddPatterns={mockOnAddPatterns}
			/>,
		);
		const addButtons = screen.getAllByText('Add');
		expect(addButtons.length).toBe(4);
	});

	it('adds single pattern on click', async () => {
		const user = userEvent.setup();
		render(
			<PatternLibraryModal
				isOpen={true}
				onClose={mockOnClose}
				onAddPatterns={mockOnAddPatterns}
			/>,
		);
		const addButtons = screen.getAllByText('Add');
		await user.click(addButtons[0]); // Add first pattern (Node.js)
		expect(mockOnAddPatterns).toHaveBeenCalledWith([
			'node_modules/',
			'.npm/',
			'package-lock.json',
		]);
	});

	it('selects and adds multiple patterns', async () => {
		const user = userEvent.setup();
		render(
			<PatternLibraryModal
				isOpen={true}
				onClose={mockOnClose}
				onAddPatterns={mockOnAddPatterns}
			/>,
		);

		// Select patterns using the toggle buttons
		const selectButtons = screen.getAllByTitle('Select');
		await user.click(selectButtons[0]); // Select Node.js
		await user.click(selectButtons[1]); // Select Git

		// Should show selection count
		expect(screen.getByText(/2 patterns selected/)).toBeInTheDocument();

		// Click Add Selected
		await user.click(screen.getByText('Add Selected'));
		expect(mockOnAddPatterns).toHaveBeenCalledWith(
			expect.arrayContaining(['node_modules/', '.npm/', '.git/']),
		);
		expect(mockOnClose).toHaveBeenCalled();
	});

	it('shows Added badge for already existing patterns', () => {
		render(
			<PatternLibraryModal
				isOpen={true}
				onClose={mockOnClose}
				onAddPatterns={mockOnAddPatterns}
				existingPatterns={['node_modules/', '.npm/', 'package-lock.json']}
			/>,
		);
		expect(screen.getByText('Added')).toBeInTheDocument();
	});

	it('shows Partially added badge when some patterns exist', () => {
		render(
			<PatternLibraryModal
				isOpen={true}
				onClose={mockOnClose}
				onAddPatterns={mockOnAddPatterns}
				existingPatterns={['node_modules/']}
			/>,
		);
		expect(screen.getByText('Partially added')).toBeInTheDocument();
	});

	it('shows loading spinner', () => {
		setupMocks({ libraryLoading: true });
		render(
			<PatternLibraryModal
				isOpen={true}
				onClose={mockOnClose}
				onAddPatterns={mockOnAddPatterns}
			/>,
		);
		const spinner = document.querySelector('.animate-spin');
		expect(spinner).toBeInTheDocument();
	});

	it('calls onClose when clicking Cancel', async () => {
		const user = userEvent.setup();
		render(
			<PatternLibraryModal
				isOpen={true}
				onClose={mockOnClose}
				onAddPatterns={mockOnAddPatterns}
			/>,
		);
		await user.click(screen.getByText('Cancel'));
		expect(mockOnClose).toHaveBeenCalled();
	});

	it('opens Create Custom form', async () => {
		const user = userEvent.setup();
		render(
			<PatternLibraryModal
				isOpen={true}
				onClose={mockOnClose}
				onAddPatterns={mockOnAddPatterns}
			/>,
		);
		await user.click(screen.getByText('Create Custom'));
		expect(screen.getByText('Create Custom Pattern')).toBeInTheDocument();
		expect(screen.getByLabelText('Name')).toBeInTheDocument();
		expect(screen.getByLabelText('Description')).toBeInTheDocument();
		expect(screen.getByLabelText('Category')).toBeInTheDocument();
		expect(
			screen.getByLabelText('Patterns (one per line)'),
		).toBeInTheDocument();
	});

	it('saves custom pattern', async () => {
		const user = userEvent.setup();
		mockCreateMutateAsync.mockResolvedValue({});
		render(
			<PatternLibraryModal
				isOpen={true}
				onClose={mockOnClose}
				onAddPatterns={mockOnAddPatterns}
			/>,
		);
		await user.click(screen.getByText('Create Custom'));

		await user.type(screen.getByLabelText('Name'), 'My Patterns');
		await user.type(
			screen.getByLabelText('Description'),
			'Custom test patterns',
		);
		await user.type(
			screen.getByLabelText('Patterns (one per line)'),
			'*.log\ntemp/*',
		);
		await user.click(screen.getByText('Save and Add'));

		expect(mockCreateMutateAsync).toHaveBeenCalledWith(
			expect.objectContaining({
				name: 'My Patterns',
				description: 'Custom test patterns',
				patterns: ['*.log', 'temp/*'],
			}),
		);
		expect(mockOnAddPatterns).toHaveBeenCalledWith(['*.log', 'temp/*']);
	});

	it('disables save button when name or patterns are empty', async () => {
		const user = userEvent.setup();
		render(
			<PatternLibraryModal
				isOpen={true}
				onClose={mockOnClose}
				onAddPatterns={mockOnAddPatterns}
			/>,
		);
		await user.click(screen.getByText('Create Custom'));
		expect(screen.getByText('Save and Add')).toBeDisabled();
	});

	it('cancels custom form and returns to library view', async () => {
		const user = userEvent.setup();
		render(
			<PatternLibraryModal
				isOpen={true}
				onClose={mockOnClose}
				onAddPatterns={mockOnAddPatterns}
			/>,
		);
		await user.click(screen.getByText('Create Custom'));
		expect(screen.getByText('Create Custom Pattern')).toBeInTheDocument();
		await user.click(screen.getAllByText('Cancel')[0]);
		expect(screen.queryByText('Create Custom Pattern')).not.toBeInTheDocument();
		expect(screen.getByText('Node.js')).toBeInTheDocument();
	});

	it('shows pattern preview codes', () => {
		render(
			<PatternLibraryModal
				isOpen={true}
				onClose={mockOnClose}
				onAddPatterns={mockOnAddPatterns}
			/>,
		);
		expect(screen.getByText('node_modules/')).toBeInTheDocument();
		expect(screen.getByText('.npm/')).toBeInTheDocument();
		expect(screen.getByText('.git/')).toBeInTheDocument();
	});

	it('excludes already-existing patterns from add', async () => {
		const user = userEvent.setup();
		render(
			<PatternLibraryModal
				isOpen={true}
				onClose={mockOnClose}
				onAddPatterns={mockOnAddPatterns}
				existingPatterns={['node_modules/']}
			/>,
		);
		// Select Node.js pattern - one pattern already exists, should only add the others
		const selectButtons = screen.getAllByTitle('Select');
		await user.click(selectButtons[0]); // Node.js
		await user.click(screen.getByText('Add Selected'));
		expect(mockOnAddPatterns).toHaveBeenCalledWith([
			'.npm/',
			'package-lock.json',
		]);
	});
});
