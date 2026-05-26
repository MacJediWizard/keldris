import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useSavedFilters', () => ({
	useCreateSavedFilter: vi.fn(),
}));

import { useCreateSavedFilter } from '../../hooks/useSavedFilters';
import { SaveFilterButton } from './SaveFilterButton';

const mutateAsync = vi.fn();

function setMocks({ isPending = false }: { isPending?: boolean } = {}) {
	vi.mocked(useCreateSavedFilter).mockReturnValue({
		mutateAsync,
		isPending,
	} as never);
}

describe('SaveFilterButton', () => {
	it('renders nothing when no meaningful filters set', () => {
		setMocks();
		const { container } = render(
			<SaveFilterButton entityType="agents" filters={{}} />,
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders nothing when all filter values are empty/default', () => {
		setMocks();
		const { container } = render(
			<SaveFilterButton
				entityType="agents"
				filters={{ status: '', priority: 'all', region: null }}
			/>,
		);
		expect(container.firstChild).toBeNull();
	});

	it('renders Save Filter button when filters present', () => {
		setMocks();
		render(
			<SaveFilterButton entityType="agents" filters={{ status: 'active' }} />,
		);
		expect(screen.getByRole('button', { name: /Save Filter/ })).toBeDefined();
	});

	it('opens modal when button clicked', async () => {
		setMocks();
		const user = userEvent.setup();
		render(
			<SaveFilterButton entityType="agents" filters={{ status: 'active' }} />,
		);
		await user.click(screen.getByRole('button', { name: /Save Filter/ }));
		expect(screen.getByText('Save Current Filter')).toBeDefined();
		expect(screen.getByLabelText('Filter Name')).toBeDefined();
	});

	it('disables Save Filter trigger when disabled prop true', () => {
		setMocks();
		render(
			<SaveFilterButton
				entityType="agents"
				filters={{ status: 'active' }}
				disabled
			/>,
		);
		expect(screen.getByRole('button', { name: /Save Filter/ })).toBeDisabled();
	});
});
