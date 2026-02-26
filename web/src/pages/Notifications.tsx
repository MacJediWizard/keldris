import { useState } from 'react';
import {
	useCreateNotificationChannel,
	useCreateNotificationPreference,
	useDeleteNotificationChannel,
	useNotificationChannels,
	useNotificationLogs,
	useNotificationPreferences,
	useUpdateNotificationChannel,
	useUpdateNotificationPreference,
} from '../hooks/useNotifications';
import { useLicense } from '../hooks/useLicense';
import type {
	DiscordChannelConfig,
	EmailChannelConfig,
	NotificationChannel,
	NotificationChannelType,
	NotificationEventType,
	NotificationLog,
	NotificationPreference,
	PagerDutyChannelConfig,
	SlackChannelConfig,
	TeamsChannelConfig,
	WebhookChannelConfig,
} from '../lib/types';
import { formatDate } from '../lib/utils';

function LoadingRow() {
	return (
		<tr className="animate-pulse">
			<td className="px-6 py-4">
				<div className="h-4 w-32 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4">
				<div className="h-6 w-16 bg-gray-200 dark:bg-gray-700 rounded-full" />
			</td>
			<td className="px-6 py-4">
				<div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded" />
			</td>
			<td className="px-6 py-4 text-right">
				<div className="h-8 w-16 bg-gray-200 dark:bg-gray-700 rounded inline-block" />
			</td>
		</tr>
	);
}

// Channel type definitions for the type selector
const CHANNEL_TYPE_OPTIONS: {
	type: NotificationChannelType;
	label: string;
	description: string;
	featureGate?: string; // backend feature name, undefined = always available
	icon: 'email' | 'slack' | 'teams' | 'discord' | 'pagerduty' | 'webhook';
}[] = [
	{
		type: 'email',
		label: 'Email',
		description: 'Send notifications via SMTP email',
		icon: 'email',
	},
	{
		type: 'slack',
		label: 'Slack',
		description: 'Post to Slack channels via webhook',
		featureGate: 'notification_slack',
		icon: 'slack',
	},
	{
		type: 'teams',
		label: 'Microsoft Teams',
		description: 'Post to Teams channels via webhook',
		featureGate: 'notification_teams',
		icon: 'teams',
	},
	{
		type: 'discord',
		label: 'Discord',
		description: 'Post to Discord channels via webhook',
		featureGate: 'notification_discord',
		icon: 'discord',
	},
	{
		type: 'pagerduty',
		label: 'PagerDuty',
		description: 'Trigger PagerDuty incidents',
		featureGate: 'notification_pagerduty',
		icon: 'pagerduty',
	},
	{
		type: 'webhook',
		label: 'Webhook',
		description: 'Send HTTP requests to any endpoint',
		icon: 'webhook',
	},
];

function ChannelTypeIcon({
	type,
	className,
}: {
	type: string;
	className?: string;
}) {
	const cls = className || 'w-5 h-5';
	switch (type) {
		case 'email':
			return (
				<svg
					className={cls}
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
					/>
				</svg>
			);
		case 'slack':
			return (
				<svg
					className={cls}
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M7 20l4-16m2 16l4-16M6 9h14M4 15h14"
					/>
				</svg>
			);
		case 'teams':
			return (
				<svg
					className={cls}
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"
					/>
				</svg>
			);
		case 'discord':
			return (
				<svg
					className={cls}
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"
					/>
				</svg>
			);
		case 'pagerduty':
			return (
				<svg
					className={cls}
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
					/>
				</svg>
			);
		case 'webhook':
			return (
				<svg
					className={cls}
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"
					/>
				</svg>
			);
		default:
			return (
				<svg
					className={cls}
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
					/>
				</svg>
			);
	}
}

function channelIconBgColor(type: string): string {
	switch (type) {
		case 'email':
			return 'bg-indigo-100 dark:bg-indigo-900/30';
		case 'slack':
			return 'bg-purple-100 dark:bg-purple-900/30';
		case 'teams':
			return 'bg-blue-100 dark:bg-blue-900/30';
		case 'discord':
			return 'bg-violet-100 dark:bg-violet-900/30';
		case 'pagerduty':
			return 'bg-amber-100 dark:bg-amber-900/30';
		case 'webhook':
			return 'bg-emerald-100 dark:bg-emerald-900/30';
		default:
			return 'bg-gray-100 dark:bg-gray-700';
	}
}

