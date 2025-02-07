let currentActionId = null;

document.addEventListener('DOMContentLoaded', () => {
    loadActions();
    loadLogs();
    startAutoRefresh();

    const savedTheme = localStorage.getItem('theme') || 'light';
    document.body.setAttribute('data-theme', savedTheme);
    
    document.querySelectorAll('.nav-item').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.nav-item').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));
            document.getElementById(btn.dataset.page + '-page').classList.add('active');
        });
    });
});

async function loadActions() {
    try {
        const response = await fetch('/actions');
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
        const actions = await response.json();
        renderActions(actions);
    } catch (error) {
        console.error('Error loading actions:', error);
        alert('Failed to load actions');
    }
}

function renderActions(actions) {
    const tbody = document.querySelector('#actions-table tbody');
    tbody.innerHTML = '';
    
    // Sort actions alphabetically by name
    actions.sort((a, b) => a.name.localeCompare(b.name));
    
    actions.forEach(action => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>${action.name}</td>
            <td class="url-cell">${action.url}</td>
            <td>${action.method}</td>
            <td>
                <button class="btn-secondary" onclick="editAction('${action.id}')">Edit</button>
                <button class="btn-secondary" onclick="deleteAction('${action.id}')">Delete</button>
                <button class="btn-primary" onclick="triggerAction('${action.name}')">Trigger</button>
            </td>
        `;
        tbody.appendChild(row);
    });
}

async function loadLogs() {
    try {
        const response = await fetch('/logs');
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
        const logs = await response.json();
        renderLogs(logs);
    } catch (error) {
        console.error('Error loading logs:', error);
        alert('Failed to load logs');
    }
}

function renderLogs(logs) {
    const tbody = document.querySelector('#logs-table tbody');
    tbody.innerHTML = '';
    
    logs.reverse().forEach(log => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>${new Date(log.timestamp).toLocaleString()}</td>
            <td>${getActionName(log.action_id)}</td>
            <td>${log.success ? '✅ Success' : '❌ Error'}</td>
            <td>${log.response}</td>
        `;
        tbody.appendChild(row);
    });
}

let existingActionNames = [];

async function fetchExistingActionNames() {
    try {
        const response = await fetch('/actions');
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
        const actions = await response.json();
        existingActionNames = actions.map(a => a.name.toLowerCase());
    } catch (error) {
        console.error('Error fetching action names:', error);
    }
}

async function openActionModal(action = null) {
    autoRefresh = false;

    // Fetch existing action names to validate uniqueness
    await fetchExistingActionNames();

    currentActionId = action?.id || null;
    document.getElementById('action-modal').style.display = 'flex';

    const nameInput = document.getElementById('action-name');
    const submitButton = document.querySelector('#action-form button[type="submit"]');
    const nameError = document.getElementById('name-error');

    if (action) {
        nameInput.value = action.name;
        document.getElementById('action-url').value = action.url;
        document.getElementById('action-method').value = action.method;
        document.getElementById('action-body').value = action.body;
        renderHeaders(action.headers);
    } else {
        document.getElementById('action-form').reset();
        renderHeaders({});
    }

    // Real-time validation for unique name
    nameInput.addEventListener('input', () => {
        const nameValue = nameInput.value.trim().toLowerCase();
        if (existingActionNames.includes(nameValue) && nameValue !== action?.name.toLowerCase()) {
            nameError.textContent = "Action name must be unique.";
            submitButton.disabled = true;
        } else {
            nameError.textContent = "";
            submitButton.disabled = false;
        }
    });
}

function closeActionModal() {
    autoRefresh = true;

    document.getElementById('action-modal').style.display = 'none';
}

function addHeaderField(key = '', value = '') {
    const container = document.getElementById('headers-container');
    const div = document.createElement('div');
    div.className = 'header-field';
    div.innerHTML = `
        <input type="text" placeholder="Header" value="${key}" class="header-key">
        <input type="text" placeholder="Value" value="${value}" class="header-value">
        <button class="btn-secondary" onclick="this.parentElement.remove()">Remove</button>
    `;
    container.appendChild(div);
}

function renderHeaders(headers) {
    const container = document.getElementById('headers-container');
    container.innerHTML = '';
    for (const [key, value] of Object.entries(headers)) {
        addHeaderField(key, value);
    }
}

document.getElementById('action-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const headers = {};
    document.querySelectorAll('.header-field').forEach(field => {
        const key = field.querySelector('.header-key').value;
        const value = field.querySelector('.header-value').value;
        if (key && value) headers[key] = value;
    });

    const action = {
        id: currentActionId, // Ensure the ID is included when updating
        name: document.getElementById('action-name').value.trim(),
        url: document.getElementById('action-url').value.trim(),
        method: document.getElementById('action-method').value.trim(),
        headers: headers,
        body: document.getElementById('action-body').value.trim()
    };

    try {
        if (currentActionId) {
            // Update existing action (PUT request)
            const response = await fetch(`/actions/${currentActionId}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(action)
            });

            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(`Update failed: ${errorText}`);
            }
        } else {
            // Create new action (POST request)
            const response = await fetch('/actions', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(action)
            });

            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(`Creation failed: ${errorText}`);
            }
        }

        closeActionModal();
        loadActions();
    } catch (error) {
        console.error('Error saving action:', error);
        alert('Error saving action: ' + error.message);
    }
});

async function deleteAction(id) {
    currentActionId = id;
    document.getElementById('confirm-modal').style.display = 'flex';
}

async function confirmDelete() {
    if (!currentActionId) return;

    try {
        const response = await fetch(`/actions/${currentActionId}`, {
            method: 'DELETE'
        });

        if (!response.ok) {
            const errorText = await response.text();
            console.error(`Error deleting action: ${errorText}`);
            throw new Error(`Delete failed: ${errorText}`);
        }

        console.log(`Action ${currentActionId} deleted successfully!`);
        closeConfirmModal();
        await loadActions();
    } catch (error) {
        console.error('Failed to delete action:', error);
        alert('Failed to delete action: ' + error.message);
    }
}

function closeConfirmModal() {
    document.getElementById('confirm-modal').style.display = 'none';
    currentActionId = null;
}

async function triggerAction(id) {
    try {
        await fetch(`/trigger/${encodeURIComponent(id)}`);
        loadLogs();
    } catch (error) {
        console.error('Error triggering action:', error);
        alert('Error triggering action: ' + error.message);
    }
}

async function editAction(id) {
    const response = await fetch(`/actions/${id}`);
    const action = await response.json();
    openActionModal(action);
}

function getActionName(id) {
    const action = Array.from(document.querySelectorAll('#actions-table tbody tr')).find(
        tr => tr.querySelector('td:last-child button').onclick.toString().includes(id)
    );
    return action ? action.querySelector('td:first-child').textContent : id;
}

// Auto-refresh every 2 seconds
let autoRefresh = true;

function startAutoRefresh() {
    setInterval(() => {
        if (autoRefresh) {
            loadActions();
            loadLogs();
        }
    }, 2000);
}

// Style
function toggleTheme() {
    const body = document.body;
    const currentTheme = body.getAttribute('data-theme');
    const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
    body.setAttribute('data-theme', newTheme);
    localStorage.setItem('theme', newTheme);
}