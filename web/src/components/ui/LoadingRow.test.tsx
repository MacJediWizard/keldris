import { render } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { LoadingRow } from './LoadingRow';

function renderRow(ui: React.ReactNode) {
	return render(
		<table>
			<tbody>{ui}</tbody>
		</table>,
	);
}

describe('LoadingRow', () => {
	it('renders one td per column', () => {
		const { container } = renderRow(
			<LoadingRow columns={[{ width: 'w-32' }, { width: 'w-20' }]} />,
		);
		expect(container.querySelectorAll('td').length).toBe(2);
	});

	it('honors pill flag with rounded-full', () => {
		const { container } = renderRow(
			<LoadingRow columns={[{ width: 'w-12', pill: true }]} />,
		);
		const bar = container.querySelector('td > div');
		expect(bar?.className.includes('rounded-full')).toBe(true);
		expect(bar?.className.includes('h-6')).toBe(true);
	});

	it('honors button flag with h-8 + inline-block', () => {
		const { container } = renderRow(
			<LoadingRow columns={[{ width: 'w-12', button: true }]} />,
		);
		const bar = container.querySelector('td > div');
		expect(bar?.className.includes('h-8')).toBe(true);
		expect(bar?.className.includes('inline-block')).toBe(true);
	});

	it('honors align=right on td', () => {
		const { container } = renderRow(
			<LoadingRow columns={[{ width: 'w-12', align: 'right' }]} />,
		);
		expect(
			container.querySelector('td')?.className.includes('text-right'),
		).toBe(true);
	});

	it('renders custom JSX when render is provided', () => {
		const { getByText } = renderRow(
			<LoadingRow columns={[{ width: 'w-12', render: <span>custom</span> }]} />,
		);
		expect(getByText('custom')).toBeDefined();
	});
});
