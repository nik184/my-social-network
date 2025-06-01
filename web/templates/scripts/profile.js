let userInfo = null;
let currentAvatarIndex = 0;

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

// Avatar gallery functions - use shared functions
function openAvatarGallery() {
    sharedApp.openAvatarGallery();
}

function closeAvatarGallery() {
    sharedApp.closeAvatarGallery();
}

function previousAvatar() {
    sharedApp.previousAvatar();
}

function nextAvatar() {
    sharedApp.nextAvatar();
}

// Modal functions
function closeNoteModal() {
    sharedApp.closeNoteModal();
}