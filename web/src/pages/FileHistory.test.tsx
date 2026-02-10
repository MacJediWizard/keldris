import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { BrowserRouter } from 'react-router-dom';

vi.mock('react-router-dom', async () => {
	const actual = await vi.importActual('react-router-dom');
	return {
		...actual,
		useSearchParams: () => [new URLSearchParams(), vi.fn()],
	};
});

vi.mock('../hooks/useAgents', () => ({
	useAgents: () => ({ data: [{ id: 'a1', hostname: 'server-1' }] }),
}));

vi.mock('../hooks/useRepositories', () => ({
	useRepositories: () => ({ data: [{ id: 'r1', name: 'repo-1' }] }),
}));

vi.mock('../hooks/useFileHistory', () => ({
	useFileHistory: vi.fn(() => ({ data: undefined, isLoading: false })),
}));

vi.mock('../hooks/useRestore', () => ({
	useCreateRestore: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

const { default: FileHistory } = await import('./FileHistory');

function renderPage() {
	return render(
		<BrowserRouter>
			<FileHistory />
		</BrowserRouter>,
	);
}

describe('FileHistory', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		renderPage();
		expect(screen.getByText('File History')).toBeInTheDocument();
	});

	it('shows search form', () => {
		renderPage();
		expect(screen.getByText(/Browse all versions/)).toBeInTheDocument();
	});

	it('renders agent selector', () => {
		renderPage();
		expect(screen.getByLabelText('Agent')).toBeInTheDocument();
	});

	it('renders repository selector', () => {
		renderPage();
		expect(screen.getByLabelText('Repository')).toBeInTheDocument();
	});
});
