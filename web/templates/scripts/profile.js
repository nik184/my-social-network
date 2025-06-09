let userInfo = null;
let currentFriend = null;
let isViewingFriend = false;

// Load initial data when page loads
document.addEventListener('DOMContentLoaded', function() {
    // Check if we're viewing a friend's profile or our own
    const peerID = getPeerIdFromUrl();
    if (peerID) {
        isViewingFriend = true;
        loadFriendProfile();
    } else {
        isViewingFriend = false;
        loadUserInfo();
        loadDocs();
    }
});

// Get peer ID from URL parameters
function getPeerIdFromUrl() {
    const urlParams = new URLSearchParams(window.location.search);
    return urlParams.get('peer_id');
}

// Load friend profile and docs
async function loadFriendProfile() {
    const peerID = getPeerIdFromUrl();
    if (!peerID) {
        sharedApp.showStatus('friendStatus', 'No peer ID provided', true);
        document.getElementById('profileName').textContent = 'Error';
        return;
    }

    try {
        // Load friend info
        const friendInfo = await loadFriendInfo(peerID);
        if (!friendInfo) {
            sharedApp.showStatus('friendStatus', 'Friend not found', true);
            return;
        }

        currentFriend = friendInfo;
        setCurrentFriend(friendInfo);
        
        // Update profile display
        document.getElementById('profileName').textContent = friendInfo.peer_name;
        document.getElementById('profileId').textContent = `Peer ID: ${peerID}`;

        // Load friend's avatar
        const avatarInfo = await sharedApp.getPeerAvatar(peerID);
        const profileAvatar = document.getElementById('profileAvatar');
        if (avatarInfo && avatarInfo.hasAvatar) {
            profileAvatar.innerHTML = `<img src="${avatarInfo.url}" alt="Avatar" />`;
        } else {
            profileAvatar.innerHTML = '👤';
        }

        // Remove onclick for friends' avatars
        profileAvatar.onclick = null;

        // Show friend-specific UI elements
        document.getElementById('backButtonSection').style.display = 'block';
        document.getElementById('downloadSection').style.display = 'block';
        document.getElementById('tabNavigation').style.display = 'block';
        
        // Hide upload buttons for friend profiles
        document.getElementById('addDocsBtn').style.display = 'none';
        document.getElementById('addPhotosBtn').style.display = 'none';

        // Load friend's docs
        await loadFriendDocs(peerID);

    } catch (error) {
        console.error('Error loading friend profile:', error);
        sharedApp.showStatus('friendStatus', 'Error loading friend profile: ' + error.message, true);
    }
}

// Load friend info from API
async function loadFriendInfo(peerID) {
    try {
        const friend = await sharedApp.fetchAPI(`/api/friends/${peerID}`);
        return friend;
    } catch (error) {
        console.error('Error loading friend info:', error);
        return null;
    }
}

// Load user information and avatar
async function loadUserInfo() {
    try {
        const data = await sharedApp.getUserInfo();
        if (!data) {
            document.getElementById('profileName').textContent = 'Error loading profile';
            return;
        }
        
        userInfo = data;

        // Update profile name
        let name = 'Unknown User';
        if (data.node && data.node.id) {
            const nodeId = data.node.id.toString();
            document.getElementById('profileId').textContent = `Peer ID: ${nodeId}`;
        }

        // Try to get user name from database/settings
        // For now, we'll use a default name
        document.getElementById('profileName').textContent = name;

        // Load avatar
        await loadAvatar();
        
        // Show upload buttons for own profile
        document.getElementById('addDocsBtn').style.display = 'inline-block';
        document.getElementById('addPhotosBtn').style.display = 'inline-block';
    } catch (error) {
        console.error('Error loading user info:', error);
        document.getElementById('profileName').textContent = 'Error loading profile';
    }
}

