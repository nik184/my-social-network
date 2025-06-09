let userInfo = null;

// Load initial data when page loads
document.addEventListener('DOMContentLoaded', function() {
    loadUserInfo();
    loadDocs();
});

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
        
        document.getElementById('docModal').style.display = 'block';
    } catch (error) {
        console.error('Error loading doc:', error);
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
        loadPhotos();
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

// Open gallery view
async function openGallery(galleryName) {
    try {
        const data = await sharedApp.fetchAPI(`/api/galleries/${encodeURIComponent(galleryName)}`);
        const images = data.images || [];
        
        if (images.length > 0) {
            // Create URL provider function for gallery images
            const urlProvider = (imageName) => 
                `/api/galleries/${encodeURIComponent(galleryName)}/${encodeURIComponent(imageName)}`;
            
            sharedApp.openImageGallery(images, `${galleryName} Gallery`, 'gallery', urlProvider);
        } else {
            alert('No images found in this gallery');
        }
    } catch (error) {
        console.error('Error loading gallery:', error);
        alert('Error loading gallery: ' + error.message);
    }
}