// Help content for form fields and concepts throughout the application
// Format: Key is a unique identifier, value contains content, optional title, and optional docs URL

export interface HelpContent {
	content: string;
	title?: string;
	docsUrl?: string;
}

// Schedule form fields
export const scheduleHelp: Record<string, HelpContent> = {
	name: {
		content:
			'A descriptive name for this backup schedule. Use something meaningful like **Daily Home Backup** or **Weekly Database Backup**.',
		title: 'Schedule Name',
	},
	agent: {
		content:
			'The agent (machine) that will execute this backup. Each agent can have multiple schedules configured.',
		title: 'Agent Selection',
		docsUrl: '/docs/agents',
	},
	repository: {
		content:
			'The storage destination for backups. You can select multiple repositories for redundancy - backups will be sent to all selected repositories.',
		title: 'Repository',
		docsUrl: '/docs/repositories',
	},
	cronExpression: {
		content:
			'Defines when backups run using cron syntax: `minute hour day month weekday`. Common patterns: `0 2 * * *` (daily 2 AM), `0 */6 * * *` (every 6 hours), `0 3 * * 0` (weekly Sunday).',
		title: 'Cron Expression',
		docsUrl: '/docs/scheduling',
	},
	paths: {
		content:
			'Directories and files to back up. Enter one path per line. Use absolute paths like `/home/user` or `/var/www`.',
		title: 'Backup Paths',
	},
	excludePatterns: {
		content:
			'Patterns to exclude from backup. Supports glob patterns like `*.log`, `**/node_modules/**`, or `.git/`.',
		title: 'Exclude Patterns',
		docsUrl: '/docs/patterns',
	},
	policyTemplate: {
		content:
			'Apply a pre-configured policy template to quickly set retention, paths, and bandwidth settings. You can customize values after applying.',
		title: 'Policy Template',
		docsUrl: '/docs/policies',
	},
};

// Retention policy fields
export const retentionHelp: Record<string, HelpContent> = {
	keepLast: {
		content:
			'Always keep at least this many of the most recent snapshots, regardless of age.',
		title: 'Keep Last',
	},
	keepDaily: {
		content:
			'Keep one snapshot per day for this many days. Useful for recovering from recent changes.',
		title: 'Keep Daily',
	},
	keepWeekly: {
		content:
			'Keep one snapshot per week for this many weeks. Balances storage with recovery options.',
		title: 'Keep Weekly',
	},
	keepMonthly: {
		content:
			'Keep one snapshot per month for this many months. Good for longer-term recovery needs.',
		title: 'Keep Monthly',
	},
	keepYearly: {
		content:
			'Keep one snapshot per year for this many years. Required for compliance or archival purposes.',
		title: 'Keep Yearly',
	},
	overview: {
		content:
			'Retention policies automatically remove old backups to save storage while maintaining recovery points. Each rule keeps the most recent snapshot within its time period.',
		title: 'Retention Policy',
		docsUrl: '/docs/retention',
	},
};

// Advanced settings fields
export const advancedSettingsHelp: Record<string, HelpContent> = {
	bandwidthLimit: {
		content:
			'Limit upload speed in KB/s to prevent backups from saturating network bandwidth. Leave empty for unlimited speed.',
		title: 'Bandwidth Limit',
	},
	backupWindow: {
		content:
			'Restrict backups to run only within specific hours. Useful for running backups during off-peak times.',
		title: 'Backup Window',
	},
	excludedHours: {
		content:
			'Hours when backups should **not** run. Click hours to toggle exclusion. Red hours are excluded.',
		title: 'Excluded Hours',
	},
	compressionLevel: {
		content:
			'**Off:** Fastest, no compression (good for pre-compressed data). **Auto:** Balanced for most data. **Max:** Smallest files, slowest (best for text/logs).',
		title: 'Compression Level',
	},
	maxFileSize: {
		content:
			'Exclude files larger than this size (in MB). Useful for skipping large media files or database dumps.',
		title: 'Max File Size',
	},
	onMountUnavailable: {
		content:
			'**Fail:** Backup fails if network path unavailable (safer). **Skip:** Backup proceeds without the path (may miss data).',
		title: 'Network Mount Behavior',
	},
};

// Repository form fields
export const repositoryHelp: Record<string, HelpContent> = {
	name: {
		content:
			'A descriptive name for this repository. Use something like **AWS S3 Primary** or **Local NAS Backup**.',
		title: 'Repository Name',
	},
	type: {
		content:
			'The storage backend type. Each type has different configuration requirements and cost characteristics.',
		title: 'Repository Type',
		docsUrl: '/docs/repositories#types',
	},
	localPath: {
		content:
			'Absolute path on the server where backups will be stored. Ensure the path exists and has appropriate permissions.',
		title: 'Local Path',
	},
	s3Bucket: {
		content:
			'The S3 bucket name. Must already exist in your AWS account. Can include a prefix for organization.',
		title: 'S3 Bucket',
	},
	s3Region: {
		content:
			'AWS region where the bucket is located, e.g., `us-east-1`, `eu-west-1`.',
		title: 'S3 Region',
	},
	s3Endpoint: {
		content:
			'Custom S3-compatible endpoint URL. Leave empty for AWS S3. Required for MinIO, Backblaze S3, etc.',
		title: 'S3 Endpoint',
	},
	escrowEnabled: {
		content:
			'Store an encrypted copy of the repository password. Enables password recovery if needed but reduces security.',
		title: 'Password Escrow',
		docsUrl: '/docs/security#escrow',
	},
};

