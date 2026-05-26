import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '../test/helpers';

vi.mock('../hooks/useClassifications', () => ({
	useClassificationSummary: vi.fn().mockReturnValue({
		data: {
			restricted_count: 2,
			confidential_count: 3,
			internal_count: 4,
			public_count: 5,
			total_schedules: 10,
			total_backups: 20,
		},
		isLoading: false,
	}),
	useClassificationRules: vi.fn().mockReturnValue({
		data: [],
		isLoading: false,
	}),
	useScheduleClassifications: vi.fn().mockReturnValue({
		data: [],
		isLoading: false,
	}),
	useCreateClassificationRule: vi.fn().mockReturnValue({
		mutateAsync: vi.fn(),
		isPending: false,
	}),
	useDeleteClassificationRule: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
	useUpdateClassificationRule: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
	useAutoClassifySchedule: vi.fn().mockReturnValue({
		mutate: vi.fn(),
		isPending: false,
	}),
}));

vi.mock('../components/ClassificationBadge', () => ({
	ClassificationBadge: () => null,
	ClassificationLevelSelect: () => null,
	DataTypeMultiSelect: () => null,
}));

import { Classifications } from './Classifications';

describe('Classifications page', () => {
	it('renders the title', () => {
		renderWithProviders(<Classifications />);
		expect(screen.getByText('Data Classification')).toBeInTheDocument();
	});

	it('renders compliance summary tab content', () => {
		renderWithProviders(<Classifications />);
		expect(screen.getByText('Classification Overview')).toBeInTheDocument();
	});
});
