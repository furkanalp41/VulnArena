import adapter from '@sveltejs/adapter-node';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	compilerOptions: {
		// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
		runes: ({ filename }) => (filename.split(/[/\\]/).includes('node_modules') ? undefined : true)
	},
	kit: {
		// adapter-node produces a standalone Node server (build/index.js) for the
		// self-hosted Docker/nginx deploy. See https://svelte.dev/docs/kit/adapter-node
		adapter: adapter({
			out: 'build',
			precompress: true
		})
	}
};

export default config;
