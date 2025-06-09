// Load avatar images and update display
async function loadAvatarImages() {
    const data = await sharedApp.loadAvatarImages();
    const avatarDisplay = document.getElementById('avatarDisplay');
    sharedApp.updateHeaderAvatar(avatarDisplay);
    return data;
}

async function createDirectory() {
    try {
        const response = await fetch('/api/create', { method: 'POST' });
        const data = await response.json();
        sharedApp.showStatus('directoryStatus', 'Directory created successfully!');
    } catch (error) {
        sharedApp.showStatus('directoryStatus', 'Error creating directory: ' + error.message, true);
    }
}

async function scanDirectory() {
    try {
        const response = await fetch('/api/scan', { method: 'POST' });
        const data = await response.json();
        sharedApp.showStatus('directoryStatus', 'Manual scan completed successfully!');
        getNodeInfo(); // Refresh the info
    } catch (error) {
        sharedApp.showStatus('directoryStatus', 'Error scanning directory: ' + error.message, true);
    }
}

async function getMonitorStatus() {
    try {
        const response = await fetch('/api/monitor');
        const data = await response.json();
        
        if (data.monitoring) {
            const lastScan = data.lastScan ? new Date(data.lastScan).toLocaleTimeString() : 'Never';
            sharedApp.showStatus('monitorStatus', `📡 Auto-monitoring active | Last scan: ${lastScan}`, false);
        } else {
            sharedApp.showStatus('monitorStatus', '❌ Auto-monitoring inactive', true);
        }
    } catch (error) {
        sharedApp.showStatus('monitorStatus', 'Error getting monitor status: ' + error.message, true);
    }
}

async function getConnectedPeers() {
    try {
        const response = await fetch('/api/peers');
        const data = await response.json();
        
        const validatedCount = data.validatedCount || data.count || 0;
        const totalCount = data.totalConnectedCount || 0;
        
        if (totalCount > validatedCount) {
            sharedApp.showStatus('discoveryStatus', `✅ ${validatedCount} app peers | 🔗 ${totalCount} total connections`, false);
        } else {
            sharedApp.showStatus('discoveryStatus', `✅ Connected to ${validatedCount} app peers`, false);
        }
        
        // Show detailed information
        const peerInfo = {
            'Application Peers (Validated)': data.validatedPeers || data.peers || [],
            'Validated Count': validatedCount,
            'Total Connections': totalCount,
            'Filtering': totalCount > validatedCount ? 'Active - Non-app peers filtered out' : 'No non-app peers detected'
        };
        
        sharedApp.showResult('discoveryResult', peerInfo);
    } catch (error) {
        sharedApp.showStatus('discoveryStatus', 'Error getting peers: ' + error.message, true);
    }
}

async function getNodeInfo() {
    try {
        const response = await fetch('/api/info');
        const data = await response.json();
        
        // Display NAT status
        if (data.isPublicNode !== undefined) {
            const natStatusMsg = data.isPublicNode 
                ? '🌐 PUBLIC NODE - Can assist with NAT traversal' 
                : '🏠 NAT\'d NODE - Seeks assistance for connections';
            sharedApp.showStatus('natStatus', natStatusMsg, !data.isPublicNode);
        }
        
        // Create a clean display object
        const displayData = {
            'Node ID': data.node?.id || 'Unknown',
            'Addresses': data.node?.addresses || [],
            'Last Seen': data.node?.lastSeen ? new Date(data.node.lastSeen).toLocaleString() : 'Unknown',
            'NAT Status': data.isPublicNode ? 'Public (Can help others)' : 'Behind NAT (Needs assistance)',
            'Directory Info': data.folderInfo || 'No directory scanned yet',
            'Connected Peers': data.connectedPeerInfo ? Object.keys(data.connectedPeerInfo).length : 0
        };
        
        sharedApp.showResult('nodeInfo', displayData);
    } catch (error) {
        sharedApp.showStatus('nodeInfo', 'Error getting node info: ' + error.message, true);
    }
}


