import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('../../hooks/useFavorites', () => ({
	useAddFavorite: vi.fn(),
	useRemoveFavorite: vi.fn(),
}));

import { useAddFavorite, useRemoveFavorite } from '../../hooks/useFavorites';
import { StarButton } from './StarButton';

const addMutate = vi.fn();
const removeMutate = vi.fn();

function setMocks({
	addPending = false,
	removePending = false,
}: { addPending?: boolean; removePending?: boolean } = {}) {
	vi.mocked(useAddFavorite).mockReturnValue({
		mutate: addMutate,
		isPending: addPending,
	} as never);
	vi.mocked(useRemoveFavorite).mockReturnValue({
		mutate: removeMutate,
		isPending: removePending,
	} as never);
}

describe('StarButton', () => {
	it('renders Add label when not favorite', () => {
		setMocks();
		render(<StarButton entityType="agent" entityId="1" isFavorite={false} />);
		expect(
			screen.getByRole('button', { name: 'Add to favorites' }),
		).toBeDefined();
	});

	it('renders Remove label when favorite', () => {
		setMocks();
		render(<StarButton entityType="agent" entityId="1" isFavorite={true} />);
		expect(
			screen.getByRole('button', { name: 'Remove from favorites' }),
		).toBeDefined();
	});

	it('fires addFavorite mutate when star clicked and not favorite', () => {
		setMocks();
		addMutate.mockReset();
		render(<StarButton entityType="agent" entityId="abc" isFavorite={false} />);
		screen.getByRole('button', { name: 'Add to favorites' }).click();
		expect(addMutate).toHaveBeenCalledWith({
			entity_type: 'agent',
			entity_id: 'abc',
		});
	});

	it('fires removeFavorite mutate when star clicked and is favorite', () => {
		setMocks();
		removeMutate.mockReset();
		render(<StarButton entityType="agent" entityId="xyz" isFavorite={true} />);
		screen.getByRole('button', { name: 'Remove from favorites' }).click();
		expect(removeMutate).toHaveBeenCalledWith({
			entityType: 'agent',
			entityId: 'xyz',
		});
	});

	it('disables button while pending', () => {
		setMocks({ addPending: true });
		render(<StarButton entityType="agent" entityId="abc" isFavorite={false} />);
		expect(
			screen.getByRole('button', { name: 'Add to favorites' }),
		).toBeDisabled();
	});
});
