const resp = await fetch('https://cdn.jsdelivr.net/npm/@kazvmoe-infra/unicode-emoji-json@0.4.0/annotations/en.json');

const data = await resp.json();
const emojis = [] as Array<{ symbol: string; name: string; keywords: string[] }>;
for (const [symbol, v] of Object.entries(data) as Array<[string, { name: string; keywords: string[] }]>) {
  const keywords = v.keywords.map((k) => k.toLowerCase());
  const name = v.name.toLowerCase();
  emojis.push({ symbol, keywords, name: name });
}

function matchEmoji(query: string, emoji: { symbol: string; name: string; keywords: string[] }) {
  const q = query.toLowerCase();
  return emoji.name.includes(q) || emoji.keywords.some((k) => k.includes(q));
}

export function findEmojis(query: string) {
  let q = query.toLowerCase();
  if (q.startsWith(':') && q.endsWith(':')) {
    q = q.slice(1, -1);
  }
  return emojis.filter((e) => matchEmoji(q, e));
}

export function findSingle(query: string) {
  const e = findEmojis(query);
  return e[0]?.symbol ?? null;
}