function channelIconTextColor(type: string): string {
	switch (type) {
		case 'email':
			return 'text-indigo-600 dark:text-indigo-400';
		case 'slack':
			return 'text-purple-600 dark:text-purple-400';
		case 'teams':
			return 'text-blue-600 dark:text-blue-400';
		case 'discord':
			return 'text-violet-600 dark:text-violet-400';
		case 'pagerduty':
			return 'text-amber-600 dark:text-amber-400';
		case 'webhook':
			return 'text-emerald-600 dark:text-emerald-400';
		default:
			return 'text-gray-600 dark:text-gray-400';
	}
}

// ---------- Per-type config forms ----------

function EmailConfigForm({
	config,
	onChange,
}: {
	config: EmailChannelConfig;
	onChange: (c: EmailChannelConfig) => void;
}) {
	const inputCls =
		'w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500';
	const labelCls =
		'block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1';

	return (
		<div className="space-y-4">
			<div className="grid grid-cols-2 gap-4">
				<div>
					<label htmlFor="email-host" className={labelCls}>
						SMTP Host
					</label>
					<input
						type="text"
						id="email-host"
						value={config.host}
						onChange={(e) =>
							onChange({ ...config, host: e.target.value })
						}
						placeholder="smtp.example.com"
						className={inputCls}
						required
					/>
				</div>
				<div>
					<label htmlFor="email-port" className={labelCls}>
						Port
					</label>
					<input
						type="number"
						id="email-port"
						value={config.port}
						onChange={(e) =>
							onChange({
								...config,
								port: Number.parseInt(e.target.value, 10) || 587,
							})
						}
						className={inputCls}
						required
					/>
				</div>
			</div>
			<div>
				<label htmlFor="email-username" className={labelCls}>
					Username
				</label>
				<input
					type="text"
					id="email-username"
					value={config.username}
					onChange={(e) =>
						onChange({ ...config, username: e.target.value })
					}
					placeholder="user@example.com"
					className={inputCls}
				/>
			</div>
			<div>
				<label htmlFor="email-password" className={labelCls}>
					Password
				</label>
				<input
					type="password"
					id="email-password"
					value={config.password}
					onChange={(e) =>
						onChange({ ...config, password: e.target.value })
					}
					className={inputCls}
				/>
			</div>
			<div>
				<label htmlFor="email-from" className={labelCls}>
					From Address
				</label>
				<input
					type="email"
					id="email-from"
					value={config.from}
					onChange={(e) =>
						onChange({ ...config, from: e.target.value })
					}
					placeholder="notifications@example.com"
					className={inputCls}
					required
				/>
			</div>
			<div className="flex items-center">
				<input
					type="checkbox"
					id="email-tls"
					checked={config.tls}
					onChange={(e) =>
						onChange({ ...config, tls: e.target.checked })
					}
					className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
				/>
				<label
					htmlFor="email-tls"
					className="ml-2 text-sm text-gray-700 dark:text-gray-300"
				>
					Use TLS
				</label>
			</div>
		</div>
	);
}

function SlackConfigForm({
	config,
	onChange,
}: {
	config: SlackChannelConfig;
	onChange: (c: SlackChannelConfig) => void;
}) {
	const inputCls =
		'w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500';
	const labelCls =
		'block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1';

	return (
		<div className="space-y-4">
			<div>
				<label htmlFor="slack-webhook" className={labelCls}>
					Webhook URL
				</label>
				<input
					type="url"
					id="slack-webhook"
					value={config.webhook_url}
					onChange={(e) =>
						onChange({ ...config, webhook_url: e.target.value })
					}
					placeholder="https://hooks.slack.com/services/..."
					className={inputCls}
					required
				/>
			</div>
			<div>
				<label htmlFor="slack-channel" className={labelCls}>
					Channel
				</label>
				<input
					type="text"
					id="slack-channel"
					value={config.channel}
					onChange={(e) =>
						onChange({ ...config, channel: e.target.value })
					}
					placeholder="#alerts"
					className={inputCls}
					required
				/>
			</div>
			<div>
				<label htmlFor="slack-username" className={labelCls}>
					Bot Username{' '}
					<span className="text-gray-400 dark:text-gray-500">
						(optional)
					</span>
				</label>
				<input
					type="text"
					id="slack-username"
					value={config.username ?? ''}
					onChange={(e) =>
						onChange({ ...config, username: e.target.value || undefined })
					}
					placeholder="Keldris Bot"
					className={inputCls}
				/>
			</div>
			<div>
				<label htmlFor="slack-emoji" className={labelCls}>
					Icon Emoji{' '}
					<span className="text-gray-400 dark:text-gray-500">
						(optional)
					</span>
				</label>
				<input
					type="text"
					id="slack-emoji"
					value={config.icon_emoji ?? ''}
					onChange={(e) =>
						onChange({
							...config,
							icon_emoji: e.target.value || undefined,
						})
					}
					placeholder=":shield:"
					className={inputCls}
				/>
			</div>
		</div>
	);
}

