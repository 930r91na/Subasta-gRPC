let currentUser = '';
let products = {};

// Connect to gRPC-Web proxy
const client = new AuctionServiceClient('http://localhost:8080');

function registerUser() {
    const username = document.getElementById('username').value;
    const request = new RegisterUserRequest();
    request.setName(username);
    
    client.registerUser(request, {}, (err, response) => {
        if (response.getSuccess()) {
            currentUser = username;
            document.getElementById('userStatus').textContent = '✓ Logged in as ' + username;
            loadCatalog();
            startBidStream();
        }
    });
}

function loadCatalog() {
    const request = new GetCatalogRequest();
    client.getCatalog(request, {}, (err, response) => {
        const productList = response.getProductsList();
        displayProducts(productList);
    });
}

function displayProducts(productList) {
    const container = document.getElementById('products');
    container.innerHTML = '';
    
    productList.forEach(product => {
        const div = document.createElement('div');
        div.className = 'product';
        div.innerHTML = `
            <h3>${product.getProduct()}</h3>
            <p>Seller: ${product.getSeller()}</p>
            <p class="price">Current: $${product.getCurrentPrice().toFixed(2)}</p>
            <input type="number" id="bid-${product.getProduct()}" 
                   placeholder="Your bid" min="${product.getCurrentPrice() + 1}">
            <button class="bid-button" onclick="placeBid('${product.getProduct()}')">
                Place Bid
            </button>
        `;
        container.appendChild(div);
    });
}

function placeBid(productName) {
    const amount = parseFloat(document.getElementById(`bid-${productName}`).value);
    const request = new PlaceBidRequest();
    request.setBuyer(currentUser);
    request.setProduct(productName);
    request.setAmount(amount);
    
    client.placeBid(request, {}, (err, response) => {
        if (response.getSuccess()) {
            alert('✓ Bid accepted!');
            loadCatalog(); // Refresh
        } else {
            alert('✗ ' + response.getMessage());
        }
    });
}

function startBidStream() {
    // Stream real-time bid updates
    const request = new StreamBidRequest();
    const stream = client.streamBidUpdates(request, {});
    
    stream.on('data', (update) => {
        addBidToHistory(update);
        updateProductDisplay(update);
    });
}

function addBidToHistory(update) {
    const history = document.getElementById('bidHistory');
    const entry = document.createElement('div');
    entry.innerHTML = `
        <strong>${update.getBuyer()}</strong> bid 
        <strong>$${update.getAmount()}</strong> on 
        <strong>${update.getProduct()}</strong>
        <small>(${new Date().toLocaleTimeString()})</small>
    `;
    history.insertBefore(entry, history.firstChild);
}
        
// Auto-refresh every 5 seconds as fallback
setInterval(loadCatalog, 5000);
