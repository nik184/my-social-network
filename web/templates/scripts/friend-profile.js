let currentFriend = null;

// Load initial data when page loads
document.addEventListener('DOMContentLoaded', function() {
    loadFriendProfile();
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
        document.getElementById('friendName').textContent = 'Error';
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
        document.getElementById('friendName').textContent = friendInfo.peer_name;
        document.getElementById('friendId').textContent = `Peer ID: ${peerID}`;

        // Load friend's avatar
        const avatarInfo = await sharedApp.getPeerAvatar(peerID);
        const friendAvatar = document.getElementById('friendAvatar');
        if (avatarInfo && avatarInfo.hasAvatar) {
            friendAvatar.innerHTML = `<img src="${avatarInfo.url}" alt="Avatar" />`;
        } else {
            friendAvatar.innerHTML = '👤';
        }

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
        
        document.getElementById('docModal').style.display = 'block';
    } catch (error) {
        console.error('Error loading friend doc:', error);
        alert('Error loading doc: ' + error.message);
    }
}

// Doc modal functions
function closeDocModal() {
    sharedApp.closeDocModal();
}

// Go back to friends page
function goBack() {
    window.location.href = '/friends';
}

// Update page title with friend name
function updatePageTitle(friendName) {
    document.title = `${friendName}'s Profile - My Social Network`;
}

// Update title when friend is loaded
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
        // Disable button and show loading state
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
        
        // Show download results
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
        
        // Show details in console for debugging
        console.log('Download result:', result);
        
        // Hide status after 5 seconds if successful
        if (errors.length === 0) {
            setTimeout(() => {
                sharedApp.hideStatus('downloadStatus');
            }, 5000);
        }
        
    } catch (error) {
        console.error('Error downloading content:', error);
        sharedApp.showStatus('downloadStatus', 'Error downloading content: ' + error.message, true);
    } finally {
        // Re-enable button
        downloadBtn.disabled = false;
        downloadBtn.textContent = '📥 Download All Content';
    }
}