/**
 * Shared skeleton loading row for tables.
 *
 * Each item in the `columns` array describes one `<td>`:
 *   - `width`  – Tailwind width class for the placeholder bar (e.g. "w-32")
 *   - `pill`   – if true, uses rounded-full + taller height (badge look)
 *   - `button` – if true, uses h-8 + inline-block (action button look)
 *   - `align`  – "right" adds `text-right` to the cell
 *   - `tdClassName` – extra classes on the `<td>` itself
 *   - `barClassName` – completely override the inner `<div>` class
 *   - `render` – custom JSX for the cell content (overrides all bar options)
 */

import type { ReactNode } from 'react';

export interface LoadingRowColumn {
	width: string;
	pill?: boolean;
	button?: boolean;
	align?: 'right';
	tdClassName?: string;
	barClassName?: string;
	render?: ReactNode;
}

interface LoadingRowProps {
	columns: LoadingRowColumn[];
}

export function LoadingRow({ columns }: LoadingRowProps) {
	return (
		<tr className="animate-pulse">
			{columns.map((col, i) => {
				const tdClass = [
					'px-6 py-4',
					col.align === 'right' ? 'text-right' : '',
					col.tdClassName ?? '',
				]
					.filter(Boolean)
					.join(' ');

				if (col.render) {
					return (
						// biome-ignore lint/suspicious/noArrayIndexKey: static skeleton columns have no stable key
						<td key={i} className={tdClass}>
							{col.render}
						</td>
					);
				}

				const height = col.pill ? 'h-6' : col.button ? 'h-8' : 'h-4';
				const rounding = col.pill ? 'rounded-full' : 'rounded';
				const display = col.button ? 'inline-block' : '';

				const barClass =
					col.barClassName ??
					`${height} ${col.width} bg-gray-200 dark:bg-gray-700 ${rounding} ${display}`.trim();

				return (
					// biome-ignore lint/suspicious/noArrayIndexKey: static skeleton columns have no stable key
					<td key={i} className={tdClass}>
						<div className={barClass} />
					</td>
				);
			})}
		</tr>
	);
}