async function getConnectionInfo() {
    try {
        const response = await fetch('/api/connection-info');
        const data = await response.json();
        
        // Create a shareable connection string
        let connectionString = '';
        let publicAddress = '';
        
        if (data.publicAddress && data.port && data.peerId) {
            connectionString = `${data.publicAddress}:${data.port}:${data.peerId}`;
            publicAddress = `${data.publicAddress}:${data.port}`;
        }
        
        // Extract P2P port from local addresses
        let p2pPort = 'Unknown';
        let localIP = 'Unknown';
        if (data.localAddresses && data.localAddresses.length > 0) {
            for (const addr of data.localAddresses) {
                if (addr.includes('/ip4/') && !addr.includes('127.0.0.1')) {
                    const parts = addr.split('/');
                    if (parts.length >= 5) {
                        localIP = parts[2];
                        p2pPort = parts[4];
                        break;
                    }
                }
            }
        }

        const connectionInfo = {
            'Peer ID': data.peerId || 'Unknown',
            'P2P Port': p2pPort + ' (Use this port for connections!)',
            'Local IP': localIP,
            'Public Address': publicAddress || 'Not auto-detected',
            'Connection String': connectionString || `Manual format: YOUR_PUBLIC_IP:${p2pPort}:${data.peerId}`,
            'Share This': connectionString ? 'Copy the connection string above and share with others' : `Replace YOUR_PUBLIC_IP with actual public IP in: YOUR_PUBLIC_IP:${p2pPort}:${data.peerId}`,
            'Local Addresses': data.localAddresses || [],
            'NAT Status': data.isPublicNode ? 'Public (can accept connections)' : 'Behind NAT (needs relay)',
            'Important': 'Use P2P port for connections, NOT the web interface port!'
        };
        
        sharedApp.showStatus('discoveryStatus', 
            data.isPublicNode 
                ? '✅ Connection info ready for sharing' 
                : '⚠️ Node behind NAT - direct connections not possible', 
            !data.isPublicNode);
        sharedApp.showResult('discoveryResult', connectionInfo);
        
    } catch (error) {
        sharedApp.showStatus('discoveryStatus', 'Error getting connection info: ' + error.message, true);
    }
}

async function getDetailedPeerInfo() {
    try {
        const response = await fetch('/api/info');
        const data = await response.json();
        
        if (data.connectedPeerInfo && Object.keys(data.connectedPeerInfo).length > 0) {
            const peerCount = Object.keys(data.connectedPeerInfo).length;
            const publicNode = data.isPublicNode;
            
            sharedApp.showStatus('peerInfoStatus', 
                `📊 ${peerCount} detailed peer record${peerCount === 1 ? '' : 's'} ${publicNode ? '(stored for relay assistance)' : ''}`, 
                false);
            
            // Format peer information for better display
            const formattedPeerInfo = {};
            for (const [peerId, info] of Object.entries(data.connectedPeerInfo)) {
                const shortId = peerId.substring(0, 12) + '...';
                const displayName = info.name && info.name !== '' && info.name !== 'unknown' 
                    ? info.name 
                    : 'Unknown';
                const peerLabel = `${displayName} (${shortId})`;
                
                formattedPeerInfo[peerLabel] = {
                    'Name': displayName,
                    'Full ID': info.id,
                    'Connection Type': info.connection_type || 'unknown',
                    'Addresses': info.addresses || [],
                    'First Seen': new Date(info.first_seen).toLocaleString(),
                    'Last Seen': new Date(info.last_seen).toLocaleString(),
                    'Validated': info.is_validated ? 'Yes' : 'No'
                };
            }
            
            showResult('detailedPeerInfo', formattedPeerInfo);
        } else {
            sharedApp.showStatus('peerInfoStatus', '📭 No detailed peer information available', true);
            document.getElementById('detailedPeerInfo').textContent = '';
        }
    } catch (error) {
        sharedApp.showStatus('peerInfoStatus', 'Error getting detailed peer info: ' + error.message, true);
    }
}

