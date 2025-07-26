// API Configuration
const API_URL = '/api';

// Helper function to truncate text
function truncateText(text, maxLength = 35) {
    if (!text) return '';
    if (text.length <= maxLength) return text;
    return text.substring(0, maxLength) + '...';
}

// Helper function to format file size
function formatFileSize(bytes) {
    if (!bytes) return '0 Bytes';
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// Helper function to format date
function formatDate(timestamp) {
    if (!timestamp) return '';
    return new Date(timestamp).toLocaleString();
}

// Display files in the list
function displayFiles(files, container, sectionTitle) {
    if (!container) return;
    
    container.innerHTML = '';

    if (!Array.isArray(files) || files.length === 0) {
        const noFilesElement = document.createElement('div');
        noFilesElement.className = 'list-group-item text-center text-muted';
        noFilesElement.textContent = 'No files found';
        container.appendChild(noFilesElement);
        return;
    }

    files.forEach(file => {
        const fileItem = createFileListItem(file);
        container.appendChild(fileItem);
    });
}

// Create file list item
function createFileListItem(file) {
    const item = document.createElement('div');
    item.className = 'list-group-item';
    
    const isPublic = file.is_public;
    const fileId = file.id || file.file_id || file.fileId || file.filename;
    const filename = file.filename;
    
    if (!fileId) {
        console.error('File ID not found:', file);
        return item;
    }
    
    item.innerHTML = `
        <div class="d-flex">
            <div class="file-name">
                <strong data-bs-toggle="tooltip" data-bs-placement="top" title="${filename}">
                    <span class="desktop-filename">${filename}</span>
                    <span class="mobile-filename">${truncateText(filename)}</span>
                </strong>
                <small class="text-muted d-block">
                    ${formatFileSize(file.size)} • ${formatDate(file.uploaded_at)}
                    ${isPublic ? ' • Public' : ''}
                    ${file.uploaded_by ? ` • Uploaded by: ${file.uploaded_by}` : ''}
                </small>
            </div>
            <div class="btn-group">
                <button class="btn btn-sm btn-primary" onclick="downloadFile('${fileId}', '${filename.replace(/'/g, "\\'")}')">
                    <i class="bi bi-download"></i> Download
                </button>
                <button class="btn btn-sm btn-danger" onclick="deleteFile('${fileId}')">
                    <i class="bi bi-trash"></i> Delete
                </button>
            </div>
            <div class="mobile-dropdown">
                <button class="btn btn-outline-secondary btn-sm w-100 dropdown-toggle" type="button" data-bs-toggle="dropdown">
                    Actions
                </button>
                <ul class="dropdown-menu">
                    <li><a class="dropdown-item" href="#" onclick="downloadFile('${fileId}', '${filename.replace(/'/g, "\\'")}')">
                        <i class="bi bi-download"></i> Download
                    </a></li>
                    <li><a class="dropdown-item text-danger" href="#" onclick="deleteFile('${fileId}')">
                        <i class="bi bi-trash"></i> Delete
                    </a></li>
                </ul>
            </div>
        </div>
    `;
    
    // Initialize tooltips for this item
    const tooltipTriggerList = item.querySelectorAll('[data-bs-toggle="tooltip"]');
    [...tooltipTriggerList].map(tooltipTriggerEl => new bootstrap.Tooltip(tooltipTriggerEl));
    
    return item;
}

// Loading overlay functions
function showLoading(message = 'Loading...') {
    let overlay = document.getElementById('loadingOverlay');
    if (!overlay) {
        overlay = document.createElement('div');
        overlay.id = 'loadingOverlay';
        overlay.className = 'loading-overlay';
        overlay.innerHTML = `
            <div class="loading-content">
                <div class="spinner-border text-primary" role="status">
                    <span class="visually-hidden">Loading...</span>
                </div>
                <div class="loading-message mt-2">${message}</div>
            </div>
        `;
        document.body.appendChild(overlay);
    } else {
        const messageElement = overlay.querySelector('.loading-message');
        if (messageElement) {
            messageElement.textContent = message;
        }
    }
    overlay.style.display = 'flex';
}

function hideLoading() {
    const overlay = document.getElementById('loadingOverlay');
    if (overlay) {
        overlay.style.display = 'none';
    }
}

// Add loading overlay styles
const style = document.createElement('style');
style.textContent = `
    .loading-overlay {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background-color: rgba(0, 0, 0, 0.5);
        display: flex;
        justify-content: center;
        align-items: center;
        z-index: 9999;
    }
    .loading-content {
        background-color: white;
        padding: 2rem;
        border-radius: 8px;
        text-align: center;
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
    }
    .loading-message {
        color: #666;
        margin-top: 1rem;
    }

    /* Filename display styles */
    .desktop-filename {
        display: inline;
    }
    .mobile-filename {
        display: none;
    }

    @media (max-width: 768px) {
        .desktop-filename {
            display: none;
        }
        .mobile-filename {
            display: inline;
        }
    }
`;
document.head.appendChild(style);

// Helper to get token or redirect to login if missing
function getAuthTokenOrRedirect() {
    const token = localStorage.getItem('token');
    if (!token) {
        showUnauthenticatedUI();
        showLoginModal();
        throw new Error('No authentication token found. Please log in.');
    }
    return token;
}

// Global loadFiles function
async function loadFiles() {
    showLoading('Loading files...');
    try {
        // Load my files
        const myFilesResponse = await fetch(`${API_URL}/files`, {
            headers: {
                'Authorization': `Bearer ${getAuthTokenOrRedirect()}`,
            },
        });
        const myFilesData = await myFilesResponse.json();
        if (!myFilesResponse.ok) {
            throw new Error(myFilesData.error || 'Failed to fetch my files');
        }
        const myFilesList = document.getElementById('myFilesList');
        if (myFilesList) {
            displayFiles(myFilesData, myFilesList, 'My Files');
        }

        // Load shared files
        const sharedFilesResponse = await fetch(`${API_URL}/files?shared=true`, {
            headers: {
                'Authorization': `Bearer ${getAuthTokenOrRedirect()}`,
            },
        });
        const sharedFilesData = await sharedFilesResponse.json();
        if (!sharedFilesResponse.ok) {
            throw new Error(sharedFilesData.error || 'Failed to fetch shared files');
        }
        const sharedFilesList = document.getElementById('sharedFilesList');
        if (sharedFilesList) {
            displayFiles(sharedFilesData, sharedFilesList, 'Public Files');
        }
    } catch (error) {
        console.error('Error loading files:', error);
        alert(error.message || 'Error loading files');
    } finally {
        hideLoading();
    }
}

// Global logout function
function logout() {
    // Clear token
    localStorage.removeItem('token');
    
    // Clear the file lists
    const myFilesList = document.getElementById('myFilesList');
    const sharedFilesList = document.getElementById('sharedFilesList');
    if (myFilesList) myFilesList.innerHTML = '';
    if (sharedFilesList) sharedFilesList.innerHTML = '';
    
    // Reset all forms
    const forms = document.querySelectorAll('form');
    forms.forEach(form => {
        form.reset();
        // Clear any validation states or error messages
        const inputs = form.querySelectorAll('input');
        inputs.forEach(input => {
            input.classList.remove('is-invalid', 'is-valid');
            // Clear any custom validation messages
            const feedback = input.nextElementSibling;
            if (feedback && feedback.classList.contains('invalid-feedback')) {
                feedback.textContent = '';
            }
        });
    });
    
    // Close any open modals
    const modals = document.querySelectorAll('.modal');
    modals.forEach(modal => {
        const modalInstance = bootstrap.Modal.getInstance(modal);
        if (modalInstance) {
            modalInstance.hide();
        }
    });
    
    // Clear any error messages or alerts
    const alerts = document.querySelectorAll('.alert');
    alerts.forEach(alert => alert.remove());
    
    // Update UI using the new function
    showUnauthenticatedUI();
    showLoginModal();
}

// Delete file
async function deleteFile(fileId) {
    if (!confirm('Are you sure you want to delete this file?')) return;

    showLoading('Deleting file...');
    try {
        const response = await fetch(`${API_URL}/file/${fileId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${getAuthTokenOrRedirect()}`,
            },
        });
        handleApiResponse(response);
        if (response.ok) {
            await loadFiles(); // Reload the file list after successful deletion
        } else {
            const data = await response.json();
            alert(data.error || 'Delete failed');
        }
    } catch (error) {
        alert(`Error during delete: ${error.message || error}`);
    } finally {
        hideLoading();
    }
}

