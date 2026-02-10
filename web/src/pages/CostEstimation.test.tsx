import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../hooks/useCostEstimation', () => ({
	useCostSummary: vi.fn(),
	useCostForecast: vi.fn(() => ({ data: undefined, isLoading: false })),
	useCostAlerts: vi.fn(() => ({ data: [], isLoading: false })),
	useCreateCostAlert: () => ({
		mutateAsync: vi.fn(),
		isPending: false,
		isError: false,
	}),
	useDeleteCostAlert: () => ({ mutateAsync: vi.fn(), isPending: false }),
	useUpdateCostAlert: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

import {
	useCostAlerts,
	useCostForecast,
	useCostSummary,
} from '../hooks/useCostEstimation';

const { default: CostEstimation } = await import('./CostEstimation');

function renderPage() {
	return render(
		<BrowserRouter>
			<CostEstimation />
		</BrowserRouter>,
	);
}

describe('CostEstimation', () => {
	beforeEach(() => vi.clearAllMocks());

	it('renders title', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: undefined,
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		renderPage();
		expect(screen.getByText('Cost Estimation')).toBeInTheDocument();
	});

	it('shows subtitle', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: undefined,
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		renderPage();
		expect(
			screen.getByText('Monitor and forecast your cloud storage costs'),
		).toBeInTheDocument();
	});

	it('shows loading state', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: undefined,
			isLoading: true,
		} as ReturnType<typeof useCostSummary>);
		renderPage();
		const skeletons = document.querySelectorAll('.animate-pulse');
		expect(skeletons.length).toBeGreaterThan(0);
	});

	it('renders cost summary cards', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: {
				total_monthly_cost: 150.5,
				total_yearly_cost: 1806,
				total_storage_size_gb: 5.0,
				repository_count: 3,
				by_type: {},
				repositories: [],
			},
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		renderPage();
		expect(screen.getByText('Monthly Cost')).toBeInTheDocument();
		expect(screen.getByText('Yearly Cost')).toBeInTheDocument();
	});

	it('shows tabs', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: undefined,
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		renderPage();
		expect(screen.getByText('Cost Forecast')).toBeInTheDocument();
	});

	it('shows cost summary values', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: {
				total_monthly_cost: 250.75,
				total_yearly_cost: 3009,
				total_storage_size_gb: 12.5,
				repository_count: 5,
				by_type: { s3: 200, local: 50.75 },
				repositories: [],
			},
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		renderPage();
		expect(screen.getByText('$250.75')).toBeInTheDocument();
		expect(screen.getByText('$3,009.00')).toBeInTheDocument();
	});

	it('shows storage size', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: {
				total_monthly_cost: 100,
				total_yearly_cost: 1200,
				total_storage_size_gb: 8.3,
				repository_count: 2,
				by_type: {},
				repositories: [],
			},
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		renderPage();
		expect(screen.getByText('Total Storage')).toBeInTheDocument();
	});

	it('shows repository count card', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: {
				total_monthly_cost: 100,
				total_yearly_cost: 1200,
				total_storage_size_gb: 5,
				repository_count: 4,
				by_type: {},
				repositories: [],
			},
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		renderPage();
		expect(screen.getByText('Repositories')).toBeInTheDocument();
	});

	it('shows cost by storage type section', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: {
				total_monthly_cost: 150,
				total_yearly_cost: 1800,
				total_storage_size_gb: 10,
				repository_count: 2,
				by_type: { s3: 100, local: 50 },
				repositories: [],
			},
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		renderPage();
		expect(screen.getByText('Cost by Storage Type')).toBeInTheDocument();
	});

	it('shows cost per repository section', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: {
				total_monthly_cost: 100,
				total_yearly_cost: 1200,
				total_storage_size_gb: 5,
				repository_count: 1,
				by_type: {},
				repositories: [],
			},
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		renderPage();
		expect(screen.getByText('Cost per Repository')).toBeInTheDocument();
	});

	it('shows no repositories message', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: {
				total_monthly_cost: 0,
				total_yearly_cost: 0,
				total_storage_size_gb: 0,
				repository_count: 0,
				by_type: {},
				repositories: [],
			},
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		renderPage();
		expect(
			screen.getByText('No repositories with cost data'),
		).toBeInTheDocument();
	});

	it('shows repository table with data', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: {
				total_monthly_cost: 150,
				total_yearly_cost: 1800,
				total_storage_size_gb: 10,
				repository_count: 1,
				by_type: { s3: 150 },
				repositories: [
					{
						repository_id: 'r1',
						repository_name: 'Primary S3',
						repository_type: 's3',
						monthly_cost: 150,
						yearly_cost: 1800,
						storage_size_bytes: 10737418240,
						cost_per_gb: 15,
					},
				],
			},
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		renderPage();
		expect(screen.getByText('Primary S3')).toBeInTheDocument();
		expect(screen.getByText('Total')).toBeInTheDocument();
	});

	it('shows repository table headers', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: {
				total_monthly_cost: 100,
				total_yearly_cost: 1200,
				total_storage_size_gb: 5,
				repository_count: 1,
				by_type: {},
				repositories: [
					{
						repository_id: 'r1',
						repository_name: 'Test',
						repository_type: 'local',
						monthly_cost: 100,
						yearly_cost: 1200,
						storage_size_bytes: 5368709120,
						cost_per_gb: 20,
					},
				],
			},
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		renderPage();
		expect(screen.getByText('Repository')).toBeInTheDocument();
		expect(screen.getByText('Storage')).toBeInTheDocument();
		expect(screen.getByText('Monthly')).toBeInTheDocument();
		expect(screen.getByText('Yearly')).toBeInTheDocument();
	});

	it('shows view storage stats link', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: {
				total_monthly_cost: 0,
				total_yearly_cost: 0,
				total_storage_size_gb: 0,
				repository_count: 0,
				by_type: {},
				repositories: [],
			},
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		renderPage();
		expect(screen.getByText('View Storage Stats')).toBeInTheDocument();
	});

	it('shows forecast loading state', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: undefined,
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		vi.mocked(useCostForecast).mockReturnValue({
			data: undefined,
			isLoading: true,
		} as ReturnType<typeof useCostForecast>);
		renderPage();
		expect(screen.getByText('Loading forecast...')).toBeInTheDocument();
	});

	it('shows forecast empty state', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: undefined,
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		vi.mocked(useCostForecast).mockReturnValue({
			data: { forecasts: [], current_monthly_cost: 0, monthly_growth_rate: 0 },
			isLoading: false,
		} as ReturnType<typeof useCostForecast>);
		renderPage();
		expect(
			screen.getByText('Insufficient data for forecasting'),
		).toBeInTheDocument();
	});

	it('shows cost alerts section with no alerts', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: undefined,
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		vi.mocked(useCostAlerts).mockReturnValue({
			data: [],
			isLoading: false,
		} as ReturnType<typeof useCostAlerts>);
		renderPage();
		expect(screen.getByText('No cost alerts configured')).toBeInTheDocument();
	});

	it('shows add alert button', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: undefined,
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		vi.mocked(useCostAlerts).mockReturnValue({
			data: [],
			isLoading: false,
		} as ReturnType<typeof useCostAlerts>);
		renderPage();
		expect(screen.getByText('Add Alert')).toBeInTheDocument();
	});

	it('shows cost alert form on add alert click', async () => {
		const user = (await import('@testing-library/user-event')).default.setup();
		vi.mocked(useCostSummary).mockReturnValue({
			data: undefined,
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		vi.mocked(useCostAlerts).mockReturnValue({
			data: [],
			isLoading: false,
		} as ReturnType<typeof useCostAlerts>);
		renderPage();
		await user.click(screen.getByText('Add Alert'));
		expect(screen.getByText('New Cost Alert')).toBeInTheDocument();
		expect(screen.getByLabelText('Alert Name')).toBeInTheDocument();
		expect(screen.getByLabelText('Monthly Threshold ($)')).toBeInTheDocument();
	});

	it('shows cost alerts when configured', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: undefined,
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		vi.mocked(useCostAlerts).mockReturnValue({
			data: [
				{
					id: 'a1',
					name: 'Budget Alert',
					monthly_threshold: 100,
					enabled: true,
					notify_on_exceed: true,
					notify_on_forecast: false,
					forecast_months: 3,
				},
			],
			isLoading: false,
		} as ReturnType<typeof useCostAlerts>);
		renderPage();
		expect(screen.getByText('Budget Alert')).toBeInTheDocument();
		expect(screen.getByText('Active')).toBeInTheDocument();
	});

	it('shows disabled alert badge', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: undefined,
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		vi.mocked(useCostAlerts).mockReturnValue({
			data: [
				{
					id: 'a1',
					name: 'Paused Alert',
					monthly_threshold: 200,
					enabled: false,
					notify_on_exceed: true,
					notify_on_forecast: false,
					forecast_months: 3,
				},
			],
			isLoading: false,
		} as ReturnType<typeof useCostAlerts>);
		renderPage();
		expect(screen.getByText('Disabled')).toBeInTheDocument();
	});

	it('shows card subtitles', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: {
				total_monthly_cost: 100,
				total_yearly_cost: 1200,
				total_storage_size_gb: 5,
				repository_count: 2,
				by_type: {},
				repositories: [],
			},
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		renderPage();
		expect(screen.getByText('Current estimated cost')).toBeInTheDocument();
		expect(screen.getByText('Projected annual cost')).toBeInTheDocument();
		expect(screen.getByText('Across all repositories')).toBeInTheDocument();
		expect(screen.getByText('Tracked repositories')).toBeInTheDocument();
	});

	it('shows forecast with data', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: undefined,
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		vi.mocked(useCostForecast).mockReturnValue({
			data: {
				current_monthly_cost: 100,
				monthly_growth_rate: 0.05,
				forecasts: [
					{ period: '3 months', projected_cost: 115, projected_size_gb: 12.5 },
					{ period: '6 months', projected_cost: 130, projected_size_gb: 15.0 },
				],
			},
			isLoading: false,
		} as ReturnType<typeof useCostForecast>);
		renderPage();
		expect(screen.getByText('3 months')).toBeInTheDocument();
		expect(screen.getByText('6 months')).toBeInTheDocument();
	});

	it('shows no cost data message for empty by_type', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: {
				total_monthly_cost: 0,
				total_yearly_cost: 0,
				total_storage_size_gb: 0,
				repository_count: 0,
				by_type: {},
				repositories: [],
			},
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		renderPage();
		expect(screen.getByText('No cost data available')).toBeInTheDocument();
	});

	it('shows type badges in cost breakdown', () => {
		vi.mocked(useCostSummary).mockReturnValue({
			data: {
				total_monthly_cost: 150,
				total_yearly_cost: 1800,
				total_storage_size_gb: 10,
				repository_count: 2,
				by_type: { s3: 100 },
				repositories: [],
			},
			isLoading: false,
		} as ReturnType<typeof useCostSummary>);
		renderPage();
		expect(screen.getAllByText('S3').length).toBeGreaterThan(0);
	});
});
