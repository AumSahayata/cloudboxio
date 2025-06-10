// API Configuration
const API_URL = 'http://localhost:3000';

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

// Global loadFiles function
async function loadFiles() {
    showLoading('Loading files...');
    try {
        // Load my files
        const myFilesResponse = await fetch(`${API_URL}/my-files`, {
            headers: {
                'Authorization': `Bearer ${localStorage.getItem('token')}`,
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
        const sharedFilesResponse = await fetch(`${API_URL}/shared-files`, {
            headers: {
                'Authorization': `Bearer ${localStorage.getItem('token')}`,
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
}

// Delete file
async function deleteFile(fileId) {
    if (!confirm('Are you sure you want to delete this file?')) return;

    showLoading('Deleting file...');
    try {
        const response = await fetch(`${API_URL}/file/${fileId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${localStorage.getItem('token')}`,
            },
        });

        if (response.ok) {
            await loadFiles(); // Reload the file list after successful deletion
        } else {
            const data = await response.json();
            alert(data.error || 'Delete failed');
        }
    } catch (error) {
        // console.error('Delete error:', error);
        alert('Error during delete: ${error}');
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
                'Authorization': `Bearer ${localStorage.getItem('token')}`,
            },
        });

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
        // console.error('Download error:', error);
        alert('Error during download: ${error}');
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
                'Authorization': `Bearer ${localStorage.getItem('token')}`,
            },
        });
        
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
    
    // Update navigation username and email
    const navUsername = document.getElementById('navUsername');
    const navEmail = document.getElementById('navEmail');
    if (navUsername) navUsername.textContent = userData.username;
    if (navEmail) navEmail.textContent = userData.email;
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

// Initialize the application
document.addEventListener('DOMContentLoaded', () => {
    // Initialize password visibility toggles
    togglePasswordVisibility('loginPassword', 'toggleLoginPassword');
    togglePasswordVisibility('signupPassword', 'toggleSignupPassword');

    // Handle login
    const loginForm = document.getElementById('loginForm');
    if (loginForm) {
        loginForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const email = document.getElementById('loginEmail').value;
            const password = document.getElementById('loginPassword').value;

            showLoading('Logging in...');
            try {
                const response = await fetch(`${API_URL}/login`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ email, password })
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

    // Handle signup
    const signupForm = document.getElementById('signupForm');
    if (signupForm) {
        signupForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const username = document.getElementById('signupUsername').value;
            const email = document.getElementById('signupEmail').value;
            const password = document.getElementById('signupPassword').value;

            showLoading('Creating account...');
            try {
                const response = await fetch(`${API_URL}/signup`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ username, email, password }),
                });

                const data = await response.json();
                if (response.ok) {
                    const modal = bootstrap.Modal.getInstance(document.getElementById('signupModal'));
                    if (modal) modal.hide();
                    showLoading('Signup complete! Please login.');
                    signupForm.reset();
                    // Wait for 2 seconds before hiding the loading message
                    setTimeout(() => {
                        hideLoading();
                    }, 2000);
                } else {
                    throw new Error(data.error || 'Signup failed');
                }
            } catch (error) {
                console.error('Signup error:', error);
                alert(error.message || 'Error during signup');
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
                formData.append('file', file);

                try {
                    const response = await fetch(uploadUrl, {
                        method: 'POST',
                        headers: {
                            'Authorization': `Bearer ${localStorage.getItem('token')}`,
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

    // Check initial authentication status
    checkAuth();
}); 