// Agent download URLs
export const AGENT_DOWNLOADS = {
	baseUrl: 'https://releases.keldris.io/agent',
	version: 'latest',
	platforms: [
		{
			os: 'Linux',
			arch: 'x86_64',
			filename: 'keldris-agent-linux-amd64',
			icon: 'linux',
		},
		{
			os: 'Linux',
			arch: 'ARM64',
			filename: 'keldris-agent-linux-arm64',
			icon: 'linux',
		},
		{
			os: 'macOS',
			arch: 'Intel',
			filename: 'keldris-agent-darwin-amd64',
			icon: 'apple',
		},
		{
			os: 'macOS',
			arch: 'Apple Silicon',
			filename: 'keldris-agent-darwin-arm64',
			icon: 'apple',
		},
		{
			os: 'Windows',
			arch: 'x86_64',
			filename: 'keldris-agent-windows-amd64.exe',
			icon: 'windows',
		},
	],
	installers: {
		linux: 'https://releases.keldris.io/install-linux.sh',
		macos: 'https://releases.keldris.io/install-macos.sh',
		windows: 'https://releases.keldris.io/install-windows.ps1',
	},
};

// Quick install commands for each platform
export const INSTALL_COMMANDS = {
	linux: 'curl -sSL https://releases.keldris.io/install-linux.sh | sudo bash',
	macos: 'curl -sSL https://releases.keldris.io/install-macos.sh | bash',
	windows: 'irm https://releases.keldris.io/install-windows.ps1 | iex',
};
