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

function updatePlayerPosition(player) {
    let playerElement = document.getElementById(player.id);
    if (!playerElement) {
        playerElement = createPlayerElement(player);
        document.getElementById('gameArea').appendChild(playerElement);
    }
    playerElement.style.left = player.x + 'px';
    playerElement.style.top = player.y + 'px';
}

function createPlayerElement(player) {
    let playerElement = document.createElement('div');
    playerElement.id = player.id;
    playerElement.className = 'player';
    playerElement.style.backgroundColor = player.color;
    return playerElement;
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
        let direction = '';
        switch (event.code) {
            case 'KeyW': // W key
                direction = 'up';
                break;
            case 'KeyS': // S key
                direction = 'down';
                break;
            case 'KeyA': // A key
                direction = 'left';
                break;
            case 'KeyD': // D key
                direction = 'right';
                break;
            case 'Space': // Space key
                direction = 'stop';
                break;
            default:
                return; // Ignore other keys
        }

        // Construct the movePayload object
        const movePayload = {
            id: playerID,
            direction: direction
        };

        // Create the signal message with type 'move'
        const signalMessage = {
            type: 'move',
            content: JSON.stringify(movePayload)
        };

        // Send the signal message to the server
        socket.send(JSON.stringify(signalMessage));
    }
});

// Function to initialize gamepad events
function setupGamepad() {
    window.addEventListener("gamepadconnected", function (e) {
        console.log("Gamepad connected at index %d: %s. %d buttons, %d axes.",
            e.gamepad.index, e.gamepad.id,
            e.gamepad.buttons.length, e.gamepad.axes.length);
        setInterval(updateGamepadState, 1000 / 60)
        requestAnimationFrame(updateGamepadState);
    });

    window.addEventListener("gamepaddisconnected", function (e) {
        console.log("Gamepad disconnected from index %d: %s",
            e.gamepad.index, e.gamepad.id);
    });
}

function updateGamepadState() {
    let gamepads = navigator.getGamepads();
    for (let i = 0; i < gamepads.length; i++) {
        let gamepad = gamepads[i];
        if (gamepad) {
            let direction = '';
            if (gamepad.axes[1] < -0.5) {
                direction = 'up';
            } else if (gamepad.axes[1] > 0.5) {
                direction = 'down';
            } else if (gamepad.axes[0] < -0.5) {
                direction = 'left';
            } else if (gamepad.axes[0] > 0.5) {
                direction = 'right';
            }

            if (direction) {
                // Construct the movePayload object
                const movePayload = {
                    id: playerID,
                    direction: direction
                };

                // Create the signal message with type 'move'
                const signalMessage = {
                    type: 'move',
                    content: JSON.stringify(movePayload)
                };

                // Send the signal message to the server
                socket.send(JSON.stringify(signalMessage));
            }
        }
    }
}


setupGamepad(); // Initialize gamepad processing
connectToWebSocket();