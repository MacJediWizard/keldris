import { useState } from 'react';
import { AGENT_DOWNLOADS, INSTALL_COMMANDS } from '../../lib/constants';

type Platform = 'linux' | 'macos' | 'windows';

interface AgentDownloadsProps {
	showInstallCommands?: boolean;
}

function PlatformIcon({ platform }: { platform: string }) {
	switch (platform) {
		case 'linux':
			return (
				<svg
					aria-hidden="true"
					className="w-5 h-5"
					viewBox="0 0 24 24"
					fill="currentColor"
				>
					<path d="M12.504 0c-.155 0-.315.008-.48.021-4.226.333-3.105 4.807-3.17 6.298-.076 1.092-.3 1.953-1.05 3.02-.885 1.051-2.127 2.75-2.716 4.521-.278.832-.41 1.684-.287 2.489a.424.424 0 00-.11.135c-.26.268-.45.6-.663.839-.199.199-.485.267-.797.4-.313.136-.658.269-.864.68-.09.189-.136.394-.132.602 0 .199.027.4.055.536.058.399.116.728.04.97-.249.68-.28 1.145-.106 1.484.174.334.535.47.94.601.81.2 1.91.135 2.774.6.926.466 1.866.67 2.616.47.526-.116.97-.464 1.208-.946.587-.003 1.23-.269 2.26-.334.699-.058 1.574.267 2.577.2.025.134.063.198.114.333l.003.003c.391.778 1.113 1.132 1.884 1.071.771-.06 1.592-.536 2.257-1.306.631-.765 1.683-1.084 2.378-1.503.348-.199.629-.469.649-.853.023-.4-.2-.811-.714-1.376v-.097l-.003-.003c-.17-.2-.25-.535-.338-.926-.085-.401-.182-.786-.492-1.046h-.003c-.059-.054-.123-.067-.188-.135a.357.357 0 00-.19-.064c.431-1.278.264-2.55-.173-3.694-.533-1.41-1.465-2.638-2.175-3.483-.796-1.005-1.576-1.957-1.56-3.368.026-2.152.236-6.133-3.544-6.139zm.529 3.405h.013c.213 0 .396.062.584.198.19.135.33.332.438.533.105.259.158.459.166.724 0-.02.006-.04.006-.06v.105a.086.086 0 01-.004-.021l-.004-.024a1.807 1.807 0 01-.15.706.953.953 0 01-.213.335.71.71 0 00-.088-.042c-.104-.045-.198-.064-.284-.133a1.312 1.312 0 00-.22-.066c.05-.06.146-.133.183-.198.053-.128.082-.264.088-.402v-.02a1.21 1.21 0 00-.061-.4c-.045-.134-.101-.2-.183-.333-.084-.066-.167-.132-.267-.132h-.016c-.093 0-.176.03-.262.132a.8.8 0 00-.205.334 1.18 1.18 0 00-.09.4v.019c.002.089.008.179.02.267-.193-.067-.438-.135-.607-.202a1.635 1.635 0 01-.018-.2v-.02a1.772 1.772 0 01.15-.768c.082-.22.232-.406.43-.533a.985.985 0 01.594-.2zm-2.962.059h.036c.142 0 .27.048.399.135.146.129.264.288.344.465.09.199.14.4.153.667v.004c.007.134.006.2-.002.266v.08c-.03.007-.056.018-.083.024-.152.055-.274.135-.393.2.012-.09.013-.18.003-.267v-.015c-.012-.133-.04-.2-.082-.333a.613.613 0 00-.166-.267.248.248 0 00-.183-.064h-.021c-.071.006-.13.04-.186.132a.552.552 0 00-.12.27.944.944 0 00-.023.33v.015c.012.135.037.2.08.334.046.134.098.2.166.268.01.009.02.018.034.024-.07.057-.117.07-.176.136a.304.304 0 01-.131.068 2.62 2.62 0 01-.275-.402 1.772 1.772 0 01-.155-.667 1.759 1.759 0 01.08-.668 1.43 1.43 0 01.283-.535c.128-.133.26-.2.418-.2zm1.37 1.706c.332 0 .733.065 1.216.399.293.2.523.269 1.052.468h.003c.255.136.405.266.478.399v-.131a.571.571 0 01.016.47c-.123.31-.516.643-1.063.842v.002c-.268.135-.501.333-.775.465-.276.135-.588.292-1.012.267a1.139 1.139 0 01-.448-.067 3.566 3.566 0 01-.322-.198c-.195-.135-.363-.332-.612-.465v-.005h-.005c-.4-.246-.616-.512-.686-.71-.07-.268-.005-.47.193-.6.224-.135.38-.271.483-.336.104-.074.143-.102.176-.131h.002v-.003c.169-.202.436-.47.839-.601.139-.036.294-.065.466-.065zm2.8 2.142c.358 1.417 1.196 3.475 1.735 4.473.286.534.855 1.659 1.102 3.024.156-.005.33.018.513.064.646-1.671-.546-3.467-1.089-3.966-.22-.2-.232-.335-.123-.335.59.534 1.365 1.572 1.646 2.757.13.535.16 1.104.021 1.67.067.028.135.06.205.067 1.032.534 1.413.938 1.23 1.537v-.043c-.06-.003-.12 0-.18 0h-.016c.151-.467-.182-.825-1.065-1.224-.915-.4-1.646-.336-1.77.465-.008.043-.013.066-.018.135-.068.023-.139.053-.209.064-.43.268-.662.669-.793 1.187-.13.533-.17 1.156-.205 1.869v.003c-.02.334-.17.838-.319 1.35-1.5 1.072-3.58 1.538-5.348.334a2.645 2.645 0 00-.402-.533 1.45 1.45 0 00-.275-.333c.182 0 .338-.03.465-.067a.615.615 0 00.314-.334c.108-.267 0-.697-.345-1.163-.345-.467-.931-.995-1.788-1.521-.63-.4-.986-.87-1.15-1.396-.165-.534-.143-1.085-.015-1.645.245-1.07.873-2.11 1.274-2.763.107-.065.037.135-.408.974-.396.751-1.14 2.497-.122 3.854a8.123 8.123 0 01.647-2.876c.564-1.278 1.743-3.504 1.836-5.268.048.036.217.135.289.202.218.133.38.333.59.465.21.201.477.335.876.335.039.003.075.006.11.006.412 0 .73-.134.997-.268.29-.134.52-.334.74-.4h.005c.467-.135.835-.402 1.044-.7zm2.185 8.958c.037.6.343 1.245.882 1.377.588.134 1.434-.333 1.791-.765l.211-.01c.315-.007.577.01.847.268l.003.003c.208.199.305.53.391.876.085.4.154.78.409 1.066.486.527.645.906.636 1.14l.003-.007v.018l-.003-.012c-.015.262-.185.396-.498.595-.63.401-1.746.712-2.457 1.57-.618.737-1.37 1.14-2.036 1.191-.664.053-1.237-.2-1.574-.898l-.005-.003c-.21-.4-.12-1.025.056-1.69.176-.668.428-1.344.463-1.897.037-.714.076-1.335.195-1.814.117-.468.32-.753.644-.93zm-9.1.422c.018.135.035.27.055.4.15.867.618 1.534 1.144 2.063.608.564 1.337 1.026 2.106 1.46.91.533 1.38.995 1.637 1.396.26.402.326.733.276 1.067-.023.066-.047.133-.082.2-.273.067-.588.135-.929.135-.693 0-1.455-.201-2.16-.733-.545-.4-1.194-.602-1.907-.668-.357-.033-.728-.066-1.075-.2-.345-.133-.655-.334-.858-.667-.146-.245-.193-.533-.078-.87.113-.333.373-.676.68-.998.155-.133.258-.267.373-.4.113-.134.14-.268.173-.467z" />
				</svg>
			);
		case 'apple':
			return (
				<svg
					aria-hidden="true"
					className="w-5 h-5"
					viewBox="0 0 24 24"
					fill="currentColor"
				>
					<path d="M12.152 6.896c-.948 0-2.415-1.078-3.96-1.04-2.04.027-3.91 1.183-4.961 3.014-2.117 3.675-.546 9.103 1.519 12.09 1.013 1.454 2.208 3.09 3.792 3.039 1.52-.065 2.09-.987 3.935-.987 1.831 0 2.35.987 3.96.948 1.637-.026 2.676-1.48 3.676-2.948 1.156-1.688 1.636-3.325 1.662-3.415-.039-.013-3.182-1.221-3.22-4.857-.026-3.04 2.48-4.494 2.597-4.559-1.429-2.09-3.623-2.324-4.39-2.376-2-.156-3.675 1.09-4.61 1.09zM15.53 3.83c.843-1.012 1.4-2.427 1.245-3.83-1.207.052-2.662.805-3.532 1.818-.78.896-1.454 2.338-1.273 3.714 1.338.104 2.715-.688 3.559-1.701z" />
				</svg>
			);
		case 'windows':
			return (
				<svg
					aria-hidden="true"
					className="w-5 h-5"
					viewBox="0 0 24 24"
					fill="currentColor"
				>
					<path d="M0 3.449L9.75 2.1v9.451H0m10.949-9.602L24 0v11.4H10.949M0 12.6h9.75v9.451L0 20.699M10.949 12.6H24V24l-12.9-1.801" />
				</svg>
			);
		default:
			return null;
	}
}

