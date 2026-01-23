import { useMutation } from '@tanstack/react-query';
import { supportApi } from '../lib/api';

export function useGenerateSupportBundle() {
	return useMutation({
		mutationFn: async () => {
			const blob = await supportApi.generateBundle();

			// Generate filename with timestamp
			const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
			const filename = `keldris-support-bundle-${timestamp}.zip`;

			// Trigger download
			const url = window.URL.createObjectURL(blob);
			const link = document.createElement('a');
			link.href = url;
			link.download = filename;
			document.body.appendChild(link);
			link.click();
			document.body.removeChild(link);
			window.URL.revokeObjectURL(url);

			return { filename };
		},
	});
}
