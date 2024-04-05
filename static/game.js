// static/game.js
const port = 8080;
const lanIP = "192.168.1.120";

const socket = new WebSocket(`ws://${lanIP}:${port}/ws`);
// const socket = new WebSocket(`ws:// ${window.location.hostname}:${port}/ws`);
const pauseButton = document.getElementById('pauseButton');
const pauseMenu = document.getElementById('pauseMenu');
const resumeButton = document.getElementById('resumeButton');
const playerNameInput = document.getElementById('playerName');
const playerColorInput = document.getElementById('playerColor');

pauseButton.addEventListener('click', () => {
    pauseMenu.style.display = 'block';
});

resumeButton.addEventListener('click', () => {
    const playerName = playerNameInput.value.trim();
    const playerColor = playerColorInput.value;

    socket.send(JSON.stringify({
        type: 'updatePlayer',
        name: playerName,
        color: playerColor
    }));

    pauseMenu.style.display = 'none';
});

socket.onmessage = function (event) {
    const gameState = JSON.parse(event.data);
    console.log("Parsed game state:", gameState);

    const gameArea = document.getElementById("gameArea");
    gameArea.innerHTML = "";

    for (const player of gameState.players) {
        const playerElement = document.createElement("div");
        playerElement.id = player.id;
        playerElement.classList.add("player");
        playerElement.style.backgroundColor = player.color;
        createGlowingEffect(playerElement);

        const playerNameElement = document.createElement("div");
        playerNameElement.classList.add("player-name");
        playerNameElement.textContent = player.name;
        playerElement.appendChild(playerNameElement);

        playerElement.style.left = player.x + "px";
        playerElement.style.top = player.y + "px";
        gameArea.appendChild(playerElement);
    }
};

fetch('https://api.ipify.org?format=json')
    .then(response => response.json())
    .then(data => {
        const ipElement = document.createElement('p');
        ipElement.textContent = `Join LAN Play at IP: 192.168.1.120:8080`;
        document.body.appendChild(ipElement);
    })
    .catch(console.error);

document.addEventListener("keydown", function (event) {
    let message = "";
    switch (event.key) {
        case "ArrowUp":
        case "w":
            message = "up";
            break;
        case "ArrowDown":
        case "s":
            message = "down";
            break;
        case "ArrowLeft":
        case "a":
            message = "left";
            break;
        case "ArrowRight":
        case "d":
            message = "right";
            break;
        case "Escape":
            if (pauseMenu.style.display === 'block') {
                pauseMenu.style.display = 'none';
            } else {
                pauseMenu.style.display = 'block';
            }
            break;
    }
    if (message !== "") {
        socket.send(message);
        console.log("Sent message:", message);
    }
});


function createGlowingEffect(playerElement) {
    const glowEffect = document.createElement('div');
    glowEffect.style.position = 'absolute';
    glowEffect.style.width = '100%';
    glowEffect.style.height = '100%';
    glowEffect.style.borderRadius = '50%';
    glowEffect.style.opacity = '0.7';
    glowEffect.style.background = 'radial-gradient(circle, inherit, transparent 70%)';
    glowEffect.style.zIndex = '-1';
    playerElement.appendChild(glowEffect);
}