// Agent fields
export const agentHelp: Record<string, HelpContent> = {
	hostname: {
		content:
			'The hostname or identifier for this agent. Should match the machine name for easy identification.',
		title: 'Hostname',
	},
	status: {
		content:
			'**Active:** Agent is online and communicating. **Offline:** Agent has not checked in recently. **Pending:** Agent registered but not yet activated.',
		title: 'Agent Status',
	},
	healthStatus: {
		content:
			'**Healthy:** All metrics normal. **Warning:** Some metrics elevated. **Critical:** Immediate attention needed.',
		title: 'Health Status',
		docsUrl: '/docs/agents#health',
	},
	debugMode: {
		content:
			'Enable verbose logging for troubleshooting. Automatically disables after the configured duration to prevent log bloat.',
		title: 'Debug Mode',
	},
};

// Dashboard concepts
export const dashboardHelp: Record<string, HelpContent> = {
	activeAgents: {
		content:
			'Number of agents currently online and communicating with the server. Offline agents may indicate connectivity or system issues.',
		title: 'Active Agents',
		docsUrl: '/docs/agents',
	},
	repositories: {
		content:
			'Total backup storage destinations configured. Each repository can store backups from multiple agents and schedules.',
		title: 'Repositories',
		docsUrl: '/docs/repositories',
	},
	scheduledJobs: {
		content:
			'Number of backup schedules currently enabled. Paused schedules are not counted.',
		title: 'Scheduled Jobs',
	},
	totalBackups: {
		content:
			'Total number of backup jobs that have been executed, including completed, failed, and running backups.',
		title: 'Total Backups',
	},
	recentBackups: {
		content:
			'The most recent backup jobs across all agents. Shows status, snapshot ID, size, and timing.',
		title: 'Recent Backups',
	},
	systemStatus: {
		content:
			'Current health of core system components. All indicators should be green for normal operation.',
		title: 'System Status',
	},
	storageEfficiency: {
		content:
			'Deduplication reduces storage by eliminating duplicate data blocks. Higher ratios mean better efficiency.',
		title: 'Storage Efficiency',
		docsUrl: '/docs/deduplication',
	},
	dedupRatio: {
		content:
			'The ratio of original data size to actual stored size. A 3.0x ratio means data is compressed to 1/3 its original size.',
		title: 'Deduplication Ratio',
	},
	spaceSaved: {
		content:
			'Total storage space saved through deduplication and compression across all repositories.',
		title: 'Space Saved',
	},
};

// Status badge explanations
export const statusHelp: Record<string, HelpContent> = {
	// Backup statuses
	backupCompleted: {
		content:
			'Backup finished successfully. All files were processed and a snapshot was created.',
		title: 'Completed',
	},
	backupRunning: {
		content:
			'Backup is currently in progress. Files are being processed and uploaded.',
		title: 'Running',
	},
	backupFailed: {
		content:
			'Backup encountered an error and could not complete. Check the error message for details.',
		title: 'Failed',
	},
	backupCanceled: {
		content: 'Backup was manually stopped before completion. No snapshot was created.',
		title: 'Canceled',
	},

	// Agent statuses
	agentActive: {
		content:
			'Agent is online, healthy, and ready to execute backups. Last check-in was recent.',
		title: 'Active',
	},
	agentOffline: {
		content:
			'Agent has not communicated with the server recently. Check network connectivity and agent service.',
		title: 'Offline',
	},
	agentPending: {
		content:
			'Agent has been registered but is waiting for initial activation or configuration.',
		title: 'Pending',
	},
	agentDisabled: {
		content:
			'Agent has been manually disabled. No backups will run until re-enabled.',
		title: 'Disabled',
	},

	// Health statuses
	healthHealthy: {
		content:
			'All system metrics are within normal ranges. CPU, memory, and disk usage are acceptable.',
		title: 'Healthy',
	},
	healthWarning: {
		content:
			'One or more metrics are elevated but not critical. Monitor closely and consider action.',
		title: 'Warning',
	},
	healthCritical: {
		content:
			'One or more metrics are at critical levels. Immediate attention required to prevent failures.',
		title: 'Critical',
	},

	// Schedule statuses
	scheduleActive: {
		content:
			'Schedule is enabled and will run backups according to its cron expression.',
		title: 'Active',
	},
	schedulePaused: {
		content:
			'Schedule is disabled and will not run automatic backups until re-enabled.',
		title: 'Paused',
	},

	// Alert statuses
	alertActive: {
		content:
			'Alert condition is currently present and requires attention.',
		title: 'Active Alert',
	},
	alertAcknowledged: {
		content:
			'Alert has been acknowledged by a user but the condition still exists.',
		title: 'Acknowledged',
	},
	alertResolved: {
		content:
			'Alert condition has been resolved either automatically or manually.',
		title: 'Resolved',
	},
};