// Load user avatar
async function loadAvatar() {
    try {
        const data = await sharedApp.loadAvatarImages();
        
        if (avatarImages.length > 0) {
            const primaryAvatar = data.primary || avatarImages[0];
            const avatarUrl = `/api/avatar/${primaryAvatar}`;
            
            document.getElementById('profileAvatar').innerHTML = 
                `<img src="${avatarUrl}" alt="Avatar" />`;
        } else {
            // No avatar, keep default icon
            document.getElementById('profileAvatar').innerHTML = '👤';
        }
    } catch (error) {
        console.error('Error loading avatar:', error);
        document.getElementById('profileAvatar').innerHTML = '👤';
    }
}

// Load docs from the server
async function loadDocs() {
    try {
        sharedApp.showStatus('docsStatus', 'Loading docs...', false);
        
        const data = await sharedApp.fetchAPI('/api/docs');
        
        displayDocs(data.docs || []);
        sharedApp.hideStatus('docsStatus');
    } catch (error) {
        console.error('Error loading docs:', error);
        sharedApp.showStatus('docsStatus', 'Error loading docs: ' + error.message, true);
        displayEmptyState('Failed to load docs');
    }
}

// Load friend's docs via P2P
async function loadFriendDocs(peerID) {
    try {
        sharedApp.showStatus('docsStatus', 'Loading docs via P2P...', false);
        
        const data = await sharedApp.fetchAPI(`/api/peer-docs/${peerID}`);
        
        displayFriendDocs(data.docs || []);
        sharedApp.hideStatus('docsStatus');
    } catch (error) {
        console.error('Error loading friend docs:', error);
        sharedApp.showStatus('docsStatus', 'Error loading docs: ' + error.message, true);
        displayFriendDocsEmptyState('Failed to load docs from friend');
    }
}

// Display docs in the grid
function displayDocs(docs) {
    const docsContent = document.getElementById('docsContent');
    
    if (docs.length === 0) {
        displayEmptyState('No docs found');
        return;
    }

    const docsGrid = document.createElement('div');
    docsGrid.className = 'docs-grid';

    docs.forEach(doc => {
        const docCard = document.createElement('div');
        docCard.className = 'doc-card';

        const modifiedDate = new Date(doc.modified_at).toLocaleDateString();
        const sizeKB = Math.round(doc.size / 1024 * 100) / 100;

        docCard.innerHTML = `
            <div class="doc-title">${sharedApp.escapeHtml(doc.title)}</div>
            <div class="doc-meta">
                <span>📅 ${modifiedDate}</span>
                <span>📄 ${sizeKB} KB</span>
            </div>
            <div class="doc-preview">${sharedApp.escapeHtml(doc.preview)}</div>
            <div class="doc-actions">
                <button class="read-more-btn" onclick="openDoc('${sharedApp.escapeHtml(doc.filename)}')">
                    Read more
                </button>
            </div>
        `;

        docsGrid.appendChild(docCard);
    });

    docsContent.innerHTML = '';
    docsContent.appendChild(docsGrid);
}

// Display empty state
function displayEmptyState(message) {
    const docsContent = document.getElementById('docsContent');
    docsContent.innerHTML = `
        <div class="empty-state">
            <div class="empty-state-icon">📝</div>
            <div>${message}</div>
            <div class="create-doc-hint">
                💡 To add docs, create .txt files in your space184/docs directory
            </div>
        </div>
    `;
}

// Display friend's docs
function displayFriendDocs(docs) {
    const docsContent = document.getElementById('docsContent');
    
    if (docs.length === 0) {
        displayFriendDocsEmptyState('No docs found');
        return;
    }

    const docsGrid = document.createElement('div');
    docsGrid.className = 'docs-grid';

    docs.forEach(doc => {
        const docCard = document.createElement('div');
        docCard.className = 'doc-card';

        const modifiedDate = new Date(doc.modified_at).toLocaleDateString();
        const sizeKB = Math.round(doc.size / 1024 * 100) / 100;

        docCard.innerHTML = `
            <div class="doc-title">${sharedApp.escapeHtml(doc.title)}</div>
            <div class="doc-meta">
                <span>📅 ${modifiedDate}</span>
                <span>📄 ${sizeKB} KB</span>
            </div>
            <div class="doc-preview">${sharedApp.escapeHtml(doc.preview)}</div>
            <div class="doc-actions">
                <button class="read-more-btn" onclick="openFriendDoc('${currentFriend.peer_id}', '${sharedApp.escapeHtml(doc.filename)}')">
                    Read more
                </button>
            </div>
        `;

        docsGrid.appendChild(docCard);
    });

    docsContent.innerHTML = '';
    docsContent.appendChild(docsGrid);
}

