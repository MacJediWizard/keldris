import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { NewOrganization } from './NewOrganization';

vi.mock('../hooks/useOrganizations', () => ({
	useCreateOrganization: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
}));

function renderPage() {
	return render(
		<BrowserRouter>
			<NewOrganization />
		</BrowserRouter>,
	);
}

describe('NewOrganization', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		renderPage();
		expect(screen.getByText('Create New Organization')).toBeInTheDocument();
	});

	it('renders name and slug inputs', () => {
		renderPage();
		expect(screen.getByLabelText('Organization Name')).toBeInTheDocument();
		expect(screen.getByLabelText('URL Slug')).toBeInTheDocument();
	});

	it('auto-generates slug from name', async () => {
		const user = userEvent.setup();
		renderPage();
		const nameInput = screen.getByLabelText('Organization Name');
		await user.type(nameInput, 'My Company');
		const slugInput = screen.getByLabelText('URL Slug') as HTMLInputElement;
		expect(slugInput.value).toBe('my-company');
	});

	it('shows info box', () => {
		renderPage();
		expect(screen.getByText('What happens next?')).toBeInTheDocument();
	});

	it('shows back to dashboard link', () => {
		renderPage();
		expect(screen.getByText('Back to Dashboard')).toBeInTheDocument();
	});

	it('shows cancel link', () => {
		renderPage();
		expect(screen.getByText('Cancel')).toBeInTheDocument();
	});

	it('shows create button', () => {
		renderPage();
		expect(screen.getByText('Create Organization')).toBeInTheDocument();
	});
});
