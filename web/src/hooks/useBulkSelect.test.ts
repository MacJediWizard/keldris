import { act, renderHook } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { useBulkSelect } from './useBulkSelect';

describe('useBulkSelect', () => {
	it('initialises with empty selection', () => {
		const { result } = renderHook(() => useBulkSelect<string>(['a', 'b', 'c']));

		expect(result.current.selectedCount).toBe(0);
		expect(result.current.isAllSelected).toBe(false);
		expect(result.current.isPartiallySelected).toBe(false);
		expect(result.current.selectedIds.size).toBe(0);
	});

	it('toggles selection of individual ids', () => {
		const { result } = renderHook(() => useBulkSelect<string>(['a', 'b', 'c']));

		act(() => {
			result.current.toggle('a');
		});

		expect(result.current.selectedCount).toBe(1);
		expect(result.current.isSelected('a')).toBe(true);
		expect(result.current.isPartiallySelected).toBe(true);

		act(() => {
			result.current.toggle('a');
		});

		expect(result.current.selectedCount).toBe(0);
		expect(result.current.isSelected('a')).toBe(false);
	});

	it('selects and deselects all ids', () => {
		const { result } = renderHook(() => useBulkSelect<string>(['a', 'b', 'c']));

		act(() => {
			result.current.selectAll(['a', 'b', 'c']);
		});

		expect(result.current.selectedCount).toBe(3);
		expect(result.current.isAllSelected).toBe(true);
		expect(result.current.isPartiallySelected).toBe(false);

		act(() => {
			result.current.deselectAll();
		});

		expect(result.current.selectedCount).toBe(0);
		expect(result.current.isAllSelected).toBe(false);
	});
});
