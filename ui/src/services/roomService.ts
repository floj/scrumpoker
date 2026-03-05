import type { Room } from '@/types';
import { apiBaseURL } from '@/utils/baseurl';

const defaultErrHandler = (err: any, url: string, options: RequestInit) => {
  console.error('Error dispatching request:', err, 'URL:', url, 'Options:', options);
  throw err;
};

const defaultStatusHandler = (status: number, url: string, options: RequestInit) => {
  throw new Error(`Request failed with status ${status}`);
};

type CreateRoomResponse = {
  name: string;
};

type JoinRoomResponse = {
  playerId: string;
  selectedCard: string;
  room: Room;
  username: string;
};

class RoomService {
  private baseUrl: string;

  errHandler: (err: any, url: string, options: RequestInit) => Promise<any> = defaultErrHandler;
  statusHandler: (status: number, url: string, options: RequestInit) => Promise<any> = defaultStatusHandler;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  async createNewRoom(): Promise<CreateRoomResponse> {
    const resp = await this.dispatchRequest('/rooms/', 'POST');
    return (await resp.json()) as CreateRoomResponse;
  }

  async joinRoom(roomId: string, username: string, playerId?: string): Promise<JoinRoomResponse> {
    const resp = await this.dispatchRequest(`/rooms/${roomId}/join`, 'POST', { username, playerId });
    return (await resp.json()) as JoinRoomResponse;
  }

  async revealCards(roomId: string): Promise<void> {
    await this.dispatchRequest(`/rooms/${roomId}/reveal`, 'POST');
  }

  async resetCards(roomId: string): Promise<void> {
    await this.dispatchRequest(`/rooms/${roomId}/reset`, 'POST');
  }

  async submitVote(roomId: string, playerId: string, card: string): Promise<void> {
    await this.dispatchRequest(`/rooms/${roomId}/vote`, 'POST', { playerId, card });
  }

  async getRoom(roomId: string): Promise<Room> {
    const resp = await this.dispatchRequest(`/rooms/${roomId}/`, 'GET');
    return (await resp.json()) as Room;
  }

  getEventStream(roomId: string): EventSource {
    const params = new URLSearchParams({ stream: roomId });
    const eventSource = new EventSource(`${this.baseUrl}/rooms/sse?${params.toString()}`);

    return eventSource;
  }

  private async dispatchRequest(url: string, method: string, body?: any, headers?: HeadersInit): Promise<Response> {
    const options: RequestInit = {
      method,
      headers: {
        'Content-Type': 'application/json',
        'X-Player-ID': localStorage.getItem('playerId') || '',
        ...headers,
      },
      body: body ? JSON.stringify(body) : undefined,
    };

    try {
      const resp = await fetch(`${this.baseUrl}${url}`, options);
      if (!resp.ok) {
        return await this.statusHandler(resp.status, url, options);
      }
      return resp;
    } catch (error) {
      return await this.errHandler(error, url, options);
    }
  }
}

const roomService = new RoomService(apiBaseURL());

export { roomService };
