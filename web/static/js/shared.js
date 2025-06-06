// Shared JavaScript functionality for both Profile and Network pages

// Global variables for unified image gallery
let galleryImages = [];
let currentGalleryIndex = 0;
let galleryType = '';
let galleryTitle = '';
let galleryUrlProvider = null;

// Legacy avatar variables (for backward compatibility)
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

// Unified Image Gallery System
function openImageGallery(images, title = 'Gallery', type = 'default', urlProvider = null) {
    if (!images || images.length === 0) {
        if (type === 'avatar') {
            createAvatarDirectory();
        }
        return;
    }
    
    galleryImages = images;
    galleryTitle = title;
    galleryType = type;
    galleryUrlProvider = urlProvider;
    currentGalleryIndex = 0;
    
    // Set title
    const titleElement = document.getElementById('galleryModalTitle');
    if (titleElement) {
        titleElement.textContent = title;
    }
    
    showGalleryImage();
    updateGalleryImageCounter();
    document.getElementById('imageGalleryModal').style.display = 'block';
}

function closeImageGallery() {
    const modal = document.getElementById('imageGalleryModal');
    if (modal) {
        modal.style.display = 'none';
    }
    
    // Reset gallery state
    galleryImages = [];
    galleryType = '';
    galleryTitle = '';
    galleryUrlProvider = null;
    currentGalleryIndex = 0;
}

function showGalleryImage() {
    if (galleryImages.length === 0) return;
    
    const imageUrl = galleryUrlProvider ? galleryUrlProvider(galleryImages[currentGalleryIndex], currentGalleryIndex) : 
                     galleryType === 'avatar' ? `/api/avatar/${galleryImages[currentGalleryIndex]}` :
                     galleryImages[currentGalleryIndex];
    
    const galleryContent = document.getElementById('galleryImageContent');
    if (galleryContent) {
        galleryContent.innerHTML = 
            `<img src="${imageUrl}" alt="${galleryTitle}" style="max-width: 100%; max-height: 400px; border-radius: 10px;" />`;
    }
}

function previousGalleryImage() {
    if (galleryImages.length > 1) {
        currentGalleryIndex = (currentGalleryIndex - 1 + galleryImages.length) % galleryImages.length;
        showGalleryImage();
        updateGalleryImageCounter();
    }
}

function nextGalleryImage() {
    if (galleryImages.length > 1) {
        currentGalleryIndex = (currentGalleryIndex + 1) % galleryImages.length;
        showGalleryImage();
        updateGalleryImageCounter();
    }
}

function updateGalleryImageCounter() {
    const counterElement = document.getElementById('galleryImageCounter');
    if (counterElement) {
        counterElement.textContent = `${currentGalleryIndex + 1} of ${galleryImages.length}`;
    }
    
    // Hide navigation if only one image
    const prevBtn = document.getElementById('prevGalleryBtn');
    const nextBtn = document.getElementById('nextGalleryBtn');
    if (prevBtn && nextBtn) {
        if (galleryImages.length <= 1) {
            prevBtn.style.display = 'none';
            nextBtn.style.display = 'none';
        } else {
            prevBtn.style.display = 'inline-block';
            nextBtn.style.display = 'inline-block';
        }
    }
}

// Legacy functions for backward compatibility
function openGallery() {
    openImageGallery(avatarImages, 'Avatar Gallery', 'avatar');
}

function closeGallery() {
    closeImageGallery();
}

function previousImage() {
    previousGalleryImage();
}

function nextImage() {
    nextGalleryImage();
}

function updateGalleryCounter() {
    updateGalleryImageCounter();
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
    const imageGalleryModal = document.getElementById('imageGalleryModal');
    const noteModal = document.getElementById('noteModal');
    
    if (imageGalleryModal && event.target === imageGalleryModal) {
        closeImageGallery();
    }
    if (noteModal && event.target === noteModal) {
        closeNoteModal();
    }
}

// Keyboard navigation
document.addEventListener('keydown', function(event) {
    const imageGalleryModal = document.getElementById('imageGalleryModal');
    const noteModal = document.getElementById('noteModal');
    
    // Image gallery modal keyboard controls
    if (imageGalleryModal && imageGalleryModal.style.display === 'block') {
        if (event.key === 'ArrowLeft') {
            previousGalleryImage();
        } else if (event.key === 'ArrowRight') {
            nextGalleryImage();
        } else if (event.key === 'Escape') {
            closeImageGallery();
        }
        event.preventDefault();
    }
    
    // Note modal keyboard controls
    if (noteModal && noteModal.style.display === 'block' && event.key === 'Escape') {
        closeNoteModal();
    }
});

// Modal close functions (profile-specific)
function closeNoteModal() {
    const modal = document.getElementById('noteModal');
    if (modal) {
        modal.style.display = 'none';
    }
}

// Avatar gallery functions (now using unified system)
function closeAvatarGallery() {
    closeImageGallery();
}

function openAvatarGallery() {
    if (avatarImages.length === 0) {
        alert('No avatar images available. Add images to your space184/images/avatar directory.');
        return;
    }
    
    openImageGallery(avatarImages, 'Avatar Gallery', 'avatar');
}

function previousAvatar() {
    previousGalleryImage();
}

function nextAvatar() {
    nextGalleryImage();
}

// Export functions for global access
window.sharedApp = {
    // Unified image gallery functions
    openImageGallery,
    closeImageGallery,
    showGalleryImage,
    previousGalleryImage,
    nextGalleryImage,
    updateGalleryImageCounter,
    
    // Legacy functions (for backward compatibility)
    loadAvatarImages,
    updateHeaderAvatar,
    openGallery,
    closeGallery,
    previousImage,
    nextImage,
    updateGalleryCounter,
    createAvatarDirectory,
    openAvatarGallery,

    // Utility functions
    getPeerAvatar,
    createPeerAvatarElement,
    fetchAPI,
    getUserInfo,
    escapeHtml,
    showStatus,
    hideStatus,
    showResult,
    closeNoteModal
};