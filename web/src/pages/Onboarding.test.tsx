import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../hooks/useOnboarding', () => ({
	useOnboardingStatus: vi.fn(),
	useCompleteOnboardingStep: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useSkipOnboarding: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock('../hooks/useAgents', () => ({
	useAgents: () => ({ data: [] }),
}));

vi.mock('../hooks/useOrganizations', () => ({
	useOrganizations: () => ({ data: [] }),
}));

vi.mock('../hooks/useRepositories', () => ({
	useRepositories: () => ({ data: [] }),
}));

vi.mock('../hooks/useSchedules', () => ({
	useSchedules: () => ({ data: [] }),
}));

vi.mock('../components/features/AgentDownloads', () => ({
	AgentDownloads: () => <div data-testid="agent-downloads">Downloads</div>,
}));

vi.mock('../components/ui/Stepper', () => ({
	VerticalStepper: ({ steps }: { steps: { label: string }[] }) => (
		<div data-testid="stepper">
			{steps.map((s: { label: string }) => (
				<div key={s.label}>{s.label}</div>
			))}
		</div>
	),
}));

import { useOnboardingStatus } from '../hooks/useOnboarding';

const { default: Onboarding } = await import('./Onboarding');

function renderPage() {
	return render(
		<BrowserRouter>
			<Onboarding />
		</BrowserRouter>,
	);
}

