export type Player = {
  name: string;
  card: string;
  voted: boolean;
};

export type Room = {
  allowedCards: Array<string>;
  players: Record<string, Player>;
  revealed: boolean;
};