// Display empty state for friend docs
function displayFriendDocsEmptyState(message) {
    const docsContent = document.getElementById('docsContent');
    docsContent.innerHTML = `
        <div class="empty-state">
            <div class="empty-state-icon">📝</div>
            <div>${message}</div>
            <div class="create-doc-hint">
                📡 Docs are requested directly from your friend via P2P connection
            </div>
        </div>
    `;
}

// Open a specific doc
async function openDoc(filename) {
    try {
        const doc = await sharedApp.fetchAPI(`/api/docs/${encodeURIComponent(filename)}`);
        
        document.getElementById('docModalTitle').textContent = doc.title;
        document.getElementById('docModalMeta').innerHTML = `
            <strong>Filename:</strong> ${sharedApp.escapeHtml(doc.filename)}<br>
            <strong>Modified:</strong> ${new Date(doc.modified_at).toLocaleString()}<br>
            <strong>Size:</strong> ${Math.round(doc.size / 1024 * 100) / 100} KB
        `;
        document.getElementById('docModalContent').textContent = doc.content;
        
        // Set current doc filename and show kebab menu for own docs
        sharedApp.setCurrentDocFilename(filename);
        
        document.getElementById('docModal').style.display = 'block';
    } catch (error) {
        console.error('Error loading doc:', error);
        alert('Error loading doc: ' + error.message);
    }
}

// Open a specific friend doc
async function openFriendDoc(peerID, filename) {
    try {
        const doc = await sharedApp.fetchAPI(`/api/peer-docs/${peerID}/${encodeURIComponent(filename)}`);
        
        document.getElementById('docModalTitle').textContent = doc.title;
        document.getElementById('docModalMeta').innerHTML = `
            <strong>From:</strong> ${sharedApp.escapeHtml(currentFriend.peer_name)}<br>
            <strong>Filename:</strong> ${sharedApp.escapeHtml(doc.filename)}<br>
            <strong>Modified:</strong> ${new Date(doc.modified_at).toLocaleString()}<br>
            <strong>Size:</strong> ${Math.round(doc.size / 1024 * 100) / 100} KB
        `;
        document.getElementById('docModalContent').textContent = doc.content;
        
        // No kebab menu for friend docs - set empty filename
        sharedApp.setCurrentDocFilename('');
        
        document.getElementById('docModal').style.display = 'block';
    } catch (error) {
        console.error('Error loading friend doc:', error);
        alert('Error loading doc: ' + error.message);
    }
}

// Avatar gallery functions - use shared functions directly

// Tab switching functionality
function switchTab(tabName) {
    // Remove active class from all tabs and buttons
    document.querySelectorAll('.tab-button').forEach(btn => btn.classList.remove('active'));
    document.querySelectorAll('.tab-content').forEach(content => content.classList.remove('active'));
    
    // Add active class to clicked button and corresponding content
    event.target.classList.add('active');
    document.getElementById(tabName + 'Tab').classList.add('active');
    
    // Load tab content if needed
    if (tabName === 'photos' && !photosLoaded) {
        if (isViewingFriend && currentFriend) {
            loadFriendPhotos(currentFriend.peer_id);
        } else {
            loadPhotos();
        }
    }
}

// Gallery variables
let photosLoaded = false;

