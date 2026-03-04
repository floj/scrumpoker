<template>
  <div class="container-sm">
    <h1 class="text-center">no-fuss scrum poker</h1>
    <div class="input-group mb-3">
      <input
        v-model="newUsername"
        type="text"
        class="form-control"
        placeholder="Username"
        aria-label="Username"
        @keydown.enter="updateUsername"
      />
      <button class="btn btn-primary" type="button" @click="updateUsername">Change</button>
    </div>

    <div class="text-center">
      <button v-if="!revealed" type="button" class="btn btn-lg btn-success w-50" @click="revealCards">Reveal</button>
      <button v-if="revealed" type="button" class="btn btn-lg btn-danger w-50" @click="clearCards">Clear</button>
    </div>

    <table class="table table-striped fs-3">
      <thead>
        <tr>
          <th scope="col">Player</th>
          <th scope="col">Card</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(player, id) in playersList" :key="id">
          <td>{{ player.name }}</td>
          <td>
            <span v-if="revealed">{{ player.card }}</span>
            <i v-if="!revealed && player.voted" class="bi bi-check"></i>
          </td>
        </tr>
      </tbody>
    </table>

    <div class="vcard-container">
      <div
        v-for="card in cardTitles"
        :key="card"
        class="vcard"
        :class="{ 'vcard-selected': selectedCard === card }"
        @click="submitVote(card)"
      >
        <div>{{ card }}</div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { apiBaseURL } from '@/utils/baseurl';
import { findSingle } from '@/utils/emojis';

type RoomData = {
  playerId: string;
  allowedCards: Array<string>;
  players: Record<string, { name: string; card: string; voted: boolean }>;
  selectedCard: string | null;
  revealed: boolean;
};

function loadConfig() {
  const username = localStorage.getItem('username') ?? '';
  const playerId = localStorage.getItem('playerId') ?? null;
  return { username, playerId };
}

export default {
  name: 'RoomView',
  data() {
    const conf = loadConfig();
    return {
      playerId: conf.playerId,
      username: conf.username,
      newUsername: conf.username,
      roomName: this.$route.params.id as string,
      allowedCards: [] as Array<string>,
      players: {} as Record<string, { name: string; card: string; voted: boolean }>,
      revealed: false,
      selectedCard: null as string | null,
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
    const params = new URLSearchParams({ stream: this.roomName });
    const source = new EventSource(`${apiBaseURL()}/api/v1/rooms/sse?${params.toString()}`);
    source.onmessage = (event) => {
      const message = JSON.parse(event.data);
      switch (message.eventName) {
        case 'room_cleared':
          console.log('Room cleared');
          this.selectedCard = null;
        // fallthrough
        case 'room_updated':
          console.log('Room updated:', message.data);
          this.updateRoom(message.data as RoomData);
          break;
        default:
          console.warn('Unknown event type:', message.eventName);
      }
    };
  },
  computed: {
    playersList() {
      const players = [] as Array<{ name: string; card: string; voted: boolean }>;
      for (const player of Object.values(this.players)) {
        players.push(player);
      }
      if (this.revealed) {
        players.sort((a, b) => {
          if (a.card === b.card) {
            return 0;
          }
          if (a.card === '?' || a.card === ':coffee:') {
            return 1;
          }
          if (b.card === '?' || b.card === ':coffee:') {
            return -1;
          }
          return parseInt(a.card) - parseInt(b.card);
        });
      }
      return players;
    },
    cardTitles() {
      const cards = [] as Array<string>;
      for (const card of this.allowedCards) {
        if (card.startsWith(':') && card.endsWith(':')) {
          cards.push(findSingle(card) ?? card);
        } else {
          cards.push(card);
        }
      }
      return cards;
    },
  },
  methods: {
    async revealCards() {
      const resp = await fetch(`${apiBaseURL()}/api/v1/rooms/${this.roomName}/reveal`, {
        method: 'POST',
      });
      if (!resp.ok) {
        console.error('Failed to reveal cards');
        return;
      }
    },
    async clearCards() {
      const resp = await fetch(`${apiBaseURL()}/api/v1/rooms/${this.roomName}/`, {
        method: 'DELETE',
      });
      if (!resp.ok) {
        console.error('Failed to clear cards');
        return;
      }
    },
    async submitVote(card: string) {
      const resp = await fetch(`${apiBaseURL()}/api/v1/rooms/${this.roomName}/vote`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          playerId: this.playerId,
          card,
        }),
      });
      if (!resp.ok) {
        console.error('Failed to submit vote');
        return;
      }
      this.selectedCard = card;
    },
    updateRoom(room: RoomData) {
      this.allowedCards = room.allowedCards;
      this.players = room.players;
      this.revealed = room.revealed;
    },
    async joinRoom() {
      const resp = await fetch(`${apiBaseURL()}/api/v1/rooms/${this.roomName}/join`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          username: this.username,
          playerId: this.playerId,
        }),
      });
      if (!resp.ok) {
        console.error('Failed to fetch room data');
        return;
      }
      if (resp.status === 404) {
        console.warn('No content returned from join endpoint');
        return;
      }
      const data = await resp.json();
      console.log('Room data:', data);
      const { playerId, selectedCard, room } = data;
      this.playerId = playerId;
      this.selectedCard = selectedCard;
      this.updateRoom(room);
    },
    updateUsername() {
      this.username = this.newUsername;
      this.joinRoom();
    },
  },
};
</script>

<style scoped>
.vcard {
  width: 5rem;
  height: 7rem;
  display: flex;
  justify-content: center;
  align-items: center;
  font-size: 2rem;
  border: 2px solid var(--bs-primary);
  border-radius: var(--bs-border-radius-lg);
  padding: 1rem;
  cursor: pointer;
  user-select: none;
  transition: transform 0.1s ease-in-out;
}

.vcard:hover {
  transform: translateY(-0.5rem);
}
.vcard-selected {
  background-color: var(--bs-primary);
  color: white;
  border-color: var(--bs-primary);
  transform: translateY(-0.5rem);
}
.vcard-container {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  justify-content: center;
  align-content: flex-start;
  gap: 1rem;
}
</style>
