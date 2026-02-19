import { Link, useParams } from 'react-router-dom';

const docs: Record<string, { title: string; content: JSX.Element }> = {
	'getting-started': {
		title: 'Getting Started',
		content: (
			<div className="space-y-6">
				<p>
					Welcome to Keldris — your self-hosted backup solution. This guide
					walks you through initial setup.
				</p>
				<h2 className="text-xl font-semibold">Prerequisites</h2>
				<ul className="list-disc list-inside space-y-1">
					<li>Docker and Docker Compose installed</li>
					<li>PostgreSQL 15+ (included in the Docker Compose stack)</li>
					<li>An OIDC provider (e.g., Authentik, Keycloak, Auth0)</li>
				</ul>
				<h2 className="text-xl font-semibold">Quick Start</h2>
				<ol className="list-decimal list-inside space-y-2">
					<li>
						Pull the latest images and start the stack with{' '}
						<code className="bg-gray-100 dark:bg-gray-700 px-1 rounded">
							docker compose up -d
						</code>
					</li>
					<li>
						Configure the environment variables (DATABASE_URL, OIDC settings,
						ENCRYPTION_KEY, SESSION_SECRET)
					</li>
					<li>Open the web UI and sign in with your OIDC provider</li>
					<li>Follow the onboarding wizard to complete setup</li>
				</ol>
				<h2 className="text-xl font-semibold">Environment Variables</h2>
				<div className="overflow-x-auto">
					<table className="min-w-full text-sm">
						<thead>
							<tr className="border-b dark:border-gray-700">
								<th className="text-left py-2 pr-4 font-medium">Variable</th>
								<th className="text-left py-2 pr-4 font-medium">Required</th>
								<th className="text-left py-2 font-medium">Description</th>
							</tr>
						</thead>
						<tbody className="divide-y dark:divide-gray-700">
							<tr>
								<td className="py-2 pr-4 font-mono text-xs">DATABASE_URL</td>
								<td className="py-2 pr-4">Yes</td>
								<td className="py-2">PostgreSQL connection string</td>
							</tr>
							<tr>
								<td className="py-2 pr-4 font-mono text-xs">ENCRYPTION_KEY</td>
								<td className="py-2 pr-4">Yes</td>
								<td className="py-2">
									Hex-encoded 32-byte key (
									<code className="bg-gray-100 dark:bg-gray-700 px-1 rounded">
										openssl rand -hex 32
									</code>
									)
								</td>
							</tr>
							<tr>
								<td className="py-2 pr-4 font-mono text-xs">SESSION_SECRET</td>
								<td className="py-2 pr-4">Yes</td>
								<td className="py-2">Session signing secret</td>
							</tr>
							<tr>
								<td className="py-2 pr-4 font-mono text-xs">OIDC_ISSUER</td>
								<td className="py-2 pr-4">Yes</td>
								<td className="py-2">OIDC provider issuer URL</td>
							</tr>
							<tr>
								<td className="py-2 pr-4 font-mono text-xs">OIDC_CLIENT_ID</td>
								<td className="py-2 pr-4">Yes</td>
								<td className="py-2">OIDC client ID</td>
							</tr>
							<tr>
								<td className="py-2 pr-4 font-mono text-xs">
									OIDC_CLIENT_SECRET
								</td>
								<td className="py-2 pr-4">Yes</td>
								<td className="py-2">OIDC client secret</td>
							</tr>
							<tr>
								<td className="py-2 pr-4 font-mono text-xs">
									OIDC_REDIRECT_URL
								</td>
								<td className="py-2 pr-4">Yes</td>
								<td className="py-2">
									Callback URL (e.g., https://keldris.example.com/auth/callback)
								</td>
							</tr>
							<tr>
								<td className="py-2 pr-4 font-mono text-xs">PORT</td>
								<td className="py-2 pr-4">No</td>
								<td className="py-2">Listen port (default: 8080)</td>
							</tr>
						</tbody>
					</table>
				</div>
			</div>
		),
	},
	organizations: {
		title: 'Organizations',
		content: (
			<div className="space-y-6">
				<p>
					Organizations are the top-level unit of multi-tenancy in Keldris. All
					resources (agents, repositories, schedules, backups) belong to an
					organization.
				</p>
				<h2 className="text-xl font-semibold">How It Works</h2>
				<ul className="list-disc list-inside space-y-2">
					<li>
						A default organization is automatically created when you first sign
						in
					</li>
					<li>
						All team members are added to the organization and share its
						resources
					</li>
					<li>
						Role-based access control (RBAC) lets you assign owner, admin,
						member, or read-only roles
					</li>
					<li>
						You can switch between organizations if you belong to more than one
					</li>
				</ul>
				<h2 className="text-xl font-semibold">Roles</h2>
				<div className="overflow-x-auto">
					<table className="min-w-full text-sm">
						<thead>
							<tr className="border-b dark:border-gray-700">
								<th className="text-left py-2 pr-4 font-medium">Role</th>
								<th className="text-left py-2 font-medium">Permissions</th>
							</tr>
						</thead>
						<tbody className="divide-y dark:divide-gray-700">
							<tr>
								<td className="py-2 pr-4 font-medium">Owner</td>
								<td className="py-2">
									Full access, manage members, delete organization
								</td>
							</tr>
							<tr>
								<td className="py-2 pr-4 font-medium">Admin</td>
								<td className="py-2">
									Manage resources, invite members, configure settings
								</td>
							</tr>
							<tr>
								<td className="py-2 pr-4 font-medium">Member</td>
								<td className="py-2">
									Create and manage own resources, view shared resources
								</td>
							</tr>
							<tr>
								<td className="py-2 pr-4 font-medium">Read-only</td>
								<td className="py-2">View all resources, no modifications</td>
							</tr>
						</tbody>
					</table>
				</div>
				<h2 className="text-xl font-semibold">Invitations</h2>
				<p>
					Organization owners and admins can invite new members by email.
					Invitations expire after 7 days. The invitee receives an email with a
					link to accept the invitation.
				</p>
			</div>
		),
	},
	repositories: {
		title: 'Repositories',
		content: (
			<div className="space-y-6">
				<p>
					Repositories define where your backups are stored. Keldris uses Restic
					as the backup engine, supporting multiple storage backends.
				</p>
				<h2 className="text-xl font-semibold">Supported Backends</h2>
				<div className="overflow-x-auto">
					<table className="min-w-full text-sm">
						<thead>
							<tr className="border-b dark:border-gray-700">
								<th className="text-left py-2 pr-4 font-medium">Backend</th>
								<th className="text-left py-2 pr-4 font-medium">Tier</th>
								<th className="text-left py-2 font-medium">Description</th>
							</tr>
						</thead>
						<tbody className="divide-y dark:divide-gray-700">
							<tr>
								<td className="py-2 pr-4 font-medium">Local</td>
								<td className="py-2 pr-4">Free</td>
								<td className="py-2">Local filesystem or mounted volume</td>
							</tr>
							<tr>
								<td className="py-2 pr-4 font-medium">REST</td>
								<td className="py-2 pr-4">Free</td>
								<td className="py-2">Restic REST server</td>
							</tr>
							<tr>
								<td className="py-2 pr-4 font-medium">Amazon S3</td>
								<td className="py-2 pr-4">Pro</td>
								<td className="py-2">
									S3 or S3-compatible storage (MinIO, Wasabi, etc.)
								</td>
							</tr>
							<tr>
								<td className="py-2 pr-4 font-medium">Backblaze B2</td>
								<td className="py-2 pr-4">Pro</td>
								<td className="py-2">Backblaze B2 Cloud Storage</td>
							</tr>
							<tr>
								<td className="py-2 pr-4 font-medium">SFTP</td>
								<td className="py-2 pr-4">Pro</td>
								<td className="py-2">Any server with SSH/SFTP access</td>
							</tr>
						</tbody>
					</table>
				</div>
				<h2 className="text-xl font-semibold">Creating a Repository</h2>
				<ol className="list-decimal list-inside space-y-2">
					<li>
						Navigate to{' '}
						<Link
							to="/repositories"
							className="text-indigo-600 hover:text-indigo-700"
						>
							Repositories
						</Link>
					</li>
					<li>Click "Add Repository"</li>
					<li>Choose a storage backend and fill in the connection details</li>
					<li>
						Keldris will initialize the Restic repository and encrypt the
						credentials
					</li>
				</ol>
				<h2 className="text-xl font-semibold">Encryption</h2>
				<p>
					All repository credentials are encrypted at rest using AES-256-GCM
					with the server's master encryption key. Restic repositories
					themselves are also encrypted with a separate repository password
					managed by Keldris.
				</p>
			</div>
		),
	},
	schedules: {
		title: 'Backup Schedules',
		content: (
			<div className="space-y-6">
				<p>
					Schedules define when and what to back up. Each schedule links an
					agent to a repository with a set of paths and a cron expression.
				</p>
				<h2 className="text-xl font-semibold">Creating a Schedule</h2>
				<ol className="list-decimal list-inside space-y-2">
					<li>
						Navigate to{' '}
						<Link
							to="/schedules"
							className="text-indigo-600 hover:text-indigo-700"
						>
							Schedules
						</Link>
					</li>
					<li>Click "Create Schedule"</li>
					<li>Select an agent, a repository, and the paths to back up</li>
					<li>Set a cron expression for the schedule (e.g., daily at 2 AM)</li>
					<li>Optionally configure retention policies and exclude patterns</li>
				</ol>
				<h2 className="text-xl font-semibold">Cron Expressions</h2>
				<div className="overflow-x-auto">
					<table className="min-w-full text-sm">
						<thead>
							<tr className="border-b dark:border-gray-700">
								<th className="text-left py-2 pr-4 font-medium">Expression</th>
								<th className="text-left py-2 font-medium">Meaning</th>
							</tr>
						</thead>
						<tbody className="divide-y dark:divide-gray-700">
							<tr>
								<td className="py-2 pr-4 font-mono text-xs">0 2 * * *</td>
								<td className="py-2">Every day at 2:00 AM</td>
							</tr>
							<tr>
								<td className="py-2 pr-4 font-mono text-xs">0 */6 * * *</td>
								<td className="py-2">Every 6 hours</td>
							</tr>
							<tr>
								<td className="py-2 pr-4 font-mono text-xs">0 3 * * 0</td>
								<td className="py-2">Every Sunday at 3:00 AM</td>
							</tr>
							<tr>
								<td className="py-2 pr-4 font-mono text-xs">*/30 * * * *</td>
								<td className="py-2">Every 30 minutes</td>
							</tr>
						</tbody>
					</table>
				</div>
				<h2 className="text-xl font-semibold">Retention</h2>
				<p>
					Configure how many snapshots to keep. Keldris supports keep-last,
					keep-daily, keep-weekly, and keep-monthly retention policies. Old
					snapshots are automatically pruned.
				</p>
			</div>
		),
	},
	'agent-installation': {
		title: 'Agent Installation',
		content: (
			<div className="space-y-6">
				<p>
					The Keldris agent runs on the machines you want to back up. It
					communicates with the Keldris server to receive backup schedules and
					report results.
				</p>
				<h2 className="text-xl font-semibold">Supported Platforms</h2>
				<ul className="list-disc list-inside space-y-1">
					<li>Linux (amd64, arm64)</li>
					<li>macOS (Intel, Apple Silicon)</li>
					<li>Windows (amd64)</li>
				</ul>
				<h2 className="text-xl font-semibold">Installation</h2>
				<h3 className="text-lg font-medium">Linux (Quick Install)</h3>
				<pre className="bg-gray-100 dark:bg-gray-700 p-4 rounded-lg text-sm overflow-x-auto">
					{
						'curl -fsSL https://github.com/MacJediWizard/keldris/releases/latest/download/install-linux.sh | sudo bash'
					}
				</pre>
				<h3 className="text-lg font-medium">Manual Download</h3>
				<p>
					Download the appropriate binary from the{' '}
					<a
						href="https://github.com/MacJediWizard/keldris/releases/latest"
						target="_blank"
						rel="noopener noreferrer"
						className="text-indigo-600 hover:text-indigo-700"
					>
						latest release
					</a>{' '}
					page. Place it in your PATH and configure the agent with your server
					URL and API key.
				</p>
				<h2 className="text-xl font-semibold">Configuration</h2>
				<p>
					After installation, configure the agent by setting the server URL and
					API key. You can generate an API key from the Agents page in the web
					UI.
				</p>
				<pre className="bg-gray-100 dark:bg-gray-700 p-4 rounded-lg text-sm overflow-x-auto">
					{`keldris-agent configure \\
  --server-url https://keldris.example.com \\
  --api-key kld_your_api_key_here`}
				</pre>
			</div>
		),
	},
	'notifications/email': {
		title: 'Email Notifications',
		content: (
			<div className="space-y-6">
				<p>
					Configure email notifications to stay informed about backup events,
					agent status changes, and storage alerts.
				</p>
				<h2 className="text-xl font-semibold">Setting Up Email</h2>
				<ol className="list-decimal list-inside space-y-2">
					<li>
						Navigate to{' '}
						<Link
							to="/notifications"
							className="text-indigo-600 hover:text-indigo-700"
						>
							Notifications
						</Link>
					</li>
					<li>Click "Add Channel" and select "Email"</li>
					<li>
						Enter your SMTP server details (host, port, username, password)
					</li>
					<li>Add recipient email addresses</li>
					<li>Send a test email to verify the configuration</li>
				</ol>
				<h2 className="text-xl font-semibold">Notification Events</h2>
				<ul className="list-disc list-inside space-y-1">
					<li>Backup completed successfully</li>
					<li>Backup failed or encountered errors</li>
					<li>Agent went offline</li>
					<li>Storage usage reached threshold</li>
					<li>Verification completed or failed</li>
				</ul>
				<h2 className="text-xl font-semibold">Preferences</h2>
				<p>
					You can customize which events trigger notifications and set quiet
					hours to avoid alerts during maintenance windows.
				</p>
			</div>
		),
	},
	'backup-verification': {
		title: 'Backup Verification',
		content: (
			<div className="space-y-6">
				<p>
					Verifying your backups ensures that your data can actually be
					restored. A backup that cannot be restored is useless.
				</p>
				<h2 className="text-xl font-semibold">Verification Methods</h2>
				<ul className="list-disc list-inside space-y-2">
					<li>
						<strong>Quick Check</strong> — Verifies repository metadata and
						snapshot integrity without downloading data
					</li>
					<li>
						<strong>Full Verify</strong> — Downloads and verifies all data
						blocks in the repository
					</li>
					<li>
						<strong>Test Restore</strong> — Restores files to a temporary
						location and validates checksums
					</li>
				</ul>
				<h2 className="text-xl font-semibold">Running a Verification</h2>
				<ol className="list-decimal list-inside space-y-2">
					<li>
						Go to the{' '}
						<Link
							to="/schedules"
							className="text-indigo-600 hover:text-indigo-700"
						>
							Schedules
						</Link>{' '}
						page
					</li>
					<li>Click "Run Now" to trigger a manual backup</li>
					<li>
						Check the{' '}
						<Link
							to="/backups"
							className="text-indigo-600 hover:text-indigo-700"
						>
							Backups
						</Link>{' '}
						page to see the result
					</li>
					<li>
						Once a backup succeeds, you can verify it from the snapshot details
					</li>
				</ol>
				<h2 className="text-xl font-semibold">Best Practices</h2>
				<ul className="list-disc list-inside space-y-1">
					<li>Run at least one manual backup before relying on schedules</li>
					<li>Periodically test restoring files to confirm data integrity</li>
					<li>
						Set up alerts so you are notified immediately if a backup fails
					</li>
				</ul>
			</div>
		),
	},
};

export function DocsPage() {
	const { '*': slug } = useParams();
	const page = slug ? docs[slug] : null;

	if (!page) {
		return (
			<div className="max-w-3xl mx-auto py-8">
				<h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-6">
					Documentation
				</h1>
				<div className="space-y-3">
					{Object.entries(docs).map(([key, doc]) => (
						<Link
							key={key}
							to={`/docs/${key}`}
							className="block p-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-indigo-300 dark:hover:border-indigo-600 transition-colors"
						>
							<span className="text-indigo-600 dark:text-indigo-400 font-medium">
								{doc.title}
							</span>
						</Link>
					))}
				</div>
			</div>
		);
	}

	return (
		<div className="max-w-3xl mx-auto py-8">
			<div className="mb-6">
				<Link
					to="/docs"
					className="text-sm text-indigo-600 dark:text-indigo-400 hover:text-indigo-700"
				>
					&larr; All docs
				</Link>
			</div>
			<h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-6">
				{page.title}
			</h1>
			<div className="prose dark:prose-invert max-w-none text-gray-700 dark:text-gray-300">
				{page.content}
			</div>
		</div>
	);
}

export default DocsPage;
