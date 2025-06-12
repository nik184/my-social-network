// Shared JavaScript functionality for both Profile and Network pages

// SPA (Single Page Application) functionality
let currentPage = '';
let isNavigating = false;

// Global variables for unified image gallery
let galleryImages = [];
let currentGalleryIndex = 0;
let galleryType = '';
let galleryTitle = '';
let galleryUrlProvider = null;
let currentGalleryName = '';
let isOwnContent = false;

// Legacy avatar variables (for backward compatibility)
let avatarImages = [];

// Global Audio Player Data
window.globalPlaylistData = {
    currentGallery: null,
    files: [],
    currentIndex: -1,
    isPlaying: false
};

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
        return {images: [], count: 0};
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
function openImageGallery(images, title = 'Gallery', type = 'default', urlProvider = null, galleryName = '', ownContent = false) {
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
    currentGalleryName = galleryName;
    isOwnContent = ownContent;
    currentGalleryIndex = 0;

    // Set title
    const titleElement = document.getElementById('galleryModalTitle');
    if (titleElement) {
        titleElement.textContent = title;
    }

    // Show/hide kebab menu based on whether this is own content
    const kebabMenu = document.getElementById('imageKebabMenu');
    if (kebabMenu) {
        kebabMenu.style.display = isOwnContent && type === 'gallery' ? 'block' : 'none';
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

    // Hide kebab dropdown
    const dropdown = document.getElementById('imageKebabDropdown');
    if (dropdown) {
        dropdown.classList.remove('show');
    }

    // Reset gallery state
    galleryImages = [];
    galleryType = '';
    galleryTitle = '';
    galleryUrlProvider = null;
    currentGalleryName = '';
    isOwnContent = false;
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

// Create avatar directory instruction
async function createAvatarDirectory() {
    try {
        await fetch('/api/create', {method: 'POST'});
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
window.onclick = function (event) {
    const imageGalleryModal = document.getElementById('imageGalleryModal');
    const docModal = document.getElementById('docModal');

    if (imageGalleryModal && event.target === imageGalleryModal) {
        closeImageGallery();
    }
    if (docModal && event.target === docModal) {
        closeDocModal();
    }

    // Close kebab dropdowns when clicking outside
    const imageKebabDropdown = document.getElementById('imageKebabDropdown');
    const docKebabDropdown = document.getElementById('docKebabDropdown');

    if (imageKebabDropdown && !event.target.closest('.kebab-menu')) {
        imageKebabDropdown.classList.remove('show');
    }
    if (docKebabDropdown && !event.target.closest('.kebab-menu')) {
        docKebabDropdown.classList.remove('show');
    }
}

// Initialize SPA navigation early
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initializeSPANavigation);
} else {
    initializeSPANavigation();
}

// Keyboard navigation
document.addEventListener('keydown', function (event) {
    const imageGalleryModal = document.getElementById('imageGalleryModal');
    const docModal = document.getElementById('docModal');

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

    // Doc modal keyboard controls
    if (docModal && docModal.style.display === 'block' && event.key === 'Escape') {
        closeDocModal();
    }
});

// Modal close functions (profile-specific)
function closeDocModal() {
    const modal = document.getElementById('docModal');
    if (modal) {
        modal.style.display = 'none';
    }
}

function openAvatarGallery() {
    if (avatarImages.length === 0) {
        alert('No avatar images available. Add images to your space184/images/avatar directory.');
        return;
    }

    openImageGallery(avatarImages, 'Avatar Gallery', 'avatar');
}

// Kebab menu functions for images
function toggleImageKebab(event) {
    event.stopPropagation();
    const dropdown = document.getElementById('imageKebabDropdown');
    if (dropdown) {
        dropdown.classList.toggle('show');
    }
}

// Kebab menu functions for docs
let currentDocFilename = '';

function toggleDocKebab(event) {
    event.stopPropagation();
    const dropdown = document.getElementById('docKebabDropdown');
    if (dropdown) {
        dropdown.classList.toggle('show');
    }
}

// Delete current image
async function deleteCurrentImage() {
    if (!isOwnContent || galleryImages.length === 0) {
        return;
    }

    const currentImage = galleryImages[currentGalleryIndex];
    if (!currentImage || !currentGalleryName) {
        alert('Unable to determine which image to delete');
        return;
    }

    if (!confirm(`Are you sure you want to delete "${currentImage}"?`)) {
        return;
    }

    try {
        const response = await fetch(`/api/delete/images/${encodeURIComponent(currentGalleryName)}/${encodeURIComponent(currentImage)}`, {
            method: 'DELETE'
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`Failed to delete image: ${errorText}`);
        }

        // Remove image from gallery array
        galleryImages.splice(currentGalleryIndex, 1);

        if (galleryImages.length === 0) {
            // No more images, close gallery
            closeImageGallery();
            alert('Image deleted successfully. Gallery is now empty.');

            // Refresh the photos tab if we're on profile page
            if (typeof loadPhotos === 'function') {
                loadPhotos();
            }
        } else {
            // Move to previous image if we were at the end
            if (currentGalleryIndex >= galleryImages.length) {
                currentGalleryIndex = galleryImages.length - 1;
            }

            // Update display
            showGalleryImage();
            updateGalleryImageCounter();

            alert('Image deleted successfully');
        }

        // Hide dropdown
        const dropdown = document.getElementById('imageKebabDropdown');
        if (dropdown) {
            dropdown.classList.remove('show');
        }

    } catch (error) {
        console.error('Error deleting image:', error);
        alert('Error deleting image: ' + error.message);
    }
}

// Delete current document
async function deleteCurrentDoc() {
    if (!currentDocFilename) {
        alert('Unable to determine which document to delete');
        return;
    }

    if (!confirm(`Are you sure you want to delete "${currentDocFilename}"?`)) {
        return;
    }

    try {
        const response = await fetch(`/api/delete/docs/${encodeURIComponent(currentDocFilename)}`, {
            method: 'DELETE'
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`Failed to delete document: ${errorText}`);
        }

        // Close modal
        closeDocModal();
        alert('Document deleted successfully');

        // Refresh the docs tab if we're on profile page
        if (typeof loadDocs === 'function') {
            loadDocs();
        }

    } catch (error) {
        console.error('Error deleting document:', error);
        alert('Error deleting document: ' + error.message);
    }
}

// Function to set current doc filename (called when opening doc modal)
function setCurrentDocFilename(filename) {
    currentDocFilename = filename;

    // Show/hide kebab menu for own documents only
    const kebabMenu = document.getElementById('docKebabMenu');
    if (kebabMenu) {
        // Show kebab menu only if we're not viewing a friend's profile
        const isViewingOwnContent = !isViewingFriend; // This variable should be available in profile.js
        kebabMenu.style.display = isViewingOwnContent ? 'block' : 'none';
    }
}

// SPA Navigation Functions
async function loadPage(url, addToHistory = true) {
    if (isNavigating) return;

    try {
        isNavigating = true;

        // Add loading indicator
        const mainContent = document.querySelector('.main-content');
        if (mainContent) {
            mainContent.style.opacity = '0.7';
        }

        // Fetch the new page content
        const response = await fetch(url);
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const html = await response.text();

        // Parse the response to extract main content
        const parser = new DOMParser();
        const doc = parser.parseFromString(html, 'text/html');

        // Extract the new page content
        const newContainer = doc.querySelector('.container');
        const newTitle = doc.querySelector('title');

        if (newContainer) {
            // Replace the container content
            const currentContainer = document.querySelector('.container');
            if (currentContainer) {
                currentContainer.innerHTML = newContainer.innerHTML;
            }
        }

        // Update page title
        if (newTitle) {
            document.title = newTitle.textContent;
        }

        // Update navigation active states
        updateNavigationState(url);

        // Add to browser history
        if (addToHistory) {
            history.pushState({url: url}, '', url);
        }

        // Execute any page-specific scripts
        executePageScripts(url);

        // Store current page
        currentPage = url;

    } catch (error) {
        console.error('Error loading page:', error);
        // Fallback to regular navigation
        window.location.href = url;
    } finally {
        isNavigating = false;

        // Remove loading indicator
        const mainContent = document.querySelector('.main-content');
        if (mainContent) {
            mainContent.style.opacity = '1';
        }
    }
}

function updateNavigationState(url) {
    // Remove active class from all nav links
    document.querySelectorAll('.nav-link').forEach(link => {
        link.classList.remove('active');
    });

    // Add active class to current page nav link
    const currentLink = document.querySelector(`a[href="${url}"]`);
    if (currentLink) {
        currentLink.classList.add('active');
    }
}

function executePageScripts(url) {
    // Execute page-specific initialization based on URL
    const path = url.split('?')[0]; // Remove query parameters

    switch (path) {
        case '/profile':
        case '/friend-profile':
            const peerID = getPeerIdFromUrl();
            if (peerID) {
                isViewingFriend = true;
                loadFriendProfile();
            } else {
                isViewingFriend = false;
                loadUserInfo();
                loadDocs();
            }
            break;
        case '/friends':
            loadFriends();
            break;
    }
}

// Intercept navigation clicks
function initializeSPANavigation() {
    // Handle navigation links
    document.addEventListener('click', function (event) {
        const link = event.target.closest('a');
        if (!link) return;

        const href = link.getAttribute('href');
        if (!href || href.startsWith('#') || href.startsWith('http') || href.includes('://')) {
            return; // Skip external links and anchors
        }

        // Only intercept internal navigation links
        if (href.startsWith('/')) {
            event.preventDefault();
            loadPage(href);
        }
    });

    // Handle browser back/forward buttons
    window.addEventListener('popstate', function (event) {
        if (event.state && event.state.url) {
            loadPage(event.state.url, false);
        } else {
            loadPage(window.location.pathname, false);
        }
    });

    // Initialize current page state
    currentPage = window.location.pathname;
    history.replaceState({url: currentPage}, '', currentPage);
}

// Function to get peer ID from URL (for profile pages)
function getPeerIdFromUrl() {
    const urlParams = new URLSearchParams(window.location.search);
    return urlParams.get('peer_id');
}

// Global Audio Player Functions
function playTrack(galleryName, trackIndex, fileName) {
    // Load all files for the gallery to enable navigation
    fetchAPI(`/api/audio-galleries/${encodeURIComponent(galleryName)}`)
        .then(data => {
            const audioFiles = data.audio_files || [];
            if (audioFiles.length > 0) {
                // Store playlist data globally
                window.globalPlaylistData = {
                    currentGallery: galleryName,
                    files: audioFiles,
                    currentIndex: trackIndex,
                    isPlaying: false
                };

                // Play the track in the global player
                playTrackInGlobalPlayer(galleryName, trackIndex, fileName);
            }
        })
        .catch(error => {
            console.error('Error loading audio files:', error);
            alert('Error playing track: ' + error.message);
        });
}

function playTrackInGlobalPlayer(galleryName, trackIndex, fileName) {
    // Get global player elements
    const globalPlayer = document.getElementById('globalAudioPlayer');
    const globalAudio = document.getElementById('globalAudio');
    const globalTitle = document.getElementById('globalPlayerTitle');
    const globalPlaylist = document.getElementById('globalPlayerPlaylist');
    const globalPlayPauseBtn = document.getElementById('globalPlayPauseBtn');
    const globalPrevBtn = document.getElementById('globalPrevBtn');
    const globalNextBtn = document.getElementById('globalNextBtn');

    if (!globalPlayer || !globalAudio || !globalTitle) return;

    // Show the global player
    globalPlayer.classList.add('active');

    // Update track highlighting
    updateTrackHighlighting(galleryName, trackIndex);

    // Set audio source
    const trackUrl = `/api/audio-galleries/${encodeURIComponent(galleryName)}/${encodeURIComponent(fileName)}`;
    globalAudio.src = trackUrl;

    // Update title and playlist info
    const trackName = fileName.replace(/\.[^/.]+$/, '');
    const displayName = galleryName === 'root_audio' ? '🎶 Root Playlist' : galleryName;
    globalTitle.textContent = trackName;
    globalPlaylist.textContent = displayName;

    // Enable controls
    globalPlayPauseBtn.disabled = false;
    updateGlobalNavigationButtons();

    // Auto-play the track
    globalAudio.play().catch(error => {
        console.log('Auto-play prevented:', error);
    });

    // Update play/pause button when audio state changes
    globalAudio.onplay = () => {
        globalPlayPauseBtn.textContent = '⏸';
        window.globalPlaylistData.isPlaying = true;
    };

    globalAudio.onpause = () => {
        globalPlayPauseBtn.textContent = '▶';
        window.globalPlaylistData.isPlaying = false;
    };

    // Auto-advance to next track when current track ends
    globalAudio.onended = () => {
        globalNextTrack();
    };
}

function updateTrackHighlighting(galleryName, currentIndex) {
    // Remove highlighting from all tracks in all playlists
    const allTracks = document.querySelectorAll('.track-item.playing');
    allTracks.forEach(track => {
        track.classList.remove('playing');
    });

    // Highlight current track in the specific gallery
    const tracksContainer = document.getElementById(`tracks-${galleryName}`);
    if (!tracksContainer) return;

    const galleryTracks = tracksContainer.querySelectorAll('.track-item');
    const currentTrack = galleryTracks[currentIndex];
    if (currentTrack) {
        currentTrack.classList.add('playing');
    }
}

function updateGlobalNavigationButtons() {
    const playlistData = window.globalPlaylistData;
    if (!playlistData || !playlistData.files) return;

    const globalPrevBtn = document.getElementById('globalPrevBtn');
    const globalNextBtn = document.getElementById('globalNextBtn');

    if (globalPrevBtn) {
        globalPrevBtn.disabled = playlistData.currentIndex <= 0;
    }

    if (globalNextBtn) {
        globalNextBtn.disabled = playlistData.currentIndex >= playlistData.files.length - 1;
    }
}

function globalTogglePlayPause() {
    const globalAudio = document.getElementById('globalAudio');
    if (!globalAudio) return;

    if (globalAudio.paused) {
        globalAudio.play();
    } else {
        globalAudio.pause();
    }
}

function globalPreviousTrack() {
    const playlistData = window.globalPlaylistData;
    if (!playlistData || !playlistData.currentGallery || playlistData.currentIndex <= 0) return;

    const newIndex = playlistData.currentIndex - 1;
    const fileName = playlistData.files[newIndex];

    playlistData.currentIndex = newIndex;
    playTrackInGlobalPlayer(playlistData.currentGallery, newIndex, fileName);
}

function globalNextTrack() {
    const playlistData = window.globalPlaylistData;
    if (!playlistData || !playlistData.currentGallery || playlistData.currentIndex >= playlistData.files.length - 1) return;

    const newIndex = playlistData.currentIndex + 1;
    const fileName = playlistData.files[newIndex];

    playlistData.currentIndex = newIndex;
    playTrackInGlobalPlayer(playlistData.currentGallery, newIndex, fileName);
}

function hideGlobalPlayer() {
    const globalPlayer = document.getElementById('globalAudioPlayer');
    const globalAudio = document.getElementById('globalAudio');

    if (globalPlayer) {
        globalPlayer.classList.remove('active');
    }

    if (globalAudio) {
        globalAudio.pause();
        globalAudio.src = '';
    }

    // Clear highlighting from all tracks
    const allTracks = document.querySelectorAll('.track-item.playing');
    allTracks.forEach(track => {
        track.classList.remove('playing');
    });

    // Reset global playlist data
    window.globalPlaylistData = {
        currentGallery: null,
        files: [],
        currentIndex: -1,
        isPlaying: false
    };
}

// Export functions for global access
window.sharedApp = {
    // SPA functions
    loadPage,
    initializeSPANavigation,

    // Unified image gallery functions
    openImageGallery,
    closeImageGallery,
    showGalleryImage,
    previousGalleryImage,
    nextGalleryImage,
    updateGalleryImageCounter,

    // Kebab menu functions
    toggleImageKebab,
    toggleDocKebab,
    deleteCurrentImage,
    deleteCurrentDoc,
    setCurrentDocFilename,

    // Legacy functions (for backward compatibility)
    loadAvatarImages,
    updateHeaderAvatar,
    openGallery,
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
    closeDocModal
};

// Make global audio player functions available globally
window.playTrack = playTrack;
window.globalTogglePlayPause = globalTogglePlayPause;
window.globalPreviousTrack = globalPreviousTrack;
window.globalNextTrack = globalNextTrack;
window.hideGlobalPlayer = hideGlobalPlayer;

// Initialize SPA when DOM is loaded
document.addEventListener('DOMContentLoaded', function () {
    initializeSPANavigation();
});