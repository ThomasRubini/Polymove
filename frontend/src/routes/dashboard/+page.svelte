<script>
	let { data } = $props();

	const sortOptions = [
		{ value: '', label: 'No sorting' },
		{ value: 'safety', label: 'Safety' },
		{ value: 'economy', label: 'Economy' },
		{ value: 'quality_of_life', label: 'Quality of Life' },
		{ value: 'culture', label: 'Culture' }
	];
</script>

<section class="panel dashboard-intro">
	<p class="eyebrow">Feature 2</p>
	<h2>Student Dashboard</h2>
	<p>Enter a student id to load profile and personalized recommended offers.</p>
</section>

<section class="panel">
	<form method="GET" class="grid-form">
		<label>
			Student ID
			<input name="student_id" type="number" min="1" value={data.filters.studentId} required />
		</label>
		<label>
			Sort by
			<select name="sort_by" value={data.filters.sortBy}>
				{#each sortOptions as option}
					<option value={option.value}>{option.label}</option>
				{/each}
			</select>
		</label>
		<label>
			Limit
			<input name="limit" type="number" min="1" value={data.filters.limit} />
		</label>
		<button type="submit">Load Dashboard</button>
	</form>
</section>

{#if data.error}
	<section class="panel message error">{data.error}</section>
{/if}

{#if data.student}
	<section class="panel">
		<h3>Student Profile</h3>
		<p><strong>ID:</strong> {data.student.id}</p>
		<p><strong>Name:</strong> {data.student.name}</p>
		<p><strong>Domain:</strong> {data.student.domain}</p>
	</section>
{/if}

{#if data.student}
	<section class="panel">
		<h3>Recommended Offers</h3>
		{#if data.offers.length === 0}
			<p>No recommendations available.</p>
		{:else}
			<div class="dashboard-grid">
				{#each data.offers as offer}
					<article class="offer-card">
						<div class="offer-head">
							<div>
								<p class="eyebrow">{offer.domain}</p>
								<h3>{offer.title}</h3>
								<p>{offer.city}</p>
							</div>
							<a href={offer.link} target="_blank" rel="noreferrer">Source</a>
						</div>
						<p><strong>Salary:</strong> {offer.salary}</p>
					</article>
				{/each}
			</div>
		{/if}
	</section>
{/if}