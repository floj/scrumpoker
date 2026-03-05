<template>
  <div class="container-sm">
    <h1 class="text-center">no-fuzz scrum poker</h1>

    <UsernameInput :username="username" @updateUsername="updateUsername"></UsernameInput>

    <CardActions :revealed="revealed" @reveal="revealCards" @reset="resetCards" />

    <PlayerList :players="players" :revealed="revealed" />

    <CardSelector :cards="allowedCards" :selectedCard="selectedCard" @vote="submitVote" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue';
import { useRoute } from 'vue-router';
import CardActions from '@/components/CardActions.vue';
import CardSelector from '@/components/CardSelector.vue';
import PlayerList from '@/components/PlayerList.vue';
import UsernameInput from '@/components/UsernameInput.vue';
import type { Player, Room } from '@/types';
import { roomService } from '@/services/roomService';
import { useLocalStorage } from '@/composables/useLocalStorage';

const route = useRoute();

const roomName = route.params.id as string;

const playerId = useLocalStorage('playerId');
const username = useLocalStorage('username');
const allowedCards = ref<string[]>([]);
const players = ref<Record<string, Player>>({});
const revealed = ref(false);
const selectedCard = ref('');
let eventSource: EventSource | null = null;

function updateRoom(room: Room) {
  allowedCards.value = room.allowedCards;
  players.value = room.players;
  revealed.value = room.revealed;
}

async function joinRoom() {
  const data = await roomService.joinRoom(roomName, username.value, playerId.value);
  console.log('joined room:', data);
  playerId.value = data.playerId;
  username.value = data.username;
  selectedCard.value = data.selectedCard;
  updateRoom(data.room);
}

async function submitVote(card: string) {
  const c = selectedCard.value === card ? '' : card;
  await roomService.submitVote(roomName, playerId.value ?? '', c);
  selectedCard.value = c;
}

function updateUsername(newUsername: string) {
  username.value = newUsername;
  joinRoom();
}

async function revealCards() {
  await roomService.revealCards(roomName);
}

async function resetCards() {
  await roomService.resetCards(roomName);
}

function onSSEMessage(event: MessageEvent) {
  try {
    const message = JSON.parse(event.data);
    switch (message.eventName) {
      case 'room_cleared':
        console.log('Room cleared');
        selectedCard.value = '';
        updateRoom(message.data as Room);
        break;
      case 'room_updated':
        console.log('Room updated:', message.data);
        updateRoom(message.data as Room);
        break;
      default:
        console.warn('Unknown event type:', message.eventName);
    }
  } catch (error) {
    console.error('Error parsing SSE message:', error);
  }
}

async function onSSEError(error: any) {
  console.error('SSE error:', error);
  const room = await roomService.getRoom(roomName);
  updateRoom(room);
}

onMounted(async () => {
  await joinRoom();
  eventSource = roomService.getEventStream(roomName);
  eventSource.onerror = onSSEError;
  eventSource.onmessage = onSSEMessage;
});

onBeforeUnmount(() => {
  if (eventSource) {
    eventSource.close();
  }
});
</script>
