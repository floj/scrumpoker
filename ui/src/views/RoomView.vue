<template>
  <div class="container-sm">
    <h1 class="text-center">no-fuzz scrum poker</h1>

    <UsernameInput :username="username" @updateUsername="updateUsername"></UsernameInput>

    <CardActions :revealed="revealed" :roomName="roomName" />

    <PlayerList :players="players" :revealed="revealed" />

    <CardSelector :cards="allowedCards" :selectedCard="selectedCard" @vote="submitVote" />
  </div>
</template>

<script lang="ts">
import CardActions from '@/components/CardActions.vue';
import CardSelector from '@/components/CardSelector.vue';
import PlayerList from '@/components/PlayerList.vue';
import UsernameInput from '@/components/UsernameInput.vue';
import { roomService } from '@/services/roomService';
import type { Player, Room } from '@/types';
import { apiBaseURL } from '@/utils/baseurl';

function loadConfig() {
  const username = localStorage.getItem('username') ?? '';
  const playerId = localStorage.getItem('playerId') ?? '';
  return { username, playerId };
}

const eventSources = new Map<string, EventSource>();

export default {
  name: 'RoomView',
  components: {
    CardActions,
    CardSelector,
    PlayerList,
    UsernameInput,
  },
  data() {
    const conf = loadConfig();
    return {
      playerId: conf.playerId,
      username: conf.username,
      roomName: this.$route.params.id as string,
      allowedCards: [] as Array<string>,
      players: {} as Record<string, Player>,
      revealed: false,
      selectedCard: '',
    };
  },
  watch: {
    playerId(newVal) {
      if (newVal) {
        localStorage.setItem('playerId', newVal);
      } else {
        localStorage.removeItem('playerId');
      }
    },
    username(newVal) {
      if (newVal) {
        localStorage.setItem('username', newVal);
      } else {
        localStorage.removeItem('username');
      }
    },
  },
  async created() {
    await this.joinRoom();
    let eventSource = eventSources.get(this.roomName);
    if (!eventSource) {
      eventSource = roomService.getEventStream(this.roomName);
      eventSources.set(this.roomName, eventSource);
    }
    eventSource.onerror = this.onSSEError;
    eventSource.onmessage = this.onSSEMessage;
  },
  beforeUnmount() {
    const eventSource = eventSources.get(this.roomName);
    if (eventSource) {
      eventSource.close();
      eventSources.delete(this.roomName);
    }
  },
  methods: {
    async joinRoom() {
      const data = await roomService.joinRoom(this.roomName, this.username, this.playerId);
      console.log('joined room:', data);
      const { playerId, selectedCard, room, username } = data;
      this.playerId = playerId;
      this.username = username;
      this.selectedCard = selectedCard;
      this.updateRoom(room);
    },
    async submitVote(card: string) {
      const c = this.selectedCard === card ? '' : card;
      await roomService.submitVote(this.roomName, this.playerId ?? '', c);
      this.selectedCard = c;
    },
    updateRoom(room: Room) {
      this.allowedCards = room.allowedCards;
      this.players = room.players;
      this.revealed = room.revealed;
    },
    updateUsername(newUsername: string) {
      this.username = newUsername;
      this.joinRoom();
    },
    onSSEMessage(event: MessageEvent) {
      const message = JSON.parse(event.data);
      switch (message.eventName) {
        case 'room_cleared':
          console.log('Room cleared');
          this.selectedCard = '';
        // intentional fallthrough
        case 'room_updated':
          console.log('Room updated:', message.data);
          this.updateRoom(message.data as Room);
          break;
        default:
          console.warn('Unknown event type:', message.eventName);
      }
    },
    async onSSEError(error: any) {
      console.error('SSE error:', error);
      const room = await roomService.getRoom(this.roomName);
      this.updateRoom(room);
    },
  },
};
</script>
