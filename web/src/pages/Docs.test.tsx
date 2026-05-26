import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '../test/helpers';
import { DocsPage } from './Docs';

describe('Docs page', () => {
	it('renders the documentation title when no slug provided', () => {
		renderWithProviders(<DocsPage />);
		expect(screen.getByText('Documentation')).toBeInTheDocument();
	});

	it('renders links to all doc sections', () => {
		renderWithProviders(<DocsPage />);
		expect(screen.getByText('Getting Started')).toBeInTheDocument();
		expect(screen.getByText('Organizations')).toBeInTheDocument();
	});
});
