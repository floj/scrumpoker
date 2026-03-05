export type Player = {
  name: string;
  card: string;
  voted: boolean;
};

export type Room = {
  playerId: string;
  allowedCards: Array<string>;
  players: Record<string, Player>;
  selectedCard: string;
  revealed: boolean;
};
