// Global state
let currentUser = '';
let isTyping = false;
let typingTimer = null;
let refreshInterval = null;
let lastCatalogHash = '';

// Configuration
const CONFIG = {
    API_URL: 'http://localhost:8080',
    REFRESH_INTERVAL: 5000,  // 5 seconds
    TYPING_COOLDOWN: 2000,   // 2 seconds after typing stops
    MAX_BID_HISTORY: 10
};

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    initializeEventListeners();
    console.log('Auction system initialized');
});

// Initialize all event listeners
function initializeEventListeners() {
    // Username input - Enter key to register
    const usernameInput = document.getElementById('username');
    if (usernameInput) {
        usernameInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                registerUser();
            }
        });
    }

    // Track typing in all input fields
    document.addEventListener('focusin', handleInputFocus);
    document.addEventListener('input', handleInputChange);
    document.addEventListener('focusout', handleInputBlur);
}

// Handle input focus
function handleInputFocus(e) {
    if (e.target.tagName === 'INPUT' && e.target.type === 'number') {
        isTyping = true;
        updateRefreshStatus('paused');
    }
}

// Handle input changes
function handleInputChange(e) {
    if (e.target.tagName === 'INPUT' && e.target.type === 'number') {
        isTyping = true;
        clearTimeout(typingTimer);
        updateRefreshStatus('paused');
        
        // Reset typing state after cooldown period
        typingTimer = setTimeout(() => {
            isTyping = false;
            updateRefreshStatus('active');
        }, CONFIG.TYPING_COOLDOWN);
    }
}

// Handle input blur
function handleInputBlur(e) {
    if (e.target.tagName === 'INPUT' && e.target.type === 'number') {
        setTimeout(() => {
            if (!document.activeElement || document.activeElement.type !== 'number') {
                isTyping = false;
                updateRefreshStatus('active');
            }
        }, 500);
    }
}

// Update refresh status indicator
function updateRefreshStatus(status) {
    const indicator = document.querySelector('.status-indicator');
    if (indicator) {
        indicator.className = `status-indicator ${status}`;
    }
}

// Register user
async function registerUser() {
    const usernameInput = document.getElementById('username');
    const username = usernameInput.value.trim();
    
    if (!username) {
        showAlert('Please enter your name', 'warning');
        return;
    }
    
    try {
        const response = await fetch(`${CONFIG.API_URL}/auction.AuctionService/RegisterUser`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({name: username})
        });
        
        const data = await response.json();
        if (data.success) {
            currentUser = username;
            document.getElementById('userStatus').textContent = '✓ Logged in as ' + username;
            usernameInput.disabled = true;
            
            await loadCatalog();
            startAutoRefresh();
            
            showAlert(`Welcome, ${username}!`, 'success');
        } else {
            showAlert('Registration failed: ' + data.message, 'error');
        }
    } catch (err) {
        console.error('Error:', err);
        showAlert('Connection error. Make sure the gRPC server is running on port 50051', 'error');
    }
}

// Load catalog from server
async function loadCatalog() {
    if (!currentUser) return;
    
    try {
        const response = await fetch(`${CONFIG.API_URL}/auction.AuctionService/GetCatalog`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({})
        });
        
        const data = await response.json();
        const products = data.products || [];
        
        // Only update if catalog changed (prevents unnecessary DOM updates)
        const catalogHash = JSON.stringify(products);
        if (catalogHash !== lastCatalogHash) {
            displayProducts(products);
            lastCatalogHash = catalogHash;
        }
        
        // Update last refresh time
        updateLastRefreshTime();
        
    } catch (err) {
        console.error('Error loading catalog:', err);
    }
}