async function getConnectionHistory() {
    try {
        const response = await fetch('/api/connection-history');
        const data = await response.json();
        
        if (data.connections && data.connections.length > 0) {
            sharedApp.showStatus('connectionHistoryStatus', `📚 Found ${data.connections.length} connection record${data.connections.length === 1 ? '' : 's'}`, false);
            
            // Create HTML for connection history with connect buttons
            let historyHtml = '<div style="display: grid; gap: 10px;">';
            
            // Load avatars for all peers in parallel
            const avatarPromises = data.connections.map(conn => sharedApp.getPeerAvatar(conn.peer_id));
            const avatars = await Promise.all(avatarPromises);
            
            data.connections.forEach((conn, index) => {
                const peerId = conn.peer_id;
                const displayName = conn.peer_name && conn.peer_name !== '' && conn.peer_name !== 'unknown' 
                    ? conn.peer_name 
                    : 'Unknown';
                const isCurrentlyConnected = conn.currently_connected || false;
                const lastSeen = new Date(conn.last_connected).toLocaleString();
                const avatarInfo = avatars[index];
                const avatarHtml = sharedApp.createPeerAvatarElement(peerId, avatarInfo, '40px');
                
                historyHtml += `
                    <div style="border: 1px solid #ddd; border-radius: 5px; padding: 10px; background: ${isCurrentlyConnected ? '#d4edda' : '#f8f9fa'};">
                        <div style="display: flex; justify-content: space-between; align-items: center;">
                            <div style="display: flex; align-items: center;">
                                ${avatarHtml}
                                <div>
                                    <strong>${displayName}</strong> (${peerId})
                                    <br>
                                    <small style="color: #666;">
                                        Last connected: ${lastSeen}
                                        ${isCurrentlyConnected ? ' • <span style="color: #155724;">Currently connected</span>' : ''}
                                    </small>
                                </div>
                            </div>
                            <div style="display: flex; gap: 10px; align-items: center;">
                                ${!isCurrentlyConnected ? 
                                    `<button class="button" onclick="reconnectToPeer('${conn.peer_id}', '${conn.address}', '${displayName}')">Reconnect</button>` : 
                                    '<span style="color: #155724; font-weight: bold;">Connected ✓</span>'
                                }
                                <button class="button" onclick="addToFriends('${conn.peer_id}', '${displayName}')">Add to Friends</button>
                            </div>
                        </div>
                    </div>
                `;
            });
            
            historyHtml += '</div>';
            document.getElementById('connectionHistory').innerHTML = historyHtml;
        } else {
            sharedApp.showStatus('connectionHistoryStatus', '📭 No connection history found', true);
            document.getElementById('connectionHistory').innerHTML = '';
        }
    } catch (error) {
        sharedApp.showStatus('connectionHistoryStatus', 'Error getting connection history: ' + error.message, true);
        document.getElementById('connectionHistory').innerHTML = '';
    }
}

async function reconnectToPeer(peerId, address, displayName) {
    try {
        // Extract IP and port from address if it's in multiaddr format
        let ip = '';
        let port = '';
        
        if (address.includes('/ip4/')) {
            const parts = address.split('/');
            if (parts.length >= 5) {
                ip = parts[2];
                port = parts[4];
            }
        } else {
            // Try to parse as IP:PORT format
            const parts = address.split(':');
            if (parts.length >= 2) {
                ip = parts[0];
                port = parts[1];
            }
        }
        
        if (!ip || !port) {
            throw new Error('Could not extract IP and port from address: ' + address);
        }
        
        sharedApp.showStatus('connectionHistoryStatus', `🔄 Reconnecting to ${displayName}...`, false);
        
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
        
        const data = await response.json();
        sharedApp.showStatus('connectionHistoryStatus', `✅ Successfully reconnected to ${displayName}!`, false);
        
        // Refresh the connection history to update status
        setTimeout(() => {
            getConnectionHistory();
            getDetailedPeerInfo();
        }, 1000);
        
    } catch (error) {
        sharedApp.showStatus('connectionHistoryStatus', `❌ Failed to reconnect to ${displayName}: ${error.message}`, true);
    }
}

async function addToFriends(peerID, displayName) {
    try {
        sharedApp.showStatus('connectionHistoryStatus', `👥 Adding ${displayName} to friends...`, false);
        
        const response = await fetch('/api/friends', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ 
                peer_id: peerID,
                peer_name: displayName 
            })
        });
        
        if (!response.ok) {
            throw new Error('Network response was not ok');
        }
        
        const data = await response.json();
        sharedApp.showStatus('connectionHistoryStatus', `✅ ${displayName} added to friends!`, false);
        
    } catch (error) {
        sharedApp.showStatus('connectionHistoryStatus', `❌ Failed to add ${displayName} to friends: ${error.message}`, true);
    }
}

