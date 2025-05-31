// Shared JavaScript functionality for both Profile and Network pages

// Global variables
let avatarImages = [];
let currentImageIndex = 0;

// Shared utility functions
function showStatus(elementId, message, isError = false) {
    const element = document.getElementById(elementId);
    if (element) {
        element.innerHTML = message;
        element.className = 'status ' + (isError ? 'error' : 'success');
        element.style.display = 'block';
    }
}

function hideStatus(elementId) {
    const element = document.getElementById(elementId);
    if (element) {
        element.style.display = 'none';
    }
}

function showResult(elementId, data) {
    const element = document.getElementById(elementId);
    if (element) {
        element.textContent = JSON.stringify(data, null, 2);
    }
}

// Avatar-related functions
async function loadAvatarImages() {
    try {
        const response = await fetch('/api/avatar');
        const data = await response.json();
        
        avatarImages = data.images || [];
        return data;
    } catch (error) {
        console.log('No avatar images found or error loading:', error.message);
        avatarImages = [];
        return { images: [], count: 0 };
    }
}

// Update header avatar display
function updateHeaderAvatar(avatarDisplay) {
    if (avatarImages.length > 0) {
        avatarDisplay.innerHTML = `<img src="/api/avatar/${avatarImages[0]}" alt="Avatar" class="avatar">`;
    } else {
        avatarDisplay.innerHTML = '👤';
        avatarDisplay.className = 'avatar-placeholder';
    }
}

// Gallery modal functions
function openGallery() {
    if (avatarImages.length === 0) {
        createAvatarDirectory();
        return;
    }
    
    currentImageIndex = 0;
    showGalleryImage();
    document.getElementById('galleryModal').style.display = 'block';
    updateGalleryCounter();
}

function closeGallery() {
    const modal = document.getElementById('galleryModal');
    if (modal) {
        modal.style.display = 'none';
    }
}

function showGalleryImage() {
    if (avatarImages.length > 0) {
        const galleryImage = document.getElementById('galleryImage');
        if (galleryImage) {
            galleryImage.src = `/api/avatar/${avatarImages[currentImageIndex]}`;
        }
    }
}

function previousImage() {
    if (avatarImages.length > 1) {
        currentImageIndex = (currentImageIndex - 1 + avatarImages.length) % avatarImages.length;
        showGalleryImage();
        updateGalleryCounter();
    }
}

function nextImage() {
    if (avatarImages.length > 1) {
        currentImageIndex = (currentImageIndex + 1) % avatarImages.length;
        showGalleryImage();
        updateGalleryCounter();
    }
}

function updateGalleryCounter() {
    const currentElement = document.getElementById('currentImageIndex');
    const totalElement = document.getElementById('totalImages');
    
    if (currentElement && totalElement) {
        currentElement.textContent = currentImageIndex + 1;
        totalElement.textContent = avatarImages.length;
    }
    
    // Hide navigation arrows if only one image
    const prevBtn = document.querySelector('.gallery-prev');
    const nextBtn = document.querySelector('.gallery-next');
    
    if (prevBtn && nextBtn) {
        if (avatarImages.length <= 1) {
            prevBtn.style.display = 'none';
            nextBtn.style.display = 'none';
        } else {
            prevBtn.style.display = 'block';
            nextBtn.style.display = 'block';
        }
    }
}

// Create avatar directory instruction
async function createAvatarDirectory() {
    try {
        await fetch('/api/create', { method: 'POST' });
        alert('Avatar directory is ready!\n\nTo add your avatar:\n1. Navigate to your space184/images/avatar folder\n2. Place one or more image files (jpg, png, gif, etc.)\n3. Refresh this page\n\nThe first image will become your avatar, and you can browse all images by clicking on it.');
    } catch (error) {
        alert('Error creating avatar directory: ' + error.message);
    }
}

// Peer avatar functions
async function getPeerAvatar(peerID) {
    try {
        const response = await fetch(`/api/peer-avatar/${peerID}`);
        if (!response.ok) {
            return null;
        }
        const data = await response.json();
        if (data.images && data.images.length > 0) {
            return {
                hasAvatar: true,
                primary: data.primary || data.images[0],
                count: data.count,
                url: `/api/peer-avatar/${peerID}/${data.primary || data.images[0]}`
            };
        }
        return null;
    } catch (error) {
        return null;
    }
}

function createPeerAvatarElement(peerID, avatarInfo, size = '32px') {
    if (avatarInfo && avatarInfo.hasAvatar) {
        return `<img src="${avatarInfo.url}" alt="Avatar" style="width: ${size}; height: ${size}; border-radius: 50%; object-fit: cover; margin-right: 10px; border: 2px solid #ddd;" />`;
    } else {
        return `<div style="width: ${size}; height: ${size}; border-radius: 50%; background: #e9ecef; display: flex; align-items: center; justify-content: center; margin-right: 10px; border: 2px solid #ddd; font-size: 16px;">👤</div>`;
    }
}