// Download file
async function downloadFile(fileId, filename) {
    if (!fileId) {
        alert('Invalid file ID');
        return;
    }

    showLoading('Preparing download...');
    try {
        const response = await fetch(`${API_URL}/file/${fileId}`, {
            headers: {
                'Authorization': `Bearer ${getAuthTokenOrRedirect()}`,
            },
        });
        handleApiResponse(response);
        if (!response.ok) {
            const data = await response.json();
            throw new Error(data.error || 'Download failed');
        }

        // Get the blob from the response
        const blob = await response.blob();
        
        // Create a temporary URL for the blob
        const url = window.URL.createObjectURL(blob);
        
        // Create a temporary link element
        const link = document.createElement('a');
        link.href = url;
        link.download = filename; // Use the passed filename
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        
        // Clean up the URL
        window.URL.revokeObjectURL(url);
    } catch (error) {
        alert(`Error during download: ${error.message || error}`);
    } finally {
        hideLoading();
    }
}

// Initialize password visibility toggles
function togglePasswordVisibility(inputId, buttonId) {
    const input = document.getElementById(inputId);
    const button = document.getElementById(buttonId);
    if (input && button) {
        button.addEventListener('click', () => {
            const type = input.type === 'password' ? 'text' : 'password';
            input.type = type;
            button.innerHTML = `<i class="bi bi-eye${type === 'password' ? '' : '-slash'}"></i>`;
        });
    }
}