// Load photos and galleries
async function loadPhotos() {
    try {
        sharedApp.showStatus('photosStatus', 'Loading galleries...', false);
        
        const data = await sharedApp.fetchAPI('/api/galleries');
        
        displayGalleries(data.galleries || []);
        photosLoaded = true;
        sharedApp.hideStatus('photosStatus');
    } catch (error) {
        console.error('Error loading galleries:', error);
        sharedApp.showStatus('photosStatus', 'Error loading galleries: ' + error.message, true);
        displayPhotosEmptyState('Failed to load galleries');
    }
}

// Load friend's photos and galleries via P2P and downloaded content
async function loadFriendPhotos(peerID) {
    try {
        sharedApp.showStatus('photosStatus', 'Loading friend\'s galleries...', false);
        
        // Load both live P2P galleries and downloaded galleries
        const [liveData, downloadedData] = await Promise.allSettled([
            sharedApp.fetchAPI(`/api/peer-galleries/${peerID}`),
            sharedApp.fetchAPI(`/api/downloaded/${peerID}/images`)
        ]);
        
        let liveGalleries = [];
        let downloadedGalleries = [];
        
        if (liveData.status === 'fulfilled') {
            liveGalleries = liveData.value.galleries || [];
        } else {
            console.warn('Failed to load live galleries:', liveData.reason);
        }
        
        if (downloadedData.status === 'fulfilled') {
            downloadedGalleries = downloadedData.value.galleries || [];
        } else {
            console.warn('Failed to load downloaded galleries:', downloadedData.reason);
        }
        
        // Combine and display galleries with source indication
        displayFriendGalleries(liveGalleries, downloadedGalleries, peerID);
        photosLoaded = true;
        sharedApp.hideStatus('photosStatus');
    } catch (error) {
        console.error('Error loading friend galleries:', error);
        sharedApp.showStatus('photosStatus', 'Error loading friend galleries: ' + error.message, true);
        displayFriendPhotosEmptyState('Failed to load galleries from friend');
    }
}

// Display galleries in the grid
function displayGalleries(galleries) {
    const photosContent = document.getElementById('photosContent');
    
    if (galleries.length === 0) {
        displayPhotosEmptyState('No photo galleries found');
        return;
    }

    const galleriesGrid = document.createElement('div');
    galleriesGrid.className = 'galleries-grid';

    galleries.forEach(gallery => {
        const galleryCard = document.createElement('div');
        galleryCard.className = 'gallery-card';
        galleryCard.onclick = () => openGallery(gallery.name);

        const preview = gallery.images.length > 0 
            ? `<img src="/api/galleries/${encodeURIComponent(gallery.name)}/${encodeURIComponent(gallery.images[0])}" alt="${sharedApp.escapeHtml(gallery.name)}" />`
            : '<div class="gallery-placeholder">📷</div>';

        galleryCard.innerHTML = `
            <div class="gallery-preview">
                ${preview}
            </div>
            <div class="gallery-info">
                <div class="gallery-name">${sharedApp.escapeHtml(gallery.name)}</div>
                <div class="gallery-count">${gallery.image_count} images</div>
            </div>
        `;

        galleriesGrid.appendChild(galleryCard);
    });

    photosContent.innerHTML = '';
    photosContent.appendChild(galleriesGrid);
}

