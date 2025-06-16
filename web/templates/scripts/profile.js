let userInfo = null;
let currentFriend = null;
let isViewingFriend = false;

// Function to initialize the profile page
function initializeProfilePage() {
    // Reset tab loading flags when switching between profiles
    photosLoaded = false;
    audioLoaded = false;
    videoLoaded = false;
    
    // Check if we're viewing a friend's profile or our own
    // Use the navigated URL if available (from SPA navigation), otherwise current location
    const urlToCheck = window.currentNavigatedURL || window.location.href;
    const peerID = getPeerIdFromUrl(urlToCheck);
    
    if (peerID) {
        isViewingFriend = true;
        window.isViewingFriend = true;
        loadFriendProfile();
    } else {
        isViewingFriend = false;
        window.isViewingFriend = false;
        loadUserInfo();
        loadDocs();
    }
}

// Load initial data when page loads (for direct page access only)
document.addEventListener('DOMContentLoaded', function() {
    // Only run automatically if this is not SPA navigation
    if (!window.currentNavigatedURL) {
        initializeProfilePage();
    }
});

// Get peer ID from URL parameters
function getPeerIdFromUrl(url) {
    // If URL is provided, parse it; otherwise use current window location
    if (url) {
        const urlObj = new URL(url, window.location.origin);
        const urlParams = new URLSearchParams(urlObj.search);
        return urlParams.get('peer_id');
    } else {
        const urlParams = new URLSearchParams(window.location.search);
        return urlParams.get('peer_id');
    }
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
        document.getElementById('addAudioBtn').style.display = 'none';
        document.getElementById('addVideoBtn').style.display = 'none';

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
        document.getElementById('addAudioBtn').style.display = 'inline-block';
        document.getElementById('addVideoBtn').style.display = 'inline-block';
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
            <div class="doc-header">
                <div class="doc-title">${sharedApp.escapeHtml(doc.title)}</div>
            </div>
            <div class="doc-body">
                <div class="doc-preview">${sharedApp.escapeHtml(doc.preview)}</div>
                <div class="doc-meta">
                    <span>📅 ${modifiedDate}</span>
                    <span>📄 ${sizeKB} KB</span>
                </div>
                <div class="doc-actions">
                    <button class="read-more-btn" onclick="openDoc('${sharedApp.escapeHtml(doc.filename)}')">
                        Read more
                    </button>
                </div>
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
            <div class="doc-header">
                <div class="doc-title">${sharedApp.escapeHtml(doc.title)}</div>
            </div>
            <div class="doc-body">
                <div class="doc-preview">${sharedApp.escapeHtml(doc.preview)}</div>
                <div class="doc-meta">
                    <span>📅 ${modifiedDate}</span>
                    <span>📄 ${sizeKB} KB</span>
                </div>
                <div class="doc-actions">
                    <button class="read-more-btn" onclick="openFriendDoc('${currentFriend.peer_id}', '${sharedApp.escapeHtml(doc.filename)}')">
                        Read more
                    </button>
                </div>
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
            <strong>Size:</strong> ${Math.round(doc.size / 1024 * 100) / 100} KB<br>
            <strong>Type:</strong> ${doc.content_type === 'html' ? 'Markdown' : 'Text'}
        `;
        
        // Render content based on type
        const contentElement = document.getElementById('docModalContent');
        if (doc.content_type === 'html') {
            contentElement.innerHTML = doc.content;
            contentElement.className = 'doc-content html-content';
        } else {
            contentElement.textContent = doc.content;
            contentElement.className = 'doc-content text-content';
        }
        
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
            <strong>Size:</strong> ${Math.round(doc.size / 1024 * 100) / 100} KB<br>
            <strong>Type:</strong> ${doc.content_type === 'html' ? 'Markdown' : 'Text'}
        `;
        
        // Render content based on type
        const contentElement = document.getElementById('docModalContent');
        if (doc.content_type === 'html') {
            contentElement.innerHTML = doc.content;
            contentElement.className = 'doc-content html-content';
        } else {
            contentElement.textContent = doc.content;
            contentElement.className = 'doc-content text-content';
        }
        
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
    } else if (tabName === 'audio' && !audioLoaded) {
        if (isViewingFriend && currentFriend) {
            loadFriendAudio(currentFriend.peer_id);
        } else {
            loadAudio();
        }
    } else if (tabName === 'video' && !videoLoaded) {
        if (isViewingFriend && currentFriend) {
            loadFriendVideo(currentFriend.peer_id);
        } else {
            loadVideo();
        }
    }
}

