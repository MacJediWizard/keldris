import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { AgentDownloads } from './AgentDownloads';

// Mock clipboard API
Object.assign(navigator, {
	clipboard: {
		writeText: vi.fn().mockResolvedValue(undefined),
	},
});

describe('AgentDownloads', () => {
	it('renders the download section title', () => {
		render(<AgentDownloads />);
		expect(screen.getByText('Download Agent')).toBeInTheDocument();
	});

	it('renders the description', () => {
		render(<AgentDownloads />);
		expect(
			screen.getByText(
				'Install the Keldris agent on your systems to enable backups',
			),
		).toBeInTheDocument();
	});

	it('renders platform tabs', () => {
		render(<AgentDownloads />);
		expect(screen.getByText('Linux')).toBeInTheDocument();
		expect(screen.getByText('macOS')).toBeInTheDocument();
		expect(screen.getByText('Windows')).toBeInTheDocument();
	});

	it('shows Linux downloads by default', () => {
		render(<AgentDownloads />);
		expect(screen.getByText('Linux (x86_64)')).toBeInTheDocument();
		expect(screen.getByText('Linux (ARM64)')).toBeInTheDocument();
	});

	it('switches to macOS downloads when tab is clicked', async () => {
		const user = userEvent.setup();
		render(<AgentDownloads />);

		await user.click(screen.getByText('macOS'));

		expect(screen.getByText('macOS (Intel)')).toBeInTheDocument();
		expect(screen.getByText('macOS (Apple Silicon)')).toBeInTheDocument();
	});

	it('switches to Windows downloads when tab is clicked', async () => {
		const user = userEvent.setup();
		render(<AgentDownloads />);

		await user.click(screen.getByText('Windows'));

		expect(screen.getByText('Windows (x86_64)')).toBeInTheDocument();
	});

	it('shows quick install commands by default', () => {
		render(<AgentDownloads />);
		expect(screen.getByText('Quick Install')).toBeInTheDocument();
	});

	it('hides quick install commands when showInstallCommands is false', () => {
		render(<AgentDownloads showInstallCommands={false} />);
		expect(screen.queryByText('Quick Install')).not.toBeInTheDocument();
	});

	it('renders the documentation link', () => {
		render(<AgentDownloads />);
		expect(
			screen.getByText('View full installation guide'),
		).toBeInTheDocument();
	});
});
