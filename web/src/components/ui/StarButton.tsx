import { useAddFavorite, useRemoveFavorite } from '../../hooks/useFavorites';
import type { FavoriteEntityType } from '../../lib/types';

interface StarButtonProps {
	entityType: FavoriteEntityType;
	entityId: string;
	isFavorite: boolean;
	size?: 'sm' | 'md';
}

export function StarButton({
	entityType,
	entityId,
	isFavorite,
	size = 'md',
}: StarButtonProps) {
	const addFavorite = useAddFavorite();
	const removeFavorite = useRemoveFavorite();

	const isLoading = addFavorite.isPending || removeFavorite.isPending;

	const handleClick = (e: React.MouseEvent) => {
		e.stopPropagation();
		e.preventDefault();

		if (isLoading) return;

		if (isFavorite) {
			removeFavorite.mutate({ entityType, entityId });
		} else {
			addFavorite.mutate({ entity_type: entityType, entity_id: entityId });
		}
	};

	const sizeClasses = size === 'sm' ? 'w-4 h-4' : 'w-5 h-5';
	const buttonClasses = size === 'sm' ? 'p-0.5 -m-0.5' : 'p-1 -m-1';

	return (
		<button
			type="button"
			onClick={handleClick}
			disabled={isLoading}
			className={`${buttonClasses} rounded hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors disabled:opacity-50`}
			title={isFavorite ? 'Remove from favorites' : 'Add to favorites'}
		>
			<svg
				aria-hidden="true"
				className={`${sizeClasses} ${
					isFavorite
						? 'text-yellow-400 fill-current'
						: 'text-gray-400 hover:text-yellow-400'
				} transition-colors`}
				fill={isFavorite ? 'currentColor' : 'none'}
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					strokeLinecap="round"
					strokeLinejoin="round"
					strokeWidth={2}
					d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
				/>
			</svg>
		</button>
	);
}
