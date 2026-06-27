import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
  preprocess: vitePreprocess(),
  kit: {
    // SPA mode: a single fallback HTML shell, all routing client-side. The Go
    // backend serves this shell for any non-/api path.
    adapter: adapter({
      fallback: 'index.html',
      pages: 'build',
      assets: 'build',
      precompress: false,
      strict: false
    })
  }
};

export default config;