// Display friend's galleries in the grid (both live and downloaded)
function displayFriendGalleries(liveGalleries, downloadedGalleries, peerID) {
    const photosContent = document.getElementById('photosContent');
    
    // Merge galleries and mark their source
    const allGalleries = [];
    const galleriesMap = new Map();
    
    // Add live galleries
    liveGalleries.forEach(gallery => {
        gallery.source = 'live';
        gallery.sourceLabel = 'Live P2P';
        galleriesMap.set(gallery.name, gallery);
        allGalleries.push(gallery);
    });
    
    // Add downloaded galleries (avoid duplicates)
    downloadedGalleries.forEach(gallery => {
        if (!galleriesMap.has(gallery.name)) {
            gallery.source = 'downloaded';
            gallery.sourceLabel = 'Downloaded';
            allGalleries.push(gallery);
        } else {
            // Mark that this gallery is also downloaded
            galleriesMap.get(gallery.name).isDownloaded = true;
        }
    });
    
    if (allGalleries.length === 0) {
        displayFriendPhotosEmptyState('No photo galleries found');
        return;
    }

    const galleriesGrid = document.createElement('div');
    galleriesGrid.className = 'galleries-grid';

    allGalleries.forEach(gallery => {
        const galleryCard = document.createElement('div');
        galleryCard.className = 'gallery-card';
        galleryCard.onclick = () => openFriendGallery(peerID, gallery.name, gallery.source);

        // Choose appropriate preview source
        let previewSrc = '';
        if (gallery.images.length > 0) {
            if (gallery.source === 'live') {
                previewSrc = `/api/peer-galleries/${encodeURIComponent(peerID)}/${encodeURIComponent(gallery.name)}/${encodeURIComponent(gallery.images[0])}`;
            } else {
                previewSrc = `/api/downloaded/${encodeURIComponent(peerID)}/images/${encodeURIComponent(gallery.name)}/${encodeURIComponent(gallery.images[0])}`;
            }
        }

        const preview = gallery.images.length > 0 
            ? `<img src="${previewSrc}" alt="${sharedApp.escapeHtml(gallery.name)}" />`
            : '<div class="gallery-placeholder">📷</div>';

        // Create source indicator
        let sourceIndicator = `<div class="gallery-source">${gallery.sourceLabel}`;
        if (gallery.isDownloaded) {
            sourceIndicator += ' (Also Downloaded)';
        }
        sourceIndicator += '</div>';

        galleryCard.innerHTML = `
            <div class="gallery-preview">
                ${preview}
            </div>
            <div class="gallery-info">
                <div class="gallery-name">${sharedApp.escapeHtml(gallery.name)}</div>
                <div class="gallery-count">${gallery.image_count} images</div>
                ${sourceIndicator}
            </div>
        `;

        galleriesGrid.appendChild(galleryCard);
    });

    photosContent.innerHTML = '';
    photosContent.appendChild(galleriesGrid);
}

// Display empty state for photos
function displayPhotosEmptyState(message) {
    const photosContent = document.getElementById('photosContent');
    photosContent.innerHTML = `
        <div class="empty-state">
            <div class="empty-state-icon">📷</div>
            <div>${message}</div>
            <div class="create-doc-hint">
                💡 To add photo galleries, create subdirectories in your space184/images directory and add images to them
            </div>
        </div>
    `;
}

// Display empty state for friend photos
function displayFriendPhotosEmptyState(message) {
    const photosContent = document.getElementById('photosContent');
    photosContent.innerHTML = `
        <div class="empty-state">
            <div class="empty-state-icon">📷</div>
            <div>${message}</div>
            <div class="create-doc-hint">
                📡 Photo galleries are requested directly from your friend via P2P connection
            </div>
        </div>
    `;
}

// Open gallery view
async function openGallery(galleryName) {
    try {
        const data = await sharedApp.fetchAPI(`/api/galleries/${encodeURIComponent(galleryName)}`);
        const images = data.images || [];
        
        if (images.length > 0) {
            // Create URL provider function for gallery images
            const urlProvider = (imageName) => 
                `/api/galleries/${encodeURIComponent(galleryName)}/${encodeURIComponent(imageName)}`;
            
            // This is own content, so show kebab menu
            const isOwnContent = !isViewingFriend;
            sharedApp.openImageGallery(images, `${galleryName} Gallery`, 'gallery', urlProvider, galleryName, isOwnContent);
        } else {
            alert('No images found in this gallery');
        }
    } catch (error) {
        console.error('Error loading gallery:', error);
        alert('Error loading gallery: ' + error.message);
    }
}

