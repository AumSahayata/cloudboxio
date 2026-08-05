// ============================================================ CloudBoxIO UI
const API_URL = '/api';

/* ----------------------------------------------------------------- themes */
const THEME_KEY = 'cbio-theme';
const THEMES = ['paper', 'carbon'];

function applyTheme(theme) {
    if (!THEMES.includes(theme)) theme = 'paper';
    document.documentElement.setAttribute('data-theme', theme);
    try { localStorage.setItem(THEME_KEY, theme); } catch (_) { }
    document.querySelectorAll('.ts-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.themeValue === theme);
    });
}

function initTheme() {
    let saved = 'paper';
    try { saved = localStorage.getItem(THEME_KEY) || 'paper'; } catch (_) { }
    applyTheme(saved);
    document.querySelectorAll('.ts-btn').forEach(btn => {
        btn.addEventListener('click', () => applyTheme(btn.dataset.themeValue));
    });
}

/* --------------------------------------------------------------- helpers */
function formatFileSize(bytes) {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

function formatDate(timestamp) {
    if (!timestamp) return '';
    const d = new Date(timestamp);
    if (isNaN(d)) return '';
    return d.toLocaleString(undefined, {
        year: 'numeric', month: 'short', day: '2-digit',
        hour: '2-digit', minute: '2-digit'
    });
}

// Helper function to truncate text
function truncateText(text, maxLength = 35) {
    if (!text) return '';
    if (text.length <= maxLength) return text;
    return text.substring(0, maxLength) + '...';
}

function iconForFilename(name) {
    const ext = (name.split('.').pop() || '').toLowerCase();
    const map = {
        pdf: 'bi-filetype-pdf', doc: 'bi-filetype-doc', docx: 'bi-filetype-docx',
        xls: 'bi-filetype-xlsx', xlsx: 'bi-filetype-xlsx', csv: 'bi-filetype-csv',
        ppt: 'bi-filetype-pptx', pptx: 'bi-filetype-pptx', txt: 'bi-filetype-txt',
        md: 'bi-filetype-md', json: 'bi-filetype-json', xml: 'bi-filetype-xml',
        html: 'bi-filetype-html', css: 'bi-filetype-css', js: 'bi-filetype-js',
        py: 'bi-filetype-py', java: 'bi-filetype-java', sh: 'bi-filetype-sh',
        png: 'bi-filetype-png', jpg: 'bi-filetype-jpg', jpeg: 'bi-filetype-jpg',
        gif: 'bi-filetype-gif', svg: 'bi-filetype-svg', webp: 'bi-file-image',
        mp3: 'bi-filetype-mp3', wav: 'bi-filetype-wav', mp4: 'bi-filetype-mp4',
        mov: 'bi-filetype-mov', zip: 'bi-file-zip', tar: 'bi-file-zip',
        gz: 'bi-file-zip', rar: 'bi-file-zip', '7z': 'bi-file-zip'
    };
    return map[ext] || 'bi-file-earmark';
}

/* ------------------------------------------------------ loading overlay */
function showLoading(message = 'Loading…') {
    const overlay = document.getElementById('loadingOverlay');
    const msg = document.getElementById('loadingMsg');
    if (msg) msg.textContent = message;
    if (overlay) overlay.classList.add('show');
}
function hideLoading() {
    const overlay = document.getElementById('loadingOverlay');
    if (overlay) overlay.classList.remove('show');
}

/* --------------------------------------------------------- safe rendering */
// Build a file row entirely with DOM APIs + textContent. User-controlled
// values (filename, uploaded_by) are NEVER interpolated into HTML, which
// removes the stored-XSS vector the old innerHTML template had.
function createFileRow(file) {
    const fileId = file.file_id || file.id || file.fileId;
    const filename = file.filename || '';

    const row = document.createElement('div');
    row.className = 'file-row';

    const icon = document.createElement('i');
    icon.className = 'bi ' + iconForFilename(filename) + ' file-icon';
    row.appendChild(icon);

    const main = document.createElement('div');
    main.className = 'file-main';

    const nameEl = document.createElement('div');
    nameEl.className = 'file-name';
    nameEl.textContent = filename;
    nameEl.title = filename;
    main.appendChild(nameEl);

    const meta = document.createElement('div');
    meta.className = 'file-meta';

    const sizeEl = document.createElement('span');
    sizeEl.textContent = formatFileSize(file.size);
    meta.appendChild(sizeEl);

    const dateStr = formatDate(file.uploaded_at);
    if (dateStr) {
        meta.appendChild(makeSep());
        const dateEl = document.createElement('span');
        dateEl.textContent = dateStr;
        meta.appendChild(dateEl);
    }
    if (file.uploaded_by && file.uploaded_by !== 'Me') {
        meta.appendChild(makeSep());
        const byEl = document.createElement('span');
        byEl.className = 'uploader';
        byEl.textContent = '@' + file.uploaded_by;
        meta.appendChild(byEl);
    }
    main.appendChild(meta);
    row.appendChild(main);

    const actions = document.createElement('div');
    actions.className = 'file-actions';

    // Desktop: full inline buttons (hidden below 720px via CSS)
    const inline = document.createElement('div');
    inline.className = 'file-actions-inline';

    const dlBtn = document.createElement('button');
    dlBtn.className = 'icon-btn';
    dlBtn.title = 'Download';
    dlBtn.innerHTML = '<i class="bi bi-download"></i>';
    dlBtn.addEventListener('click', () => downloadFile(fileId, filename));
    inline.appendChild(dlBtn);

    // Share button lands here later — another .icon-btn, same pattern.

    const delBtn = document.createElement('button');
    delBtn.className = 'icon-btn danger';
    delBtn.title = 'Delete';
    delBtn.innerHTML = '<i class="bi bi-trash3"></i>';
    delBtn.addEventListener('click', () => deleteFile(fileId));
    inline.appendChild(delBtn);

    actions.appendChild(inline);

    // Mobile: 3-dot overflow menu (hidden at 720px and above via CSS)
    const mobileActions = document.createElement('div');
    mobileActions.className = 'file-actions-dropdown dropdown';

    const moreBtn = document.createElement('button');
    moreBtn.className = 'icon-btn dropdown-toggle';
    moreBtn.title = 'Actions';
    moreBtn.innerHTML = '<i class="bi bi-three-dots-vertical"></i>';
    mobileActions.appendChild(moreBtn);

    const menu = document.createElement('div');
    menu.className = 'dropdown-menu dropdown-menu-end';

    const dlItem = document.createElement('button');
    dlItem.className = 'dropdown-item';
    dlItem.innerHTML = '<i class="bi bi-download"></i>Download';
    dlItem.addEventListener('click', () => downloadFile(fileId, filename));
    menu.appendChild(dlItem);

    // Share action lands here later — same dropdown, one more dropdown-item.

    const delItem = document.createElement('button');
    delItem.className = 'dropdown-item text-danger';
    delItem.innerHTML = '<i class="bi bi-trash3"></i>Delete';
    delItem.addEventListener('click', () => deleteFile(fileId));
    menu.appendChild(delItem);

    mobileActions.appendChild(menu);
    actions.appendChild(mobileActions);
    row.appendChild(actions);
    return row;
}

function makeSep() {
    const s = document.createElement('span');
    s.className = 'sep';
    s.textContent = '·';
    return s;
}

function displayFiles(files, container, countEl) {
    if (!container) return;
    container.textContent = '';
    const list = Array.isArray(files) ? files : [];
    if (countEl) countEl.textContent = String(list.length);

    if (list.length === 0) {
        const empty = document.createElement('div');
        empty.className = 'empty-row';
        empty.textContent = 'No files yet';
        container.appendChild(empty);
        return;
    }
    list.forEach(f => container.appendChild(createFileRow(f)));
}

/* ------------------------------------------------------------ auth token */
function getAuthTokenOrRedirect() {
    const token = localStorage.getItem('token');
    if (!token) {
        showUnauthenticatedUI();
        showLoginModal();
        throw new Error('No authentication token found. Please log in.');
    }
    return token;
}

function authHeaders(extra) {
    return Object.assign({ 'Authorization': `Bearer ${getAuthTokenOrRedirect()}` }, extra || {});
}

/* --------------------------------------------------------------- files */
async function loadFiles() {
    showLoading('Loading files…');
    try {
        const myResp = await fetch(`${API_URL}/files`, { headers: authHeaders() });
        handleApiResponse(myResp);
        const myData = await myResp.json();
        if (!myResp.ok) throw new Error(myData.error || 'Failed to fetch files');
        displayFiles(myData, document.getElementById('myFilesList'),
            document.getElementById('myFilesCount'));

        const shResp = await fetch(`${API_URL}/files?public=true`, { headers: authHeaders() });
        handleApiResponse(shResp);
        const shData = await shResp.json();
        if (!shResp.ok) throw new Error(shData.error || 'Failed to fetch public files');
        displayFiles(shData, document.getElementById('publicFilesList'),
            document.getElementById('publicFilesCount'));
    } catch (error) {
        console.error('Error loading files:', error);
    } finally {
        hideLoading();
    }
}

async function deleteFile(fileId) {
    if (!fileId) return;
    if (!confirm('Delete this file?')) return;
    showLoading('Deleting…');
    try {
        const response = await fetch(`${API_URL}/file/${encodeURIComponent(fileId)}`, {
            method: 'DELETE', headers: authHeaders()
        });
        handleApiResponse(response);
        if (response.ok) {
            await loadFiles();
        } else {
            const data = await response.json().catch(() => ({}));
            alert(data.error || 'Delete failed');
        }
    } catch (error) {
        alert(`Error during delete: ${error.message || error}`);
    } finally {
        hideLoading();
    }
}

async function downloadFile(fileId, filename) {
    if (!fileId) { alert('Invalid file ID'); return; }
    showLoading('Preparing download…');
    try {
        const response = await fetch(`${API_URL}/file/${encodeURIComponent(fileId)}`, {
            headers: authHeaders()
        });
        handleApiResponse(response);
        if (!response.ok) {
            const data = await response.json().catch(() => ({}));
            throw new Error(data.error || 'Download failed');
        }
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.download = filename || 'download';
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        window.URL.revokeObjectURL(url);
    } catch (error) {
        alert(`Error during download: ${error.message || error}`);
    } finally {
        hideLoading();
    }
}

/* -------------------------------------------------------------- session */
function logout() {
    localStorage.removeItem('token');

    const myFiles = document.getElementById('myFilesList');
    const shFiles = document.getElementById('publicFilesList');
    if (myFiles) myFiles.textContent = '';
    if (shFiles) shFiles.textContent = '';

    document.querySelectorAll('form').forEach(form => {
        form.reset();
        form.querySelectorAll('input').forEach(i => i.classList.remove('is-invalid', 'is-valid'));
    });

    document.querySelectorAll('.modal.open').forEach(modal => closeModal(modal.id));

    showUnauthenticatedUI();
    showLoginModal();
}

function handleApiResponse(response) {
    if (response.status === 498 || response.status === 401) {
        localStorage.removeItem('token');
        showUnauthenticatedUI();
        showLoginModal();
        throw new Error('Session expired. Please log in again.');
    }
    return response;
}

/* --------------------------------------------------------- user details */
async function fetchUserDetails() {
    try {
        const response = await fetch(`${API_URL}/user-info`, { headers: authHeaders() });
        if (response.status === 404) {
            localStorage.removeItem('token');
            showUnauthenticatedUI();
            showLoginModal();
            return;
        }
        handleApiResponse(response);
        if (!response.ok) throw new Error('Failed to fetch user details');
        displayUserDetails(await response.json());
    } catch (error) {
        console.error('Error fetching user details:', error);
    }
}

function displayUserDetails(userData) {
    if (!userData) return;
    const navUsername = document.getElementById('navUsername');
    if (navUsername) {
        navUsername.textContent = userData.username;   // safe: no HTML
        if (userData.is_admin) {
            const badge = document.createElement('span');
            badge.className = 'badge-admin';
            badge.textContent = 'admin';
            navUsername.appendChild(badge);
        }
    }
    const adminEls = ['createUserNavItem', 'createUserDivider', 'showUsersPanelNavItem'];
    adminEls.forEach(id => {
        const el = document.getElementById(id);
        if (el) el.style.display = userData.is_admin ? 'block' : 'none';
    });
}

/* ------------------------------------------------------------- UI state */
function showAuthenticatedUI() {
    const nav = document.getElementById('userProfileNav');
    if (nav) nav.classList.remove('d-none');
    document.getElementById('authSection').style.display = 'none';
    document.getElementById('mainSection').style.display = 'block';
    fetchUserDetails();
    loadFiles();
}

function showUnauthenticatedUI() {
    const nav = document.getElementById('userProfileNav');
    if (nav) nav.classList.add('d-none');
    document.getElementById('authSection').style.display = 'block';
    document.getElementById('mainSection').style.display = 'none';
    const navUsername = document.getElementById('navUsername');
    if (navUsername) navUsername.textContent = 'User';
}

function showLoginModal() {
    if (document.getElementById('loginModal')) openModal('loginModal');
}

function checkAuth() {
    if (localStorage.getItem('token')) showAuthenticatedUI();
    else showUnauthenticatedUI();
}

/* ------------------------------------------------------- users (admin) */
function renderUsersPanel(users) {
    const usersList = document.getElementById('usersList');
    if (!usersList) return;
    usersList.textContent = '';

    (users || []).forEach(user => {
        const row = document.createElement('div');
        row.className = 'user-row';

        const left = document.createElement('span');
        left.className = 'uname';
        const strong = document.createElement('strong');
        strong.textContent = user.username;          // safe
        left.appendChild(strong);
        if (user.is_admin) {
            const badge = document.createElement('span');
            badge.className = 'badge-admin';
            badge.textContent = 'admin';
            left.appendChild(badge);
        }
        row.appendChild(left);

        const del = document.createElement('button');
        del.className = 'btn btn-danger';
        del.innerHTML = '<i class="bi bi-trash3 me-1"></i>Delete';
        del.addEventListener('click', () => deleteUser(user.id, del));
        row.appendChild(del);

        usersList.appendChild(row);
    });
}

async function fetchAllUsers() {
    try {
        showLoading('Loading users…');
        const response = await fetch(`${API_URL}/users`, { headers: authHeaders() });
        handleApiResponse(response);
        if (!response.ok) throw new Error('Failed to fetch users');
        renderUsersPanel(await response.json());
    } catch (error) {
        console.error('Error fetching users:', error);
        alert(error.message || 'Failed to load users');
    } finally {
        hideLoading();
    }
}

async function deleteUser(userId, btn) {
    if (!confirm('Delete this user?')) return;
    btn.disabled = true;
    try {
        showLoading('Deleting user…');
        const response = await fetch(`${API_URL}/users/${encodeURIComponent(userId)}`, {
            method: 'DELETE', headers: authHeaders()
        });
        handleApiResponse(response);
        if (!response.ok) {
            const data = await response.json().catch(() => ({}));
            throw new Error(data.error || 'Delete failed');
        }
        const row = btn.closest('.user-row');
        if (row) row.remove();
    } catch (error) {
        alert(`Error during delete: ${error.message || error}`);
    } finally {
        hideLoading();
        btn.disabled = false;
    }
}

/* ---------------------------------------------------- password reveal */
function togglePasswordVisibility(inputId, buttonId) {
    const input = document.getElementById(inputId);
    const button = document.getElementById(buttonId);
    if (input && button) {
        button.addEventListener('click', () => {
            const show = input.type === 'password';
            input.type = show ? 'text' : 'password';
            button.innerHTML = `<i class="bi bi-eye${show ? '-slash' : ''}"></i>`;
        });
    }
}

/* --------------------------------------------------------------- init */
document.addEventListener('DOMContentLoaded', () => {
    initTheme();

    togglePasswordVisibility('loginPassword', 'toggleLoginPassword');
    togglePasswordVisibility('currentPassword', 'toggleCurrentPassword');
    togglePasswordVisibility('newPassword', 'toggleNewPassword');
    togglePasswordVisibility('newUserPassword', 'toggleNewUserPassword');

    // Login
    const loginForm = document.getElementById('loginForm');
    if (loginForm) {
        loginForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const username = document.getElementById('loginUsername').value;
            const password = document.getElementById('loginPassword').value;
            showLoading('Signing in…');
            try {
                const response = await fetch(`${API_URL}/login`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ username, password })
                });
                const data = await response.json();
                if (response.ok) {
                    localStorage.setItem('token', data.token);
                    closeModal('loginModal');
                    loginForm.reset();
                    showAuthenticatedUI();
                    hideLoading();
                } else {
                    throw new Error(data.error || 'Login failed');
                }
            } catch (error) {
                console.error('Login error:', error);
                alert(error.message || 'Error during login');
                hideLoading();
            }
        });
    }

    // Upload
    const uploadForm = document.getElementById('uploadForm');
    const fileInput = document.getElementById('fileInput');
    if (uploadForm && fileInput) {
        uploadForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const files = fileInput.files;
            if (files.length === 0) return;
            const isPublic = document.getElementById('publicCheckbox')?.checked || false;
            const uploadUrl = `${API_URL}/upload${isPublic ? '?public=true' : ''}`;

            showLoading('Uploading…');
            let ok = true;
            for (const file of files) {
                const formData = new FormData();
                formData.append('files', file);
                try {
                    const response = await fetch(uploadUrl, {
                        method: 'POST', headers: authHeaders(), body: formData
                    });
                    if (!response.ok) {
                        const data = await response.json().catch(() => ({}));
                        alert(data.error || `Failed to upload ${file.name}`);
                        ok = false;
                    }
                } catch (error) {
                    console.error('Upload error:', error);
                    alert(`Error uploading ${file.name}`);
                    ok = false;
                }
            }
            fileInput.value = '';
            uploadForm.reset();
            await loadFiles();
            if (ok) {
                showLoading('Uploaded');
                setTimeout(hideLoading, 1200);
            } else {
                hideLoading();
            }
        });
    }

    // Reset password
    const resetPasswordForm = document.getElementById('resetPasswordForm');
    if (resetPasswordForm) {
        resetPasswordForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const currentPassword = document.getElementById('currentPassword').value;
            const newPassword = document.getElementById('newPassword').value;
            if (newPassword.length < 8) { alert('New password must be at least 8 characters'); return; }
            showLoading('Updating password…');
            try {
                const response = await fetch(`${API_URL}/reset-password`, {
                    method: 'PUT',
                    headers: authHeaders({ 'Content-Type': 'application/json' }),
                    body: JSON.stringify({ current_password: currentPassword, new_password: newPassword })
                });
                const data = await response.json();
                if (response.ok) {
                    closeModal('resetPasswordModal')
                    resetPasswordForm.reset();
                    showLoading('Password updated');
                    setTimeout(hideLoading, 1200);
                } else {
                    throw new Error(data.error || 'Password reset failed');
                }
            } catch (error) {
                console.error('Password reset error:', error);
                alert(error.message || 'Error during password reset');
                hideLoading();
            }
        });
    }

    // Create user
    const createUserForm = document.getElementById('createUserForm');
    if (createUserForm) {
        createUserForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const username = document.getElementById('newUsername').value;
            const password = document.getElementById('newUserPassword').value;
            const isAdmin = document.getElementById('isAdminCheckbox').checked;
            if (password.length < 8) { alert('Password must be at least 8 characters'); return; }
            showLoading('Creating user…');
            try {
                const response = await fetch(`${API_URL}/signup`, {
                    method: 'POST',
                    headers: authHeaders({ 'Content-Type': 'application/json' }),
                    body: JSON.stringify({ username, password, is_admin: isAdmin })
                });
                const data = await response.json();
                if (response.ok) {
                    createUserForm.reset();
                    showLoading('User created');
                    setTimeout(hideLoading, 1200);
                } else {
                    throw new Error(data.error || 'Failed to create user');
                }
            } catch (error) {
                console.error('Create user error:', error);
                alert(error.message || 'Error creating user');
                hideLoading();
            }
        });
    }

    // Search
    const fileSearchForm = document.getElementById('fileSearchForm');
    const fileSearchInput = document.getElementById('fileSearchInput');
    if (fileSearchForm && fileSearchInput) {
        fileSearchForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const keyword = fileSearchInput.value.trim();
            if (!keyword) { await loadFiles(); return; }
            showLoading('Searching…');
            try {
                const myResp = await fetch(`${API_URL}/files?keyword=${encodeURIComponent(keyword)}`, { headers: authHeaders() });
                handleApiResponse(myResp);
                const myData = await myResp.json();
                if (!myResp.ok) throw new Error(myData.error || 'Search failed');
                displayFiles(myData, document.getElementById('myFilesList'),
                    document.getElementById('myFilesCount'));

                const shResp = await fetch(`${API_URL}/files?public=true&keyword=${encodeURIComponent(keyword)}`, { headers: authHeaders() });
                handleApiResponse(shResp);
                const shData = await shResp.json();
                if (!shResp.ok) throw new Error(shData.error || 'Search failed');
                displayFiles(shData, document.getElementById('publicFilesList'),
                    document.getElementById('publicFilesCount'));
            } catch (error) {
                console.error('Error searching files:', error);
                alert(error.message || 'Error searching files');
            } finally {
                hideLoading();
            }
        });
        fileSearchForm.addEventListener('reset', () => { loadFiles(); });
    }

    // Users modal
    const usersModal = document.getElementById('usersModal');
    if (usersModal) {
        usersModal.addEventListener('modal:show', fetchAllUsers);
    }

    initModals();
    initDropdowns();
    checkAuth();
});