function TeamsConfigForm({
	config,
	onChange,
}: {
	config: TeamsChannelConfig;
	onChange: (c: TeamsChannelConfig) => void;
}) {
	const inputCls =
		'w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500';
	const labelCls =
		'block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1';

	return (
		<div className="space-y-4">
			<div>
				<label htmlFor="teams-webhook" className={labelCls}>
					Webhook URL
				</label>
				<input
					type="url"
					id="teams-webhook"
					value={config.webhook_url}
					onChange={(e) =>
						onChange({ ...config, webhook_url: e.target.value })
					}
					placeholder="https://outlook.office.com/webhook/..."
					className={inputCls}
					required
				/>
			</div>
		</div>
	);
}

function DiscordConfigForm({
	config,
	onChange,
}: {
	config: DiscordChannelConfig;
	onChange: (c: DiscordChannelConfig) => void;
}) {
	const inputCls =
		'w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500';
	const labelCls =
		'block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1';

	return (
		<div className="space-y-4">
			<div>
				<label htmlFor="discord-webhook" className={labelCls}>
					Webhook URL
				</label>
				<input
					type="url"
					id="discord-webhook"
					value={config.webhook_url}
					onChange={(e) =>
						onChange({ ...config, webhook_url: e.target.value })
					}
					placeholder="https://discord.com/api/webhooks/..."
					className={inputCls}
					required
				/>
			</div>
			<div>
				<label htmlFor="discord-username" className={labelCls}>
					Bot Username{' '}
					<span className="text-gray-400 dark:text-gray-500">
						(optional)
					</span>
				</label>
				<input
					type="text"
					id="discord-username"
					value={config.username ?? ''}
					onChange={(e) =>
						onChange({ ...config, username: e.target.value || undefined })
					}
					placeholder="Keldris Bot"
					className={inputCls}
				/>
			</div>
			<div>
				<label htmlFor="discord-avatar" className={labelCls}>
					Avatar URL{' '}
					<span className="text-gray-400 dark:text-gray-500">
						(optional)
					</span>
				</label>
				<input
					type="url"
					id="discord-avatar"
					value={config.avatar_url ?? ''}
					onChange={(e) =>
						onChange({
							...config,
							avatar_url: e.target.value || undefined,
						})
					}
					placeholder="https://example.com/avatar.png"
					className={inputCls}
				/>
			</div>
		</div>
	);
}

function PagerDutyConfigForm({
	config,
	onChange,
}: {
	config: PagerDutyChannelConfig;
	onChange: (c: PagerDutyChannelConfig) => void;
}) {
	const inputCls =
		'w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500';
	const labelCls =
		'block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1';
	const selectCls =
		'w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500';

	return (
		<div className="space-y-4">
			<div>
				<label htmlFor="pd-routing-key" className={labelCls}>
					Routing Key
				</label>
				<input
					type="text"
					id="pd-routing-key"
					value={config.routing_key}
					onChange={(e) =>
						onChange({ ...config, routing_key: e.target.value })
					}
					placeholder="Integration or routing key"
					className={inputCls}
					required
				/>
			</div>
			<div>
				<label htmlFor="pd-severity" className={labelCls}>
					Severity
				</label>
				<select
					id="pd-severity"
					value={config.severity ?? 'error'}
					onChange={(e) =>
						onChange({ ...config, severity: e.target.value })
					}
					className={selectCls}
				>
					<option value="critical">Critical</option>
					<option value="error">Error</option>
					<option value="warning">Warning</option>
					<option value="info">Info</option>
				</select>
			</div>
			<div>
				<label htmlFor="pd-component" className={labelCls}>
					Component{' '}
					<span className="text-gray-400 dark:text-gray-500">
						(optional)
					</span>
				</label>
				<input
					type="text"
					id="pd-component"
					value={config.component ?? ''}
					onChange={(e) =>
						onChange({
							...config,
							component: e.target.value || undefined,
						})
					}
					placeholder="e.g., backup-service"
					className={inputCls}
				/>
			</div>
			<div>
				<label htmlFor="pd-group" className={labelCls}>
					Group{' '}
					<span className="text-gray-400 dark:text-gray-500">
						(optional)
					</span>
				</label>
				<input
					type="text"
					id="pd-group"
					value={config.group ?? ''}
					onChange={(e) =>
						onChange({
							...config,
							group: e.target.value || undefined,
						})
					}
					placeholder="e.g., infrastructure"
					className={inputCls}
				/>
			</div>
			<div>
				<label htmlFor="pd-class" className={labelCls}>
					Class{' '}
					<span className="text-gray-400 dark:text-gray-500">
						(optional)
					</span>
				</label>
				<input
					type="text"
					id="pd-class"
					value={config.class ?? ''}
					onChange={(e) =>
						onChange({
							...config,
							class: e.target.value || undefined,
						})
					}
					placeholder="e.g., backup-failure"
					className={inputCls}
				/>
			</div>
		</div>
	);
}

