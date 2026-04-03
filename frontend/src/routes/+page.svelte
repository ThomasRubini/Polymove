<script>
	let { data, form } = $props();

	const scoreRows = [
		{ key: 'safety', label: 'Safety' },
		{ key: 'economy', label: 'Economy' },
		{ key: 'qol', label: 'Quality of Life' },
		{ key: 'culture', label: 'Culture' }
	];
	const scoreMax = 2000;

	function scorePercent(value) {
		if (typeof value !== 'number') return 0;
		return Math.max(0, Math.min(100, Math.round((value / scoreMax) * 100)));
	}

	function scoreValue(scores, key) {
		if (!scores) return 'N/A';
		const value = scores[key];
		return typeof value === 'number' ? value.toFixed(1) : 'N/A';
	}
</script>

<section class="panel intro">
	<p class="eyebrow">Feature 1</p>
	<h2>Offers Explorer</h2>
	<p>
		Browse aggregated internship offers from Erasmumu and MI8 city intelligence through the Polytech API
		Gateway.
	</p>
</section>

<section class="panel filters">
	<form method="GET" class="grid-form">
		<label>
			City
			<input name="city" value={data.filters.city} placeholder="Lyon" />
		</label>
		<label>
			Domain
			<input name="domain" value={data.filters.domain} placeholder="Computer Science" />
		</label>
		<label>
			Limit
			<input name="limit" value={data.filters.limit} type="number" min="1" placeholder="10" />
		</label>
		<label>
			Student ID (for Apply)
			<input name="student_id" value={data.filters.studentId} type="number" min="1" placeholder="1" />
		</label>
		<button type="submit">Apply Filters</button>
	</form>
</section>

{#if data.error}
	<section class="panel message error">{data.error}</section>
{/if}

{#if form?.applyResult}
	<section class="panel message {form.applyResult.ok ? 'success' : 'error'}">
		{form.applyResult.message}
	</section>
{/if}

<section class="offers">
	{#if data.offers.length === 0}
		<div class="panel">No offers found for the selected filters.</div>
	{:else}
		{#each data.offers as offer}
			<article class="offer-card">
				<div class="offer-head">
					<div>
						<p class="eyebrow">{offer.domain}</p>
						<h3>{offer.title}</h3>
						<p>{offer.city}</p>
					</div>
				</div>

				<div class="meta-grid">
					<div><strong>Salary:</strong> {offer.salary}</div>
					<div><strong>Start:</strong> {offer.startDate}</div>
					<div><strong>End:</strong> {offer.endDate}</div>
					<div><strong>Open:</strong> {offer.available ? 'Yes' : 'No'}</div>
				</div>

				<div class="scores">
					<h4>City Scores</h4>
					{#each scoreRows as row}
						<div class="score-row">
							<div class="score-label">
								<span>{row.label}</span>
								<span>{scoreValue(offer.scores, row.key)}</span>
							</div>
							<div class="track">
								<div class="fill" style={`width: ${scorePercent(offer.scores?.[row.key])}%`}></div>
							</div>
						</div>
					{/each}
				</div>

				<div class="news">
					<h4>Latest News</h4>
					{#if offer.latest_news?.length}
						<ul>
							{#each offer.latest_news as item}
								<li>{item.title}</li>
							{/each}
						</ul>
					{:else}
						<p>No city news available.</p>
					{/if}
				</div>

				<form method="POST" action="?/apply" class="apply-form">
					<input type="hidden" name="offer_id" value={offer.id} />
					<input
						type="number"
						name="student_id"
						min="1"
						required
						value={data.filters.studentId}
						placeholder="Student ID"
					/>
					<button type="submit">Apply</button>
				</form>
			</article>
		{/each}
	{/if}
</section>
