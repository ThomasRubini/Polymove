<script>
	let { data, form } = $props();
	let selectedChannel = $state(data.preferences?.channel || 'email');
</script>

<section class="panel dashboard-intro">
	<p class="eyebrow">Feature 3</p>
	<h2>La Poste Preferences</h2>
	<p>Configure how internship alerts are delivered for your student profile.</p>
</section>

<section class="panel">
	<form method="GET" class="grid-form">
		<label>
			Student ID
			<input name="student_id" type="number" min="1" value={data.filters.studentId} required />
		</label>
		<button type="submit">Load Preferences</button>
	</form>
</section>

{#if data.error}
	<section class="panel message error">{data.error}</section>
{/if}

{#if form?.prefsResult}
	<section class="panel message {form.prefsResult.ok ? 'success' : 'error'}">
		{form.prefsResult.message}
	</section>
{/if}

{#if data.student}
	<section class="panel">
		<h3>Student Profile</h3>
		<p><strong>ID:</strong> {data.student.id}</p>
		<p><strong>Name:</strong> {data.student.name}</p>
		<p><strong>Domain:</strong> {data.student.domain}</p>
		<p><a href={`/dashboard?student_id=${data.student.id}`}>Back to dashboard</a></p>
	</section>
{/if}

{#if data.student}
	<section class="panel">
		<h3>Notification Preferences</h3>
		{#if data.preferencesError}
			<p class="message error">{data.preferencesError}</p>
		{/if}

		<form method="POST" action="?/savePreferences" class="settings-form">
			<input type="hidden" name="student_id" value={data.student.id} />

			<label>
				Domain
				<input
					name="domain"
					value={data.preferences?.domain || data.student.domain}
					placeholder="Computer Science"
				/>
			</label>

			<label>
				Channel
				<select name="channel" bind:value={selectedChannel} required>
					<option value="email">
						Email
					</option>
					<option value="sms">SMS</option>
				</select>
			</label>

			<label>
				Contact
				<input
					name="contact"
					value={data.preferences?.contact || ''}
					placeholder={selectedChannel === 'sms' ? '+33 6 12 34 56 78' : 'student@example.com'}
				/>
			</label>

			<label>
				Enabled
				<select name="enabled">
					<option value="true" selected={(data.preferences?.enabled ?? true) === true}>Yes</option>
					<option value="false" selected={(data.preferences?.enabled ?? true) === false}>No</option>
				</select>
			</label>

			<button type="submit">Save Preferences</button>
		</form>

		<p class="helper-text">
			Channel is required by La Poste API. Use unsubscribe to completely stop notifications.
		</p>

		<form method="POST" action="?/unsubscribe" class="action-row">
			<input type="hidden" name="student_id" value={data.student.id} />
			<button type="submit" class="btn-small btn-ghost">Unsubscribe</button>
		</form>
	</section>
{/if}