function WebhookConfigForm({
	config,
	onChange,
}: {
	config: WebhookChannelConfig;
	onChange: (c: WebhookChannelConfig) => void;
}) {
	const inputCls =
		'w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500';
	const labelCls =
		'block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1';
	const selectCls =
		'w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500';

	return (
		<div className="space-y-4">
			<div>
				<label htmlFor="wh-url" className={labelCls}>
					URL
				</label>
				<input
					type="url"
					id="wh-url"
					value={config.url}
					onChange={(e) =>
						onChange({ ...config, url: e.target.value })
					}
					placeholder="https://api.example.com/webhook"
					className={inputCls}
					required
				/>
			</div>
			<div className="grid grid-cols-2 gap-4">
				<div>
					<label htmlFor="wh-method" className={labelCls}>
						Method
					</label>
					<select
						id="wh-method"
						value={config.method ?? 'POST'}
						onChange={(e) =>
							onChange({ ...config, method: e.target.value })
						}
						className={selectCls}
					>
						<option value="GET">GET</option>
						<option value="POST">POST</option>
						<option value="PUT">PUT</option>
					</select>
				</div>
				<div>
					<label htmlFor="wh-content-type" className={labelCls}>
						Content Type
					</label>
					<select
						id="wh-content-type"
						value={config.content_type ?? 'application/json'}
						onChange={(e) =>
							onChange({ ...config, content_type: e.target.value })
						}
						className={selectCls}
					>
						<option value="application/json">application/json</option>
						<option value="application/x-www-form-urlencoded">
							application/x-www-form-urlencoded
						</option>
						<option value="text/plain">text/plain</option>
					</select>
				</div>
			</div>
			<div>
				<label htmlFor="wh-auth-type" className={labelCls}>
					Authentication
				</label>
				<select
					id="wh-auth-type"
					value={config.auth_type ?? 'none'}
					onChange={(e) =>
						onChange({
							...config,
							auth_type: e.target.value,
							auth_token:
								e.target.value === 'none'
									? undefined
									: config.auth_token,
						})
					}
					className={selectCls}
				>
					<option value="none">None</option>
					<option value="bearer">Bearer Token</option>
					<option value="basic">Basic Auth</option>
				</select>
			</div>
			{config.auth_type && config.auth_type !== 'none' && (
				<div>
					<label htmlFor="wh-auth-token" className={labelCls}>
						{config.auth_type === 'bearer'
							? 'Bearer Token'
							: 'Credentials (user:pass)'}
					</label>
					<input
						type="password"
						id="wh-auth-token"
						value={config.auth_token ?? ''}
						onChange={(e) =>
							onChange({
								...config,
								auth_token: e.target.value || undefined,
							})
						}
						placeholder={
							config.auth_type === 'bearer'
								? 'Token value'
								: 'username:password'
						}
						className={inputCls}
						required
					/>
				</div>
			)}
		</div>
	);
}

// Default config factories
function defaultEmailConfig(): EmailChannelConfig {
	return { host: '', port: 587, username: '', password: '', from: '', tls: true };
}
function defaultSlackConfig(): SlackChannelConfig {
	return { webhook_url: '', channel: '' };
}
function defaultTeamsConfig(): TeamsChannelConfig {
	return { webhook_url: '' };
}
function defaultDiscordConfig(): DiscordChannelConfig {
	return { webhook_url: '' };
}
function defaultPagerDutyConfig(): PagerDutyChannelConfig {
	return { routing_key: '', severity: 'error' };
}
function defaultWebhookConfig(): WebhookChannelConfig {
	return { url: '', method: 'POST', auth_type: 'none', content_type: 'application/json' };
}

