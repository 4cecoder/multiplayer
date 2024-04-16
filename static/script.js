const port = 8080;
// const siteURL = 'http://isdlife.com';
const siteURL = 'http://localhost';
let socket;
let playerID = null;

function connectToWebSocket() {
    socket = new WebSocket('ws://' + siteURL.replace('http://', '') + ':' + port + '/ws');
    console.log('WebSocket connection opened:', socket);

    // Listen for messages
    socket.onmessage = function (event) {
        const instruction = JSON.parse(event.data);
        handleRenderInstruction(instruction);
    };

    // Listen for connection opening
    socket.addEventListener('open', function (event) {
        console.log('WebSocket connection opened:', event);
    });

    // Listen for errors
    socket.addEventListener('error', function (event) {
        console.error('WebSocket error observed:', event);
    });

    // Listen for connection closing
    socket.addEventListener('close', function (event) {
        console.log('WebSocket connection closed:', event);
        // Attempt to reconnect after a short delay
        setTimeout(connectToWebSocket, 5000);
    });
}

function handleRenderInstruction(instruction) {
    switch (instruction.type) {
        case 'updatePlayer':
            updatePlayerPosition(instruction.payload);
            updatePlayerTrail(instruction.payload);
            playerID = instruction.payload.id;
            break;
        case 'captureTerritory':
            updateTerritory(instruction.payload);
            break;
        case 'removePlayer':
            removePlayer(instruction.payload);
            break;
        case 'newPlayer':
            createPlayerElement(instruction.payload);
            break;
    }
}

function createPlayerElement(player) {
    let playerElement = document.createElement('div');
    playerElement.id = player.id;
    playerElement.className = 'player';
    playerElement.style.backgroundColor = player.color;
    document.getElementById('gameArea').appendChild(playerElement);
}

function updatePlayerPosition(player) {
    let playerElement = document.getElementById(player.id);
    if (!playerElement) {
        playerElement = createPlayerElement(player);
        document.getElementById('gameArea').appendChild(playerElement);
    }
    playerElement.style.left = player.x + 'px';
    playerElement.style.top = player.y + 'px';
}

function updatePlayerTrail(player) {
    // Assuming each player has a 'trail' element to display their path
    let trailElement = document.getElementById(player.id + '-trail');
    if (!trailElement) {
        trailElement = document.createElement('div');
        trailElement.id = player.id + '-trail';
        document.getElementById('gameArea').appendChild(trailElement);
    }
    // Add new point to the trail
    let pointElement = document.createElement('div');
    pointElement.className = 'trail-point';
    pointElement.style.left = player.x + 'px';
    pointElement.style.top = player.y + 'px';
    trailElement.appendChild(pointElement);
}

function updateTerritory(player) {
    player.landCapture.forEach((row, y) => {
        row.forEach((captured, x) => {
            if (captured) {
                let cellId = `cell-${x}-${y}`;
                let cell = document.getElementById(cellId);
                if (!cell) {
                    cell = document.createElement('div');
                    cell.id = cellId;
                    cell.className = 'territory-cell';
                    document.getElementById('gameArea').appendChild(cell);
                }
                cell.style.backgroundColor = player.color;
            }
        });
    });
}

function removePlayer(player) {
    let playerElement = document.getElementById(player.id);
    if (playerElement) {
        playerElement.parentNode.removeChild(playerElement);
    }
    let trailElement = document.getElementById(player.id + '-trail');
    if (trailElement) {
        trailElement.parentNode.removeChild(trailElement);
    }
}



document.addEventListener('keydown', function (event) {
    if (playerID !== null) {
        let message = '';
        switch (event.key) {
            case 'ArrowUp':
            case 'w':
                message = 'up';
                break;
            case 'ArrowDown':
            case 's':
                message = 'down';
                break;
            case 'ArrowLeft':
            case 'a':
                message = 'left';
                break;
            case 'ArrowRight':
            case 'd':
                message = 'right';
                break;
            case ' ':
                message = 'stop';
                break;
        }
        if (message !== '') {
            socket.send(message);
        }
    }
});

connectToWebSocket();