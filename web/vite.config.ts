import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
	plugins: [react()],
	server: {
		port: 5173,
		proxy: {
			'/api': 'http://localhost:8080',
			'/auth': 'http://localhost:8080',
		},
	},
	build: {
		rollupOptions: {
			output: {
				manualChunks: {
					'vendor-react': ['react', 'react-dom'],
					'vendor-router': ['react-router-dom'],
					'vendor-query': ['@tanstack/react-query'],
					'vendor-i18n': [
						'i18next',
						'react-i18next',
						'i18next-browser-languagedetector',
					],
				},
			},
		},
	},
	test: {
		globals: true,
		environment: 'jsdom',
		setupFiles: ['./src/test/setup.ts'],
		css: false,
		coverage: {
			provider: 'v8',
			reporter: ['text', 'lcov'],
			include: ['src/**/*.{ts,tsx}'],
			exclude: [
				'src/test/**',
				'src/main.tsx',
				'src/**/*.d.ts',
				'src/vite-env.d.ts',
			],
		},
	},
});
