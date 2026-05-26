import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '../test/helpers';
import { NotFound } from './NotFound';

describe('NotFound', () => {
	it('renders 404 heading', () => {
		renderWithProviders(<NotFound />);
		expect(screen.getByText('404')).toBeInTheDocument();
	});

	it('renders page not found message', () => {
		renderWithProviders(<NotFound />);
		expect(screen.getByText('Page Not Found')).toBeInTheDocument();
	});

	it('renders dashboard link', () => {
		renderWithProviders(<NotFound />);
		expect(screen.getByText('Go to Dashboard')).toBeInTheDocument();
	});

	it('renders search files link', () => {
		renderWithProviders(<NotFound />);
		expect(screen.getByText('Search Files')).toBeInTheDocument();
	});
});
