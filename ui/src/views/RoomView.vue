<template>
  <div class="container-sm">
    <UsernameInput :username="username" @updateUsername="updateUsername"></UsernameInput>

    <CardActions :revealed="revealed" @reveal="revealCards" @reset="resetCards" />

    <PlayerList :players="players" :revealed="revealed" />

    <CardSelector :cards="allowedCards" :selectedCard="selectedCard" @vote="submitVote" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, computed } from 'vue';
import { useRoute } from 'vue-router';
import CardActions from '@/components/CardActions.vue';
import CardSelector from '@/components/CardSelector.vue';
import PlayerList from '@/components/PlayerList.vue';
import UsernameInput from '@/components/UsernameInput.vue';
import type { Player, Room, RoomEventMessage as RoomEventMessage } from '@/types';
import { RoomService } from '@/services/roomService';
import { useLocalStorage } from '@/composables/useLocalStorage';
import { showToast } from '@/utils/toasts';

const roomService = new RoomService();
roomService.errHandler = async (err: any, url: string) => {
  console.error('error calling room api', 'url', url, 'error', err);
  showToast('an error occurred while communicating with the server, maybe try refreshing the page?');
  throw err;
};

const route = useRoute();

const roomName = computed(() => route.params.id as string);

const playerId = ref('');
const username = useLocalStorage('username', '');
const authToken = useLocalStorage(() => `authToken-${roomName.value}`);

const allowedCards = ref<string[]>([]);
const players = ref<Record<string, Player>>({});
const revealed = ref(false);
const selectedCard = ref('');

let websocket: WebSocket | null = null;
let reconnectAttempt = 0;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let intentionalClose = false;

function connectWebSocket() {
  if (websocket) {
    websocket.onmessage = null;
    websocket.onerror = null;
    websocket.onclose = null;
    websocket.onopen = null;
    websocket.close();
  }
  websocket = roomService.getWebSocket(roomName.value);
  websocket.onmessage = onRoomEventMessage;
  websocket.onerror = onRoomEventError;
  websocket.onclose = onRoomEventClose;
  websocket.onopen = () => (reconnectAttempt = 0);
}

function updateRoom(room: Room) {
  allowedCards.value = room.allowedCards;
  players.value = room.players;
  revealed.value = room.revealed;
}

async function joinRoom(user: string): Promise<boolean> {
  try {
    const data = await roomService.joinRoom(roomName.value, user, authToken.value);
    playerId.value = data.playerId;
    username.value = data.username;
    authToken.value = data.authToken;
    selectedCard.value = data.selectedCard;
    return true;
  } catch {
    return false;
  }
}

async function submitVote(card: string) {
  const c = selectedCard.value === card ? '' : card;
  try {
    await roomService.submitVote(roomName.value, c, authToken.value);
    selectedCard.value = c;
  } catch {
    showToast('Failed to submit vote');
  }
}

async function updateUsername(newUsername: string) {
  await joinRoom(newUsername);
}

async function revealCards() {
  try {
    await roomService.revealCards(roomName.value, authToken.value);
  } catch {
    showToast('Failed to reveal cards');
  }
}

async function resetCards() {
  try {
    await roomService.resetCards(roomName.value, authToken.value);
  } catch {
    showToast('Failed to reset cards');
  }
}

function onRoomEventMessage(event: MessageEvent) {
  try {
    const message = JSON.parse(event.data) as RoomEventMessage;
    switch (message.eventName) {
      case 'room_cleared':
        console.log('Room cleared', message.data);
        selectedCard.value = '';
        updateRoom(message.data);
        break;
      case 'room_updated':
        console.log('Room updated:', message.data);
        updateRoom(message.data);
        break;
      default:
        showToast('Received unknown event from the server, maybe try refreshing the page?');
        console.warn('Unknown event type:', (message as { eventName: string }).eventName);
    }
  } catch (error) {
    console.error('Error parsing event:', error, event.data);
  }
}

async function onRoomEventError(error: any) {
  console.error('WebSocket error:', error);
  const room = await roomService.getRoom(roomName.value);
  updateRoom(room);
}

function onRoomEventClose() {
  if (intentionalClose) {
    return;
  }
  // exponential backoff with a max delay of 10 seconds
  const delay = Math.min(1_000 * 2 ** reconnectAttempt, 10_000);
  reconnectAttempt++;
  console.warn(`WebSocket closed, reconnecting in ${delay}ms (attempt ${reconnectAttempt})`);
  showToast('Connection lost. Reconnecting…');
  reconnectTimer = setTimeout(() => connectWebSocket(), delay);
}

onMounted(async () => {
  document.title = `no-fuzz estimates - Room ${roomName.value}`;
  if (await joinRoom(username.value)) {
    connectWebSocket();
  }
});

onBeforeUnmount(() => {
  document.title = 'no-fuzz estimates';
  intentionalClose = true;
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
  }
  if (websocket) {
    websocket.close();
  }
});
</script>
