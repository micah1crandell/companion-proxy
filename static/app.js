let currentActionId = null;

document.addEventListener('DOMContentLoaded', () => {
    loadActions();
    loadLogs();
    startAutoRefresh();
    
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

function openActionModal(action = null) {
    autoRefresh = false;

    currentActionId = action?.id || null;
    document.getElementById('action-modal').style.display = 'flex';
    
    if (action) {
        document.getElementById('action-name').value = action.name;
        document.getElementById('action-url').value = action.url;
        document.getElementById('action-method').value = action.method;
        document.getElementById('action-body').value = action.body;
        renderHeaders(action.headers);
    } else {
        document.getElementById('action-form').reset();
        renderHeaders({});
    }
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
        name: document.getElementById('action-name').value,
        url: document.getElementById('action-url').value,
        method: document.getElementById('action-method').value,
        headers: headers,
        body: document.getElementById('action-body').value
    };

    try {
        if (currentActionId) {
            await fetch(`/actions/${currentActionId}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(action)
            });
        } else {
            await fetch('/actions', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(action)
            });
        }
        closeActionModal();
        loadActions();
    } catch (error) {
        console.error('Error saving action:', error);
        alert('Error saving action: ' + error.message);
    }
});

async function deleteAction(id) {
    if (confirm('Are you sure you want to delete this action?')) {
        try {
            await fetch(`/actions/${id}`, { method: 'DELETE' });
            loadActions();
        } catch (error) {
            console.error('Error deleting action:', error);
            alert('Failed to delete action');
        }
    }
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