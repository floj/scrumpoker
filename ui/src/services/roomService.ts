import type { Room } from '@/types';
import { apiBaseURL } from '@/utils/baseurl';

const defaultErrHandler = async (err: any, url: string, options: RequestInit) => {
  console.error('Error dispatching request:', err, 'URL:', url, 'Options:', options);
  throw err;
};

const defaultStatusHandler = async (status: number, url: string, options: RequestInit) => {
  throw new Error(`Request failed with status ${status}`);
};

type CreateRoomResponse = {
  name: string;
};

type JoinRoomResponse = {
  playerId: string;
  authToken: string;
  username: string;
  selectedCard: string;
};

function withAuthToken(token: string) {
  return (options: RequestInit) => {
    options.headers = Object.assign(options.headers ?? {}, { 'X-Auth-Token': token });
  };
}

class RoomService {
  private baseUrl: string;

  errHandler: (err: any, url: string, options: RequestInit) => Promise<any> = defaultErrHandler;
  statusHandler: (status: number, url: string, options: RequestInit) => Promise<any> = defaultStatusHandler;

  constructor(baseUrl?: string) {
    this.baseUrl = baseUrl ?? apiBaseURL();
  }

  async createNewRoom(): Promise<CreateRoomResponse> {
    const resp = await this.dispatchRequest(['rooms'], 'POST');
    return (await resp.json()) as CreateRoomResponse;
  }

  async joinRoom(roomId: string, username: string, authToken?: string): Promise<JoinRoomResponse> {
    const resp = await this.dispatchRequest([`rooms`, roomId, 'join'], 'POST', { username, authToken });
    return (await resp.json()) as JoinRoomResponse;
  }

  async revealCards(roomId: string, authToken: string): Promise<void> {
    await this.dispatchRequest([`rooms`, roomId, 'reveal'], 'POST', undefined, withAuthToken(authToken));
  }

  async resetCards(roomId: string, authToken: string): Promise<void> {
    await this.dispatchRequest([`rooms`, roomId, 'reset'], 'POST', undefined, withAuthToken(authToken));
  }

  async submitVote(roomId: string, card: string, authToken: string): Promise<void> {
    await this.dispatchRequest([`rooms`, roomId, 'vote'], 'POST', { card }, withAuthToken(authToken));
  }

  async getRoom(roomId: string): Promise<Room> {
    const resp = await this.dispatchRequest([`rooms`, roomId], 'GET');
    return (await resp.json()) as Room;
  }

  getWebSocket(roomId: string): WebSocket {
    const wsUrl = this.baseUrl.replace(/^http/, 'ws') + `/rooms/${encodeURIComponent(roomId)}/ws`;
    return new WebSocket(wsUrl);
  }

  private async dispatchRequest(
    url: string[],
    method: 'GET' | 'POST' | 'PUT' | 'DELETE',
    body?: Record<string, unknown>,
    ...modifiers: ((r: RequestInit) => void)[]
  ): Promise<Response> {
    const options: RequestInit = {
      method,
      headers: {},
      signal: AbortSignal.timeout(10000), // 10 seconds timeout for all requests
    };

    if (body) {
      options.headers = {
        'Content-Type': 'application/json',
      };
      options.body = JSON.stringify(body);
    }

    modifiers.forEach((modifier) => modifier(options));
    const path = url.map(encodeURIComponent).join('/');

    try {
      const resp = await fetch(`${this.baseUrl}/${path}`, options);
      if (!resp.ok) {
        return await this.statusHandler(resp.status, path, options);
      }
      return resp;
    } catch (error) {
      return await this.errHandler(error, path, options);
    }
  }
}
export { RoomService };