// API helper functions
async function fetchAPI(url, options = {}) {
    try {
        const response = await fetch(url, options);
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        return await response.json();
    } catch (error) {
        console.error(`API Error for ${url}:`, error);
        throw error;
    }
}

// User info functions
async function getUserInfo() {
    try {
        return await fetchAPI('/api/info');
    } catch (error) {
        console.error('Error loading user info:', error);
        return null;
    }
}

// Utility functions
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Global event handlers
window.onclick = function(event) {
    const galleryModal = document.getElementById('galleryModal');
    const noteModal = document.getElementById('noteModal');
    const avatarModal = document.getElementById('avatarModal');
    
    if (galleryModal && event.target === galleryModal) {
        closeGallery();
    }
    if (noteModal && event.target === noteModal) {
        closeNoteModal();
    }
    if (avatarModal && event.target === avatarModal) {
        closeAvatarGallery();
    }
}

// Keyboard navigation
document.addEventListener('keydown', function(event) {
    const galleryModal = document.getElementById('galleryModal');
    const noteModal = document.getElementById('noteModal');
    const avatarModal = document.getElementById('avatarModal');
    
    // Gallery modal keyboard controls
    if (galleryModal && galleryModal.style.display === 'block') {
        if (event.key === 'ArrowLeft') {
            previousImage();
        } else if (event.key === 'ArrowRight') {
            nextImage();
        } else if (event.key === 'Escape') {
            closeGallery();
        }
    }
    
    // Note modal keyboard controls
    if (noteModal && noteModal.style.display === 'block' && event.key === 'Escape') {
        closeNoteModal();
    }
    
    // Avatar modal keyboard controls
    if (avatarModal && avatarModal.style.display === 'block') {
        if (event.key === 'Escape') {
            closeAvatarGallery();
        } else if (event.key === 'ArrowLeft') {
            previousAvatar();
        } else if (event.key === 'ArrowRight') {
            nextAvatar();
        }
    }
});

// Modal close functions (profile-specific)
function closeNoteModal() {
    const modal = document.getElementById('noteModal');
    if (modal) {
        modal.style.display = 'none';
    }
}

function closeAvatarGallery() {
    const modal = document.getElementById('avatarModal');
    if (modal) {
        modal.style.display = 'none';
    }
}

function openAvatarGallery() {
    if (avatarImages.length === 0) {
        alert('No avatar images available. Add images to your space184/images/avatar directory.');
        return;
    }
    
    showAvatarImage();
    updateAvatarCounter();
    document.getElementById('avatarModal').style.display = 'block';
}

function showAvatarImage() {
    if (avatarImages.length === 0) return;
    
    const image = avatarImages[currentAvatarIndex];
    const imageUrl = `/api/avatar/${image}`;
    
    const galleryContent = document.getElementById('avatarGalleryContent');
    if (galleryContent) {
        galleryContent.innerHTML = 
            `<img src="${imageUrl}" alt="Avatar" style="max-width: 100%; max-height: 400px; border-radius: 10px;" />`;
    }
}

function updateAvatarCounter() {
    const counterElement = document.getElementById('avatarCounter');
    if (counterElement) {
        counterElement.textContent = `${currentAvatarIndex + 1} of ${avatarImages.length}`;
    }
    
    // Hide navigation if only one image
    const prevBtn = document.getElementById('prevAvatarBtn');
    const nextBtn = document.getElementById('nextAvatarBtn');
    if (prevBtn && nextBtn) {
        if (avatarImages.length <= 1) {
            prevBtn.style.display = 'none';
            nextBtn.style.display = 'none';
        } else {
            prevBtn.style.display = 'inline-block';
            nextBtn.style.display = 'inline-block';
        }
    }
}

function previousAvatar() {
    if (avatarImages.length > 1) {
        currentAvatarIndex = (currentAvatarIndex - 1 + avatarImages.length) % avatarImages.length;
        showAvatarImage();
        updateAvatarCounter();
    }
}

function nextAvatar() {
    if (avatarImages.length > 1) {
        currentAvatarIndex = (currentAvatarIndex + 1) % avatarImages.length;
        showAvatarImage();
        updateAvatarCounter();
    }
}

// Export functions for global access
window.sharedApp = {
    loadAvatarImages,
    updateHeaderAvatar,
    openGallery,
    closeGallery,
    showGalleryImage,
    previousImage,
    nextImage,
    updateGalleryCounter,
    createAvatarDirectory,
    getPeerAvatar,
    createPeerAvatarElement,
    fetchAPI,
    getUserInfo,
    escapeHtml,
    showStatus,
    hideStatus,
    showResult,
    closeNoteModal,
    closeAvatarGallery,
    openAvatarGallery,
    showAvatarImage,
    updateAvatarCounter,
    previousAvatar,
    nextAvatar
};