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
	const sortBy = readTextParam(url, 'sort_by');
	const limit = readTextParam(url, 'limit') || '5';

	let student = null;
	let offers = [];
	let notifications = [];
	let preferences = null;
	let error = '';
	let notificationsError = '';
	let preferencesError = '';

	if (!studentId) {
		return {
			student,
			offers,
			notifications,
			preferences,
			error,
			notificationsError,
			preferencesError,
			filters: { studentId, sortBy, limit }
		};
	}

	try {
		const studentRes = await fetch(`${POLYTECH_BASE_URL}/student/${studentId}`);
		if (!studentRes.ok) {
			const payload = await studentRes.json().catch(() => ({}));
			error = parseErrorMessage(payload, 'Failed to fetch student profile');
			return {
				student,
				offers,
				notifications,
				preferences,
				error,
				notificationsError,
				preferencesError,
				filters: { studentId, sortBy, limit }
			};
		}
		student = await studentRes.json();

		const recQuery = new URLSearchParams();
		recQuery.set('limit', limit);
		if (sortBy) recQuery.set('sort_by', sortBy);

		const recRes = await fetch(
			`${POLYTECH_BASE_URL}/students/${studentId}/recommended-offers?${recQuery.toString()}`
		);

		if (!recRes.ok) {
			const payload = await recRes.json().catch(() => ({}));
			error = parseErrorMessage(payload, 'Failed to fetch recommendations');
			return {
				student,
				offers,
				notifications,
				preferences,
				error,
				notificationsError,
				preferencesError,
				filters: { studentId, sortBy, limit }
			};
		}

		offers = await recRes.json();

		const notificationsRes = await fetch(`${POLYTECH_BASE_URL}/students/${studentId}/notifications`);
		if (!notificationsRes.ok) {
			const payload = await notificationsRes.json().catch(() => ({}));
			notificationsError = parseErrorMessage(payload, 'Failed to fetch notifications');
		} else {
			notifications = await notificationsRes.json();
		}

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
		error = 'Polytech API is unreachable.';
	}

	return {
		student,
		offers,
		notifications,
		preferences,
		error,
		notificationsError,
		preferencesError,
		filters: { studentId, sortBy, limit }
	};
}

export const actions = {
	markRead: async ({ fetch, request }) => {
		const formData = await request.formData();
		const notificationID = String(formData.get('notification_id') || '').trim();

		if (!notificationID) {
			return {
				markReadResult: {
					ok: false,
					message: 'Notification ID is required.'
				}
			};
		}

		try {
			const response = await fetch(`${POLYTECH_BASE_URL}/notifications/${notificationID}/read`, {
				method: 'PUT'
			});

			const payload = await response.json().catch(() => ({}));
			if (!response.ok) {
				return {
					markReadResult: {
						ok: false,
						message: parseErrorMessage(payload, 'Failed to mark notification as read')
					}
				};
			}

			return {
				markReadResult: {
					ok: true,
					message: 'Notification marked as read.'
				}
			};
		} catch {
			return {
				markReadResult: {
					ok: false,
					message: 'Polytech API is unreachable.'
				}
			};
		}
	}
};