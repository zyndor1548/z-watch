class ZWatch {
	constructor() {
		this.username = localStorage.getItem('zwatch_username');
		this.token = localStorage.getItem('zwatch_token');
		this.apiBase = window.location.origin;

		this.init();
	}

	init() {
		this.setupEventListeners();
		this.checkAuthStatus();
	}

	setupEventListeners() {
		// Auth tabs
		document.querySelectorAll('.tab').forEach(tab => {
			tab.addEventListener('click', () => this.switchTab(tab.dataset.tab));
		});

		// Auth forms
		document.getElementById('login-form').addEventListener('submit', (e) => {
			e.preventDefault();
			this.handleLogin();
		});

		document.getElementById('register-form').addEventListener('submit', (e) => {
			e.preventDefault();
			this.handleRegister();
		});

		// Logout
		document.getElementById('logout-btn').addEventListener('click', () => {
			this.handleLogout();
		});

		// Quick check
		document.getElementById('quick-check-form').addEventListener('submit', (e) => {
			e.preventDefault();
			this.handleQuickCheck();
		});

		// Add site
		document.getElementById('add-site-form').addEventListener('submit', (e) => {
			e.preventDefault();
			this.handleAddSite();
		});

		// Refresh logs
		document.getElementById('refresh-logs').addEventListener('click', () => {
			this.loadLogs();
		});
	}

	switchTab(tab) {
		// Update tab buttons
		document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
		document.querySelector(`[data-tab="${tab}"]`).classList.add('active');

		// Update forms
		if (tab === 'login') {
			document.getElementById('login-form').classList.remove('hidden');
			document.getElementById('register-form').classList.add('hidden');
		} else {
			document.getElementById('login-form').classList.add('hidden');
			document.getElementById('register-form').classList.remove('hidden');
		}
	}

	checkAuthStatus() {
		if (this.username && this.token) {
			this.showDashboard();
		} else {
			this.showAuth();
		}
	}

	showAuth() {
		document.getElementById('auth-section').classList.remove('hidden');
		document.getElementById('dashboard-section').classList.add('hidden');
		document.getElementById('user-info').classList.add('hidden');
	}

	showDashboard() {
		document.getElementById('auth-section').classList.add('hidden');
		document.getElementById('dashboard-section').classList.remove('hidden');
		document.getElementById('user-info').classList.remove('hidden');
		document.getElementById('username-display').textContent = this.username;
		this.loadLogs();
	}

	async handleLogin() {
		const username = document.getElementById('login-username').value.trim();
		const password = document.getElementById('login-password').value;
		const messageEl = document.getElementById('login-message');

		if (!username || !password) {
			this.showMessage(messageEl, 'Please enter username and password', 'error');
			return;
		}

		try {
			const response = await fetch(`${this.apiBase}/login`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ username, password })
			});

			if (!response.ok) {
				this.showMessage(messageEl, 'Login failed. Please check your credentials.', 'error');
				return;
			}

			const data = await response.json();

			this.username = data.username;
			this.token = data.token;

			localStorage.setItem('zwatch_username', this.username);
			localStorage.setItem('zwatch_token', this.token);

			this.showMessage(messageEl, 'Login successful!', 'success');

			setTimeout(() => {
				this.showDashboard();
			}, 500);

		} catch (error) {
			console.error('Login error:', error);
			this.showMessage(messageEl, 'Network error. Please try again.', 'error');
		}
	}

	async handleRegister() {
		const username = document.getElementById('register-username').value.trim();
		const password = document.getElementById('register-password').value;
		const messageEl = document.getElementById('register-message');

		if (!username || !password) {
			this.showMessage(messageEl, 'Please enter username and password', 'error');
			return;
		}

		if (password.length < 6) {
			this.showMessage(messageEl, 'Password must be at least 6 characters', 'error');
			return;
		}

		try {
			const response = await fetch(`${this.apiBase}/register`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ username, password })
			});

			if (!response.ok) {
				this.showMessage(messageEl, 'Registration failed. Username may already exist.', 'error');
				return;
			}

			const data = await response.json();

			this.username = data.username;
			this.token = data.token;

			localStorage.setItem('zwatch_username', this.username);
			localStorage.setItem('zwatch_token', this.token);

			this.showMessage(messageEl, 'Registration successful!', 'success');

			setTimeout(() => {
				this.showDashboard();
			}, 500);

		} catch (error) {
			console.error('Register error:', error);
			this.showMessage(messageEl, 'Network error. Please try again.', 'error');
		}
	}

	handleLogout() {
		if (confirm('Are you sure you want to logout?')) {
			this.username = null;
			this.token = null;
			localStorage.removeItem('zwatch_username');
			localStorage.removeItem('zwatch_token');
			this.showAuth();
		}
	}

	async handleQuickCheck() {
		const url = document.getElementById('check-url').value.trim();
		const resultEl = document.getElementById('check-result');

		if (!url) {
			this.showMessage(resultEl, 'Please enter a URL', 'error');
			return;
		}

		resultEl.innerHTML = '<p class="loading">Checking...</p>';

		try {
			const response = await fetch(`${this.apiBase}/check`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ url })
			});

			if (!response.ok) {
				resultEl.innerHTML = '<p class="message error">Check failed</p>';
				return;
			}

			const data = await response.json();
			this.displayCheckResult(data);

		} catch (error) {
			console.error('Check error:', error);
			resultEl.innerHTML = '<p class="message error">Network error</p>';
		}
	}

	displayCheckResult(data) {
		const resultEl = document.getElementById('check-result');

		if (!data || !data.data) {
			resultEl.innerHTML = '<p class="message error">No data received</p>';
			return;
		}

		const result = data.data;
		const html = `
            <h4>Results for: ${data.url}</h4>
            <div class="result-grid">
                <div class="result-item">
                    <h4>Status</h4>
                    <p class="${result.status === 'up' ? 'status-up' : 'status-down'}">
                        ${result.status || 'Unknown'}
                    </p>
                </div>
                <div class="result-item">
                    <h4>Latency</h4>
                    <p>${result.latency || 'N/A'}</p>
                </div>
            </div>
        `;

		resultEl.innerHTML = html;
	}

	async handleAddSite() {
		const url = document.getElementById('site-url').value.trim();
		const messageEl = document.getElementById('add-site-message');

		if (!url) {
			this.showMessage(messageEl, 'Please enter a URL', 'error');
			return;
		}

		if (!this.token) {
			this.showMessage(messageEl, 'Please login first', 'error');
			return;
		}

		try {
			const response = await fetch(`${this.apiBase}/addsite`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					'Authorization': `Bearer ${this.token}`
				},
				body: JSON.stringify({ url })
			});

			if (!response.ok) {
				this.showMessage(messageEl, 'Failed to add site', 'error');
				return;
			}

			const data = await response.json();

			if (data.success) {
				this.showMessage(messageEl, 'Site added successfully!', 'success');
				document.getElementById('site-url').value = '';

				setTimeout(() => {
					this.loadLogs();
				}, 1000);
			} else {
				this.showMessage(messageEl, 'Failed to add site', 'error');
			}

		} catch (error) {
			console.error('Add site error:', error);
			this.showMessage(messageEl, 'Network error', 'error');
		}
	}

	async loadLogs() {
		const logsEl = document.getElementById('logs-container');

		if (!this.token) {
			logsEl.innerHTML = '<p class="empty-state">Please login to view logs</p>';
			return;
		}

		logsEl.innerHTML = '<p class="loading">Loading logs...</p>';

		try {
			const response = await fetch(`${this.apiBase}/getlog`, {
				method: 'GET',
				headers: {
					'Authorization': `Bearer ${this.token}`
				}
			});

			if (!response.ok) {
				logsEl.innerHTML = '<p class="empty-state">Failed to load logs</p>';
				return;
			}

			const logs = await response.json();
			this.displayLogs(logs);

		} catch (error) {
			console.error('Load logs error:', error);
			logsEl.innerHTML = '<p class="empty-state">Network error</p>';
		}
	}

	displayLogs(urlCheckLogs) {
		const logsEl = document.getElementById('logs-container');

		if (!urlCheckLogs || urlCheckLogs.length === 0) {
			logsEl.innerHTML = '<p class="empty-state">No logs yet. Add a site to start monitoring.</p>';
			return;
		}

		const html = urlCheckLogs.map(urlLog => {
			// Determine cube color based on status and latency
			const getCubeClass = (log) => {
				if (log.status === 'down') return 'cube-down';

				// Parse latency to determine if it's slow
				const latencyMs = this.parseLatency(log.latency);
				if (latencyMs > 1000) return 'cube-slow'; // Yellow if > 1 second

				return 'cube-up'; // Green for normal
			};

			return `
				<div class="url-group">
					<div class="url-header">
						<h4 class="url-title">${urlLog.url || 'Unknown URL'}</h4>
						<span class="check-count">${urlLog.logs ? urlLog.logs.length : 0} checks</span>
					</div>
					<div class="timeline-container">
						${urlLog.logs && urlLog.logs.length > 0 ? urlLog.logs.map(log => `
							<div class="timeline-cube ${getCubeClass(log)}" 
								data-status="${log.status || 'unknown'}" 
								data-latency="${log.latency || 'N/A'}">
							</div>
						`).join('') : '<p class="empty-state">No logs for this URL yet.</p>'}
					</div>
				</div>
			`;
		}).join('');

		logsEl.innerHTML = html;
		this.attachTooltipListeners();
	}



	parseLatency(latencyStr) {
		// Parse latency string like "100ms" or "1.5s" to milliseconds
		if (!latencyStr) return 0;

		const match = latencyStr.match(/([\d.]+)(ms|s|µs)/);
		if (!match) return 0;

		const value = parseFloat(match[1]);
		const unit = match[2];

		if (unit === 's') return value * 1000;
		if (unit === 'µs') return value / 1000;
		return value; // ms
	}

	attachTooltipListeners() {
		const cubes = document.querySelectorAll('.timeline-cube');
		const tooltip = document.getElementById('cube-tooltip');

		if (!tooltip) return;

		cubes.forEach(cube => {
			cube.addEventListener('mouseenter', (e) => {
				const status = cube.dataset.status;
				const latency = cube.dataset.latency;

				// Update tooltip content
				const statusClass = status === 'up' ? 'status-up' : 'status-down';
				tooltip.innerHTML = `
					<div class="tooltip-row">
						<span class="tooltip-label">Status:</span>
						<span class="tooltip-value ${statusClass}">${status}</span>
					</div>
					<div class="tooltip-row">
						<span class="tooltip-label">Latency:</span>
						<span class="tooltip-value">${latency}</span>
					</div>
				`;

				// Position tooltip
				const rect = cube.getBoundingClientRect();
				tooltip.style.left = `${rect.left + rect.width / 2}px`;
				tooltip.style.top = `${rect.top - 10}px`;
				tooltip.classList.add('visible');
			});

			cube.addEventListener('mouseleave', () => {
				tooltip.classList.remove('visible');
			});
		});
	}

	showMessage(element, message, type) {
		element.innerHTML = `<p class="message ${type}">${message}</p>`;

		setTimeout(() => {
			element.innerHTML = '';
		}, 5000);
	}

	formatDate(dateString) {
		if (!dateString) return 'Unknown';

		const date = new Date(dateString);
		return new Intl.DateTimeFormat('en-US', {
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		}).format(date);
	}
}

// Initialize app
document.addEventListener('DOMContentLoaded', () => {
	new ZWatch();
});
