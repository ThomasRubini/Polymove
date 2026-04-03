const API_BASE_URL = process.env.POLYTECH_BASE_URL || 'http://localhost:8080';

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
	const sortBy = readTextParam(url, 'sort_by');
	const limit = readTextParam(url, 'limit') || '5';

	let student = null;
	let offers = [];
	let error = '';

	if (!studentId) {
		return {
			student,
			offers,
			error,
			filters: { studentId, sortBy, limit }
		};
	}

	try {
		const studentRes = await fetch(`${API_BASE_URL}/student/${studentId}`);
		if (!studentRes.ok) {
			const payload = await studentRes.json().catch(() => ({}));
			error = parseErrorMessage(payload, 'Failed to fetch student profile');
			return { student, offers, error, filters: { studentId, sortBy, limit } };
		}
		student = await studentRes.json();

		const recQuery = new URLSearchParams();
		recQuery.set('limit', limit);
		if (sortBy) recQuery.set('sort_by', sortBy);

		const recRes = await fetch(
			`${API_BASE_URL}/students/${studentId}/recommended-offers?${recQuery.toString()}`
		);

		if (!recRes.ok) {
			const payload = await recRes.json().catch(() => ({}));
			error = parseErrorMessage(payload, 'Failed to fetch recommendations');
			return { student, offers, error, filters: { studentId, sortBy, limit } };
		}

		offers = await recRes.json();
	} catch {
		error = 'Polytech API is unreachable.';
	}

	return {
		student,
		offers,
		error,
		filters: { studentId, sortBy, limit }
	};
}