export function apiBaseURL() {
  const { protocol, hostname, host } = window.location;
  if (import.meta.env.DEV) {
    return `${protocol}//${hostname}:1323/api/v1`;
  }
  return `${protocol}//${host}/api/v1`;
}
