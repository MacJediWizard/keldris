import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { Pagination } from './Pagination';

describe('Pagination', () => {
	it('renders nothing when totalPages is 1', () => {
		const { container } = render(
			<Pagination currentPage={1} totalPages={1} onPageChange={vi.fn()} />,
		);
		expect(container).toBeEmptyDOMElement();
	});

	it('renders Previous and Next buttons', () => {
		render(
			<Pagination currentPage={2} totalPages={5} onPageChange={vi.fn()} />,
		);
		expect(screen.getByText('Previous')).toBeInTheDocument();
		expect(screen.getByText('Next')).toBeInTheDocument();
	});

	it('disables Previous on first page', () => {
		render(
			<Pagination currentPage={1} totalPages={5} onPageChange={vi.fn()} />,
		);
		expect(screen.getByText('Previous')).toBeDisabled();
	});

	it('disables Next on last page', () => {
		render(
			<Pagination currentPage={5} totalPages={5} onPageChange={vi.fn()} />,
		);
		expect(screen.getByText('Next')).toBeDisabled();
	});

	it('calls onPageChange when page is clicked', () => {
		const onPageChange = vi.fn();
		render(
			<Pagination currentPage={1} totalPages={5} onPageChange={onPageChange} />,
		);
		fireEvent.click(screen.getByText('3'));
		expect(onPageChange).toHaveBeenCalledWith(3);
	});

	it('calls onPageChange when Next is clicked', () => {
		const onPageChange = vi.fn();
		render(
			<Pagination currentPage={2} totalPages={5} onPageChange={onPageChange} />,
		);
		fireEvent.click(screen.getByText('Next'));
		expect(onPageChange).toHaveBeenCalledWith(3);
	});

	it('calls onPageChange when Previous is clicked', () => {
		const onPageChange = vi.fn();
		render(
			<Pagination currentPage={3} totalPages={5} onPageChange={onPageChange} />,
		);
		fireEvent.click(screen.getByText('Previous'));
		expect(onPageChange).toHaveBeenCalledWith(2);
	});

	it('marks current page with aria-current', () => {
		render(
			<Pagination currentPage={2} totalPages={5} onPageChange={vi.fn()} />,
		);
		expect(screen.getByText('2')).toHaveAttribute('aria-current', 'page');
	});

	it('highlights current page with active styling', () => {
		render(
			<Pagination currentPage={2} totalPages={5} onPageChange={vi.fn()} />,
		);
		expect(screen.getByText('2')).toHaveClass('bg-indigo-600', 'text-white');
	});
});