// Alert and notification help
export const alertHelp: Record<string, HelpContent> = {
	agentOffline: {
		content:
			'Triggered when an agent has not communicated with the server for the configured threshold period.',
		title: 'Agent Offline Alert',
	},
	backupSla: {
		content:
			'Triggered when a schedule has not had a successful backup within the configured SLA period.',
		title: 'Backup SLA Alert',
	},
	storageUsage: {
		content:
			'Triggered when repository storage usage exceeds the configured percentage threshold.',
		title: 'Storage Usage Alert',
	},
	healthWarning: {
		content:
			'Triggered when agent health metrics enter warning thresholds (elevated CPU, memory, or disk).',
		title: 'Health Warning Alert',
	},
	healthCritical: {
		content:
			'Triggered when agent health metrics reach critical levels requiring immediate attention.',
		title: 'Health Critical Alert',
	},
};

// Classification and compliance
export const classificationHelp: Record<string, HelpContent> = {
	level: {
		content:
			'**Public:** No sensitivity. **Internal:** Business use only. **Confidential:** Sensitive data. **Restricted:** Highest protection required.',
		title: 'Classification Level',
		docsUrl: '/docs/compliance#classification',
	},
	dataTypes: {
		content:
			'**PII:** Personal identifiable info. **PHI:** Health info (HIPAA). **PCI:** Payment card data. **Proprietary:** Trade secrets.',
		title: 'Data Types',
	},
};

// Immutability and legal hold
export const immutabilityHelp: Record<string, HelpContent> = {
	lock: {
		content:
			'Prevents snapshot deletion until the lock expires. Use for compliance requirements or ransomware protection.',
		title: 'Immutability Lock',
		docsUrl: '/docs/immutability',
	},
	legalHold: {
		content:
			'Indefinite hold on a snapshot for legal/compliance purposes. Cannot be removed without proper authorization.',
		title: 'Legal Hold',
	},
	remainingDays: {
		content:
			'Days remaining until the immutability lock expires and the snapshot can be deleted.',
		title: 'Remaining Days',
	},
};

// Cost estimation
export const costHelp: Record<string, HelpContent> = {
	monthlyEstimate: {
		content:
			'Projected monthly storage cost based on current data size and configured pricing rates.',
		title: 'Monthly Estimate',
	},
	forecast: {
		content:
			'Projected costs based on historical growth rates. Helps plan budget and capacity.',
		title: 'Cost Forecast',
		docsUrl: '/docs/costs',
	},
	costAlert: {
		content:
			'Notifications when storage costs exceed configured thresholds or forecasts predict overages.',
		title: 'Cost Alerts',
	},
};

// Verification
export const verificationHelp: Record<string, HelpContent> = {
	check: {
		content:
			'Validates repository integrity by checking data structures and metadata without reading actual data.',
		title: 'Repository Check',
	},
	checkReadData: {
		content:
			'Full verification that reads and validates actual backup data. More thorough but slower.',
		title: 'Check with Read',
	},
	testRestore: {
		content:
			'Performs a test restoration to verify backup data can actually be recovered.',
		title: 'Test Restore',
		docsUrl: '/docs/verification',
	},
};

// DR Runbooks
export const drHelp: Record<string, HelpContent> = {
	rto: {
		content:
			'**Recovery Time Objective:** Maximum acceptable time to restore operations after a disaster.',
		title: 'RTO',
	},
	rpo: {
		content:
			'**Recovery Point Objective:** Maximum acceptable data loss measured in time (e.g., 4 hours of data).',
		title: 'RPO',
	},
	runbook: {
		content:
			'Step-by-step procedures for recovering from specific disaster scenarios.',
		title: 'DR Runbook',
		docsUrl: '/docs/disaster-recovery',
	},
};

// Geo-replication
export const geoReplicationHelp: Record<string, HelpContent> = {
	overview: {
		content:
			'Automatically copies backups between geographic regions for disaster recovery and compliance.',
		title: 'Geo-Replication',
		docsUrl: '/docs/geo-replication',
	},
	lag: {
		content:
			'Time or number of snapshots behind between source and target regions. High lag may indicate sync issues.',
		title: 'Replication Lag',
	},
	status: {
		content:
			'**Synced:** Fully up-to-date. **Syncing:** Transfer in progress. **Pending:** Waiting to sync. **Failed:** Sync error occurred.',
		title: 'Replication Status',
	},
};
