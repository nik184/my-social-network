// Function to initialize the friends page
function initializeFriendsPage() {
    // Check if the friends page elements exist before initializing
    const friendsContent = document.getElementById('friendsContent');
    const friendsStatus = document.getElementById('friendsStatus');

    if (!friendsContent) {
        // Friends page elements not found, try again later
        setTimeout(initializeFriendsPage, 50);
        return;
    }

    loadFriends();
    // Show connection status initially
    if (typeof sharedApp !== 'undefined') {
        sharedApp.showStatus('connectionStatus', '', false);
        sharedApp.hideStatus('connectionStatus');
    }
}

// Load initial data when page loads (for direct page access only)
document.addEventListener('DOMContentLoaded', function() {
    // Always run for friends page since it doesn't have complex SPA state
    initializeFriendsPage();
});

// Make functions globally accessible for SPA navigation
window.loadFriends = loadFriends;
window.initializeFriendsPage = initializeFriendsPage;

// Load friends from the server
async function loadFriends() {
    try {
        sharedApp.showStatus('friendsStatus', 'Loading friends...', false);
        
        const data = await sharedApp.fetchAPI('/api/friends');
        
        displayFriends(data.friends || []);
        sharedApp.hideStatus('friendsStatus');
    } catch (error) {
        console.error('Error loading friends:', error);
        sharedApp.showStatus('friendsStatus', 'Error loading friends: ' + error.message, true);
        displayEmptyState('Failed to load friends');
    }
}

// Display friends in the list
function displayFriends(friends) {
    const friendsContent = document.getElementById('friendsContent');
    
    if (friends.length === 0) {
        displayEmptyState('No friends found');
        return;
    }

    let friendsHtml = '<div style="display: grid; gap: 15px;">';

    friends.forEach(async (friend, index) => {
        const addedDate = new Date(friend.added_at).toLocaleDateString();
        const lastSeenText = friend.last_seen 
            ? new Date(friend.last_seen).toLocaleString()
            : 'Never';
        const onlineStatus = friend.is_online ? 'Online' : 'Offline';
        const statusColor = friend.is_online ? '#155724' : '#721c24';

        // Load friend's avatar
        const avatarInfo = await sharedApp.getPeerAvatar(friend.peer_id);
        const avatarHtml = sharedApp.createPeerAvatarElement(friend.peer_id, avatarInfo, '50px');

        const friendCard = `
            <div style="border: 1px solid #ddd; border-radius: 5px; padding: 15px; background: #f8f9fa; cursor: pointer;" onclick="viewFriendProfile('${friend.peer_id}')">
                <div style="display: flex; justify-content: space-between; align-items: center;">
                    <div style="display: flex; align-items: center;">
                        ${avatarHtml}
                        <div style="margin-left: 15px;">
                            <strong style="font-size: 18px;">${sharedApp.escapeHtml(friend.peer_name)}</strong>
                            <br>
                            <small style="color: #666;">
                                Added: ${addedDate} • Last seen: ${lastSeenText}
                                <br>
                                Status: <span style="color: ${statusColor}; font-weight: bold;">${onlineStatus}</span>
                            </small>
                        </div>
                    </div>
                    <div style="display: flex; gap: 10px;">
                        <button class="button" onclick="event.stopPropagation(); viewFriendProfile('${friend.peer_id}')">View Profile</button>
                        <button class="button" onclick="event.stopPropagation(); removeFriend('${friend.peer_id}', '${sharedApp.escapeHtml(friend.peer_name)}')" style="background-color: #dc3545;">Remove</button>
                    </div>
                </div>
            </div>
        `;

        // Add the card to the container
        if (index === 0) {
            friendsHtml = friendCard;
            friendsContent.innerHTML = friendsHtml;
        } else {
            friendsContent.innerHTML += friendCard;
        }
    });
}

// Display empty state
function displayEmptyState(message) {
    const friendsContent = document.getElementById('friendsContent');
    friendsContent.innerHTML = `
        <div class="empty-state">
            <div class="empty-state-icon">👥</div>
            <div>${message}</div>
            <div class="create-doc-hint">
                💡 Add friends from the Network page's Connection History
            </div>
        </div>
    `;
}

// Remove a friend
async function removeFriend(peerID, friendName) {
    if (!confirm(`Are you sure you want to remove ${friendName} from your friends?`)) {
        return;
    }

    try {
        sharedApp.showStatus('friendsStatus', `Removing ${friendName}...`, false);
        
        const response = await fetch(`/api/friends/${encodeURIComponent(peerID)}`, {
            method: 'DELETE'
        });
        
        if (!response.ok) {
            throw new Error('Failed to remove friend');
        }
        
        sharedApp.showStatus('friendsStatus', `✅ ${friendName} removed from friends`, false);
        
        // Reload friends list
        setTimeout(() => {
            loadFriends();
        }, 1000);
        
    } catch (error) {
        console.error('Error removing friend:', error);
        sharedApp.showStatus('friendsStatus', `❌ Failed to remove ${friendName}: ${error.message}`, true);
    }
}

// Navigate to friend profile page using SPA navigation
function viewFriendProfile(peerID) {
    if (typeof sharedApp !== 'undefined' && sharedApp.loadPage) {
        sharedApp.loadPage(`/friend-profile?peer_id=${encodeURIComponent(peerID)}`);
    } else {
        window.location.href = `/friend-profile?peer_id=${encodeURIComponent(peerID)}`;
    }
}

// Connect to a peer using connection string and add to friends
async function connectAndAddFriend() {
    const connectionString = document.getElementById('connectionStringInput').value;
    
    if (!connectionString) {
        sharedApp.showStatus('connectionStatus', 'Please enter a connection string', true);
        return;
    }

    // Parse connection string (format: IP:PORT:PEER_ID)
    const parts = connectionString.split(':');
    if (parts.length < 3) {
        sharedApp.showStatus('connectionStatus', 'Invalid connection string format. Use IP:PORT:PEER_ID', true);
        return;
    }

    const ip = parts[0];
    const port = parts[1];
    const peerId = parts.slice(2).join(':'); // In case peer ID contains colons

    try {
        sharedApp.showStatus('connectionStatus', 'Connecting to peer...', false);
        
        const response = await fetch('/api/connect-ip', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ 
                ip: ip,
                port: parseInt(port),
                peerId: peerId 
            })
        });
        
        if (!response.ok) {
            throw new Error('Network response was not ok');
        }
        
        const connectionData = await response.json();
        sharedApp.showStatus('connectionStatus', 'Successfully connected! Adding to friends...', false);
        
        // Now add the peer to friends
        const friendResponse = await fetch('/api/friends', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ 
                peer_id: peerId,
                peer_name: connectionData.peer_name || 'Unknown'
            })
        });
        
        if (!friendResponse.ok) {
            throw new Error('Failed to add to friends');
        }
        
        const friendData = await friendResponse.json();
        sharedApp.showStatus('connectionStatus', `✅ Successfully connected and added ${connectionData.peer_name || 'peer'} to friends!`, false);
        
        // Clear the input field
        document.getElementById('connectionStringInput').value = '';
        
        // Reload friends list to show the new friend
        setTimeout(() => {
            loadFriends();
            sharedApp.hideStatus('connectionStatus');
        }, 2000);
        
    } catch (error) {
        sharedApp.showStatus('connectionStatus', 'Error connecting or adding friend: ' + error.message, true);
    }
}