// Gallery variables
let photosLoaded = false;
let audioLoaded = false;
let videoLoaded = false;

// Load photos and galleries
async function loadPhotos() {
    try {
        sharedApp.showStatus('photosStatus', 'Loading galleries...', false);
        
        const data = await sharedApp.fetchAPI('/api/media/image/galleries');
        
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

        const preview = gallery.files && gallery.files.length > 0 
            ? `<img src="/api/media/image/galleries/${encodeURIComponent(gallery.name)}/${encodeURIComponent(gallery.files[0])}" alt="${sharedApp.escapeHtml(gallery.name)}" />`
            : '<div class="gallery-placeholder">📷</div>';

        // Display user-friendly name for root gallery
        const displayName = gallery.name === 'root_images' ? '📁 Root Images' : gallery.name;

        galleryCard.innerHTML = `
            <div class="gallery-preview">
                ${preview}
            </div>
            <div class="gallery-info">
                <div class="gallery-name">${sharedApp.escapeHtml(displayName)}</div>
                <div class="gallery-count">${gallery.file_count} images</div>
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
        if (gallery.files && gallery.files.length > 0) {
            if (gallery.source === 'live') {
                previewSrc = `/api/peer-galleries/${encodeURIComponent(peerID)}/${encodeURIComponent(gallery.name)}/${encodeURIComponent(gallery.files[0])}`;
            } else {
                previewSrc = `/api/downloaded/${encodeURIComponent(peerID)}/images/${encodeURIComponent(gallery.name)}/${encodeURIComponent(gallery.files[0])}`;
            }
        }

        const preview = gallery.files && gallery.files.length > 0 
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
                <div class="gallery-count">${gallery.file_count} images</div>
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
        const data = await sharedApp.fetchAPI(`/api/media/image/galleries/${encodeURIComponent(galleryName)}`);
        const images = data.files || [];
        
        if (images.length > 0) {
            // Create URL provider function for gallery images
            const urlProvider = (imageName) => 
                `/api/media/image/galleries/${encodeURIComponent(galleryName)}/${encodeURIComponent(imageName)}`;
            
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
        const images = data.files || [];
        
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

// Load audio galleries
async function loadAudio() {
    try {
        sharedApp.showStatus('audioStatus', 'Loading audio collections...', false);
        
        const data = await sharedApp.fetchAPI('/api/media/audio/galleries');
        
        displayAudioGalleries(data.galleries || []);
        audioLoaded = true;
        sharedApp.hideStatus('audioStatus');
    } catch (error) {
        console.error('Error loading audio galleries:', error);
        sharedApp.showStatus('audioStatus', 'Error loading audio collections: ' + error.message, true);
        displayAudioEmptyState('Failed to load audio collections');
    }
}

// Load friend's audio galleries
async function loadFriendAudio(peerID) {
    try {
        sharedApp.showStatus('audioStatus', 'Loading friend\'s audio collections...', false);
        
        // For now, just show empty state - P2P audio not implemented yet
        displayFriendAudioEmptyState('Audio sharing via P2P not yet implemented');
        audioLoaded = true;
        sharedApp.hideStatus('audioStatus');
    } catch (error) {
        console.error('Error loading friend audio:', error);
        sharedApp.showStatus('audioStatus', 'Error loading friend audio: ' + error.message, true);
        displayFriendAudioEmptyState('Failed to load audio from friend');
    }
}

// Display audio galleries as playlists
function displayAudioGalleries(audioGalleries) {
    const audioContent = document.getElementById('audioContent');
    
    if (audioGalleries.length === 0) {
        displayAudioEmptyState('No audio collections found');
        return;
    }

    const audioContainer = document.createElement('div');
    audioContainer.className = 'audio-container';

    audioGalleries.forEach(gallery => {
        // Display user-friendly name for root gallery
        const displayName = gallery.name === 'root_audio' ? '🎶 Root Playlist' : gallery.name;
        
        const playlistSection = document.createElement('div');
        playlistSection.className = 'playlist-section';
        
        playlistSection.innerHTML = `
            <div class="playlist-header">
                <div class="playlist-title">
                    🎵 ${sharedApp.escapeHtml(displayName)}
                </div>
                <div class="playlist-count">${gallery.file_count} tracks</div>
            </div>
            <div class="playlist-tracks" id="tracks-${gallery.name}">
                <div style="text-align: center; padding: 20px; color: #666;">
                    Loading tracks...
                </div>
            </div>
        `;
        
        audioContainer.appendChild(playlistSection);
        
        // Load tracks for this playlist
        loadPlaylistTracks(gallery.name, displayName);
    });

    audioContent.innerHTML = '';
    audioContent.appendChild(audioContainer);
}

// Load tracks for a specific playlist
async function loadPlaylistTracks(galleryName, displayName) {
    try {
        const data = await sharedApp.fetchAPI(`/api/media/audio/galleries/${encodeURIComponent(galleryName)}`);
        const audioFiles = data.files || [];
        
        const tracksContainer = document.getElementById(`tracks-${galleryName}`);
        if (!tracksContainer) return;
        
        if (audioFiles.length === 0) {
            tracksContainer.innerHTML = `
                <div style="text-align: center; padding: 20px; color: #666;">
                    No tracks found in this playlist
                </div>
            `;
            return;
        }
        
        tracksContainer.innerHTML = '';
        
        audioFiles.forEach((fileName, index) => {
            const trackItem = document.createElement('div');
            trackItem.className = 'track-item';
            
            // Remove file extension for display name
            const trackName = fileName.replace(/\.[^/.]+$/, '');
            
            trackItem.innerHTML = `
                <div class="track-number">${index + 1}</div>
                <div class="track-info">
                    <div class="track-name">${sharedApp.escapeHtml(trackName)}</div>
                    <div class="track-duration">Audio File</div>
                </div>
                <button class="track-play-btn" onclick="playTrack('${galleryName}', ${index}, '${sharedApp.escapeHtml(fileName)}')">
                    ▶
                </button>
            `;
            
            // Store track data for playlist navigation
            trackItem.dataset.galleryName = galleryName;
            trackItem.dataset.trackIndex = index;
            trackItem.dataset.fileName = fileName;
            
            tracksContainer.appendChild(trackItem);
        });
        
    } catch (error) {
        console.error('Error loading playlist tracks:', error);
        const tracksContainer = document.getElementById(`tracks-${galleryName}`);
        if (tracksContainer) {
            tracksContainer.innerHTML = `
                <div style="text-align: center; padding: 20px; color: #e74c3c;">
                    Error loading tracks: ${error.message}
                </div>
            `;
        }
    }
}

// Display empty state for audio
function displayAudioEmptyState(message) {
    const audioContent = document.getElementById('audioContent');
    audioContent.innerHTML = `
        <div class="audio-empty-state">
            <div class="audio-empty-state-icon">🎵</div>
            <div style="font-size: 18px; margin-bottom: 10px;">${message}</div>
            <div style="font-size: 14px;">
                💡 To add audio collections, create subdirectories in your space184/audio directory and add audio files to them
            </div>
        </div>
    `;
}

// Display empty state for friend audio
function displayFriendAudioEmptyState(message) {
    const audioContent = document.getElementById('audioContent');
    audioContent.innerHTML = `
        <div class="empty-state">
            <div class="empty-state-icon">🎵</div>
            <div>${message}</div>
            <div class="create-doc-hint">
                📡 Audio files would be requested directly from your friend via P2P connection
            </div>
        </div>
    `;
}


// Load video galleries
async function loadVideo() {
    try {
        sharedApp.showStatus('videoStatus', 'Loading video collections...', false);
        
        const data = await sharedApp.fetchAPI('/api/media/video/galleries');
        
        displayVideoGalleries(data.galleries || []);
        videoLoaded = true;
        sharedApp.hideStatus('videoStatus');
    } catch (error) {
        console.error('Error loading video galleries:', error);
        sharedApp.showStatus('videoStatus', 'Error loading video collections: ' + error.message, true);
        displayVideoEmptyState('Failed to load video collections');
    }
}

// Load friend's video galleries
async function loadFriendVideo(peerID) {
    try {
        sharedApp.showStatus('videoStatus', 'Loading friend\'s video collections...', false);
        
        // For now, just show empty state - P2P video not implemented yet
        displayFriendVideoEmptyState('Video sharing via P2P not yet implemented');
        videoLoaded = true;
        sharedApp.hideStatus('videoStatus');
    } catch (error) {
        console.error('Error loading friend video:', error);
        sharedApp.showStatus('videoStatus', 'Error loading friend video: ' + error.message, true);
        displayFriendVideoEmptyState('Failed to load video from friend');
    }
}

// Display video galleries
function displayVideoGalleries(videoGalleries) {
    const videoContent = document.getElementById('videoContent');
    
    if (videoGalleries.length === 0) {
        displayVideoEmptyState('No video collections found');
        return;
    }

    const galleriesGrid = document.createElement('div');
    galleriesGrid.className = 'galleries-grid';

    videoGalleries.forEach(gallery => {
        const galleryCard = document.createElement('div');
        galleryCard.className = 'gallery-card';
        galleryCard.onclick = () => openVideoGallery(gallery.name);

        // Display user-friendly name for root gallery
        const displayName = gallery.name === 'root_video' ? '🎥 Root Videos' : gallery.name;

        galleryCard.innerHTML = `
            <div class="gallery-preview">
                <div class="gallery-placeholder">🎬</div>
            </div>
            <div class="gallery-info">
                <div class="gallery-name">${sharedApp.escapeHtml(displayName)}</div>
                <div class="gallery-count">${gallery.file_count} video files</div>
            </div>
        `;

        galleriesGrid.appendChild(galleryCard);
    });

    videoContent.innerHTML = '';
    videoContent.appendChild(galleriesGrid);
}

// Display empty state for video
function displayVideoEmptyState(message) {
    const videoContent = document.getElementById('videoContent');
    videoContent.innerHTML = `
        <div class="empty-state">
            <div class="empty-state-icon">🎬</div>
            <div>${message}</div>
            <div class="create-doc-hint">
                💡 To add video collections, create subdirectories in your space184/video directory and add video files to them
            </div>
        </div>
    `;
}

// Display empty state for friend video
function displayFriendVideoEmptyState(message) {
    const videoContent = document.getElementById('videoContent');
    videoContent.innerHTML = `
        <div class="empty-state">
            <div class="empty-state-icon">🎬</div>
            <div>${message}</div>
            <div class="create-doc-hint">
                📡 Video files would be requested directly from your friend via P2P connection
            </div>
        </div>
    `;
}

// Open video gallery
async function openVideoGallery(galleryName) {
    try {
        const data = await sharedApp.fetchAPI(`/api/media/video/galleries/${encodeURIComponent(galleryName)}`);
        const videoFiles = data.files || [];
        
        if (videoFiles.length > 0) {
            // Open video player modal
            openVideoPlayer(videoFiles, galleryName);
        } else {
            alert('No video files found in this collection');
        }
    } catch (error) {
        console.error('Error loading video gallery:', error);
        alert('Error loading video gallery: ' + error.message);
    }
}

// Open video player modal
function openVideoPlayer(videoFiles, galleryName) {
    // Create video player modal
    const modal = document.createElement('div');
    modal.className = 'modal';
    modal.style.display = 'block';
    
    // Display user-friendly name for root gallery
    const displayName = galleryName === 'root_video' ? '🎥 Root Videos' : galleryName;
    
    modal.innerHTML = `
        <div class="modal-content" style="max-width: 90%; width: 1000px;">
            <div class="modal-header">
                <div class="modal-title">🎬 ${sharedApp.escapeHtml(displayName)} Collection</div>
                <span class="close" onclick="this.closest('.modal').remove()">&times;</span>
            </div>
            <div class="modal-body" style="text-align: center;">
                <div id="videoPlayerContent">
                    <h4 id="currentVideoTitle">${sharedApp.escapeHtml(videoFiles[0])}</h4>
                    <video controls style="width: 100%; max-width: 800px; height: auto; margin: 20px 0;">
                        <source src="/api/media/video/galleries/${encodeURIComponent(galleryName)}/${encodeURIComponent(videoFiles[0])}" type="video/mp4">
                        Your browser does not support the video element.
                    </video>
                    <div style="margin-top: 20px;">
                        <button onclick="previousVideo()" class="read-more-btn" style="margin-right: 10px;">← Previous</button>
                        <span id="videoCounter">1 of ${videoFiles.length}</span>
                        <button onclick="nextVideo()" class="read-more-btn" style="margin-left: 10px;">Next →</button>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    document.body.appendChild(modal);
    
    // Store video data in window for navigation
    window.currentVideoData = {
        files: videoFiles,
        galleryName: galleryName,
        currentIndex: 0,
        modal: modal
    };
    
    // Close modal when clicking outside
    modal.onclick = function(event) {
        if (event.target === modal) {
            modal.remove();
            window.currentVideoData = null;
        }
    };
}

// Video navigation functions
function nextVideo() {
    if (!window.currentVideoData) return;
    
    const data = window.currentVideoData;
    data.currentIndex = (data.currentIndex + 1) % data.files.length;
    updateVideoPlayer();
}

function previousVideo() {
    if (!window.currentVideoData) return;
    
    const data = window.currentVideoData;
    data.currentIndex = (data.currentIndex - 1 + data.files.length) % data.files.length;
    updateVideoPlayer();
}

function updateVideoPlayer() {
    if (!window.currentVideoData) return;
    
    const data = window.currentVideoData;
    const currentFile = data.files[data.currentIndex];
    
    // Update title
    const titleElement = data.modal.querySelector('#currentVideoTitle');
    if (titleElement) {
        titleElement.textContent = currentFile;
    }
    
    // Update video source
    const videoElement = data.modal.querySelector('video');
    if (videoElement) {
        videoElement.src = `/api/media/video/galleries/${encodeURIComponent(data.galleryName)}/${encodeURIComponent(currentFile)}`;
        videoElement.load();
    }
    
    // Update counter
    const counterElement = data.modal.querySelector('#videoCounter');
    if (counterElement) {
        counterElement.textContent = `${data.currentIndex + 1} of ${data.files.length}`;
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
async function openUploadModal(type) {
    if (type === 'docs') {
        document.getElementById('uploadDocsModal').style.display = 'block';
        await populateDocsSubdirectories();
    } else if (type === 'photos') {
        document.getElementById('uploadPhotosModal').style.display = 'block';
        await populateImageGalleries();
    } else if (type === 'audio') {
        document.getElementById('uploadAudioModal').style.display = 'block';
        await populateAudioGalleries();
    } else if (type === 'video') {
        document.getElementById('uploadVideoModal').style.display = 'block';
        await populateVideoGalleries();
    }
}

function closeUploadModal() {
    document.getElementById('uploadDocsModal').style.display = 'none';
    document.getElementById('uploadPhotosModal').style.display = 'none';
    document.getElementById('uploadAudioModal').style.display = 'none';
    document.getElementById('uploadVideoModal').style.display = 'none';
    
    // Reset forms
    document.getElementById('uploadDocsForm').reset();
    document.getElementById('uploadPhotosForm').reset();
    document.getElementById('uploadAudioForm').reset();
    document.getElementById('uploadVideoForm').reset();
    
    // Hide status messages
    sharedApp.hideStatus('uploadDocsStatus');
    sharedApp.hideStatus('uploadPhotosStatus');
    sharedApp.hideStatus('uploadAudioStatus');
    sharedApp.hideStatus('uploadVideoStatus');
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
    
    const uploadAudioForm = document.getElementById('uploadAudioForm');
    if (uploadAudioForm) {
        uploadAudioForm.addEventListener('submit', async function(e) {
            e.preventDefault();
            await handleFileUpload('audio');
        });
    }
    
    const uploadVideoForm = document.getElementById('uploadVideoForm');
    if (uploadVideoForm) {
        uploadVideoForm.addEventListener('submit', async function(e) {
            e.preventDefault();
            await handleFileUpload('video');
        });
    }
});

// Handle file upload for docs, photos, audio, and video
async function handleFileUpload(type) {
    const isPhotos = type === 'photos';
    const isAudio = type === 'audio';
    const isVideo = type === 'video';
    
    let formId, filesInputId, subdirInputId, statusId;
    
    if (isPhotos) {
        formId = 'uploadPhotosForm';
        filesInputId = 'photosFiles';
        subdirInputId = 'photosSubdirectory';
        statusId = 'uploadPhotosStatus';
    } else if (isAudio) {
        formId = 'uploadAudioForm';
        filesInputId = 'audioFiles';
        subdirInputId = 'audioSubdirectory';
        statusId = 'uploadAudioStatus';
    } else if (isVideo) {
        formId = 'uploadVideoForm';
        filesInputId = 'videoFiles';
        subdirInputId = 'videoSubdirectory';
        statusId = 'uploadVideoStatus';
    } else {
        formId = 'uploadDocsForm';
        filesInputId = 'docsFiles';
        subdirInputId = 'docsSubdirectory';
        statusId = 'uploadDocsStatus';
    }
    
    const form = document.getElementById(formId);
    const filesInput = document.getElementById(filesInputId);
    const subdirInput = document.getElementById(subdirInputId);
    
    // Validate files are selected
    if (!filesInput.files || filesInput.files.length === 0) {
        sharedApp.showStatus(statusId, 'Please select at least one file to upload', true);
        return;
    }
    
    // Validate file types
    let allowedExtensions;
    if (isPhotos) {
        allowedExtensions = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp', 'svg'];
    } else if (isAudio) {
        allowedExtensions = ['mp3', 'wav', 'flac', 'aac', 'ogg', 'm4a', 'wma', 'opus'];
    } else if (isVideo) {
        allowedExtensions = ['mp4', 'avi', 'mkv', 'mov', 'wmv', 'flv', 'webm', 'm4v', '3gp', 'mpg', 'mpeg'];
    } else {
        allowedExtensions = ['md', 'pdf', 'txt', 'html', 'djvu', 'doc', 'docx'];
    }
    
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
        let uploadType;
        if (isPhotos) {
            uploadType = 'photos';
        } else if (isAudio) {
            uploadType = 'audio files';
        } else if (isVideo) {
            uploadType = 'video files';
        } else {
            uploadType = 'documents';
        }
        
        sharedApp.showStatus(statusId, `📤 Uploading ${uploadType}...`, false);
        
        // Disable form submit button
        const submitBtn = form.querySelector('button[type="submit"]');
        submitBtn.disabled = true;
        submitBtn.textContent = '📤 Uploading...';
        
        // Make upload request
        let endpoint;
        if (isPhotos) {
            endpoint = '/api/media/image/upload';
        } else if (isAudio) {
            endpoint = '/api/media/audio/upload';
        } else if (isVideo) {
            endpoint = '/api/media/video/upload';
        } else {
            endpoint = '/api/upload/docs';
        }
        
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
            } else if (isAudio) {
                audioLoaded = false;
                loadAudio();
            } else if (isVideo) {
                videoLoaded = false;
                loadVideo();
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
        
        let buttonText;
        if (isPhotos) {
            buttonText = '📤 Upload Photos';
        } else if (isAudio) {
            buttonText = '📤 Upload Audio';
        } else if (isVideo) {
            buttonText = '📤 Upload Video';
        } else {
            buttonText = '📤 Upload Documents';
        }
        submitBtn.textContent = buttonText;
    }
}

// Populate docs subdirectories for dropdown suggestions
async function populateDocsSubdirectories() {
    try {
        const response = await sharedApp.fetchAPI('/api/subdirectories/docs');
        const subdirectories = response.subdirectories || [];
        
        const datalist = document.getElementById('docsSubdirectoryList');
        if (datalist) {
            // Clear existing options
            datalist.innerHTML = '';
            
            // Add options for each existing subdirectory
            subdirectories.forEach(subdir => {
                const option = document.createElement('option');
                option.value = subdir;
                datalist.appendChild(option);
            });
        }
    } catch (error) {
        console.error('Error loading docs subdirectories:', error);
        // Continue silently - not critical for upload functionality
    }
}

// Populate image galleries for dropdown suggestions
async function populateImageGalleries() {
    try {
        const response = await sharedApp.fetchAPI('/api/media/image/galleries');
        const galleries = response.galleries || [];
        const galleryNames = galleries.map(gallery => gallery.name);
        
        const datalist = document.getElementById('photosSubdirectoryList');
        if (datalist) {
            // Clear existing options
            datalist.innerHTML = '';
            
            // Add options for each existing gallery
            galleryNames.forEach(galleryName => {
                const option = document.createElement('option');
                option.value = galleryName;
                datalist.appendChild(option);
            });
        }
    } catch (error) {
        console.error('Error loading image galleries:', error);
        // Continue silently - not critical for upload functionality
    }
}

// Populate audio galleries for dropdown suggestions
async function populateAudioGalleries() {
    try {
        const response = await sharedApp.fetchAPI('/api/media/audio/galleries');
        const galleries = response.galleries || [];
        const galleryNames = galleries.map(gallery => gallery.name);
        
        const datalist = document.getElementById('audioSubdirectoryList');
        if (datalist) {
            // Clear existing options
            datalist.innerHTML = '';
            
            // Add options for each existing gallery
            galleryNames.forEach(galleryName => {
                const option = document.createElement('option');
                option.value = galleryName;
                datalist.appendChild(option);
            });
        }
    } catch (error) {
        console.error('Error loading audio galleries:', error);
        // Continue silently - not critical for upload functionality
    }
}

// Populate video galleries for dropdown suggestions
async function populateVideoGalleries() {
    try {
        const response = await sharedApp.fetchAPI('/api/media/video/galleries');
        const galleries = response.galleries || [];
        const galleryNames = galleries.map(gallery => gallery.name);
        
        const datalist = document.getElementById('videoSubdirectoryList');
        if (datalist) {
            // Clear existing options
            datalist.innerHTML = '';
            
            // Add options for each existing gallery
            galleryNames.forEach(galleryName => {
                const option = document.createElement('option');
                option.value = galleryName;
                datalist.appendChild(option);
            });
        }
    } catch (error) {
        console.error('Error loading video galleries:', error);
        // Continue silently - not critical for upload functionality
    }
}

// Close modal when clicking outside of it
window.onclick = function(event) {
    const docsModal = document.getElementById('uploadDocsModal');
    const photosModal = document.getElementById('uploadPhotosModal');
    const audioModal = document.getElementById('uploadAudioModal');
    const videoModal = document.getElementById('uploadVideoModal');
    
    if (event.target === docsModal || event.target === photosModal || 
        event.target === audioModal || event.target === videoModal) {
        closeUploadModal();
    }
}

// Expose functions for SPA navigation
window.loadUserInfo = loadUserInfo;
window.loadFriendProfile = loadFriendProfile;
window.loadDocs = loadDocs;
window.initializeProfilePage = initializeProfilePage;