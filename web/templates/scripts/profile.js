let userInfo = null;

// Load initial data when page loads
document.addEventListener('DOMContentLoaded', function() {
    loadUserInfo();
    loadNotes();
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

// Load notes from the server
async function loadNotes() {
    try {
        sharedApp.showStatus('notesStatus', 'Loading notes...', false);
        
        const data = await sharedApp.fetchAPI('/api/notes');
        
        displayNotes(data.notes || []);
        sharedApp.hideStatus('notesStatus');
    } catch (error) {
        console.error('Error loading notes:', error);
        sharedApp.showStatus('notesStatus', 'Error loading notes: ' + error.message, true);
        displayEmptyState('Failed to load notes');
    }
}

// Display notes in the grid
function displayNotes(notes) {
    const notesContent = document.getElementById('notesContent');
    
    if (notes.length === 0) {
        displayEmptyState('No notes found');
        return;
    }

    const notesGrid = document.createElement('div');
    notesGrid.className = 'notes-grid';

    notes.forEach(note => {
        const noteCard = document.createElement('div');
        noteCard.className = 'note-card';

        const modifiedDate = new Date(note.modified_at).toLocaleDateString();
        const sizeKB = Math.round(note.size / 1024 * 100) / 100;

        noteCard.innerHTML = `
            <div class="note-title">${sharedApp.escapeHtml(note.title)}</div>
            <div class="note-meta">
                <span>📅 ${modifiedDate}</span>
                <span>📄 ${sizeKB} KB</span>
            </div>
            <div class="note-preview">${sharedApp.escapeHtml(note.preview)}</div>
            <div class="note-actions">
                <button class="read-more-btn" onclick="openNote('${sharedApp.escapeHtml(note.filename)}')">
                    Read more
                </button>
            </div>
        `;

        notesGrid.appendChild(noteCard);
    });

    notesContent.innerHTML = '';
    notesContent.appendChild(notesGrid);
}

// Display empty state
function displayEmptyState(message) {
    const notesContent = document.getElementById('notesContent');
    notesContent.innerHTML = `
        <div class="empty-state">
            <div class="empty-state-icon">📝</div>
            <div>${message}</div>
            <div class="create-note-hint">
                💡 To add notes, create .txt files in your space184/notes directory
            </div>
        </div>
    `;
}

// Open a specific note
async function openNote(filename) {
    try {
        const note = await sharedApp.fetchAPI(`/api/notes/${encodeURIComponent(filename)}`);
        
        document.getElementById('noteModalTitle').textContent = note.title;
        document.getElementById('noteModalMeta').innerHTML = `
            <strong>Filename:</strong> ${sharedApp.escapeHtml(note.filename)}<br>
            <strong>Modified:</strong> ${new Date(note.modified_at).toLocaleString()}<br>
            <strong>Size:</strong> ${Math.round(note.size / 1024 * 100) / 100} KB
        `;
        document.getElementById('noteModalContent').textContent = note.content;
        
        document.getElementById('noteModal').style.display = 'block';
    } catch (error) {
        console.error('Error loading note:', error);
        alert('Error loading note: ' + error.message);
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
            <div class="create-note-hint">
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