describe('Onboarding', () => {
	beforeEach(() => vi.clearAllMocks());

	it('shows loading state', () => {
		vi.mocked(useOnboardingStatus).mockReturnValue({
			data: undefined,
			isLoading: true,
		} as ReturnType<typeof useOnboardingStatus>);
		renderPage();
		const spinner = document.querySelector('.animate-spin');
		expect(spinner).not.toBeNull();
	});

	it('renders welcome step', () => {
		vi.mocked(useOnboardingStatus).mockReturnValue({
			data: { current_step: 'welcome', completed_steps: [] },
			isLoading: false,
		} as ReturnType<typeof useOnboardingStatus>);
		renderPage();
		expect(screen.getByText(/Welcome to Keldris/)).toBeInTheDocument();
	});

	it('shows skip button', () => {
		vi.mocked(useOnboardingStatus).mockReturnValue({
			data: { current_step: 'welcome', completed_steps: [] },
			isLoading: false,
		} as ReturnType<typeof useOnboardingStatus>);
		renderPage();
		expect(screen.getByText('Skip for now')).toBeInTheDocument();
	});

	it('renders stepper with steps', () => {
		vi.mocked(useOnboardingStatus).mockReturnValue({
			data: { current_step: 'welcome', completed_steps: [] },
			isLoading: false,
		} as ReturnType<typeof useOnboardingStatus>);
		renderPage();
		expect(screen.getByText('Welcome')).toBeInTheDocument();
		expect(screen.getByText('Organization')).toBeInTheDocument();
	});

	it('shows get started button on welcome step', () => {
		vi.mocked(useOnboardingStatus).mockReturnValue({
			data: { current_step: 'welcome', completed_steps: [] },
			isLoading: false,
		} as ReturnType<typeof useOnboardingStatus>);
		renderPage();
		expect(screen.getByText("Let's get started")).toBeInTheDocument();
	});

	it('shows what you will set up list on welcome', () => {
		vi.mocked(useOnboardingStatus).mockReturnValue({
			data: { current_step: 'welcome', completed_steps: [] },
			isLoading: false,
		} as ReturnType<typeof useOnboardingStatus>);
		renderPage();
		expect(screen.getByText('Organization for your team')).toBeInTheDocument();
		expect(screen.getByText('Backup storage repository')).toBeInTheDocument();
		expect(
			screen.getByText('Backup agent on your machine'),
		).toBeInTheDocument();
		expect(screen.getByText('Automated backup schedule')).toBeInTheDocument();
	});

	it('renders organization step', () => {
		vi.mocked(useOnboardingStatus).mockReturnValue({
			data: { current_step: 'organization', completed_steps: ['welcome'] },
			isLoading: false,
		} as ReturnType<typeof useOnboardingStatus>);
		renderPage();
		expect(screen.getByText('Create Your Organization')).toBeInTheDocument();
	});

	it('renders SMTP step', () => {
		vi.mocked(useOnboardingStatus).mockReturnValue({
			data: {
				current_step: 'smtp',
				completed_steps: ['welcome', 'organization'],
			},
			isLoading: false,
		} as ReturnType<typeof useOnboardingStatus>);
		renderPage();
		expect(
			screen.getByText('Configure Email Notifications'),
		).toBeInTheDocument();
	});

	it('renders repository step', () => {
		vi.mocked(useOnboardingStatus).mockReturnValue({
			data: {
				current_step: 'repository',
				completed_steps: ['welcome', 'organization', 'smtp'],
			},
			isLoading: false,
		} as ReturnType<typeof useOnboardingStatus>);
		renderPage();
		expect(screen.getByText(/Create a Repository/i)).toBeInTheDocument();
	});

	it('renders agent step with downloads', () => {
		vi.mocked(useOnboardingStatus).mockReturnValue({
			data: {
				current_step: 'agent',
				completed_steps: ['welcome', 'organization', 'smtp', 'repository'],
			},
			isLoading: false,
		} as ReturnType<typeof useOnboardingStatus>);
		renderPage();
		expect(screen.getByTestId('agent-downloads')).toBeInTheDocument();
	});

	it('renders schedule step', () => {
		vi.mocked(useOnboardingStatus).mockReturnValue({
			data: {
				current_step: 'schedule',
				completed_steps: [
					'welcome',
					'organization',
					'smtp',
					'repository',
					'agent',
				],
			},
			isLoading: false,
		} as ReturnType<typeof useOnboardingStatus>);
		renderPage();
		expect(screen.getByText('Create a Backup Schedule')).toBeInTheDocument();
	});

	it('renders verify step', () => {
		vi.mocked(useOnboardingStatus).mockReturnValue({
			data: {
				current_step: 'verify',
				completed_steps: [
					'welcome',
					'organization',
					'smtp',
					'repository',
					'agent',
					'schedule',
				],
			},
			isLoading: false,
		} as ReturnType<typeof useOnboardingStatus>);
		renderPage();
		expect(screen.getByText('Verify Your Backup Works')).toBeInTheDocument();
	});

	it('shows stepper with all step labels', () => {
		vi.mocked(useOnboardingStatus).mockReturnValue({
			data: { current_step: 'welcome', completed_steps: [] },
			isLoading: false,
		} as ReturnType<typeof useOnboardingStatus>);
		renderPage();
		expect(screen.getByText('Email Setup')).toBeInTheDocument();
		expect(screen.getByText('Repository')).toBeInTheDocument();
		expect(screen.getByText('Install Agent')).toBeInTheDocument();
		expect(screen.getByText('Schedule')).toBeInTheDocument();
		expect(screen.getByText('Verify')).toBeInTheDocument();
	});

	it('shows documentation link', () => {
		vi.mocked(useOnboardingStatus).mockReturnValue({
			data: { current_step: 'organization', completed_steps: ['welcome'] },
			isLoading: false,
		} as ReturnType<typeof useOnboardingStatus>);
		renderPage();
		expect(
			screen.getByText('Learn more about organizations'),
		).toBeInTheDocument();
	});

	it('shows SMTP step description', () => {
		vi.mocked(useOnboardingStatus).mockReturnValue({
			data: {
				current_step: 'smtp',
				completed_steps: ['welcome', 'organization'],
			},
			isLoading: false,
		} as ReturnType<typeof useOnboardingStatus>);
		renderPage();
		expect(
			screen.getByText(/optional and can be configured later/),
		).toBeInTheDocument();
	});
});
