const API_BASE_URL = process.env.POLYTECH_BASE_URL || 'http://localhost:8080';

function readTextParam(url, key) {
	return (url.searchParams.get(key) || '').trim();
}

function readLimit(url, defaultValue) {
	const raw = readTextParam(url, 'limit');
	if (!raw) return defaultValue;
	const parsed = Number.parseInt(raw, 10);
	if (Number.isNaN(parsed)) return defaultValue;
	return parsed;
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
	const city = readTextParam(url, 'city');
	const domain = readTextParam(url, 'domain');
	const studentId = readTextParam(url, 'student_id');
	const limit = readLimit(url, 10);

	const query = new URLSearchParams();
	if (city) query.set('city', city);
	if (domain) query.set('domain', domain);
	query.set('limit', String(limit));

	let offers = [];
	let error = '';

	try {
		const response = await fetch(`${API_BASE_URL}/offers?${query.toString()}`);
		if (!response.ok) {
			const payload = await response.json().catch(() => ({}));
			error = parseErrorMessage(payload, 'Failed to fetch offers');
		} else {
			offers = await response.json();
		}
	} catch {
		error = 'Polytech API is unreachable.';
	}

	return {
		offers,
		error,
		filters: {
			city,
			domain,
			limit: String(limit),
			studentId
		}
	};
}

export const actions = {
	apply: async ({ fetch, request }) => {
		const formData = await request.formData();
		const studentID = Number.parseInt(String(formData.get('student_id') || ''), 10);
		const offerID = Number.parseInt(String(formData.get('offer_id') || ''), 10);

		if (Number.isNaN(studentID) || Number.isNaN(offerID)) {
			return {
				applyResult: {
					ok: false,
					message: 'Student ID and Offer ID are required.'
				}
			};
		}

		try {
			const response = await fetch(`${API_BASE_URL}/internship`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ student_id: studentID, offer_id: offerID })
			});

			const payload = await response.json().catch(() => ({}));
			if (!response.ok) {
				return {
					applyResult: {
						ok: false,
						message: parseErrorMessage(payload, 'Application failed')
					}
				};
			}

			return {
				applyResult: {
					ok: true,
					message: `Application submitted (internship id: ${payload.id})`
				}
			};
		} catch {
			return {
				applyResult: {
					ok: false,
					message: 'Polytech API is unreachable.'
				}
			};
		}
	}
};