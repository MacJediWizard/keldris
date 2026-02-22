import { useCallback, useState } from 'react';

export interface BulkSelectState<T extends string = string> {
	selectedIds: Set<T>;
	isAllSelected: boolean;
	isPartiallySelected: boolean;
	selectedCount: number;
}

export interface BulkSelectActions<T extends string = string> {
	select: (id: T) => void;
	deselect: (id: T) => void;
	toggle: (id: T) => void;
	selectAll: (ids: T[]) => void;
	deselectAll: () => void;
	toggleAll: (ids: T[]) => void;
	isSelected: (id: T) => boolean;
	clear: () => void;
}

export function useBulkSelect<T extends string = string>(
	allIds: T[] = [],
): BulkSelectState<T> & BulkSelectActions<T> {
	const [selectedIds, setSelectedIds] = useState<Set<T>>(new Set());

	const select = useCallback((id: T) => {
		setSelectedIds((prev) => new Set([...prev, id]));
	}, []);

	const deselect = useCallback((id: T) => {
		setSelectedIds((prev) => {
			const next = new Set(prev);
			next.delete(id);
			return next;
		});
	}, []);

	const toggle = useCallback((id: T) => {
		setSelectedIds((prev) => {
			const next = new Set(prev);
			if (next.has(id)) {
				next.delete(id);
			} else {
				next.add(id);
			}
			return next;
		});
	}, []);

	const selectAll = useCallback((ids: T[]) => {
		setSelectedIds(new Set(ids));
	}, []);

	const deselectAll = useCallback(() => {
		setSelectedIds(new Set());
	}, []);

	const toggleAll = useCallback(
		(ids: T[]) => {
			const allCurrentlySelected = ids.every((id) => selectedIds.has(id));
			if (allCurrentlySelected) {
				setSelectedIds(new Set());
			} else {
				setSelectedIds(new Set(ids));
			}
		},
		[selectedIds],
	);

	const isSelected = useCallback(
		(id: T) => {
			return selectedIds.has(id);
		},
		[selectedIds],
	);

	const clear = useCallback(() => {
		setSelectedIds(new Set());
	}, []);

	const selectedCount = selectedIds.size;
	const isAllSelected = allIds.length > 0 && selectedCount === allIds.length;
	const isPartiallySelected = selectedCount > 0 && !isAllSelected;

	return {
		selectedIds,
		isAllSelected,
		isPartiallySelected,
		selectedCount,
		select,
		deselect,
		toggle,
		selectAll,
		deselectAll,
		toggleAll,
		isSelected,
		clear,
	};
}