// Display products with input preservation
function displayProducts(products) {
    const container = document.getElementById('products');
    
    if (products.length === 0) {
        container.innerHTML = '<div class="empty-state">No products available. Add some products to start bidding!</div>';
        return;
    }
    
    // Save current input values and focus state
    const savedInputs = {};
    const focusedElement = document.activeElement;
    const focusedId = focusedElement ? focusedElement.id : null;
    
    document.querySelectorAll('.product input[type="number"]').forEach(input => {
        savedInputs[input.id] = {
            value: input.value,
            selectionStart: input.selectionStart,
            selectionEnd: input.selectionEnd
        };
    });
    
    // Clear and rebuild
    container.innerHTML = '';
    
    products.forEach(product => {
        const div = document.createElement('div');
        div.className = 'product';
        const inputId = `bid-${product.product}`;
        const savedData = savedInputs[inputId] || { value: '' };
        
        div.innerHTML = `
            <h3>📦 ${escapeHtml(product.product)}</h3>
            <p><strong>Seller:</strong> ${escapeHtml(product.seller)}</p>
            <p><strong>Starting Price:</strong> $${product.initial_price.toFixed(2)}</p>
            <p class="price">💰 Current Bid: $${product.current_price.toFixed(2)}</p>
            <div class="bid-section">
                <input type="number" 
                       id="${inputId}" 
                       placeholder="Enter your bid (min $${(product.current_price + 0.01).toFixed(2)})" 
                       min="${product.current_price + 0.01}"
                       step="0.01"
                       value="${savedData.value}">
                <button class="bid-button" onclick="placeBid('${escapeHtml(product.product)}')">
                    Place Bid
                </button>
            </div>
        `;
        container.appendChild(div);
    });
    
    // Restore focus and cursor position
    if (focusedId && savedInputs[focusedId]) {
        const element = document.getElementById(focusedId);
        if (element) {
            element.focus();
            const saved = savedInputs[focusedId];
            element.setSelectionRange(saved.selectionStart, saved.selectionEnd);
        }
    }
}

// Place a bid
async function placeBid(productName) {
    if (!currentUser) {
        showAlert('Please register first', 'warning');
        return;
    }
    
    const inputId = `bid-${productName}`;
    const amountInput = document.getElementById(inputId);
    const amount = parseFloat(amountInput.value);
    
    if (!amount || amount <= 0) {
        showAlert('Please enter a valid bid amount', 'warning');
        return;
    }
    
    try {
        const response = await fetch(`${CONFIG.API_URL}/auction.AuctionService/PlaceBid`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                buyer: currentUser,
                product: productName,
                amount: amount
            })
        });
        
        const data = await response.json();
        if (data.success) {
            addBidToHistory(currentUser, productName, amount);
            amountInput.value = '';
            
            // Force immediate refresh after successful bid
            await loadCatalog();
            
            showAlert(`Bid accepted! Current price: $${data.current_price.toFixed(2)}`, 'success');
        } else {
            showAlert(data.message, 'error');
        }
    } catch (err) {
        console.error('Error placing bid:', err);
        showAlert('Error placing bid. Please try again.', 'error');
    }
}

// Add bid to history
function addBidToHistory(buyer, product, amount) {
    const history = document.getElementById('bidHistory');
    
    // Remove empty state if present
    const emptyState = history.querySelector('.empty-state');
    if (emptyState) {
        history.innerHTML = '';
    }
    
    const entry = document.createElement('div');
    entry.className = 'bid-entry';
    entry.innerHTML = `
        <strong>${escapeHtml(buyer)}</strong> bid 
        <strong>$${amount.toFixed(2)}</strong> on 
        <strong>${escapeHtml(product)}</strong>
        <small>${new Date().toLocaleTimeString()}</small>
    `;
    history.insertBefore(entry, history.firstChild);
    
    // Keep only last N bids
    while (history.children.length > CONFIG.MAX_BID_HISTORY) {
        history.removeChild(history.lastChild);
    }
}

// Start auto-refresh
function startAutoRefresh() {
    // Clear any existing interval
    if (refreshInterval) {
        clearInterval(refreshInterval);
    }
    
    // Start new interval
    refreshInterval = setInterval(() => {
        // Only refresh if:
        // 1. User is logged in
        // 2. User is not currently typing
        if (currentUser && !isTyping) {
            loadCatalog();
        }
    }, CONFIG.REFRESH_INTERVAL);
    
    updateRefreshStatus('active');
}

// Manual refresh button handler
function manualRefresh() {
    loadCatalog();
    showAlert('Catalog refreshed', 'info');
}

// Update last refresh time display
function updateLastRefreshTime() {
    const lastUpdateElement = document.getElementById('lastUpdate');
    if (lastUpdateElement) {
        const now = new Date();
        lastUpdateElement.textContent = `Last updated: ${now.toLocaleTimeString()}`;
    }
}

// Show alert message
function showAlert(message, type = 'info') {
    // For now, use browser alerts
    // In production, you'd want a nicer toast notification system
    alert(message);
}

// Escape HTML to prevent XSS
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Export functions to global scope for inline onclick handlers
window.registerUser = registerUser;
window.placeBid = placeBid;
window.manualRefresh = manualRefresh;