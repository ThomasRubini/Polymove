const POLYTECH_BASE_URL = process.env.POLYTECH_BASE_URL || 'http://localhost:8080';
const LAPOSTE_BASE_URL = process.env.LAPOSTE_BASE_URL || 'http://localhost:8083';

function readTextParam(url, key) {
	return (url.searchParams.get(key) || '').trim();
}

function parseErrorMessage(payload, fallback) {
	if (payload && typeof payload.message === 'string' && payload.message.length > 0) {
		return payload.message;
	}
	if (payload && typeof payload.error === 'string' && payload.error.length > 0) {
		return payload.error;
	}
	return fallback;
}

export async function load({ fetch, url }) {
	const studentId = readTextParam(url, 'student_id');

	let student = null;
	let preferences = null;
	let error = '';
	let preferencesError = '';

	if (!studentId) {
		return {
			student,
			preferences,
			error,
			preferencesError,
			filters: { studentId }
		};
	}

	try {
		const studentRes = await fetch(`${POLYTECH_BASE_URL}/student/${studentId}`);
		if (!studentRes.ok) {
			const payload = await studentRes.json().catch(() => ({}));
			error = parseErrorMessage(payload, 'Failed to fetch student profile');
			return {
				student,
				preferences,
				error,
				preferencesError,
				filters: { studentId }
			};
		}
		student = await studentRes.json();
	} catch {
		error = 'Polytech API is unreachable.';
	}

	if (error) {
		return {
			student,
			preferences,
			error,
			preferencesError,
			filters: { studentId }
		};
	}

	try {
		const preferencesRes = await fetch(`${LAPOSTE_BASE_URL}/subscribers/${studentId}`);
		if (preferencesRes.status === 404) {
			preferences = null;
		} else if (!preferencesRes.ok) {
			const payload = await preferencesRes.json().catch(() => ({}));
			preferencesError = parseErrorMessage(payload, 'Failed to fetch La Poste preferences');
		} else {
			preferences = await preferencesRes.json();
		}
	} catch {
		preferencesError = 'La Poste API is unreachable.';
	}

	return {
		student,
		preferences,
		error,
		preferencesError,
		filters: { studentId }
	};
}

export const actions = {
	savePreferences: async ({ fetch, request }) => {
		const formData = await request.formData();
		const studentId = String(formData.get('student_id') || '').trim();
		const domain = String(formData.get('domain') || '').trim();
		const channel = String(formData.get('channel') || '').trim();
		const contact = String(formData.get('contact') || '').trim();
		const enabled = String(formData.get('enabled') || '') === 'true';

		if (!studentId) {
			return {
				prefsResult: {
					ok: false,
					message: 'Student ID is required.'
				}
			};
		}

		if (!channel) {
			return {
				prefsResult: {
					ok: false,
					message: 'Channel is required.'
				}
			};
		}

		try {
			const response = await fetch(`${LAPOSTE_BASE_URL}/subscribers/${studentId}`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					domain,
					channel,
					contact,
					enabled
				})
			});

			const payload = await response.json().catch(() => ({}));
			if (!response.ok) {
				return {
					prefsResult: {
						ok: false,
						message: parseErrorMessage(payload, 'Failed to save preferences')
					}
				};
			}

			return {
				prefsResult: {
					ok: true,
					message: 'Preferences updated.'
				}
			};
		} catch {
			return {
				prefsResult: {
					ok: false,
					message: 'La Poste API is unreachable.'
				}
			};
		}
	},

	unsubscribe: async ({ fetch, request }) => {
		const formData = await request.formData();
		const studentId = String(formData.get('student_id') || '').trim();

		if (!studentId) {
			return {
				prefsResult: {
					ok: false,
					message: 'Student ID is required.'
				}
			};
		}

		try {
			const response = await fetch(`${LAPOSTE_BASE_URL}/subscribers/${studentId}`, {
				method: 'DELETE'
			});

			if (!response.ok && response.status !== 404) {
				const payload = await response.json().catch(() => ({}));
				return {
					prefsResult: {
						ok: false,
						message: parseErrorMessage(payload, 'Failed to unsubscribe')
					}
				};
			}

			return {
				prefsResult: {
					ok: true,
					message: 'Unsubscribed successfully.'
				}
			};
		} catch {
			return {
				prefsResult: {
					ok: false,
					message: 'La Poste API is unreachable.'
				}
			};
		}
	}
};