function CopyButton({ text, className }: { text: string; className?: string }) {
	const [copied, setCopied] = useState(false);

	const handleCopy = async () => {
		await navigator.clipboard.writeText(text);
		setCopied(true);
		setTimeout(() => setCopied(false), 2000);
	};

	return (
		<button
			type="button"
			onClick={handleCopy}
			className={`p-1.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 hover:bg-gray-200 dark:hover:bg-gray-700 rounded transition-colors ${className}`}
			title={copied ? 'Copied!' : 'Copy to clipboard'}
		>
			{copied ? (
				<svg
					aria-hidden="true"
					className="w-4 h-4 text-green-500"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M5 13l4 4L19 7"
					/>
				</svg>
			) : (
				<svg
					aria-hidden="true"
					className="w-4 h-4"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
					/>
				</svg>
			)}
		</button>
	);
}

export function AgentDownloads({
	showInstallCommands = true,
}: AgentDownloadsProps) {
	const [selectedPlatform, setSelectedPlatform] = useState<Platform>('linux');

	const platformTabs: { key: Platform; label: string; icon: string }[] = [
		{ key: 'linux', label: 'Linux', icon: 'linux' },
		{ key: 'macos', label: 'macOS', icon: 'apple' },
		{ key: 'windows', label: 'Windows', icon: 'windows' },
	];

	const getDownloadUrl = (filename: string) =>
		`${AGENT_DOWNLOADS.baseUrl}/${filename}`;

	const filteredPlatforms = AGENT_DOWNLOADS.platforms.filter((p) => {
		if (selectedPlatform === 'linux') return p.os === 'Linux';
		if (selectedPlatform === 'macos') return p.os === 'macOS';
		if (selectedPlatform === 'windows') return p.os === 'Windows';
		return false;
	});

	return (
		<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
			<div className="p-4 border-b border-gray-200 dark:border-gray-700">
				<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
					Download Agent
				</h3>
				<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
					Install the Keldris agent on your systems to enable backups
				</p>
			</div>

			{/* Platform tabs */}
			<div className="flex border-b border-gray-200 dark:border-gray-700">
				{platformTabs.map((tab) => (
					<button
						key={tab.key}
						type="button"
						onClick={() => setSelectedPlatform(tab.key)}
						className={`flex items-center gap-2 px-4 py-3 text-sm font-medium transition-colors ${
							selectedPlatform === tab.key
								? 'text-indigo-600 dark:text-indigo-400 border-b-2 border-indigo-600 dark:border-indigo-400 -mb-px'
								: 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200'
						}`}
					>
						<PlatformIcon platform={tab.icon} />
						{tab.label}
					</button>
				))}
			</div>

			<div className="p-4 space-y-4">
				{/* Quick install command */}
				{showInstallCommands && (
					<div>
						<span className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
							Quick Install
						</span>
						<div className="flex items-center gap-2 bg-gray-900 rounded-lg p-3">
							<code className="text-sm text-green-400 font-mono flex-1 overflow-x-auto whitespace-nowrap">
								{INSTALL_COMMANDS[selectedPlatform]}
							</code>
							<CopyButton
								text={INSTALL_COMMANDS[selectedPlatform]}
								className="text-gray-400 hover:text-gray-200 hover:bg-gray-700"
							/>
						</div>
						<p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
							{selectedPlatform === 'windows'
								? 'Run in PowerShell as Administrator'
								: selectedPlatform === 'linux'
									? 'Requires sudo for installation'
									: 'Downloads and installs automatically'}
						</p>
					</div>
				)}

				{/* Direct downloads */}
				<div>
					<span className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
						Direct Download
					</span>
					<div className="space-y-2">
						{filteredPlatforms.map((platform) => (
							<a
								key={platform.filename}
								href={getDownloadUrl(platform.filename)}
								className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 hover:bg-gray-100 dark:hover:bg-gray-600 rounded-lg transition-colors group"
							>
								<div className="flex items-center gap-3">
									<PlatformIcon platform={platform.icon} />
									<div>
										<div className="font-medium text-gray-900 dark:text-white">
											{platform.os} ({platform.arch})
										</div>
										<div className="text-sm text-gray-500 dark:text-gray-400">
											{platform.filename}
										</div>
									</div>
								</div>
								<svg
									aria-hidden="true"
									className="w-5 h-5 text-gray-400 group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
									/>
								</svg>
							</a>
						))}
					</div>
				</div>

				{/* Documentation link */}
				<div className="pt-2 border-t border-gray-200 dark:border-gray-700">
					<a
						href="/docs/agent-installation"
						target="_blank"
						rel="noopener noreferrer"
						className="text-sm text-indigo-600 dark:text-indigo-400 hover:text-indigo-700 dark:hover:text-indigo-300 font-medium"
					>
						View full installation guide
					</a>
				</div>
			</div>
		</div>
	);
}