function defaultConfigForType(
	type: NotificationChannelType,
):
	| EmailChannelConfig
	| SlackChannelConfig
	| TeamsChannelConfig
	| DiscordChannelConfig
	| PagerDutyChannelConfig
	| WebhookChannelConfig {
	switch (type) {
		case 'email':
			return defaultEmailConfig();
		case 'slack':
			return defaultSlackConfig();
		case 'teams':
			return defaultTeamsConfig();
		case 'discord':
			return defaultDiscordConfig();
		case 'pagerduty':
			return defaultPagerDutyConfig();
		case 'webhook':
			return defaultWebhookConfig();
	}
}

interface AddChannelModalProps {
	isOpen: boolean;
	onClose: () => void;
}

function AddChannelModal({ isOpen, onClose }: AddChannelModalProps) {
	const [selectedType, setSelectedType] =
		useState<NotificationChannelType | null>(null);
	const [name, setName] = useState('');
	const [config, setConfig] = useState<Record<string, unknown>>({});

	const { data: license } = useLicense();
	const licenseFeatures = new Set(license?.features ?? []);

	const createChannel = useCreateNotificationChannel();

	const isFeatureAvailable = (featureGate?: string): boolean => {
		if (!featureGate) return true;
		// If there is no license or no features loaded, allow free-tier channels only
		if (licenseFeatures.size === 0) return false;
		return licenseFeatures.has(featureGate);
	};

	const handleSelectType = (type: NotificationChannelType) => {
		setSelectedType(type);
		setConfig(defaultConfigForType(type) as unknown as Record<string, unknown>);
	};

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		if (!selectedType) return;
		try {
			await createChannel.mutateAsync({
				name,
				type: selectedType,
				config,
			});
			resetForm();
			onClose();
		} catch {
			// Error handled by mutation
		}
	};

	const resetForm = () => {
		setSelectedType(null);
		setName('');
		setConfig({});
	};

	const handleClose = () => {
		resetForm();
		onClose();
	};

	const handleBack = () => {
		setSelectedType(null);
		setName('');
		setConfig({});
	};

	if (!isOpen) return null;

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
				{/* Step 1: Type selector */}
				{!selectedType && (
					<>
						<h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
							Add Notification Channel
						</h3>
						<p className="text-sm text-gray-500 dark:text-gray-400 mb-4">
							Choose a channel type to get started.
						</p>
						<div className="grid grid-cols-2 gap-3">
							{CHANNEL_TYPE_OPTIONS.map((opt) => {
								const available = isFeatureAvailable(opt.featureGate);
								return (
									<button
										key={opt.type}
										type="button"
										disabled={!available}
										onClick={() => handleSelectType(opt.type)}
										className={`relative flex flex-col items-center gap-2 p-4 rounded-lg border-2 text-center transition-colors ${
											available
												? 'border-gray-200 dark:border-gray-600 hover:border-indigo-500 dark:hover:border-indigo-400 cursor-pointer'
												: 'border-gray-200 dark:border-gray-700 opacity-60 cursor-not-allowed'
										}`}
									>
										{!available && (
											<div className="absolute top-2 right-2 flex items-center gap-1">
												<svg
													className="w-3.5 h-3.5 text-gray-400 dark:text-gray-500"
													fill="none"
													stroke="currentColor"
													viewBox="0 0 24 24"
													aria-hidden="true"
												>
													<path
														strokeLinecap="round"
														strokeLinejoin="round"
														strokeWidth={2}
														d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
													/>
												</svg>
												<span className="text-[10px] font-medium text-gray-400 dark:text-gray-500">
													Pro
												</span>
											</div>
										)}
										<div
											className={`p-2 rounded-lg ${channelIconBgColor(opt.icon)}`}
										>
											<ChannelTypeIcon
												type={opt.icon}
												className={`w-6 h-6 ${channelIconTextColor(opt.icon)}`}
											/>
										</div>
										<span className="text-sm font-medium text-gray-900 dark:text-white">
											{opt.label}
										</span>
										<span className="text-xs text-gray-500 dark:text-gray-400">
											{opt.description}
										</span>
									</button>
								);
							})}
						</div>
						<div className="flex justify-end mt-6">
							<button
								type="button"
								onClick={handleClose}
								className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
							>
								Cancel
							</button>
						</div>
					</>
				)}

				{/* Step 2: Config form */}
				{selectedType && (
					<>
						<div className="flex items-center gap-3 mb-4">
							<button
								type="button"
								onClick={handleBack}
								className="p-1 rounded hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
								aria-label="Back to channel type selection"
							>
								<svg
									className="w-5 h-5 text-gray-500 dark:text-gray-400"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
									aria-hidden="true"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M15 19l-7-7 7-7"
									/>
								</svg>
							</button>
							<div
								className={`p-1.5 rounded-lg ${channelIconBgColor(selectedType)}`}
							>
								<ChannelTypeIcon
									type={selectedType}
									className={`w-4 h-4 ${channelIconTextColor(selectedType)}`}
								/>
							</div>
							<h3 className="text-lg font-semibold text-gray-900 dark:text-white">
								Add{' '}
								{CHANNEL_TYPE_OPTIONS.find(
									(o) => o.type === selectedType,
								)?.label ?? selectedType}{' '}
								Channel
							</h3>
						</div>
						<form onSubmit={handleSubmit}>
							<div className="space-y-4">
								<div>
									<label
										htmlFor="channel-name"
										className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
									>
										Channel Name
									</label>
									<input
										type="text"
										id="channel-name"
										value={name}
										onChange={(e) => setName(e.target.value)}
										placeholder={`e.g., Primary ${
											CHANNEL_TYPE_OPTIONS.find(
												(o) => o.type === selectedType,
											)?.label ?? ''
										}`}
										className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
										required
									/>
								</div>

								{selectedType === 'email' && (
									<EmailConfigForm
										config={config as unknown as EmailChannelConfig}
										onChange={(c) =>
											setConfig(
												c as unknown as Record<string, unknown>,
											)
										}
									/>
								)}
								{selectedType === 'slack' && (
									<SlackConfigForm
										config={config as unknown as SlackChannelConfig}
										onChange={(c) =>
											setConfig(
												c as unknown as Record<string, unknown>,
											)
										}
									/>
								)}
								{selectedType === 'teams' && (
									<TeamsConfigForm
										config={config as unknown as TeamsChannelConfig}
										onChange={(c) =>
											setConfig(
												c as unknown as Record<string, unknown>,
											)
										}
									/>
								)}
								{selectedType === 'discord' && (
									<DiscordConfigForm
										config={config as unknown as DiscordChannelConfig}
										onChange={(c) =>
											setConfig(
												c as unknown as Record<string, unknown>,
											)
										}
									/>
								)}
								{selectedType === 'pagerduty' && (
									<PagerDutyConfigForm
										config={
											config as unknown as PagerDutyChannelConfig
										}
										onChange={(c) =>
											setConfig(
												c as unknown as Record<string, unknown>,
											)
										}
									/>
								)}
								{selectedType === 'webhook' && (
									<WebhookConfigForm
										config={
											config as unknown as WebhookChannelConfig
										}
										onChange={(c) =>
											setConfig(
												c as unknown as Record<string, unknown>,
											)
										}
									/>
								)}
							</div>
							{createChannel.isError && (
								<p className="text-sm text-red-600 dark:text-red-400 mt-4">
									Failed to create channel. Please try again.
								</p>
							)}
							<div className="flex justify-end gap-3 mt-6">
								<button
									type="button"
									onClick={handleClose}
									className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
								>
									Cancel
								</button>
								<button
									type="submit"
									disabled={createChannel.isPending}
									className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
								>
									{createChannel.isPending
										? 'Creating...'
										: 'Create Channel'}
								</button>
							</div>
						</form>
					</>
				)}
			</div>
		</div>
	);
}

