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
};

const route = useRoute();

const roomName = computed(() => route.params.id as string);

const playerId = ref('');
const username = useLocalStorage('username');
const authToken = useLocalStorage(`authToken-${roomName.value}`);

const allowedCards = ref<string[]>([]);
const players = ref<Record<string, Player>>({});
const revealed = ref(false);
const selectedCard = ref('');

let websocket: WebSocket | null = null;

function updateRoom(room: Room) {
  allowedCards.value = room.allowedCards;
  players.value = room.players;
  revealed.value = room.revealed;
}

async function joinRoom(): Promise<Room | null> {
  try {
    const data = await roomService.joinRoom(roomName.value, username.value, authToken.value);
    playerId.value = data.playerId;
    username.value = data.username;
    authToken.value = data.authToken;
    selectedCard.value = data.selectedCard;
    return data.room;
  } catch {
    showToast('Failed to join room');
    return null;
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
  username.value = newUsername;
  const room = await joinRoom();
  if (room) {
    updateRoom(room);
  }
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
        console.log('Room cleared');
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

onMounted(async () => {
  document.title = `no-fuzz estimates - Room ${roomName.value}`;
  const room = await joinRoom();
  if (room != null) {
    websocket = roomService.getWebSocket(roomName.value);
    websocket.onerror = onRoomEventError;
    websocket.onmessage = onRoomEventMessage;
  }
});

onBeforeUnmount(() => {
  document.title = 'no-fuzz estimates';
  if (websocket) {
    websocket.close();
  }
});
</script>
