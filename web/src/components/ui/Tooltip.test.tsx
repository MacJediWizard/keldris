import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Tooltip } from './Tooltip';

function getWrapper() {
	const el = screen.getByText('Trigger').closest('div');
	if (!el) throw new Error('Wrapper not found');
	return el;
}

describe('Tooltip', () => {
	it('renders children', () => {
		render(
			<Tooltip content="Help text">
				<button type="button">Hover me</button>
			</Tooltip>,
		);
		expect(screen.getByText('Hover me')).toBeInTheDocument();
	});

	it('does not show tooltip by default', () => {
		render(
			<Tooltip content="Help text">
				<span>Trigger</span>
			</Tooltip>,
		);
		expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();
	});

	it('shows tooltip on mouse enter', () => {
		render(
			<Tooltip content="Help text">
				<span>Trigger</span>
			</Tooltip>,
		);
		fireEvent.mouseEnter(getWrapper());
		expect(screen.getByRole('tooltip')).toBeInTheDocument();
		expect(screen.getByText('Help text')).toBeInTheDocument();
	});

	it('hides tooltip on mouse leave', () => {
		render(
			<Tooltip content="Help text">
				<span>Trigger</span>
			</Tooltip>,
		);
		const wrapper = getWrapper();
		fireEvent.mouseEnter(wrapper);
		expect(screen.getByRole('tooltip')).toBeInTheDocument();
		fireEvent.mouseLeave(wrapper);
		expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();
	});

	it('positions tooltip at top by default', () => {
		render(
			<Tooltip content="Help text">
				<span>Trigger</span>
			</Tooltip>,
		);
		fireEvent.mouseEnter(getWrapper());
		expect(screen.getByRole('tooltip')).toHaveClass('bottom-full');
	});

	it('positions tooltip at bottom', () => {
		render(
			<Tooltip content="Help text" position="bottom">
				<span>Trigger</span>
			</Tooltip>,
		);
		fireEvent.mouseEnter(getWrapper());
		expect(screen.getByRole('tooltip')).toHaveClass('top-full');
	});
});
