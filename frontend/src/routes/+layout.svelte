<script>
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>Polymove Gateway UI</title>
	<link rel="preconnect" href="https://fonts.googleapis.com" />
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous" />
	<link
		href="https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@400;500;700&family=Newsreader:opsz,wght@6..72,500;6..72,700&display=swap"
		rel="stylesheet"
	/>
</svelte:head>

<div class="shell">
	<header class="topbar">
		<div class="brand">
			<p>Polymove</p>
			<h1>API Gateway Console</h1>
		</div>
		<nav>
			<a href="/">Offers Explorer</a>
			<a href="/dashboard">Student Dashboard</a>
		</nav>
	</header>

	<main>
		{@render children()}
	</main>
</div>