// Check authentication status
function checkAuth() {
    const token = localStorage.getItem('token');
    if (token) {
        showAuthenticatedUI();
    } else {
        showUnauthenticatedUI();
    }
}

// Fetch and display user details
async function fetchUserDetails() {
    try {
        const response = await fetch(`${API_URL}/user-info`, {
            headers: {
                'Authorization': `Bearer ${getAuthTokenOrRedirect()}`,
            },
        });
        handleApiResponse(response);
        if (response.status === 404) {
            // User not found, treat as unauthenticated
            localStorage.removeItem('token');
            showUnauthenticatedUI();
            showLoginModal();
            throw new Error('User not found. Please log in again.');
        }
        if (!response.ok) {
            throw new Error('Failed to fetch user details');
        }
        const userData = await response.json();
        displayUserDetails(userData);
    } catch (error) {
        console.error('Error fetching user details:', error);
        alert('Failed to load user details');
    }
}

// Display user details in the UI
function displayUserDetails(userData) {
    if (!userData) return;
    // Update navigation username
    const navUsername = document.getElementById('navUsername');
    if (navUsername) {
        // Add admin badge if user is admin
        navUsername.textContent = userData.username;
        if (userData.is_admin) {
            navUsername.innerHTML += ' <span class="badge bg-warning">Admin</span>';
            // Show admin-only options
            document.getElementById('createUserNavItem').style.display = 'block';
            document.getElementById('createUserDivider').style.display = 'block';
            // Show users panel nav item
            const showUsersPanelNavItem = document.getElementById('showUsersPanelNavItem');
            if (showUsersPanelNavItem) showUsersPanelNavItem.style.display = 'block';
        } else {
            // Hide admin-only options
            document.getElementById('createUserNavItem').style.display = 'none';
            document.getElementById('createUserDivider').style.display = 'none';
            // Hide users panel nav item
            const showUsersPanelNavItem = document.getElementById('showUsersPanelNavItem');
            if (showUsersPanelNavItem) showUsersPanelNavItem.style.display = 'none';
        }
    }
}

// Show authenticated UI
function showAuthenticatedUI() {
    // Show user profile
    const userProfileNav = document.getElementById('userProfileNav');
    if (userProfileNav) userProfileNav.classList.remove('d-none');

    // Show main content
    document.getElementById('authSection').style.display = 'none';
    document.getElementById('mainSection').style.display = 'block';
    
    // Fetch and display user details
    fetchUserDetails();
    
    // Load files
    loadFiles();
}

// Show unauthenticated UI
function showUnauthenticatedUI() {
    // Hide user profile
    const userProfileNav = document.getElementById('userProfileNav');
    if (userProfileNav) userProfileNav.classList.add('d-none');

    // Show auth content
    document.getElementById('authSection').style.display = 'block';
    document.getElementById('mainSection').style.display = 'none';
}

// Helper to show the login modal
function showLoginModal() {
    const loginModalElement = document.getElementById('loginModal');
    if (loginModalElement) {
        const loginModal = new bootstrap.Modal(loginModalElement);
        loginModal.show();
    }
}

// Update handleApiResponse to show login modal on 498
function handleApiResponse(response) {
    if (response.status === 498) {
        // Token missing or expired
        localStorage.removeItem('token');
        showUnauthenticatedUI();
        showLoginModal();
        throw new Error('Session expired. Please log in again.');
    }
    return response;
}