/* ------------------------------------------------- modal (no bootstrap) */
function openModal(id) {
    const modal = document.getElementById(id);
    if (!modal) return;
    let backdrop = document.querySelector('.modal-backdrop');
    if (!backdrop) {
        backdrop = document.createElement('div');
        backdrop.className = 'modal-backdrop';
        document.body.appendChild(backdrop);
        backdrop.addEventListener('click', () => {
            document.querySelectorAll('.modal.open').forEach(m => closeModal(m.id));
        });
    }
    requestAnimationFrame(() => backdrop.classList.add('show'));
    modal.classList.add('open');
    modal.dispatchEvent(new CustomEvent('modal:show'));
}

function closeModal(id) {
    const modal = document.getElementById(id);
    if (!modal) return;
    modal.classList.remove('open');
    if (!document.querySelector('.modal.open')) {
        const backdrop = document.querySelector('.modal-backdrop');
        if (backdrop) backdrop.remove();
    }
}

function initModals() {
    document.querySelectorAll('[data-modal-target]').forEach(btn => {
        btn.addEventListener('click', () => openModal(btn.dataset.modalTarget));
    });
    document.querySelectorAll('[data-modal-dismiss]').forEach(btn => {
        btn.addEventListener('click', () => closeModal(btn.closest('.modal').id));
    });
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') document.querySelectorAll('.modal.open').forEach(m => closeModal(m.id));
    });
}

/* ---------------------------------------------- dropdown (no bootstrap) */
function initDropdowns() {
    document.addEventListener('click', (e) => {
        const toggle = e.target.closest('.dropdown-toggle');
        if (toggle) {
            e.stopPropagation();
            const menu = toggle.nextElementSibling;
            const isOpen = menu.classList.contains('show');
            document.querySelectorAll('.dropdown-menu.show').forEach(m => m.classList.remove('show'));
            if (!isOpen) menu.classList.add('show');
            return;
        }
        document.querySelectorAll('.dropdown-menu.show').forEach(m => m.classList.remove('show'));
    });
}