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

// Load friend profile and notes
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

        // Load friend's notes
        await loadFriendNotes(peerID);

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

// Load friend's notes via P2P
async function loadFriendNotes(peerID) {
    try {
        sharedApp.showStatus('notesStatus', 'Loading notes via P2P...', false);
        
        const data = await sharedApp.fetchAPI(`/api/peer-notes/${peerID}`);
        
        displayFriendNotes(data.notes || []);
        sharedApp.hideStatus('notesStatus');
    } catch (error) {
        console.error('Error loading friend notes:', error);
        sharedApp.showStatus('notesStatus', 'Error loading notes: ' + error.message, true);
        displayFriendNotesEmptyState('Failed to load notes from friend');
    }
}

// Display friend's notes
function displayFriendNotes(notes) {
    const notesContent = document.getElementById('notesContent');
    
    if (notes.length === 0) {
        displayFriendNotesEmptyState('No notes found');
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
                <button class="read-more-btn" onclick="openFriendNote('${currentFriend.peer_id}', '${sharedApp.escapeHtml(note.filename)}')">
                    Read more
                </button>
            </div>
        `;

        notesGrid.appendChild(noteCard);
    });

    notesContent.innerHTML = '';
    notesContent.appendChild(notesGrid);
}

// Display empty state for friend notes
function displayFriendNotesEmptyState(message) {
    const notesContent = document.getElementById('notesContent');
    notesContent.innerHTML = `
        <div class="empty-state">
            <div class="empty-state-icon">📝</div>
            <div>${message}</div>
            <div class="create-note-hint">
                📡 Notes are requested directly from your friend via P2P connection
            </div>
        </div>
    `;
}

// Open a specific friend note
async function openFriendNote(peerID, filename) {
    try {
        const note = await sharedApp.fetchAPI(`/api/peer-notes/${peerID}/${encodeURIComponent(filename)}`);
        
        document.getElementById('noteModalTitle').textContent = note.title;
        document.getElementById('noteModalMeta').innerHTML = `
            <strong>From:</strong> ${sharedApp.escapeHtml(currentFriend.peer_name)}<br>
            <strong>Filename:</strong> ${sharedApp.escapeHtml(note.filename)}<br>
            <strong>Modified:</strong> ${new Date(note.modified_at).toLocaleString()}<br>
            <strong>Size:</strong> ${Math.round(note.size / 1024 * 100) / 100} KB
        `;
        document.getElementById('noteModalContent').textContent = note.content;
        
        document.getElementById('noteModal').style.display = 'block';
    } catch (error) {
        console.error('Error loading friend note:', error);
        alert('Error loading note: ' + error.message);
    }
}

// Note modal functions
function closeNoteModal() {
    sharedApp.closeNoteModal();
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