// Helper to render the users panel (admin only)
function renderUsersPanel(users) {
    const usersList = document.getElementById('usersList');
    if (!usersList) return;
    usersList.innerHTML = '';
    users.forEach(user => {
        const userItem = document.createElement('div');
        userItem.className = 'list-group-item d-flex justify-content-between align-items-center';
        userItem.innerHTML = `
            <span><strong>${user.username}</strong> ${user.is_admin ? '<span class="badge bg-warning">Admin</span>' : ''}</span>
            <button class="btn btn-sm btn-danger delete-user-btn" data-user-id="${user.id}"><i class="bi bi-trash"></i> Delete</button>
        `;
        usersList.appendChild(userItem);
    });
    // Attach event listeners for delete buttons
    usersList.querySelectorAll('.delete-user-btn').forEach(btn => {
        btn.addEventListener('click', function() {
            const userId = this.getAttribute('data-user-id');
            deleteUser(userId, this);
        });
    });
}

// Fetch all users (admin only)
async function fetchAllUsers() {
    try {
        showLoading('Loading users...');
        const response = await fetch(`${API_URL}/users`, {
            headers: {
                'Authorization': `Bearer ${getAuthTokenOrRedirect()}`,
            },
        });
        handleApiResponse(response);
        if (!response.ok) {
            throw new Error('Failed to fetch users');
        }
        const users = await response.json();
        renderUsersPanel(users);
    } catch (error) {
        console.error('Error fetching users:', error);
        alert(error.message || 'Failed to load users');
    } finally {
        hideLoading();
    }
}

// Update deleteUser to use id in the API call
async function deleteUser(userId, btn) {
    if (!confirm('Are you sure you want to delete this user?')) return;
    btn.disabled = true;
    try {
        showLoading('Deleting user...');
        const response = await fetch(`${API_URL}/users/${encodeURIComponent(userId)}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${getAuthTokenOrRedirect()}`,
            },
        });
        handleApiResponse(response);
        if (!response.ok) {
            const data = await response.json();
            throw new Error(data.error || 'Delete failed');
        }
        // Remove user from the list
        btn.closest('.list-group-item').remove();
    } catch (error) {
        alert(`Error during delete: ${error.message || error}`);
    } finally {
        hideLoading();
        btn.disabled = false;
    }
}

// Fetch users when the usersModal is shown
const usersModal = document.getElementById('usersModal');
if (usersModal) {
    usersModal.addEventListener('show.bs.modal', () => {
        fetchAllUsers();
    });
}