// Open friend gallery view
async function openFriendGallery(peerID, galleryName, source = 'live') {
    try {
        let apiUrl, urlProvider;
        
        if (source === 'downloaded') {
            // Use downloaded content API
            apiUrl = `/api/downloaded/${encodeURIComponent(peerID)}/images/${encodeURIComponent(galleryName)}`;
            urlProvider = (imageName) => 
                `/api/downloaded/${encodeURIComponent(peerID)}/images/${encodeURIComponent(galleryName)}/${encodeURIComponent(imageName)}`;
        } else {
            // Use live P2P API
            apiUrl = `/api/peer-galleries/${encodeURIComponent(peerID)}/${encodeURIComponent(galleryName)}`;
            urlProvider = (imageName) => 
                `/api/peer-galleries/${encodeURIComponent(peerID)}/${encodeURIComponent(galleryName)}/${encodeURIComponent(imageName)}`;
        }
        
        const data = await sharedApp.fetchAPI(apiUrl);
        const images = data.images || [];
        
        if (images.length > 0) {
            const friendName = currentFriend ? currentFriend.peer_name : 'Friend';
            const sourceLabel = source === 'downloaded' ? ' (Downloaded)' : ' (Live)';
            // Friend galleries don't get kebab menu (not own content)
            sharedApp.openImageGallery(images, `${friendName}'s ${galleryName} Gallery${sourceLabel}`, 'friend-gallery', urlProvider, galleryName, false);
        } else {
            alert('No images found in this gallery');
        }
    } catch (error) {
        console.error('Error loading friend gallery:', error);
        alert('Error loading friend gallery: ' + error.message);
    }
}

// Friend-specific functions
function goBack() {
    window.location.href = '/friends';
}

function updatePageTitle(friendName) {
    document.title = `${friendName}'s Profile - My Social Network`;
}

function setCurrentFriend(friend) {
    currentFriend = friend;
    if (friend && friend.peer_name) {
        updatePageTitle(friend.peer_name);
    }
}

// Download all content from the current friend
async function downloadAllContent() {
    if (!currentFriend) {
        alert('No friend profile loaded');
        return;
    }

    const peerID = currentFriend.peer_id;
    const downloadBtn = document.getElementById('downloadContentBtn');
    
    try {
        downloadBtn.disabled = true;
        downloadBtn.textContent = '📥 Downloading...';
        sharedApp.showStatus('downloadStatus', 'Starting download of all content...', false);
        
        const response = await fetch(`/api/peer-docs/${peerID}/download`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
        });
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        const result = await response.json();
        
        const docsCount = result.docs_downloaded || 0;
        const imagesCount = result.images_downloaded || 0;
        const totalFiles = docsCount + imagesCount;
        const errors = result.errors || [];
        
        let statusMessage = `✅ Download completed! ${totalFiles} files saved (${docsCount} docs, ${imagesCount} images)`;
        
        if (errors.length > 0) {
            statusMessage += ` | ${errors.length} errors occurred`;
            console.warn('Download errors:', errors);
        }
        
        sharedApp.showStatus('downloadStatus', statusMessage, errors.length > 0);
        
        console.log('Download result:', result);
        
        if (errors.length === 0) {
            setTimeout(() => {
                sharedApp.hideStatus('downloadStatus');
            }, 5000);
        }
        
    } catch (error) {
        console.error('Error downloading content:', error);
        sharedApp.showStatus('downloadStatus', 'Error downloading content: ' + error.message, true);
    } finally {
        downloadBtn.disabled = false;
        downloadBtn.textContent = '📥 Download All Content';
    }
}

// Upload Modal Functions
function openUploadModal(type) {
    if (type === 'docs') {
        document.getElementById('uploadDocsModal').style.display = 'block';
    } else if (type === 'photos') {
        document.getElementById('uploadPhotosModal').style.display = 'block';
    }
}

function closeUploadModal() {
    document.getElementById('uploadDocsModal').style.display = 'none';
    document.getElementById('uploadPhotosModal').style.display = 'none';
    
    // Reset forms
    document.getElementById('uploadDocsForm').reset();
    document.getElementById('uploadPhotosForm').reset();
    
    // Hide status messages
    sharedApp.hideStatus('uploadDocsStatus');
    sharedApp.hideStatus('uploadPhotosStatus');
}

