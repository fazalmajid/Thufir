/** @type {import('tailwindcss').Config} */
export default {
	content: ['./src/**/*.{html,js,svelte,ts}'],
	darkMode: 'media',
	theme: {
		extend: {}
	},
	plugins: [
		require('@tailwindcss/typography')
	]
};
