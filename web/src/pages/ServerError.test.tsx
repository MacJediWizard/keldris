import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '../test/helpers';
import { ServerError } from './ServerError';

describe('ServerError', () => {
	it('renders 500 heading', () => {
		renderWithProviders(<ServerError />);
		expect(screen.getByText('500')).toBeInTheDocument();
	});

	it('renders something went wrong text', () => {
		renderWithProviders(<ServerError />);
		expect(screen.getByText('Something Went Wrong')).toBeInTheDocument();
	});

	it('renders try again button', () => {
		renderWithProviders(<ServerError />);
		expect(screen.getByText('Try Again')).toBeInTheDocument();
	});

	it('renders error details when error is provided', () => {
		const error = new Error('Database connection failed');
		renderWithProviders(<ServerError error={error} />);
		expect(screen.getByText('Error Details:')).toBeInTheDocument();
		expect(screen.getByText('Database connection failed')).toBeInTheDocument();
	});
});