// Handle document upload form submission
document.addEventListener('DOMContentLoaded', function() {
    const uploadDocsForm = document.getElementById('uploadDocsForm');
    if (uploadDocsForm) {
        uploadDocsForm.addEventListener('submit', async function(e) {
            e.preventDefault();
            await handleFileUpload('docs');
        });
    }
    
    const uploadPhotosForm = document.getElementById('uploadPhotosForm');
    if (uploadPhotosForm) {
        uploadPhotosForm.addEventListener('submit', async function(e) {
            e.preventDefault();
            await handleFileUpload('photos');
        });
    }
});

// Handle file upload for both docs and photos
async function handleFileUpload(type) {
    const isPhotos = type === 'photos';
    const formId = isPhotos ? 'uploadPhotosForm' : 'uploadDocsForm';
    const filesInputId = isPhotos ? 'photosFiles' : 'docsFiles';
    const subdirInputId = isPhotos ? 'photosSubdirectory' : 'docsSubdirectory';
    const statusId = isPhotos ? 'uploadPhotosStatus' : 'uploadDocsStatus';
    
    const form = document.getElementById(formId);
    const filesInput = document.getElementById(filesInputId);
    const subdirInput = document.getElementById(subdirInputId);
    
    // Validate files are selected
    if (!filesInput.files || filesInput.files.length === 0) {
        sharedApp.showStatus(statusId, 'Please select at least one file to upload', true);
        return;
    }
    
    // Validate file types
    const allowedExtensions = isPhotos 
        ? ['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp', 'svg']
        : ['md', 'pdf', 'txt', 'html', 'djvu', 'doc', 'docx'];
    
    for (let file of filesInput.files) {
        const extension = file.name.split('.').pop().toLowerCase();
        if (!allowedExtensions.includes(extension)) {
            sharedApp.showStatus(statusId, `Invalid file type: ${file.name}. Allowed: ${allowedExtensions.join(', ')}`, true);
            return;
        }
    }
    
    // Create FormData
    const formData = new FormData();
    for (let file of filesInput.files) {
        formData.append('files', file);
    }
    
    const subdirectory = subdirInput.value.trim();
    if (subdirectory) {
        formData.append('subdirectory', subdirectory);
    }
    
    try {
        // Show loading status
        const uploadType = isPhotos ? 'photos' : 'documents';
        sharedApp.showStatus(statusId, `📤 Uploading ${uploadType}...`, false);
        
        // Disable form submit button
        const submitBtn = form.querySelector('button[type="submit"]');
        submitBtn.disabled = true;
        submitBtn.textContent = '📤 Uploading...';
        
        // Make upload request
        const endpoint = isPhotos ? '/api/upload/photos' : '/api/upload/docs';
        const response = await fetch(endpoint, {
            method: 'POST',
            body: formData
        });
        
        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`Upload failed: ${errorText}`);
        }
        
        const result = await response.json();
        
        // Show success message
        const fileCount = filesInput.files.length;
        const successMsg = `✅ Successfully uploaded ${fileCount} ${uploadType}!`;
        sharedApp.showStatus(statusId, successMsg, false);
        
        // Close modal after a delay
        setTimeout(() => {
            closeUploadModal();
            // Refresh the appropriate tab content
            if (isPhotos) {
                loadPhotos();
            } else {
                loadDocs();
            }
        }, 2000);
        
    } catch (error) {
        console.error('Upload error:', error);
        sharedApp.showStatus(statusId, `❌ Upload failed: ${error.message}`, true);
    } finally {
        // Re-enable form submit button
        const submitBtn = form.querySelector('button[type="submit"]');
        submitBtn.disabled = false;
        submitBtn.textContent = isPhotos ? '📤 Upload Photos' : '📤 Upload Documents';
    }
}

// Close modal when clicking outside of it
window.onclick = function(event) {
    const docsModal = document.getElementById('uploadDocsModal');
    const photosModal = document.getElementById('uploadPhotosModal');
    
    if (event.target === docsModal || event.target === photosModal) {
        closeUploadModal();
    }
}