interface ChannelRowProps {
	channel: NotificationChannel;
	preferences: NotificationPreference[];
	onDelete: (id: string) => void;
	isDeleting: boolean;
}

const EVENT_TYPES: { type: NotificationEventType; label: string }[] = [
	{ type: 'backup_success', label: 'Backup Success' },
	{ type: 'backup_failed', label: 'Backup Failed' },
	{ type: 'agent_offline', label: 'Agent Offline' },
];

function ChannelRow({
	channel,
	preferences,
	onDelete,
	isDeleting,
}: ChannelRowProps) {
	const updateChannel = useUpdateNotificationChannel();
	const createPreference = useCreateNotificationPreference();
	const updatePreference = useUpdateNotificationPreference();

	const handleToggleChannel = async () => {
		await updateChannel.mutateAsync({
			id: channel.id,
			data: { enabled: !channel.enabled },
		});
	};

	const handleToggleEvent = async (eventType: NotificationEventType) => {
		const pref = preferences.find((p) => p.event_type === eventType);
		if (pref) {
			await updatePreference.mutateAsync({
				id: pref.id,
				data: { enabled: !pref.enabled },
			});
		} else {
			await createPreference.mutateAsync({
				channel_id: channel.id,
				event_type: eventType,
				enabled: true,
			});
		}
	};

	const isEventEnabled = (eventType: NotificationEventType) => {
		const pref = preferences.find((p) => p.event_type === eventType);
		return pref?.enabled ?? false;
	};

	return (
		<tr className="hover:bg-gray-50 dark:hover:bg-gray-700">
			<td className="px-6 py-4">
				<div className="flex items-center gap-3">
					<div
						className={`p-2 rounded-lg ${channelIconBgColor(channel.type)}`}
					>
						<ChannelTypeIcon
							type={channel.type}
							className={`w-5 h-5 ${channelIconTextColor(channel.type)}`}
						/>
					</div>
					<div>
						<p className="font-medium text-gray-900 dark:text-white">
							{channel.name}
						</p>
						<p className="text-sm text-gray-500 dark:text-gray-400">
							{channel.type}
						</p>
					</div>
				</div>
			</td>
			<td className="px-6 py-4">
				<button
					type="button"
					onClick={handleToggleChannel}
					disabled={updateChannel.isPending}
					className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-indigo-600 focus:ring-offset-2 ${
						channel.enabled ? 'bg-indigo-600' : 'bg-gray-200'
					}`}
				>
					<span
						className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
							channel.enabled ? 'translate-x-5' : 'translate-x-0'
						}`}
					/>
				</button>
			</td>
			<td className="px-6 py-4">
				<div className="flex flex-wrap gap-2">
					{EVENT_TYPES.map(({ type, label }) => {
						const enabled = isEventEnabled(type);
						return (
							<button
								key={type}
								type="button"
								onClick={() => handleToggleEvent(type)}
								disabled={
									createPreference.isPending || updatePreference.isPending
								}
								className={`px-2 py-1 text-xs font-medium rounded-full transition-colors ${
									enabled
										? 'bg-green-100 text-green-800 hover:bg-green-200 dark:bg-green-900/30 dark:text-green-400 dark:hover:bg-green-900/50'
										: 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-400 dark:hover:bg-gray-600'
								}`}
							>
								{label}
							</button>
						);
					})}
				</div>
			</td>
			<td className="px-6 py-4 text-right">
				<button
					type="button"
					onClick={() => onDelete(channel.id)}
					disabled={isDeleting}
					className="text-red-600 hover:text-red-800 text-sm font-medium disabled:opacity-50"
				>
					Delete
				</button>
			</td>
		</tr>
	);
}

