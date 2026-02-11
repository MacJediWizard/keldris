import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { Tab, TabList, TabPanel, Tabs } from './Tabs';

function renderTabs(onChange?: (id: string) => void) {
	return render(
		<Tabs defaultTab="tab1" onChange={onChange}>
			<TabList>
				<Tab id="tab1">Tab 1</Tab>
				<Tab id="tab2">Tab 2</Tab>
			</TabList>
			<TabPanel id="tab1">Panel 1 content</TabPanel>
			<TabPanel id="tab2">Panel 2 content</TabPanel>
		</Tabs>,
	);
}

describe('Tabs', () => {
	it('renders tab buttons', () => {
		renderTabs();
		expect(screen.getByText('Tab 1')).toBeInTheDocument();
		expect(screen.getByText('Tab 2')).toBeInTheDocument();
	});

	it('shows default tab panel', () => {
		renderTabs();
		expect(screen.getByText('Panel 1 content')).toBeInTheDocument();
		expect(screen.queryByText('Panel 2 content')).not.toBeInTheDocument();
	});

	it('switches panels on tab click', () => {
		renderTabs();
		fireEvent.click(screen.getByText('Tab 2'));
		expect(screen.queryByText('Panel 1 content')).not.toBeInTheDocument();
		expect(screen.getByText('Panel 2 content')).toBeInTheDocument();
	});

	it('calls onChange when tab is switched', () => {
		const onChange = vi.fn();
		renderTabs(onChange);
		fireEvent.click(screen.getByText('Tab 2'));
		expect(onChange).toHaveBeenCalledWith('tab2');
	});

	it('marks active tab with aria-selected', () => {
		renderTabs();
		const tab1 = screen.getByText('Tab 1');
		const tab2 = screen.getByText('Tab 2');
		expect(tab1).toHaveAttribute('aria-selected', 'true');
		expect(tab2).toHaveAttribute('aria-selected', 'false');
	});

	it('applies active styling to selected tab', () => {
		renderTabs();
		expect(screen.getByText('Tab 1')).toHaveClass('border-indigo-500');
	});

	it('renders tabpanel with correct role', () => {
		renderTabs();
		expect(screen.getByRole('tabpanel')).toBeInTheDocument();
	});

	it('renders tablist with correct role', () => {
		renderTabs();
		expect(screen.getByRole('tablist')).toBeInTheDocument();
	});
});
