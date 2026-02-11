import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { type Column, DataTable } from './DataTable';

interface TestRow {
	id: string;
	name: string;
	status: string;
	[key: string]: unknown;
}

const columns: Column<TestRow>[] = [
	{ key: 'name', header: 'Name', sortable: true },
	{ key: 'status', header: 'Status' },
];

const data: TestRow[] = [
	{ id: '1', name: 'Alpha', status: 'Active' },
	{ id: '2', name: 'Charlie', status: 'Inactive' },
	{ id: '3', name: 'Bravo', status: 'Active' },
];

describe('DataTable', () => {
	it('renders column headers', () => {
		render(<DataTable columns={columns} data={data} keyField="id" />);
		expect(screen.getByText('Name')).toBeInTheDocument();
		expect(screen.getByText('Status')).toBeInTheDocument();
	});

	it('renders data rows', () => {
		render(<DataTable columns={columns} data={data} keyField="id" />);
		expect(screen.getByText('Alpha')).toBeInTheDocument();
		expect(screen.getByText('Charlie')).toBeInTheDocument();
		expect(screen.getByText('Bravo')).toBeInTheDocument();
	});

	it('shows loading spinner when loading', () => {
		const { container } = render(
			<DataTable columns={columns} data={[]} keyField="id" loading />,
		);
		expect(container.querySelector('.animate-spin')).toBeInTheDocument();
	});

	it('shows empty message when no data', () => {
		render(<DataTable columns={columns} data={[]} keyField="id" />);
		expect(screen.getByText('No data available')).toBeInTheDocument();
	});

	it('shows custom empty message', () => {
		render(
			<DataTable
				columns={columns}
				data={[]}
				keyField="id"
				emptyMessage="Nothing here"
			/>,
		);
		expect(screen.getByText('Nothing here')).toBeInTheDocument();
	});

	it('sorts data when sortable column header is clicked', () => {
		render(<DataTable columns={columns} data={data} keyField="id" />);
		fireEvent.click(screen.getByText('Name'));
		const cells = screen.getAllByRole('cell');
		const nameValues = cells
			.filter((_, i) => i % 2 === 0)
			.map((c) => c.textContent);
		expect(nameValues).toEqual(['Alpha', 'Bravo', 'Charlie']);
	});

	it('reverses sort direction on second click', () => {
		render(<DataTable columns={columns} data={data} keyField="id" />);
		fireEvent.click(screen.getByText('Name'));
		fireEvent.click(screen.getByText('Name'));
		const cells = screen.getAllByRole('cell');
		const nameValues = cells
			.filter((_, i) => i % 2 === 0)
			.map((c) => c.textContent);
		expect(nameValues).toEqual(['Charlie', 'Bravo', 'Alpha']);
	});

	it('renders custom cell content via render function', () => {
		const columnsWithRender: Column<TestRow>[] = [
			{
				key: 'name',
				header: 'Name',
				render: (row) => <strong>{row.name}</strong>,
			},
			{ key: 'status', header: 'Status' },
		];
		render(<DataTable columns={columnsWithRender} data={data} keyField="id" />);
		expect(screen.getByText('Alpha').tagName).toBe('STRONG');
	});

	it('paginates data', () => {
		render(
			<DataTable columns={columns} data={data} keyField="id" pageSize={2} />,
		);
		expect(screen.getByText('Alpha')).toBeInTheDocument();
		expect(screen.getByText('Charlie')).toBeInTheDocument();
		expect(screen.queryByText('Bravo')).not.toBeInTheDocument();
	});

	it('navigates to next page', () => {
		render(
			<DataTable columns={columns} data={data} keyField="id" pageSize={2} />,
		);
		fireEvent.click(screen.getByText('Next'));
		expect(screen.getByText('Bravo')).toBeInTheDocument();
		expect(screen.queryByText('Alpha')).not.toBeInTheDocument();
	});
});