// Initialize the application
document.addEventListener('DOMContentLoaded', () => {
    // Initialize password visibility toggles
    togglePasswordVisibility('loginPassword', 'toggleLoginPassword');
    togglePasswordVisibility('currentPassword', 'toggleCurrentPassword');
    togglePasswordVisibility('newPassword', 'toggleNewPassword');
    togglePasswordVisibility('newUserPassword', 'toggleNewUserPassword');

    // Handle login
    const loginForm = document.getElementById('loginForm');
    if (loginForm) {
        loginForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const username = document.getElementById('loginUsername').value;
            const password = document.getElementById('loginPassword').value;

            showLoading('Logging in...');
            try {
                const response = await fetch(`${API_URL}/login`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ username, password })
                });

                const data = await response.json();
                if (response.ok) {
                    // Store only the token
                    localStorage.setItem('token', data.token);
                    
                    const modal = bootstrap.Modal.getInstance(document.getElementById('loginModal'));
                    if (modal) modal.hide();
                    loginForm.reset();
                    
                    // Update UI using the new function
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

    // Handle file upload
    const uploadForm = document.getElementById('uploadForm');
    const fileInput = document.getElementById('fileInput');
    if (uploadForm && fileInput) {
        uploadForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const files = fileInput.files;
            if (files.length === 0) return;

            // Check if shared checkbox is checked
            const isShared = document.getElementById('sharedCheckbox')?.checked || false;
            const uploadUrl = `${API_URL}/upload${isShared ? '?shared=true' : ''}`;

            showLoading('Uploading files...');
            let uploadSuccess = true;
            // Upload each file individually
            for (let file of files) {
                const formData = new FormData();
                formData.append('files', file);

                try {
                    const response = await fetch(uploadUrl, {
                        method: 'POST',
                        headers: {
                            'Authorization': `Bearer ${getAuthTokenOrRedirect()}`,
                        },
                        body: formData,
                    });

                    if (!response.ok) {
                        const data = await response.json();
                        alert(data.error || `Failed to upload ${file.name}`);
                        uploadSuccess = false;
                    }
                } catch (error) {
                    console.error('Upload error:', error);
                    alert(`Error uploading ${file.name}`);
                    uploadSuccess = false;
                }
            }

            // Clear the input and reload files
            fileInput.value = '';
            uploadForm.reset(); // Reset the entire form including the shared checkbox
            await loadFiles();

            if (uploadSuccess) {
                showLoading('Upload successful!');
                // Wait for 2 seconds before hiding the loading message
                setTimeout(() => {
                    hideLoading();
                }, 2000);
            } else {
                hideLoading();
            }
        });
    }

    // Handle reset password
    const resetPasswordForm = document.getElementById('resetPasswordForm');
    if (resetPasswordForm) {
        resetPasswordForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const currentPassword = document.getElementById('currentPassword').value;
            const newPassword = document.getElementById('newPassword').value;

            // Validate new password length
            if (newPassword.length < 8) {
                alert('New password must be at least 8 characters long');
                return;
            }

            showLoading('Resetting password...');
            try {
                const response = await fetch(`${API_URL}/reset-password`, {
                    method: 'PUT',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${getAuthTokenOrRedirect()}`,
                    },
                    body: JSON.stringify({ current_password: currentPassword, new_password: newPassword })
                });

                const data = await response.json();
                if (response.ok) {
                    const modal = bootstrap.Modal.getInstance(document.getElementById('resetPasswordModal'));
                    if (modal) modal.hide();
                    resetPasswordForm.reset();
                    showLoading('Password reset successful!');
                    setTimeout(() => {
                        hideLoading();
                    }, 2000);
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

    // Handle create user
    const createUserForm = document.getElementById('createUserForm');
    if (createUserForm) {
        createUserForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const username = document.getElementById('newUsername').value;
            const password = document.getElementById('newUserPassword').value;
            const isAdmin = document.getElementById('isAdminCheckbox').checked;

            // Validate password length
            if (password.length < 8) {
                alert('Password must be at least 8 characters long');
                return;
            }

            showLoading('Creating user...');
            try {
                const response = await fetch(`${API_URL}/signup`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${getAuthTokenOrRedirect()}`,
                    },
                    body: JSON.stringify({ username, password, is_admin: isAdmin })
                });

                const data = await response.json();
                if (response.ok) {
                    const modal = bootstrap.Modal.getInstance(document.getElementById('createUserModal'));
                    if (modal) modal.hide();
                    createUserForm.reset();
                    showLoading('User created successfully!');
                    setTimeout(() => {
                        hideLoading();
                    }, 2000);
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

    // Handle file search
    const fileSearchForm = document.getElementById('fileSearchForm');
    const fileSearchInput = document.getElementById('fileSearchInput');
    if (fileSearchForm && fileSearchInput) {
        fileSearchForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const keyword = fileSearchInput.value.trim();
            if (!keyword) {
                // If search is empty, reload all files
                await loadFiles();
                return;
            }
            showLoading('Searching files...');
            try {
                // Search my files
                const myFilesResponse = await fetch(`${API_URL}/files?keyword=${encodeURIComponent(keyword)}`, {
                    headers: {
                        'Authorization': `Bearer ${getAuthTokenOrRedirect()}`,
                    },
                });
                handleApiResponse(myFilesResponse);
                const myFilesData = await myFilesResponse.json();
                if (!myFilesResponse.ok) {
                    throw new Error(myFilesData.error || 'Failed to search my files');
                }
                const myFilesList = document.getElementById('myFilesList');
                if (myFilesList) {
                    displayFiles(myFilesData, myFilesList, 'My Files');
                }

                // Search shared files
                const sharedFilesResponse = await fetch(`${API_URL}/files?shared=true&keyword=${encodeURIComponent(keyword)}`, {
                    headers: {
                        'Authorization': `Bearer ${getAuthTokenOrRedirect()}`,
                    },
                });
                handleApiResponse(sharedFilesResponse);
                const sharedFilesData = await sharedFilesResponse.json();
                if (!sharedFilesResponse.ok) {
                    throw new Error(sharedFilesData.error || 'Failed to search shared files');
                }
                const sharedFilesList = document.getElementById('sharedFilesList');
                if (sharedFilesList) {
                    displayFiles(sharedFilesData, sharedFilesList, 'Public Files');
                }
            } catch (error) {
                console.error('Error searching files:', error);
                alert(error.message || 'Error searching files');
            } finally {
                hideLoading();
            }
        });
    }

    // Check initial authentication status
    checkAuth();
}); 