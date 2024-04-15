//const lanIP = "192.168.1.120"
const port = 8089;
const socket = new WebSocket('ws://' + window.location.host + '/ws');
//const socket = new WebSocket('ws://' + lanIP + ':' + port + '/ws');

socket.onmessage = function (event) {
    const instruction = JSON.parse(event.data);
    handleRenderInstruction(instruction);
};

function handleRenderInstruction(instruction) {
    switch (instruction.type) {
        case 'updatePlayer':
            const player = instruction.payload;
            let playerElement = document.getElementById(player.id);
            if (!playerElement) {
                playerElement = document.createElement('div');
                playerElement.id = player.id;
                playerElement.classList.add('player');
                playerElement.style.backgroundColor = player.color;
                document.getElementById('gameArea').appendChild(playerElement);
            }
            playerElement.style.left = player.x + 'px';
            playerElement.style.top = player.y + 'px';

            let playerNameElement = playerElement.querySelector('.player-name');
            if (!playerNameElement) {
                playerNameElement = document.createElement('div');
                playerNameElement.classList.add('player-name');
                playerElement.appendChild(playerNameElement);
            }
            playerNameElement.textContent = player.name;

            // Render the player's trail only if the player is outside their territory
            const trailElements = document.querySelectorAll('.player-trail');
            trailElements.forEach(trailElement => trailElement.remove());

            if (!instruction.payload.landCapture[Math.floor(player.y / 20)][Math.floor(player.x / 20)]) {
                instruction.payload.playerTrail.forEach(point => {
                    const trailElement = document.createElement('div');
                    trailElement.classList.add('player-trail');
                    trailElement.style.backgroundColor = player.color;
                    trailElement.style.left = point.x + 'px';
                    trailElement.style.top = point.y + 'px';
                    document.getElementById('gameArea').appendChild(trailElement);
                });
            }

            // Render the player's territory
            for (let i = 0; i < instruction.payload.landCapture.length; i++) {
                for (let j = 0; j < instruction.payload.landCapture[i].length; j++) {
                    if (instruction.payload.landCapture[i][j]) {
                        const territoryElement = document.createElement('div');
                        territoryElement.classList.add('player-territory');
                        territoryElement.style.backgroundColor = player.color;
                        territoryElement.style.left = j * 20 + 'px';
                        territoryElement.style.top = i * 20 + 'px';
                        document.getElementById('gameArea').appendChild(territoryElement);

                        // Check if the current block is the center of the square
                        if (i === Math.floor((player.y - 10) / 20) && j === Math.floor((player.x - 10) / 20)) {
                            territoryElement.style.opacity = '1';
                        }
                    }
                }
            }

            // Render the player's starting territory
            for (let i = 0; i < player.startingLand.length; i++) {
                for (let j = 0; j < player.startingLand[i].length; j++) {
                    if (player.startingLand[i][j]) {
                        const startingTerritoryElement = document.createElement('div');
                        startingTerritoryElement.classList.add('starting-territory');
                        startingTerritoryElement.style.backgroundColor = player.color;
                        startingTerritoryElement.style.left = (player.startingPosition.x - 20 + j * 20) + 'px';
                        startingTerritoryElement.style.top = (player.startingPosition.y - 20 + i * 20) + 'px';
                        document.getElementById('gameArea').appendChild(startingTerritoryElement);
                    }
                }
            }

            break;

        case 'removePlayer':
            const removePlayerId = instruction.payload.id;
            const removePlayerElement = document.getElementById(removePlayerId);
            if (removePlayerElement) {
                removePlayerElement.remove();
            }
            break;
    }
}

let isMoving = false;

document.addEventListener('keydown', function (event) {
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
    }
    if (message !== '') {
        socket.send(message);
    }
});

document.addEventListener('keyup', function (event) {
    if (event.key === ' ') {
        socket.send('stop');
    }
});