async function getSecondDegreeConnections() {
    try {
        sharedApp.showStatus('secondDegreeStatus', '🔍 Discovering second-degree connections...', false);
        
        const response = await fetch('/api/second-degree-peers');
        const data = await response.json();
        
        if (data.peers && data.peers.length > 0) {
            sharedApp.showStatus('secondDegreeStatus', `🔗 Found ${data.peers.length} second-degree peer${data.peers.length === 1 ? '' : 's'}`, false);
            
            // Create HTML for second-degree peers with connect buttons
            let peersHtml = '<div style="display: grid; gap: 10px;">';
            
            // Load avatars for all peers in parallel
            const avatarPromises = data.peers.map(peer => sharedApp.getPeerAvatar(peer.peer_id));
            const avatars = await Promise.all(avatarPromises);
            
            data.peers.forEach((peer, index) => {
                const shortId = peer.peer_id.substring(0, 12) + '...';
                const displayName = peer.peer_name && peer.peer_name !== '' && peer.peer_name !== 'unknown' 
                    ? peer.peer_name 
                    : 'Unknown';
                const connectionPath = peer.connection_path ? ` (via ${peer.connection_path})` : '';
                const avatarInfo = avatars[index];
                const avatarHtml = sharedApp.createPeerAvatarElement(peer.peer_id, avatarInfo, '40px');
                
                peersHtml += `
                    <div style="border: 1px solid #ddd; border-radius: 5px; padding: 10px; background: #f8f9fa;">
                        <div style="display: flex; justify-content: space-between; align-items: center;">
                            <div style="display: flex; align-items: center;">
                                ${avatarHtml}
                                <div>
                                    <strong>${displayName}</strong> (${shortId})
                                    <br>
                                    <small style="color: #666;">
                                        Connected via: ${peer.via_peer_name || 'Unknown'}${connectionPath}
                                        <br>
                                        Distance: 2 hops away
                                    </small>
                                </div>
                            </div>
                            <div>
                                <button class="button" onclick="connectToSecondDegreePeer('${peer.peer_id}', '${peer.via_peer_id}', '${displayName}')">Connect</button>
                            </div>
                        </div>
                    </div>
                `;
            });
            
            peersHtml += '</div>';
            document.getElementById('secondDegreeConnections').innerHTML = peersHtml;
        } else {
            sharedApp.showStatus('secondDegreeStatus', '📭 No second-degree connections found', true);
            document.getElementById('secondDegreeConnections').innerHTML = '';
        }
    } catch (error) {
        sharedApp.showStatus('secondDegreeStatus', 'Error discovering second-degree connections: ' + error.message, true);
        document.getElementById('secondDegreeConnections').innerHTML = '';
    }
}

async function connectToSecondDegreePeer(targetPeerId, viaPeerId, displayName) {
    try {
        sharedApp.showStatus('secondDegreeStatus', `🔄 Connecting to ${displayName} via hole punching...`, false);
        
        const response = await fetch('/api/connect-second-degree', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ 
                targetPeerId: targetPeerId,
                viaPeerId: viaPeerId
            })
        });
        
        if (!response.ok) {
            throw new Error('Network response was not ok');
        }
        
        const data = await response.json();
        sharedApp.showStatus('secondDegreeStatus', `✅ Successfully connected to ${displayName}!`, false);
        
        // Refresh the lists to update connection status
        setTimeout(() => {
            getSecondDegreeConnections();
            getDetailedPeerInfo();
        }, 1000);
        
    } catch (error) {
        sharedApp.showStatus('secondDegreeStatus', `❌ Failed to connect to ${displayName}: ${error.message}`, true);
    }
}

// Load initial data
window.onload = function() {
    loadAvatarImages();
    getNodeInfo();
    getMonitorStatus();
    getDetailedPeerInfo();
    getConnectionHistory();
    getSecondDegreeConnections();
};