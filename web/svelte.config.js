import vercel from '@sveltejs/adapter-vercel';
import node from '@sveltejs/adapter-node';

// Deploy target is selected at build time:
//   - On Vercel (the VERCEL=1 env var is injected automatically) → adapter-vercel,
//     which emits .vercel/output. This is the hybrid deploy: the frontend runs on
//     Vercel (vulnarena.com) and talks to the Go API at api.vulnarena.com via the
//     absolute PUBLIC_API_URL.
//   - Anywhere else → adapter-node (standalone build/index.js) so the whole stack
//     can still be self-hosted behind nginx if needed.
const adapter = process.env.VERCEL
	? vercel()
	: node({ out: 'build', precompress: true });

/** @type {import('@sveltejs/kit').Config} */
const config = {
	compilerOptions: {
		// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
		runes: ({ filename }) => (filename.split(/[/\\]/).includes('node_modules') ? undefined : true)
	},
	kit: {
		adapter
	}
};

export default config;