function getStatusColor(status: string): { bg: string; text: string } {
	switch (status) {
		case 'sent':
			return {
				bg: 'bg-green-100 dark:bg-green-900/30',
				text: 'text-green-800 dark:text-green-400',
			};
		case 'failed':
			return {
				bg: 'bg-red-100 dark:bg-red-900/30',
				text: 'text-red-800 dark:text-red-400',
			};
		case 'queued':
			return {
				bg: 'bg-yellow-100 dark:bg-yellow-900/30',
				text: 'text-yellow-800 dark:text-yellow-400',
			};
		default:
			return {
				bg: 'bg-gray-100 dark:bg-gray-700',
				text: 'text-gray-800 dark:text-gray-300',
			};
	}
}

function LogRow({ log }: { log: NotificationLog }) {
	const statusColor = getStatusColor(log.status);
	return (
		<tr className="hover:bg-gray-50 dark:hover:bg-gray-700">
			<td className="px-6 py-4 text-sm text-gray-900 dark:text-white">
				{log.subject || log.event_type}
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
				{log.recipient}
			</td>
			<td className="px-6 py-4">
				<span
					className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor.bg} ${statusColor.text}`}
				>
					{log.status}
				</span>
			</td>
			<td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
				{formatDate(log.sent_at || log.created_at)}
			</td>
			<td className="px-6 py-4 text-sm text-red-500 dark:text-red-400">
				{log.error_message || '-'}
			</td>
		</tr>
	);
}

export function Notifications() {
	const [showAddModal, setShowAddModal] = useState(false);
	const [activeTab, setActiveTab] = useState<'channels' | 'logs'>('channels');

	const {
		data: channels,
		isLoading: channelsLoading,
		isError: channelsError,
	} = useNotificationChannels();
	const { data: preferences } = useNotificationPreferences();
	const {
		data: logs,
		isLoading: logsLoading,
		isError: logsError,
	} = useNotificationLogs();
	const deleteChannel = useDeleteNotificationChannel();

	const handleDelete = (id: string) => {
		if (confirm('Are you sure you want to delete this notification channel?')) {
			deleteChannel.mutate(id);
		}
	};

	const getPreferencesForChannel = (channelId: string) =>
		preferences?.filter((p) => p.channel_id === channelId) ?? [];

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold text-gray-900 dark:text-white">
						Notifications
					</h1>
					<p className="text-gray-500 mt-1">
						Configure notification channels for backup events
					</p>
				</div>
				<button
					type="button"
					onClick={() => setShowAddModal(true)}
					className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors flex items-center gap-2"
				>
					<svg
						className="w-5 h-5"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
						aria-hidden="true"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M12 4v16m8-8H4"
						/>
					</svg>
					Add Channel
				</button>
			</div>

			{/* Tabs */}
			<div className="border-b border-gray-200 dark:border-gray-700">
				<nav className="-mb-px flex space-x-8">
					<button
						type="button"
						onClick={() => setActiveTab('channels')}
						className={`py-4 px-1 border-b-2 font-medium text-sm ${
							activeTab === 'channels'
								? 'border-indigo-500 text-indigo-600'
								: 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
						}`}
					>
						Channels
					</button>
					<button
						type="button"
						onClick={() => setActiveTab('logs')}
						className={`py-4 px-1 border-b-2 font-medium text-sm ${
							activeTab === 'logs'
								? 'border-indigo-500 text-indigo-600'
								: 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
						}`}
					>
						History
					</button>
				</nav>
			</div>

			{activeTab === 'channels' && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
					{channelsError ? (
						<div className="p-12 text-center text-red-500 dark:text-red-400">
							<p className="font-medium">
								Failed to load notification channels
							</p>
							<p className="text-sm mt-1">Please try refreshing the page</p>
						</div>
					) : channelsLoading ? (
						<table className="w-full">
							<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Channel
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Enabled
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Events
									</th>
									<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Actions
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
								{[1, 2, 3].map((i) => (
									<LoadingRow key={i} />
								))}
							</tbody>
						</table>
					) : channels && channels.length > 0 ? (
						<table className="w-full">
							<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Channel
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Enabled
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Events
									</th>
									<th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Actions
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
								{channels.map((channel) => (
									<ChannelRow
										key={channel.id}
										channel={channel}
										preferences={getPreferencesForChannel(channel.id)}
										onDelete={handleDelete}
										isDeleting={deleteChannel.isPending}
									/>
								))}
							</tbody>
						</table>
					) : (
						<div className="p-12 text-center text-gray-500 dark:text-gray-400">
							<div className="inline-flex items-center justify-center w-12 h-12 bg-gray-100 dark:bg-gray-700 rounded-full mb-4">
								<svg
									className="w-6 h-6 text-gray-400"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
									aria-hidden="true"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
									/>
								</svg>
							</div>
							<p className="font-medium">No notification channels configured</p>
							<p className="text-sm mt-1">
								Add a channel to receive backup notifications
							</p>
							<button
								type="button"
								onClick={() => setShowAddModal(true)}
								className="mt-4 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
							>
								Add Channel
							</button>
						</div>
					)}
				</div>
			)}

			{activeTab === 'logs' && (
				<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
					{logsError ? (
						<div className="p-12 text-center text-red-500 dark:text-red-400">
							<p className="font-medium">Failed to load notification history</p>
							<p className="text-sm mt-1">Please try refreshing the page</p>
						</div>
					) : logsLoading ? (
						<table className="w-full">
							<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Subject
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Recipient
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Status
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Sent
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Error
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
								{[1, 2, 3].map((i) => (
									<LoadingRow key={i} />
								))}
							</tbody>
						</table>
					) : logs && logs.length > 0 ? (
						<table className="w-full">
							<thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
								<tr>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Subject
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Recipient
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Status
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Sent
									</th>
									<th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
										Error
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-gray-200 dark:divide-gray-700">
								{logs.map((log) => (
									<LogRow key={log.id} log={log} />
								))}
							</tbody>
						</table>
					) : (
						<div className="p-12 text-center text-gray-500 dark:text-gray-400">
							<div className="inline-flex items-center justify-center w-12 h-12 bg-gray-100 dark:bg-gray-700 rounded-full mb-4">
								<svg
									className="w-6 h-6 text-gray-400"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
									aria-hidden="true"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
									/>
								</svg>
							</div>
							<p className="font-medium">No notification history</p>
							<p className="text-sm mt-1">
								Notifications will appear here once they are sent
							</p>
						</div>
					)}
				</div>
			)}

			<AddChannelModal
				isOpen={showAddModal}
				onClose={() => setShowAddModal(false)}
			/>
		</div>
	);
}

export default Notifications;
